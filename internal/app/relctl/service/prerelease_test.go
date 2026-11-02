package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextReleaseCandidate(t *testing.T) {
	var gotPrefix string
	list := func(existing []string, err error) func(string) ([]string, error) {
		return func(prefix string) ([]string, error) {
			gotPrefix = prefix
			return existing, err
		}
	}

	v, err := nextReleaseCandidate(list(nil, nil), "1.4.0")
	require.NoError(t, err)
	require.Equal(t, "1.4.0-rc.1", v)
	require.Equal(t, "1.4.0-rc.", gotPrefix)

	v, err = nextReleaseCandidate(list([]string{"1.4.0-rc.1", "1.4.0-rc.2"}, nil), "1.4.0")
	require.NoError(t, err)
	require.Equal(t, "1.4.0-rc.3", v)

	_, err = nextReleaseCandidate(list(nil, errors.New("boom")), "1.4.0")
	require.ErrorContains(t, err, "boom")
}

func TestReleaseTarget(t *testing.T) {
	require.Equal(t, "abc123", releaseTarget(&ReleaseArgs{MergeCommitSHA: "abc123", ReleaseBranch: "main"}),
		"the merge commit wins over the branch")
	require.Equal(t, "main", releaseTarget(&ReleaseArgs{ReleaseBranch: "main"}))
	require.Equal(t, "", releaseTarget(&ReleaseArgs{}), "empty means the default branch")
}

func TestReleaseBody(t *testing.T) {
	failing := func() (string, error) { return "", errors.New("api down") }
	generated := func() (string, error) { return "## What's Changed\n* a", nil }
	neverCalled := func() (string, error) { t.Fatal("must not generate when a text is given"); return "", nil }

	if got := releaseBody("my notes", neverCalled); got != "my notes" {
		t.Errorf("given text must win, got %q", got)
	}
	if got := releaseBody("  \n", generated); got != "## What's Changed\n* a" {
		t.Errorf("blank text must be replaced, got %q", got)
	}
	if got := releaseBody("", failing); got != "" {
		t.Errorf("a failure must not stop the release, got %q", got)
	}
}
