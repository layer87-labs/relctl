package service

import (
	"errors"
	"testing"

	"github.com/coreos/go-semver/semver"
	"github.com/layer87-labs/relctl/internal/pkg/tools"

	"github.com/layer87-labs/relctl/internal/app/relctl/ces"

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

func TestExistingReleaseEnv(t *testing.T) {
	base := []ces.KeyValue{{Name: "RELCTL_VERSION", Value: "1.0.0"}}

	draft := existingReleaseEnv(base, 42, true)
	require.Len(t, draft, 2)
	require.Equal(t, "RELCTL_RELEASE_ID", draft[1].Name)
	require.Equal(t, "42", draft[1].Value)

	published := existingReleaseEnv(base, 42, false)
	require.Len(t, published, 1, "a published release must not hand out an id")
	for _, kv := range published {
		require.NotEqual(t, "RELCTL_RELEASE_ID", kv.Name)
	}
}

func TestReleaseEnv(t *testing.T) {
	values := func(kvs []ces.KeyValue) map[string]string {
		m := map[string]string{}
		for _, kv := range kvs {
			m[kv.Name] = kv.Value
		}
		return m
	}

	t.Run("a release", func(t *testing.T) {
		got := values(releaseEnv(releaseVars{Version: "1.4.0", Next: "1.4.0", Latest: "1.3.2"}))
		require.Equal(t, "1.4.0", got["RELCTL_VERSION"])
		require.Equal(t, "1.4.0", got["RELCTL_TAG"])
		require.Equal(t, "1.4.0", got["RELCTL_NEXT_VERSION"])
		require.Equal(t, "1.3.2", got["RELCTL_LATEST_VERSION"])
		require.Equal(t, "false", got["RELCTL_PRERELEASE"])
		require.Equal(t, "", got["RELCTL_COMPONENT"])
	})

	t.Run("a release candidate: next is the release it is a candidate for, not the candidate", func(t *testing.T) {
		got := values(releaseEnv(releaseVars{Version: "1.4.0-rc.3", Next: "1.4.0", Latest: "1.3.2", Prerelease: true}))
		require.Equal(t, "1.4.0-rc.3", got["RELCTL_VERSION"])
		require.Equal(t, "1.4.0", got["RELCTL_NEXT_VERSION"])
		require.Equal(t, "1.3.2", got["RELCTL_LATEST_VERSION"], "a candidate is never the latest version")
		require.Equal(t, "true", got["RELCTL_PRERELEASE"])
	})

	t.Run("first release has no latest version", func(t *testing.T) {
		got := values(releaseEnv(releaseVars{Version: "0.1.0", Next: "0.1.0"}))
		require.Equal(t, "", got["RELCTL_LATEST_VERSION"])
	})

	t.Run("the version stays plain, the tag carries the prefix", func(t *testing.T) {
		got := values(releaseEnv(releaseVars{Version: "202611.3.2", Next: "202611.3.2", Latest: "202611.3.1", TagPrefix: "whisper/", Component: "whisper"}))
		require.Equal(t, "202611.3.2", got["RELCTL_VERSION"])
		require.Equal(t, "whisper/202611.3.2", got["RELCTL_TAG"])
		require.Equal(t, "202611.3.1", got["RELCTL_LATEST_VERSION"])
		require.Equal(t, "whisper", got["RELCTL_COMPONENT"])
	})
}

func TestLatestReleaseTag(t *testing.T) {
	v := func(tag, version string) tools.TaggedVersion {
		return tools.TaggedVersion{Tag: tag, Version: semver.New(version)}
	}

	tags := []tools.TaggedVersion{
		v("whisper/1.0.0", "1.0.0"),
		v("whisper/1.1.0", "1.1.0"),
		v("whisper/1.2.0-rc.1", "1.2.0-rc.1"), // a candidate of an unreleased version is not a release
		v("whisper/1.2.0-rc.2", "1.2.0-rc.2"),
	}
	got, ok := latestReleaseTag(tags)
	require.True(t, ok)
	require.Equal(t, "whisper/1.1.0", got.Tag)

	_, ok = latestReleaseTag(tags[2:])
	require.False(t, ok, "only candidates: no release yet")
	_, ok = latestReleaseTag(nil)
	require.False(t, ok)
}
