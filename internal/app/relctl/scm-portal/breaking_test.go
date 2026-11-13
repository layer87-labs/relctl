package scmportal

import (
	"testing"

	"github.com/layer87-labs/relctl/internal/pkg/conventionalcommits"
	"github.com/layer87-labs/relctl/internal/pkg/semver"
)

func TestDecidePatchLevel(t *testing.T) {
	minor := semver.Feature
	commit := func(msg string) []conventionalcommits.Commit {
		return []conventionalcommits.Commit{{SHA: "0123456789abcdef", Message: msg}}
	}
	cases := []struct {
		name       string
		in         levelInput
		wantLevel  semver.PatchLevel
		wantMarker string
		wantWarn   bool
	}{
		{name: "fix branch, breaking title with scope",
			in:        levelInput{Branch: "fix/x", Title: "fix(forms)!: y"},
			wantLevel: semver.Major, wantMarker: "the pull request title"},
		{name: "feat branch, breaking footer in a commit body",
			in:        levelInput{Branch: "feat/x", Title: "feat: y", Commits: commit("feat: y\n\nBREAKING CHANGE: z")},
			wantLevel: semver.Major, wantMarker: "commit 01234567"},
		{name: "fix branch, BREAKING-CHANGE footer in the description",
			in:        levelInput{Branch: "fix/x", Title: "fix: y", Body: "Some text.\n\nBREAKING-CHANGE: z"},
			wantLevel: semver.Major, wantMarker: "the pull request description"},
		{name: "fix branch, breaking commit subject",
			in:        levelInput{Branch: "fix/x", Title: "fix: y", Commits: commit("refactor!: drop the old flag")},
			wantLevel: semver.Major, wantMarker: "commit 01234567"},
		{name: "fix branch without markers",
			in:        levelInput{Branch: "fix/x", Title: "fix: y", Body: "plain", Commits: commit("fix: y")},
			wantLevel: semver.Bugfix},
		{name: "feat branch without markers",
			in:        levelInput{Branch: "feat/x", Title: "feat: y"},
			wantLevel: semver.Feature},
		{name: "major branch",
			in:        levelInput{Branch: "major/x", Title: "chore: y"},
			wantLevel: semver.Major},
		{name: "breaking marker only in a commit that is not part of the pull request",
			// The commits are the pull request's own (PullRequestCommits); a marker in
			// the merge commit or in the target branch never reaches this input.
			in:        levelInput{Branch: "fix/x", Title: "fix: y", Commits: commit("fix: y")},
			wantLevel: semver.Bugfix},
		{name: "exclamation mark in the description of the title",
			in:        levelInput{Branch: "fix/x", Title: "fix: handle foo!", Commits: commit("fix: handle foo!\n\nreally!: yes")},
			wantLevel: semver.Bugfix},
		{name: "BREAKING CHANGE mid-sentence",
			in:        levelInput{Branch: "fix/x", Title: "fix: y", Body: "This is no BREAKING CHANGE: it only fixes a typo."},
			wantLevel: semver.Bugfix},
		{name: "BREAKING CHANGE inside a fenced code block",
			in:        levelInput{Branch: "fix/x", Title: "fix: y", Body: "Example commit:\n\n```\nfeat: x\n\nBREAKING CHANGE: z\n```\n"},
			wantLevel: semver.Bugfix},
		{name: "comment override is raised, never lowered",
			in:        levelInput{Branch: "fix/x", Override: &minor, Title: "fix!: y"},
			wantLevel: semver.Major, wantMarker: "the pull request title"},
		{name: "comment override without a marker",
			in:        levelInput{Branch: "fix/x", Override: &minor, Title: "fix: y"},
			wantLevel: semver.Feature},
		{name: "warn keeps the level and warns",
			in:        levelInput{Branch: "fix/x", Title: "fix(forms)!: y", Mode: "warn"},
			wantLevel: semver.Bugfix, wantMarker: "the pull request title", wantWarn: true},
		{name: "warn on a major branch does not warn",
			in:        levelInput{Branch: "major/x", Title: "fix!: y", Mode: "warn"},
			wantLevel: semver.Major, wantMarker: "the pull request title"},
		{name: "off ignores the marker",
			in:        levelInput{Branch: "fix/x", Title: "fix(forms)!: y", Body: "BREAKING CHANGE: z", Mode: "off"},
			wantLevel: semver.Bugfix},
		{name: "explicit major mode",
			in:        levelInput{Branch: "feat/x", Title: "feat!: y", Mode: "major"},
			wantLevel: semver.Major, wantMarker: "the pull request title"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decidePatchLevel(tc.in)
			if got.Level != tc.wantLevel {
				t.Errorf("level = %q, want %q", got.Level, tc.wantLevel)
			}
			if got.Marker != tc.wantMarker {
				t.Errorf("marker = %q, want %q", got.Marker, tc.wantMarker)
			}
			if (got.Warning != "") != tc.wantWarn {
				t.Errorf("warning = %q, want a warning: %v", got.Warning, tc.wantWarn)
			}
		})
	}
}

func TestDecidePatchLevelWarningText(t *testing.T) {
	got := decidePatchLevel(levelInput{Branch: "fix/x", Title: "fix(forms)!: y", Mode: "warn"}).Warning
	want := `The pull request title marks a breaking change, but this release is a bugfix bump, not major ` +
		`(breaking_markers: warn). Use a major/ branch prefix or the comment "relctl_patch_level: major" for a breaking change.`
	if got != want {
		t.Errorf("warning =\n%s\nwant\n%s", got, want)
	}
}
