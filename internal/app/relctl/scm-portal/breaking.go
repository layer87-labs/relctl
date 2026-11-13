package scmportal

import (
	"fmt"
	"strings"

	"github.com/layer87-labs/relctl/internal/pkg/conventionalcommits"
	"github.com/layer87-labs/relctl/internal/pkg/relctlconfig"
	"github.com/layer87-labs/relctl/internal/pkg/semver"

	log "github.com/sirupsen/logrus"
)

// levelInput is everything that decides the SemVer bump of a merged pull request.
type levelInput struct {
	// Branch is the head branch; its prefix gives the base level.
	Branch string
	// Override is the level of a relctl_patch_level comment, nil if there is none.
	Override *semver.PatchLevel
	// Title, Body and Commits are the pull request title, its description and its
	// own commits (not the merge commit, not commits of the target branch).
	Title   string
	Body    string
	Commits []conventionalcommits.Commit
	// Mode is the breaking_markers setting: "major", "warn" (also when empty) or "off".
	Mode string
}

// levelDecision is the bump of a pull request and why.
type levelDecision struct {
	Level semver.PatchLevel
	// Marker says where a breaking-change marker was found; empty if none or Mode is "off".
	Marker string
	// Warning is set when a marker was found but did not make the bump major.
	Warning string
}

// botBranchPrefixes are the branches of dependency bots. Their pull request
// descriptions quote upstream release notes, so only the title and the commits
// are checked for a breaking-change marker.
var botBranchPrefixes = []string{"renovate/", "dependabot/"}

// basePatchLevel is the level of the branch prefix, replaced by a
// relctl_patch_level comment if there is one.
func basePatchLevel(branch string, override *semver.PatchLevel) semver.PatchLevel {
	if override != nil {
		return *override
	}
	level, err := semver.ParsePatchLevelFormBranch(branch)
	if err != nil {
		log.Warnln(err)
	}
	return level
}

// decidePatchLevel computes the bump of a pull request in the semver scheme. The
// branch prefix gives the base level and a relctl_patch_level comment replaces it.
// A Conventional Commits breaking-change marker then raises the level to major
// (breaking_markers: major) or only warns (warn, the default). A level is never
// lowered.
func decidePatchLevel(in levelInput) levelDecision {
	level := basePatchLevel(in.Branch, in.Override)
	if in.Mode == relctlconfig.BreakingMarkersOff {
		return levelDecision{Level: level}
	}
	body := in.Body
	for _, prefix := range botBranchPrefixes {
		if strings.HasPrefix(in.Branch, prefix) {
			body = ""
		}
	}
	marker := conventionalcommits.BreakingMarker(in.Title, body, in.Commits)
	if marker == "" || level == semver.Major {
		return levelDecision{Level: level, Marker: marker}
	}
	if in.Mode == relctlconfig.BreakingMarkersMajor {
		return levelDecision{Level: semver.Major, Marker: marker}
	}
	return levelDecision{
		Level:  level,
		Marker: marker,
		Warning: fmt.Sprintf("%s marks a breaking change, but this release is a %s bump, not major "+
			"(breaking_markers: warn). Use a major/ branch prefix or the comment "+
			"\"relctl_patch_level: major\" for a breaking change.", strings.ToUpper(marker[:1])+marker[1:], level),
	}
}

// collectCommits reads the commits of a pull request for the breaking-change
// check. If they cannot be read completely, breaking_markers: major fails (a
// breaking change must not slip through as a patch); warn continues without
// them and returns a warning.
func collectCommits(fetch func() ([]conventionalcommits.Commit, error), mode string) ([]conventionalcommits.Commit, string, error) {
	commits, err := fetch()
	if err == nil {
		return commits, "", nil
	}
	if mode == relctlconfig.BreakingMarkersMajor {
		return nil, "", fmt.Errorf("breaking_markers: major needs the commits of the pull request: %w. "+
			"The comment \"relctl_patch_level: major\" or \"relctl_version_override: <version>\" skips the check", err)
	}
	return commits, fmt.Sprintf("The commits of the pull request were not checked for a breaking-change marker: %v.", err), nil
}
