package service

import (
	"errors"
	"fmt"

	scmportal "github.com/layer87-labs/relctl/internal/app/relctl/scm-portal"
	"github.com/layer87-labs/relctl/internal/pkg/conventionalcommits"
	"github.com/layer87-labs/relctl/internal/pkg/semver"
)

// ErrNoPreviousRelease is returned when the conventional-commits scheme
// cannot find any previously published release to bump from.
var ErrNoPreviousRelease = errors.New("conventional-commits: no previous published release found")

// ErrNoRelevantCommits is returned when none of the commits since the last
// published release carry a recognised Conventional Commits type.
var ErrNoRelevantCommits = errors.New("conventional-commits: no relevant commits since last published release")

// latestPublishedReleaseGetter is satisfied by *scmportal.SCMLayer. It is
// declared here so tests can supply a fake instead of a real SCM layer.
type latestPublishedReleaseGetter interface {
	GetLatestPublishedRelease() (*scmportal.Release, error)
}

// commitMessagesSinceFunc matches conventionalcommits.CommitMessagesSince.
// It is injected so tests can supply a fake instead of reading a real
// on-disk Git repository.
type commitMessagesSinceFunc func(repoPath, sinceTag string) ([]string, error)

// conventionalCommitsVersion computes the next version for the
// conventional-commits scheme:
//  1. find the latest published (non-draft, non-prerelease) release,
//  2. collect the commit messages between that release's tag and HEAD,
//  3. classify them per Conventional Commits and bump the tag accordingly.
func conventionalCommitsVersion(rl latestPublishedReleaseGetter, commitMessagesSince commitMessagesSinceFunc, repoPath string) (string, error) {
	latest, err := rl.GetLatestPublishedRelease()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNoPreviousRelease, err)
	}
	if latest == nil || latest.TagName == "" {
		return "", ErrNoPreviousRelease
	}

	messages, err := commitMessagesSince(repoPath, latest.TagName)
	if err != nil {
		return "", err
	}

	bump := conventionalcommits.Classify(messages)
	if bump == conventionalcommits.BumpNone {
		return "", fmt.Errorf("%w (%s..HEAD)", ErrNoRelevantCommits, latest.TagName)
	}

	patchLevel, err := bumpToPatchLevel(bump)
	if err != nil {
		return "", err
	}

	return semver.IncreaseVersion(patchLevel, latest.TagName)
}

// bumpToPatchLevel maps a conventionalcommits.Bump to the semver.PatchLevel
// relctl's existing SemVer machinery understands. relctl does not gate a
// major bump behind any repository setting today (see
// semver.ParsePatchLevel and ReleaseArgs) — the conventional-commits scheme
// deliberately mirrors that: a "!" or BREAKING CHANGE footer always produces
// a major bump, with no separate opt-in.
func bumpToPatchLevel(b conventionalcommits.Bump) (semver.PatchLevel, error) {
	switch b {
	case conventionalcommits.BumpPatch:
		return semver.Bugfix, nil
	case conventionalcommits.BumpMinor:
		return semver.Feature, nil
	case conventionalcommits.BumpMajor:
		return semver.Major, nil
	default:
		return "", fmt.Errorf("conventional-commits: unexpected bump level %v", b)
	}
}
