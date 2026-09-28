// Package conventionalcommits classifies commit messages according to the
// Conventional Commits specification (https://www.conventionalcommits.org/)
// and derives the SemVer bump level that a set of commits implies.
package conventionalcommits

import (
	"regexp"
	"strings"
)

// Bump is the SemVer bump level a set of commits implies.
type Bump int

const (
	// BumpNone means none of the inspected commits carried a recognised
	// Conventional Commits type.
	BumpNone Bump = iota
	// BumpPatch means the highest-ranking commit was a recognised type
	// other than "feat" (fix, perf, refactor, docs, style, test, chore,
	// ci, build, revert), with no breaking-change marker.
	BumpPatch
	// BumpMinor means the highest-ranking commit was a "feat" commit,
	// with no breaking-change marker.
	BumpMinor
	// BumpMajor means at least one commit carried a breaking-change
	// marker: "!" after the type/scope, or a "BREAKING CHANGE:" /
	// "BREAKING-CHANGE:" footer in the body.
	BumpMajor
)

// String returns a human-readable name for the bump level.
func (b Bump) String() string {
	switch b {
	case BumpPatch:
		return "patch"
	case BumpMinor:
		return "minor"
	case BumpMajor:
		return "major"
	default:
		return "none"
	}
}

// patchLevelTypes are the Conventional Commits types that, on their own
// (without a breaking-change marker), imply a patch-level bump. "feat" is
// handled separately because it implies a minor bump.
var patchLevelTypes = map[string]bool{
	"fix":      true,
	"perf":     true,
	"refactor": true,
	"docs":     true,
	"style":    true,
	"test":     true,
	"chore":    true,
	"ci":       true,
	"build":    true,
	"revert":   true,
}

// headerRe matches a Conventional Commits header:
//
//	<type>[(<scope>)][!]: <description>
//
// Capture groups: 1 = type, 2 = scope (with parens, may be empty), 3 = "!" (may be empty).
var headerRe = regexp.MustCompile(`^([a-zA-Z]+)(\([^)]*\))?(!)?:\s`)

// breakingFooterRe matches a "BREAKING CHANGE:" or "BREAKING-CHANGE:" footer
// anywhere in the commit body, on its own line, per the Conventional Commits
// spec.
var breakingFooterRe = regexp.MustCompile(`(?m)^BREAKING[ -]CHANGE:`)

// ClassifyMessage returns the SemVer bump level a single commit message
// implies. msg is the full commit message (header + optional body/footers).
//
// A message that happens to start with "Merge pull request #123 from
// org/branch" does not match a Conventional Commits header and is therefore
// classified as BumpNone. In practice CommitMessagesSince never hands this
// function an actual merge commit's message: it refuses to walk past a
// commit with more than one parent (ErrMergeCommitInRange) before it gets
// here, because relctl's conventional-commits scheme only understands
// squash-merged commits, where the PR title/commit subject itself carries
// the type.
func ClassifyMessage(msg string) Bump {
	header, _, _ := strings.Cut(msg, "\n")
	header = strings.TrimSpace(header)

	m := headerRe.FindStringSubmatch(header)
	if m == nil {
		return BumpNone
	}

	commitType := strings.ToLower(m[1])
	hasBang := m[3] == "!"

	bump := BumpNone
	switch {
	case commitType == "feat":
		bump = BumpMinor
	case patchLevelTypes[commitType]:
		bump = BumpPatch
	default:
		// Not a recognised Conventional Commits type.
		return BumpNone
	}

	if hasBang || breakingFooterRe.MatchString(msg) {
		return BumpMajor
	}
	return bump
}

// Classify returns the highest bump level implied by any of the given
// commit messages. It returns BumpNone if messages is empty or none of the
// messages carry a recognised Conventional Commits type.
func Classify(messages []string) Bump {
	result := BumpNone
	for _, msg := range messages {
		if b := ClassifyMessage(msg); b > result {
			result = b
		}
	}
	return result
}
