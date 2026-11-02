package scmportal

import (
	"github.com/layer87-labs/relctl/internal/app/relctl/scm-portal/github"
	"github.com/layer87-labs/relctl/internal/pkg/prerelease"
	"github.com/layer87-labs/relctl/internal/pkg/uploadasset"
)

func (lay SCMLayer) GetLatestReleaseVersion() (release *Release, err error) {

	grc := lay.Grc.(*github.GitHubRichClient)
	latestRelease, err := grc.GetLatestReleaseVersion()
	if err != nil {
		return nil, err
	}
	return &Release{
		TagName:     *latestRelease.TagName,
		Name:        *latestRelease.Name,
		Commit:      *latestRelease.TargetCommitish,
		CreatedAt:   &latestRelease.CreatedAt.Time,
		PublishedAt: &latestRelease.PublishedAt.Time,
	}, nil
}

// GetLatestPublishedRelease returns the latest release that is neither a
// draft nor a prerelease.
func (lay SCMLayer) GetLatestPublishedRelease() (release *Release, err error) {

	grc := lay.Grc.(*github.GitHubRichClient)
	latestRelease, err := grc.GetLatestPublishedRelease()
	if err != nil {
		return nil, err
	}
	return &Release{
		TagName:     *latestRelease.TagName,
		Name:        *latestRelease.Name,
		Commit:      *latestRelease.TargetCommitish,
		CreatedAt:   &latestRelease.CreatedAt.Time,
		PublishedAt: &latestRelease.PublishedAt.Time,
	}, nil
}

func (lay SCMLayer) CreateRelease(tagName string, releasePrefix string, branch string, body string, prerelease bool) (createdRelease *Release, err error) {

	grc := lay.Grc.(*github.GitHubRichClient)
	createdRel, err := grc.CreateRelease(tagName, releasePrefix, branch, body, prerelease)
	if err != nil {
		return nil, err
	}
	return &Release{
		ID:          *createdRel.ID,
		TagName:     *createdRel.TagName,
		Name:        *createdRel.Name,
		Commit:      *createdRel.TargetCommitish,
		CreatedAt:   &createdRel.CreatedAt.Time,
		PublishedAt: nil,
	}, nil
}

func (lay SCMLayer) PublishRelease(tagName string, releasePrefix string, branch string, body string, releaseID int64, assets []uploadasset.UploadAsset) (publishedRelease *Release, err error) {
	grc := lay.Grc.(*github.GitHubRichClient)
	_, err = grc.PublishRelease(tagName, releasePrefix, branch, body, releaseID, assets)
	if err != nil {
		return nil, err
	}
	return nil, nil

}

// TagExists reports whether the given tag exists in the repository.
func (lay SCMLayer) TagExists(tag string) (bool, error) {
	return lay.Grc.(*github.GitHubRichClient).TagExists(tag)
}

// FindReleaseByTag returns the release (draft or published) for the tag, or
// nil if there is none. The second return value reports whether it is a draft.
func (lay SCMLayer) FindReleaseByTag(tag string) (*Release, bool, error) {
	r, err := lay.Grc.(*github.GitHubRichClient).FindReleaseByTag(tag)
	if err != nil || r == nil {
		return nil, false, err
	}
	rel := &Release{ID: r.GetID(), TagName: r.GetTagName(), Name: r.GetName(), Commit: r.GetTargetCommitish()}
	if r.PublishedAt != nil {
		rel.PublishedAt = &r.PublishedAt.Time
	}
	return rel, r.GetDraft(), nil
}

// ListTagAndReleaseNames returns the names of all tags and releases (drafts
// included) that start with prefix.
func (lay SCMLayer) ListTagAndReleaseNames(prefix string) ([]string, error) {
	return lay.Grc.(*github.GitHubRichClient).ListTagAndReleaseNames(prefix)
}

// CandidateCommits maps the tag of every existing release candidate of version
// to the commit it points to. Candidates that only exist as a draft release (no
// tag yet) are not included.
func (lay SCMLayer) CandidateCommits(version string) (map[string]string, error) {
	grc := lay.Grc.(*github.GitHubRichClient)
	names, err := grc.ListTagAndReleaseNames(prerelease.Prefix(version))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, name := range names {
		sha, found, err := grc.TagCommit(name)
		if err != nil {
			return nil, err
		}
		if found {
			out[name] = sha
		}
	}
	return out, nil
}
