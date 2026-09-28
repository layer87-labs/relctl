package conventionalcommits

import (
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func commitFile(t *testing.T, wt *git.Worktree, repo *git.Repository, path, content, message string) string {
	t.Helper()
	if err := writeFile(repo, path, content); err != nil {
		t.Fatalf("writing file: %v", err)
	}
	if _, err := wt.Add(path); err != nil {
		t.Fatalf("git add: %v", err)
	}
	sig := &object.Signature{Name: "test", Email: "test@example.com", When: time.Now()}
	hash, err := wt.Commit(message, &git.CommitOptions{Author: sig})
	if err != nil {
		t.Fatalf("git commit: %v", err)
	}
	return hash.String()
}

func writeFile(repo *git.Repository, path, content string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	f, err := wt.Filesystem.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write([]byte(content))
	return err
}

func TestCommitMessagesSince(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	commitFile(t, wt, repo, "a.txt", "1", "chore: initial commit")
	tagHash := commitFile(t, wt, repo, "a.txt", "2", "fix: release baseline")

	if _, err := repo.CreateTag("1.0.0", plumbing.NewHash(tagHash), nil); err != nil {
		t.Fatalf("create tag: %v", err)
	}

	commitFile(t, wt, repo, "a.txt", "3", "feat: add thing")
	commitFile(t, wt, repo, "a.txt", "4", "fix: fix thing")

	messages, err := CommitMessagesSince(dir, "1.0.0")
	if err != nil {
		t.Fatalf("CommitMessagesSince: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("got %d messages, want 2: %v", len(messages), messages)
	}

	got := Classify(messages)
	if got != BumpMinor {
		t.Errorf("Classify(messages since tag) = %v, want %v", got, BumpMinor)
	}
}

func TestCommitMessagesSince_UnknownTag(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	commitFile(t, wt, repo, "a.txt", "1", "chore: initial commit")

	_, err = CommitMessagesSince(dir, "does-not-exist")
	if err == nil {
		t.Fatal("expected error for unknown tag, got nil")
	}
}
