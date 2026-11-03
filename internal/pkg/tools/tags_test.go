package tools

import (
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// repoWithTags creates a repository with one commit per tag, oldest first.
func repoWithTags(t *testing.T, tags ...string) *git.Repository {
	t.Helper()
	repo, err := git.PlainInit(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tag := range tags {
		when = when.Add(time.Hour)
		hash, err := wt.Commit("change for "+tag, &git.CommitOptions{
			AllowEmptyCommits: true,
			Author:            &object.Signature{Name: "t", Email: "t@example.test", When: when},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.CreateTag(tag, hash, nil); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

func tagNames(tags []TaggedVersion) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		out = append(out, tag.Tag)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestGetGitTagsUpToHeadWithPrefix(t *testing.T) {
	repo := repoWithTags(t,
		"1.0.0", "whisper/202611.2.1", "v2.0.0", "whisper/202611.3.1", "router/1.1.0", "1.1.0", "whisper/202611.3.2")

	cases := []struct {
		prefix string
		want   []string
	}{
		{"", []string{"1.0.0", "1.1.0"}},
		{"whisper/", []string{"whisper/202611.2.1", "whisper/202611.3.1", "whisper/202611.3.2"}},
		{"router/", []string{"router/1.1.0"}},
		{"v", []string{"v2.0.0"}},
		{"unknown/", []string{}},
	}
	for _, c := range cases {
		got, err := GetGitTagsUpToHeadWithPrefix(repo, c.prefix)
		if err != nil {
			t.Fatalf("prefix %q: %v", c.prefix, err)
		}
		if !equal(tagNames(got), c.want) {
			t.Errorf("prefix %q: got %v, want %v", c.prefix, tagNames(got), c.want)
		}
	}
}

func TestGetGitTagsUpToHeadWithPrefixOrdersByVersion(t *testing.T) {
	// created in this order, but 1.10.0 is higher than 1.9.0
	repo := repoWithTags(t, "v1.10.0", "v1.9.0")
	got, err := GetGitTagsUpToHeadWithPrefix(repo, "v")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"v1.9.0", "v1.10.0"}; !equal(tagNames(got), want) {
		t.Errorf("got %v, want %v", tagNames(got), want)
	}
}

func TestGetGitTagsUpToHeadKeepsTheOldBehaviour(t *testing.T) {
	repo := repoWithTags(t, "1.0.0", "whisper/1.0.0", "1.1.0")
	got, err := GetGitTagsUpToHead(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].String() != "1.0.0" || got[1].String() != "1.1.0" {
		t.Errorf("unexpected %v", got)
	}
}
