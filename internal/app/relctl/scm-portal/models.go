package scmportal

import (
	"strings"
	"time"

	"github.com/layer87-labs/relctl/internal/app/relctl/ces"
	"github.com/layer87-labs/relctl/internal/pkg/semver"
)

type SCMLayer struct {
	Grc interface{}
	CES ces.CES
	// TagPrefix is put in front of every tag and release name this layer reads or
	// writes. Callers work with versions; the layer maps them to tags. It is set
	// for a repository that tags "v1.2.3" and for every component ("whisper/...").
	TagPrefix string
}

// tag returns the tag of a version.
func (lay SCMLayer) tag(version string) string { return lay.TagPrefix + version }

// version returns the version of a tag, without the tag prefix.
func (lay SCMLayer) version(tag string) string { return strings.TrimPrefix(tag, lay.TagPrefix) }

// BaseVersion is the version of the release without the tag prefix. A release that
// was built without a layer (Version empty) falls back to the tag name.
func (r Release) BaseVersion() string {
	if r.Version != "" {
		return r.Version
	}
	return r.TagName
}

type PrMrRequestInfos struct {
	Number         int
	Owner          string
	Repo           string
	PatchLevel     semver.PatchLevel
	LatestVersion  string
	NextVersion    string
	Sha            string
	ShaShort       string
	BranchName     string
	MergeCommitSha string
}

type Release struct {
	ID          int64      `json:"id"`           // GitHub: ID, GitLab: not available!!
	Version     string     `json:"version"`      // the tag without the tag prefix
	TagName     string     `json:"tag_name"`     // the full tag name, prefix included
	Name        string     `json:"name"`         // equaly
	Commit      string     `json:"commit"`       // GitHub: TargetCommitish; GitLab: Commit
	CreatedAt   *time.Time `json:"created_at"`   // equaly
	PublishedAt *time.Time `json:"published_at"` // GitHub: PublishedAt; GitLab: ReleasedAt
}
