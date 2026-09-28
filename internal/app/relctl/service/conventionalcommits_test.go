package service

import (
	"errors"
	"strings"
	"testing"

	scmportal "github.com/layer87-labs/relctl/internal/app/relctl/scm-portal"
)

// fakeReleaseGetter is a test double for latestPublishedReleaseGetter.
type fakeReleaseGetter struct {
	release *scmportal.Release
	err     error
}

func (f *fakeReleaseGetter) GetLatestPublishedRelease() (*scmportal.Release, error) {
	return f.release, f.err
}

func TestConventionalCommitsVersion(t *testing.T) {
	tests := []struct {
		name          string
		release       *scmportal.Release
		releaseErr    error
		commitMsgs    []string
		commitErr     error
		wantVersion   string
		wantErr       error // checked with errors.Is when non-nil
		wantErrSubstr string
	}{
		{
			name:        "single fix bumps patch",
			release:     &scmportal.Release{TagName: "1.2.3"},
			commitMsgs:  []string{"fix: correct heartbeat timestamp parsing"},
			wantVersion: "1.2.4",
		},
		{
			name:        "feat bumps minor",
			release:     &scmportal.Release{TagName: "1.2.3"},
			commitMsgs:  []string{"fix: a", "feat: add node filtering by type"},
			wantVersion: "1.3.0",
		},
		{
			name:        "bang bumps major",
			release:     &scmportal.Release{TagName: "1.2.3"},
			commitMsgs:  []string{"feat: a", "fix!: drop deprecated field"},
			wantVersion: "2.0.0",
		},
		{
			name:        "breaking change footer bumps major",
			release:     &scmportal.Release{TagName: "1.2.3"},
			commitMsgs:  []string{"chore: cleanup\n\nBREAKING CHANGE: old config keys removed"},
			wantVersion: "2.0.0",
		},
		{
			name:       "no relevant commits errors",
			release:    &scmportal.Release{TagName: "1.2.3"},
			commitMsgs: []string{"Merge pull request #3 from org/x", "wip: nothing conventional here"},
			wantErr:    ErrNoRelevantCommits,
		},
		{
			name:       "empty commit set errors",
			release:    &scmportal.Release{TagName: "1.2.3"},
			commitMsgs: nil,
			wantErr:    ErrNoRelevantCommits,
		},
		{
			name:       "no previous release errors",
			release:    nil,
			wantErr:    ErrNoPreviousRelease,
		},
		{
			name:       "release lookup failure wraps ErrNoPreviousRelease",
			releaseErr: errors.New("API rate limited"),
			wantErr:    ErrNoPreviousRelease,
		},
		{
			name:          "commit lookup failure propagates",
			release:       &scmportal.Release{TagName: "1.2.3"},
			commitErr:     errors.New("boom"),
			wantErrSubstr: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := &fakeReleaseGetter{release: tt.release, err: tt.releaseErr}
			commitMessagesSince := func(repoPath, sinceTag string) ([]string, error) {
				if tt.release != nil && sinceTag != tt.release.TagName {
					t.Errorf("commitMessagesSince called with tag %q, want %q", sinceTag, tt.release.TagName)
				}
				return tt.commitMsgs, tt.commitErr
			}

			got, err := conventionalCommitsVersion(rl, commitMessagesSince, "/repo")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want errors.Is(_, %v)", err, tt.wantErr)
				}
				return
			}
			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantVersion {
				t.Errorf("version = %q, want %q", got, tt.wantVersion)
			}
		})
	}
}
