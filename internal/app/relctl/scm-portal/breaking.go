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
	// Mode is the breaking_markers setting: "major" (also when empty), "warn" or "off".
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

// decidePatchLevel computes the bump of a pull request in the semver scheme. The
// branch prefix gives the base level and a relctl_patch_level comment replaces it.
// A Conventional Commits breaking-change marker then raises the level to major,
// or only warns with breaking_markers: warn. A level is never lowered.
func decidePatchLevel(in levelInput) levelDecision {
	level, err := semver.ParsePatchLevelFormBranch(in.Branch)
	if err != nil {
		log.Warnln(err)
	}
	if in.Override != nil {
		level = *in.Override
	}

	if in.Mode == relctlconfig.BreakingMarkersOff {
		return levelDecision{Level: level}
	}
	marker := conventionalcommits.BreakingMarker(in.Title, in.Body, in.Commits)
	if marker == "" || level == semver.Major {
		return levelDecision{Level: level, Marker: marker}
	}
	if in.Mode == relctlconfig.BreakingMarkersWarn {
		return levelDecision{
			Level:  level,
			Marker: marker,
			Warning: fmt.Sprintf("%s marks a breaking change, but this release is a %s bump, not major "+
				"(breaking_markers: warn). Use a major/ branch prefix or the comment "+
				"\"relctl_patch_level: major\" for a breaking change.", strings.ToUpper(marker[:1])+marker[1:], level),
		}
	}
	return levelDecision{Level: semver.Major, Marker: marker}
}
