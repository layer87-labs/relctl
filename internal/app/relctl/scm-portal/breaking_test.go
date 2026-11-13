package scmportal

import (
	"errors"
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
		// breaking_markers: major
		{name: "major: fix branch, breaking title with scope",
			in:        levelInput{Mode: "major", Branch: "fix/x", Title: "fix(forms)!: y"},
			wantLevel: semver.Major, wantMarker: "the pull request title"},
		{name: "major: feat branch, breaking footer in a commit body",
			in:        levelInput{Mode: "major", Branch: "feat/x", Title: "feat: y", Commits: commit("feat: y\n\nBREAKING CHANGE: z")},
			wantLevel: semver.Major, wantMarker: "commit 01234567"},
		{name: "major: BREAKING-CHANGE footer in the description",
			in:        levelInput{Mode: "major", Branch: "fix/x", Title: "fix: y", Body: "Some text.\n\nBREAKING-CHANGE: z"},
			wantLevel: semver.Major, wantMarker: "the pull request description"},
		{name: "major: breaking commit subject",
			in:        levelInput{Mode: "major", Branch: "fix/x", Title: "fix: y", Commits: commit("refactor!: drop the old flag")},
			wantLevel: semver.Major, wantMarker: "commit 01234567"},
		{name: "major: fix branch without markers",
			in:        levelInput{Mode: "major", Branch: "fix/x", Title: "fix: y", Body: "plain", Commits: commit("fix: y")},
			wantLevel: semver.Bugfix},
		{name: "major: feat branch without markers",
			in:        levelInput{Mode: "major", Branch: "feat/x", Title: "feat: y"},
			wantLevel: semver.Feature},
		{name: "major branch",
			in:        levelInput{Mode: "major", Branch: "major/x", Title: "chore: y"},
			wantLevel: semver.Major},
		{name: "major: marker only in a commit that is not part of the pull request",
			// The commits are the pull request's own (PullRequestCommits); a marker in
			// the merge commit or in the target branch never reaches this input.
			in:        levelInput{Mode: "major", Branch: "fix/x", Title: "fix: y", Commits: commit("fix: y")},
			wantLevel: semver.Bugfix},
		{name: "major: exclamation mark in the description of the title",
			in:        levelInput{Mode: "major", Branch: "fix/x", Title: "fix: handle foo!", Commits: commit("fix: handle foo!\n\nreally!: yes")},
			wantLevel: semver.Bugfix},
		{name: "major: BREAKING CHANGE mid-sentence",
			in:        levelInput{Mode: "major", Branch: "fix/x", Title: "fix: y", Body: "This is no BREAKING CHANGE: it only fixes a typo."},
			wantLevel: semver.Bugfix},
		{name: "major: BREAKING CHANGE inside a fenced code block",
			in:        levelInput{Mode: "major", Branch: "fix/x", Title: "fix: y", Body: "Example commit:\n\n```\nfeat: x\n\nBREAKING CHANGE: z\n```\n"},
			wantLevel: semver.Bugfix},
		{name: "major: comment override is raised, never lowered",
			in:        levelInput{Mode: "major", Branch: "fix/x", Override: &minor, Title: "fix!: y"},
			wantLevel: semver.Major, wantMarker: "the pull request title"},
		{name: "major: comment override without a marker",
			in:        levelInput{Mode: "major", Branch: "fix/x", Override: &minor, Title: "fix: y"},
			wantLevel: semver.Feature},
		{name: "major: renovate description is ignored",
			in:        levelInput{Mode: "major", Branch: "renovate/lib-2.x", Title: "fix(deps): update lib to v2", Body: "Release notes\n\nBREAKING CHANGE: lib drops Go 1.20"},
			wantLevel: semver.Bugfix},
		{name: "major: dependabot description is ignored",
			in:        levelInput{Mode: "major", Branch: "dependabot/go_modules/lib-2.0.0", Title: "fix(deps): bump lib", Body: "BREAKING-CHANGE: upstream"},
			wantLevel: semver.Bugfix},
		{name: "major: bot title still counts",
			in:        levelInput{Mode: "major", Branch: "renovate/lib-2.x", Title: "fix(deps)!: update lib to v2", Body: "BREAKING CHANGE: z"},
			wantLevel: semver.Major, wantMarker: "the pull request title"},
		{name: "major: bot commits still count",
			in:        levelInput{Mode: "major", Branch: "dependabot/npm/x", Title: "fix(deps): bump x", Commits: commit("fix(deps): bump x\n\nBREAKING CHANGE: z")},
			wantLevel: semver.Major, wantMarker: "commit 01234567"},
		// breaking_markers: warn (default)
		{name: "default is warn: keeps the level and warns",
			in:        levelInput{Branch: "fix/x", Title: "fix(forms)!: y"},
			wantLevel: semver.Bugfix, wantMarker: "the pull request title", wantWarn: true},
		{name: "warn on a fix branch",
			in:        levelInput{Mode: "warn", Branch: "fix/x", Title: "fix(forms)!: y"},
			wantLevel: semver.Bugfix, wantMarker: "the pull request title", wantWarn: true},
		{name: "warn on a feat branch keeps minor",
			in:        levelInput{Mode: "warn", Branch: "feat/x", Title: "feat!: y"},
			wantLevel: semver.Feature, wantMarker: "the pull request title", wantWarn: true},
		{name: "warn on a major branch does not warn",
			in:        levelInput{Mode: "warn", Branch: "major/x", Title: "fix!: y"},
			wantLevel: semver.Major, wantMarker: "the pull request title"},
		{name: "warn without a marker",
			in:        levelInput{Mode: "warn", Branch: "feat/x", Title: "feat: y"},
			wantLevel: semver.Feature},
		// breaking_markers: off
		{name: "off ignores the marker",
			in:        levelInput{Mode: "off", Branch: "fix/x", Title: "fix(forms)!: y", Body: "BREAKING CHANGE: z"},
			wantLevel: semver.Bugfix},
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

func TestCollectCommits(t *testing.T) {
	partial := []conventionalcommits.Commit{{SHA: "aaaa", Message: "fix: x"}}
	failing := func() ([]conventionalcommits.Commit, error) { return nil, errors.New("500 Internal Server Error") }
	capped := func() ([]conventionalcommits.Commit, error) { return partial, errors.New("capped") }
	ok := func() ([]conventionalcommits.Commit, error) { return partial, nil }

	t.Run("major fails closed on an API error", func(t *testing.T) {
		_, warning, err := collectCommits(failing, "major")
		if err == nil || warning != "" {
			t.Fatalf("err = %v, warning = %q; want an error and no warning", err, warning)
		}
	})
	t.Run("major fails closed on a capped list", func(t *testing.T) {
		if _, _, err := collectCommits(capped, "major"); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("warn continues on an API error with a warning", func(t *testing.T) {
		commits, warning, err := collectCommits(failing, "warn")
		if err != nil || commits != nil {
			t.Fatalf("err = %v, commits = %v; want neither", err, commits)
		}
		want := "The commits of the pull request were not checked for a breaking-change marker: 500 Internal Server Error."
		if warning != want {
			t.Errorf("warning = %q, want %q", warning, want)
		}
	})
	t.Run("warn keeps the listed commits of a capped list", func(t *testing.T) {
		commits, warning, err := collectCommits(capped, "")
		if err != nil || len(commits) != 1 || warning == "" {
			t.Fatalf("commits = %v, warning = %q, err = %v", commits, warning, err)
		}
	})
	t.Run("no error", func(t *testing.T) {
		commits, warning, err := collectCommits(ok, "major")
		if err != nil || warning != "" || len(commits) != 1 {
			t.Fatalf("commits = %v, warning = %q, err = %v", commits, warning, err)
		}
	})
}
