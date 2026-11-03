package github

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/layer87-labs/relctl/internal/pkg/tools"
	"github.com/layer87-labs/relctl/internal/pkg/uploadasset"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v86/github"
	"github.com/sirupsen/logrus"
)

// CreateRelease creates a new release in the GitHub repository.
//
// Parameters:
// - tagName: the name of the tag for the release.
// - releasePrefix: the prefix for the release name.
// - releaseBranch: the branch for the release (default: git default).
// - body: the body of the release (markdown string or file).
//
// Returns:
// - createdRelease: the created repository release.
// - err: an error if there was a problem creating the release.
func (ghrc *GitHubRichClient) CreateRelease(tagName string, releasePrefix string, releaseBranch string, body string, prerelease bool) (createdRelease *github.RepositoryRelease, err error) {
	draft := true
	relName := fmt.Sprintf("%s %s", releasePrefix, tagName)
	if releaseBranch == "" {
		releaseBranch = tools.GetDefaultBranch()
	}

	releaseObject := github.RepositoryRelease{
		TargetCommitish: &releaseBranch,
		TagName:         &tagName,
		Name:            &relName,
		Draft:           &draft,
		Prerelease:      &prerelease,
		Body:            &body,
	}
	createdRelease, _, err = ghrc.Client.Repositories.CreateRelease(
		ctx,
		ghrc.Owner,
		ghrc.Repository,
		&releaseObject)
	if err != nil {
		err = fmt.Errorf("error at creating github release: %v", err)
		return
	}
	return
}

// PublishRelease publishes a GitHub release.
//
// Parameters:
// - tagName: the name of the tag for the release.
// - releasePrefix: the prefix for the release name.
// - releaseBranch: the branch for the release.
// - body: the body of the release.
// - releaseID: the ID of the existing release (0 if not found).
// - uploadArtifacts: the list of artifacts to upload.
//
// Returns:
// - releaseAssets: the list of release assets.
// - err: an error if any occurred.
func (ghrc *GitHubRichClient) PublishRelease(
	tagName string,
	releasePrefix string,
	releaseBranch string,
	body string,
	releaseID int64,
	uploadArtifacts []uploadasset.UploadAsset) (releaseAssets []*github.ReleaseAsset, err error) {

	if releaseID == 0 {
		logrus.Infoln("no release found, creating one...")
		release, err := ghrc.CreateRelease(tagName, releasePrefix, releaseBranch, body, false)
		if err != nil {
			logrus.Fatalln(err)
		}
		releaseID = *release.ID
	}

	existingRelease, _, err := ghrc.Client.Repositories.GetRelease(
		ctx,
		ghrc.Owner,
		ghrc.Repository,
		releaseID)
	if err != nil {
		return
	}

	// upload any given artifacts
	var uploaded []assetRow
	for _, asset := range uploadArtifacts {
		logrus.Infof("uploading %s as asset to release\n", asset.Name)
		file, err := os.Open(asset.FilePath)
		if err != nil {
			return releaseAssets, err
		}
		// Upload assets to GitHub Release
		relAsset, _, err := ghrc.Client.Repositories.UploadReleaseAsset(
			ctx,
			ghrc.Owner,
			ghrc.Repository,
			releaseID,
			&github.UploadOptions{
				Name: asset.Name,
			},
			file)
		_ = file.Close()
		if err != nil {
			logrus.Warnln("error at uploading asset to release: ", err)
			continue
		}
		uploaded = append(uploaded, assetRow{
			Name: asset.Name,
			URL:  relAsset.GetBrowserDownloadURL(),
			Size: int64(relAsset.GetSize()),
			Hash: asset.Hash,
		})
		releaseAssets = append(releaseAssets, relAsset)
	}

	// the text of an existing draft wins over the given body
	base := existingRelease.GetBody()
	if base == "" {
		base = body
	}
	newReleaseBody := composeReleaseBody(base, uploaded)
	existingRelease.Body = &newReleaseBody

	// publishing release
	*existingRelease.Draft = false
	_, _, err = ghrc.Client.Repositories.EditRelease(
		ctx,
		ghrc.Owner,
		ghrc.Repository,
		releaseID,
		existingRelease)
	if err != nil {
		return releaseAssets, err
	}

	// A draft's download links point to a temporary "untagged-…" address that stops working
	// once the tag exists. Now that the release is published, link to the final addresses.
	if len(uploaded) > 0 {
		if err := ghrc.relinkAssets(releaseID, base, uploaded); err != nil {
			logrus.Warnf("could not replace the asset links in the release text: %v", err)
		}
	}

	return
}

// relinkAssets rewrites the asset table of a published release with the final download links.
func (ghrc *GitHubRichClient) relinkAssets(releaseID int64, text string, uploaded []assetRow) error {
	published, _, err := ghrc.Client.Repositories.GetRelease(ctx, ghrc.Owner, ghrc.Repository, releaseID)
	if err != nil {
		return err
	}
	urls := make(map[string]string, len(published.Assets))
	for _, a := range published.Assets {
		urls[a.GetName()] = a.GetBrowserDownloadURL()
	}
	rows := make([]assetRow, len(uploaded))
	for i, row := range uploaded {
		rows[i] = row
		if u, ok := urls[row.Name]; ok && u != "" {
			rows[i].URL = u
		}
	}
	body := composeReleaseBody(text, rows)
	published.Body = &body
	_, _, err = ghrc.Client.Repositories.EditRelease(ctx, ghrc.Owner, ghrc.Repository, releaseID, published)
	return err
}

// GetLatestReleaseVersion retrieves the latest release version from the GitHub repository.
//
// It returns the latest non-draft release and any error encountered during the process.
// The latest release is mapped to its tag name in the releaseMap.
// The function queries the releases in the repository in pages of 100, starting from page 1.
// For each release, it checks if it is not a draft and adds it to the releaseMap.
// The function continues querying releases until there are no more pages.
// Finally, it calls the findLatestRelease function to find the latest release based on the releaseMap.
//
// Parameters:
// - ghrc: a pointer to the GitHubRichClient struct.
//
// Returns:
// - latestRelease: a pointer to the latest non-draft release.
// - err: an error if any occurred during the process.
func (ghrc *GitHubRichClient) GetLatestReleaseVersion(tagPrefix string) (latestRelease *github.RepositoryRelease, err error) {
	return ghrc.getLatestRelease(false, tagPrefix)
}

// GetLatestPublishedRelease retrieves the latest *published* release from
// the GitHub repository: not a draft, and not marked as a prerelease.
//
// It exists alongside GetLatestReleaseVersion (which only excludes drafts)
// because some callers — e.g. the conventional-commits version scheme —
// need a strictly "this is what users are running" reference point, where
// a prerelease would give a misleading base version to bump from.
func (ghrc *GitHubRichClient) GetLatestPublishedRelease(tagPrefix string) (latestRelease *github.RepositoryRelease, err error) {
	return ghrc.getLatestRelease(true, tagPrefix)
}

// getLatestRelease retrieves the latest release, always excluding drafts and
// additionally excluding prereleases when excludePrerelease is true.
func (ghrc *GitHubRichClient) getLatestRelease(excludePrerelease bool, tagPrefix string) (latestRelease *github.RepositoryRelease, err error) {
	var releaseMap = make(map[string]*github.RepositoryRelease)
	var loadReleasesFromRepo func(page int)

	loadReleasesFromRepo = func(page int) {
		logrus.Tracef("querying release page %d", page)
		releases, response, err := ghrc.Client.Repositories.ListReleases(ctx, ghrc.Owner, ghrc.Repository, &github.ListOptions{
			PerPage: 100,
			Page:    page,
		})
		if err != nil {
			logrus.Fatalf("error at loading repos from: %v", err)
		}

		logrus.Tracef("found %d releases at page %d begin with mapping...", len(releases), page)
		for _, release := range releases {
			if *release.Draft {
				logrus.Infof("ignoring draft release %s", *release.Name)
				continue
			}
			if excludePrerelease && release.Prerelease != nil && *release.Prerelease {
				logrus.Infof("ignoring prerelease %s", *release.Name)
				continue
			}
			releaseMap[*release.TagName] = release
		}

		logrus.Traceln("####### next page:", response.NextPage)
		if response.NextPage == 0 {
			logrus.Traceln("ended with paging through releases")
			return
		} else {
			loadReleasesFromRepo(page + 1)
		}
	}

	// starting recusive function
	loadReleasesFromRepo(1)

	return ghrc.findLatestRelease(`.`, releaseMap, tagPrefix)
}

// findLatestRelease finds the latest release in the given directory based on the tags in the local git repository.
//
// Parameters:
// - directory: the directory of the local git repository.
// - githubReleaseMap: a map of tags to github.RepositoryRelease objects.
//
// Returns:
// - latestRelease: the latest release found in the directory, or nil if no release is found.
// - err: an error if any occurred during the process.
func (ghrc *GitHubRichClient) findLatestRelease(directory string, githubReleaseMap map[string]*github.RepositoryRelease, tagPrefix string) (latestRelease *github.RepositoryRelease, err error) {
	logrus.Traceln("open local git repository...")
	gitRepo, err := git.PlainOpen(directory)
	if err != nil {
		return nil, err
	}
	logrus.Traceln("...opened local git repository")

	tags, err := tools.GetGitTagsUpToHeadWithPrefix(gitRepo, tagPrefix)

	logrus.Traceln(tags)
	if err != nil {
		return nil, err
	}

	for i := len(tags) - 1; i >= 0; i-- {
		if latestRelease, found := githubReleaseMap[tags[i].Tag]; found {
			return latestRelease, nil
		}
	}

	return nil, errors.New("could not find latest release")
}

// TagExists reports whether the given tag exists in the repository.
func (ghrc *GitHubRichClient) TagExists(tag string) (bool, error) {
	_, resp, err := ghrc.Client.Git.GetRef(ctx, ghrc.Owner, ghrc.Repository, "tags/"+tag)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false, nil
		}
		return false, fmt.Errorf("error at checking tag %q: %v", tag, err)
	}
	return true, nil
}

// FindReleaseByTag returns the release (draft or published) for the given tag
// name, or nil if there is none.
func (ghrc *GitHubRichClient) FindReleaseByTag(tag string) (*github.RepositoryRelease, error) {
	for page := 1; page != 0; {
		releases, resp, err := ghrc.Client.Repositories.ListReleases(ctx, ghrc.Owner, ghrc.Repository, &github.ListOptions{PerPage: 100, Page: page})
		if err != nil {
			return nil, fmt.Errorf("error at listing releases: %v", err)
		}
		for _, r := range releases {
			if r.GetTagName() == tag {
				return r, nil
			}
		}
		page = resp.NextPage
	}
	return nil, nil
}

// ListTagAndReleaseNames returns the names of all tags and of all releases
// (drafts included) that start with prefix. Release candidates exist as a tag
// once published and only as a release while they are still drafts, so both
// are needed to avoid reusing a number.
func (ghrc *GitHubRichClient) ListTagAndReleaseNames(prefix string) ([]string, error) {
	names := map[string]bool{}

	refs, _, err := ghrc.Client.Git.ListMatchingRefs(ctx, ghrc.Owner, ghrc.Repository, "tags/"+prefix)
	if err != nil {
		return nil, fmt.Errorf("error at listing tags: %v", err)
	}
	for _, r := range refs {
		names[strings.TrimPrefix(r.GetRef(), "refs/tags/")] = true
	}

	for page := 1; page != 0; {
		releases, resp, err := ghrc.Client.Repositories.ListReleases(ctx, ghrc.Owner, ghrc.Repository, &github.ListOptions{PerPage: 100, Page: page})
		if err != nil {
			return nil, fmt.Errorf("error at listing releases: %v", err)
		}
		for _, r := range releases {
			if strings.HasPrefix(r.GetTagName(), prefix) {
				names[r.GetTagName()] = true
			}
		}
		page = resp.NextPage
	}

	out := make([]string, 0, len(names))
	for n := range names {
		out = append(out, n)
	}
	return out, nil
}

// TagCommit returns the commit a tag points to (annotated tags are resolved),
// and false if the tag does not exist.
func (ghrc *GitHubRichClient) TagCommit(tag string) (string, bool, error) {
	ref, resp, err := ghrc.Client.Git.GetRef(ctx, ghrc.Owner, ghrc.Repository, "tags/"+tag)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return "", false, nil
		}
		return "", false, fmt.Errorf("error at reading tag %q: %v", tag, err)
	}
	obj := ref.GetObject()
	if obj.GetType() == "tag" {
		t, _, err := ghrc.Client.Git.GetTag(ctx, ghrc.Owner, ghrc.Repository, obj.GetSHA())
		if err != nil {
			return "", false, fmt.Errorf("error at reading annotated tag %q: %v", tag, err)
		}
		return t.GetObject().GetSHA(), true, nil
	}
	return obj.GetSHA(), true, nil
}

// GenerateReleaseNotes asks GitHub for the notes of a release that does not exist yet.
// GitHub lists the merged pull requests since previousTag and applies the categories of
// .github/release.yml, if the repository has one. An empty previousTag lets GitHub choose.
func (ghrc *GitHubRichClient) GenerateReleaseNotes(tag, target, previousTag string) (string, error) {
	opts := &github.GenerateNotesOptions{TagName: tag}
	if target != "" {
		opts.TargetCommitish = &target
	}
	if previousTag != "" {
		opts.PreviousTagName = &previousTag
	}
	notes, _, err := ghrc.Client.Repositories.GenerateReleaseNotes(ctx, ghrc.Owner, ghrc.Repository, opts)
	if err != nil {
		return "", fmt.Errorf("error at generating release notes: %v", err)
	}
	return strings.TrimSpace(notes.Body), nil
}

// assetRow is one uploaded file of a release.
type assetRow struct {
	Name string
	URL  string
	Size int64
	Hash []byte
}

// composeReleaseBody joins the release text and the table of uploaded assets.
func composeReleaseBody(text string, assets []assetRow) string {
	text = strings.TrimSpace(text)
	table := renderAssetsTable(assets)
	switch {
	case table == "":
		return text
	case text == "":
		return table
	default:
		return text + "\n\n" + table
	}
}

// renderAssetsTable renders the uploaded files as a markdown table.
func renderAssetsTable(assets []assetRow) string {
	if len(assets) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("### Assets\n\n| File | Size | SHA-256 |\n| --- | ---: | --- |\n")
	for _, a := range assets {
		fmt.Fprintf(&b, "| [%s](%s) | %s | `%x` |\n", a.Name, a.URL, humanSize(a.Size), a.Hash)
	}
	return strings.TrimRight(b.String(), "\n")
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGT"[exp])
}
