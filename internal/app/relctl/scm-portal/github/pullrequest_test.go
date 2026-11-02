package github

import (
	"testing"
	"time"

	"github.com/google/go-github/v86/github"
	"github.com/stretchr/testify/require"
)

func pr(number int, sha string, mergedAt *time.Time) *github.PullRequest {
	p := &github.PullRequest{Number: &number, MergeCommitSHA: &sha}
	if mergedAt != nil {
		p.MergedAt = &github.Timestamp{Time: *mergedAt}
	}
	return p
}

func TestPickPullRequest(t *testing.T) {
	t0 := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)
	t2 := t0.Add(2 * time.Hour)

	t.Run("no candidates", func(t *testing.T) {
		require.Nil(t, pickPullRequest(nil, "abc"))
	})
	t.Run("ignores unmerged", func(t *testing.T) {
		require.Nil(t, pickPullRequest([]*github.PullRequest{pr(1, "abc", nil)}, "abc"))
	})
	t.Run("most recent merge wins among equal merge commits", func(t *testing.T) {
		got := pickPullRequest([]*github.PullRequest{pr(1, "abc", &t0), pr(2, "abc", &t1)}, "abc")
		require.Equal(t, 2, got.GetNumber())
	})
	t.Run("exact merge commit beats newer foreign merge", func(t *testing.T) {
		got := pickPullRequest([]*github.PullRequest{pr(1, "abc", &t0), pr(2, "zzz", &t2)}, "abc")
		require.Equal(t, 1, got.GetNumber())
	})
	t.Run("equal merge time is decided by lower number, independent of order", func(t *testing.T) {
		a := pickPullRequest([]*github.PullRequest{pr(7, "abc", &t0), pr(5, "abc", &t0)}, "abc")
		b := pickPullRequest([]*github.PullRequest{pr(5, "abc", &t0), pr(7, "abc", &t0)}, "abc")
		require.Equal(t, 5, a.GetNumber())
		require.Equal(t, 5, b.GetNumber())
	})
	t.Run("falls back to newest merged", func(t *testing.T) {
		got := pickPullRequest([]*github.PullRequest{pr(1, "x", &t0), pr(2, "y", &t1)}, "abc")
		require.Equal(t, 2, got.GetNumber())
	})
}
