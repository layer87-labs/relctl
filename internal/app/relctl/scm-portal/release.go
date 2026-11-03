package scmportal

import (
	"strings"

	"github.com/layer87-labs/relctl/internal/app/relctl/scm-portal/github"
	"github.com/layer87-labs/relctl/internal/pkg/prerelease"
	"github.com/layer87-labs/relctl/internal/pkg/uploadasset"
)

// All methods take and return versions. The layer adds and removes TagPrefix, so a
// repository that tags "v1.2.3" or a component that tags "whisper/2026.11.1" works
// with the same code as a repository without a prefix.

// releaseOf converts a GitHub release; its TagName stays the full tag.
func (lay SCMLayer) releaseOf(id int64, tag, name, commit string) *Release {
	return &Release{ID: id, Version: lay.version(tag), TagName: tag, Name: name, Commit: commit}
}

func (lay SCMLayer) GetLatestReleaseVersion() (release *Release, err error) {
	grc := lay.Grc.(*github.GitHubRichClient)
	latestRelease, err := grc.GetLatestReleaseVersion(lay.TagPrefix)
	if err != nil {
		return nil, err
	}
	rel := lay.releaseOf(0, *latestRelease.TagName, *latestRelease.Name, *latestRelease.TargetCommitish)
	rel.CreatedAt = &latestRelease.CreatedAt.Time
	rel.PublishedAt = &latestRelease.PublishedAt.Time
	return rel, nil
}

// GetLatestPublishedRelease returns the latest release that is neither a
// draft nor a prerelease.
func (lay SCMLayer) GetLatestPublishedRelease() (release *Release, err error) {
	grc := lay.Grc.(*github.GitHubRichClient)
	latestRelease, err := grc.GetLatestPublishedRelease(lay.TagPrefix)
	if err != nil {
		return nil, err
	}
	rel := lay.releaseOf(0, *latestRelease.TagName, *latestRelease.Name, *latestRelease.TargetCommitish)
	rel.CreatedAt = &latestRelease.CreatedAt.Time
	rel.PublishedAt = &latestRelease.PublishedAt.Time
	return rel, nil
}

func (lay SCMLayer) CreateRelease(version string, releasePrefix string, branch string, body string, prerelease bool) (createdRelease *Release, err error) {
	grc := lay.Grc.(*github.GitHubRichClient)
	createdRel, err := grc.CreateRelease(lay.tag(version), releasePrefix, branch, body, prerelease)
	if err != nil {
		return nil, err
	}
	rel := lay.releaseOf(*createdRel.ID, *createdRel.TagName, *createdRel.Name, *createdRel.TargetCommitish)
	rel.CreatedAt = &createdRel.CreatedAt.Time
	return rel, nil
}

func (lay SCMLayer) PublishRelease(version string, releasePrefix string, branch string, body string, releaseID int64, assets []uploadasset.UploadAsset) (publishedRelease *Release, err error) {
	grc := lay.Grc.(*github.GitHubRichClient)
	_, err = grc.PublishRelease(lay.tag(version), releasePrefix, branch, body, releaseID, assets)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// TagExists reports whether the tag of the version exists in the repository.
func (lay SCMLayer) TagExists(version string) (bool, error) {
	return lay.Grc.(*github.GitHubRichClient).TagExists(lay.tag(version))
}

// FindReleaseByTag returns the release (draft or published) of the version, or
// nil if there is none. The second return value reports whether it is a draft.
func (lay SCMLayer) FindReleaseByTag(version string) (*Release, bool, error) {
	r, err := lay.Grc.(*github.GitHubRichClient).FindReleaseByTag(lay.tag(version))
	if err != nil || r == nil {
		return nil, false, err
	}
	rel := lay.releaseOf(r.GetID(), r.GetTagName(), r.GetName(), r.GetTargetCommitish())
	if r.PublishedAt != nil {
		rel.PublishedAt = &r.PublishedAt.Time
	}
	return rel, r.GetDraft(), nil
}

// ListTagAndReleaseNames returns the versions of all tags and releases (drafts
// included) that start with versionPrefix.
func (lay SCMLayer) ListTagAndReleaseNames(versionPrefix string) ([]string, error) {
	names, err := lay.Grc.(*github.GitHubRichClient).ListTagAndReleaseNames(lay.tag(versionPrefix))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if strings.HasPrefix(n, lay.TagPrefix) {
			out = append(out, lay.version(n))
		}
	}
	return out, nil
}

// CandidateCommits maps the version of every existing release candidate of version
// to the commit its tag points to. Candidates that only exist as a draft release
// (no tag yet) are not included.
func (lay SCMLayer) CandidateCommits(version string) (map[string]string, error) {
	grc := lay.Grc.(*github.GitHubRichClient)
	names, err := lay.ListTagAndReleaseNames(prerelease.Prefix(version))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, name := range names {
		sha, found, err := grc.TagCommit(lay.tag(name))
		if err != nil {
			return nil, err
		}
		if found {
			out[name] = sha
		}
	}
	return out, nil
}

// RepositoryDefaultBranch returns the default branch of the repository.
func (lay SCMLayer) RepositoryDefaultBranch() (string, error) {
	return lay.Grc.(*github.GitHubRichClient).RepositoryDefaultBranch()
}

// BranchSHA returns the commit a branch points to.
func (lay SCMLayer) BranchSHA(branch string) (string, error) {
	return lay.Grc.(*github.GitHubRichClient).BranchSHA(branch)
}

// CompareStatus compares head with base ("identical", "ahead", "behind", "diverged").
func (lay SCMLayer) CompareStatus(base, head string) (string, error) {
	return lay.Grc.(*github.GitHubRichClient).CompareStatus(base, head)
}

// FastForwardBranch moves a branch to sha without force.
func (lay SCMLayer) FastForwardBranch(branch, sha string) error {
	return lay.Grc.(*github.GitHubRichClient).FastForwardBranch(branch, sha)
}

// FindOpenPullRequest returns the number of an open pull request from head into base, or 0.
func (lay SCMLayer) FindOpenPullRequest(head, base string) (int, error) {
	return lay.Grc.(*github.GitHubRichClient).FindOpenPullRequest(head, base)
}

// CreatePullRequest opens a pull request from head into base.
func (lay SCMLayer) CreatePullRequest(head, base, title, body string) (int, error) {
	return lay.Grc.(*github.GitHubRichClient).CreatePullRequest(head, base, title, body)
}

// GenerateReleaseNotes returns the notes GitHub generates for a release of the version at
// target. The notes start after the latest published release (never a release candidate),
// so a candidate and the final release both list everything since the last real release.
func (lay SCMLayer) GenerateReleaseNotes(version, target string) (string, error) {
	grc := lay.Grc.(*github.GitHubRichClient)
	previous := ""
	if latest, err := grc.GetLatestPublishedRelease(lay.TagPrefix); err == nil && latest != nil && latest.TagName != nil {
		previous = *latest.TagName
	}
	return grc.GenerateReleaseNotes(lay.tag(version), target, previous)
}
