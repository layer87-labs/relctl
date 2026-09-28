package conventionalcommits

import (
	"errors"
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

// synthesizeMergeCommit crafts a commit object with two parents (a real
// merge commit) reusing the tree of the current HEAD, stores it, and moves
// the current branch to point at it. This avoids needing go-git's higher
// level (and, depending on version, absent) merge support just to get a
// commit with NumParents() > 1 into a test repository.
func synthesizeMergeCommit(t *testing.T, repo *git.Repository, secondParent plumbing.Hash, message string) plumbing.Hash {
	t.Helper()

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("head commit object: %v", err)
	}

	sig := object.Signature{Name: "test", Email: "test@example.com", When: time.Now()}
	mergeCommit := &object.Commit{
		Author:       sig,
		Committer:    sig,
		Message:      message,
		TreeHash:     headCommit.TreeHash,
		ParentHashes: []plumbing.Hash{head.Hash(), secondParent},
	}

	obj := repo.Storer.NewEncodedObject()
	if err := mergeCommit.Encode(obj); err != nil {
		t.Fatalf("encode merge commit: %v", err)
	}
	newHash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		t.Fatalf("store merge commit: %v", err)
	}

	branchRef := plumbing.NewHashReference(head.Name(), newHash)
	if err := repo.Storer.SetReference(branchRef); err != nil {
		t.Fatalf("move branch to merge commit: %v", err)
	}
	return newHash
}

func TestCommitMessagesSince_MergeCommitInRange(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	tagHash := commitFile(t, wt, repo, "a.txt", "1", "chore: initial commit")
	if _, err := repo.CreateTag("1.0.0", plumbing.NewHash(tagHash), nil); err != nil {
		t.Fatalf("create tag: %v", err)
	}

	sideHash := commitFile(t, wt, repo, "a.txt", "2", "fix: on a side branch")
	// Reset the branch back onto the tag before adding the "mainline" commit,
	// so the eventual merge commit's first parent is a normal, single-parent
	// commit sitting directly on top of the tag.
	if err := wt.Reset(&git.ResetOptions{Commit: plumbing.NewHash(tagHash), Mode: git.HardReset}); err != nil {
		t.Fatalf("reset: %v", err)
	}
	commitFile(t, wt, repo, "b.txt", "1", "feat: on the mainline")

	synthesizeMergeCommit(t, repo, plumbing.NewHash(sideHash), "Merge pull request #1 from org/side-branch")

	_, err = CommitMessagesSince(dir, "1.0.0")
	if !errors.Is(err, ErrMergeCommitInRange) {
		t.Fatalf("error = %v, want errors.Is(_, ErrMergeCommitInRange)", err)
	}
}

func TestCommitMessagesSince_TagNotOnFirstParentChain(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	baseHash := commitFile(t, wt, repo, "a.txt", "1", "chore: base")

	// A tag that lives on a side branch, never merged as anyone's
	// first parent into the branch HEAD walks from.
	if err := wt.Reset(&git.ResetOptions{Commit: plumbing.NewHash(baseHash), Mode: git.HardReset}); err != nil {
		t.Fatalf("reset: %v", err)
	}
	sideHash := commitFile(t, wt, repo, "side.txt", "1", "fix: side branch tip")
	if _, err := repo.CreateTag("1.0.0", plumbing.NewHash(sideHash), nil); err != nil {
		t.Fatalf("create tag: %v", err)
	}

	if err := wt.Reset(&git.ResetOptions{Commit: plumbing.NewHash(baseHash), Mode: git.HardReset}); err != nil {
		t.Fatalf("reset: %v", err)
	}
	commitFile(t, wt, repo, "main.txt", "1", "feat: mainline commit")

	_, err = CommitMessagesSince(dir, "1.0.0")
	if !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("error = %v, want errors.Is(_, ErrTagNotFound)", err)
	}
}

func TestCommitMessagesSince_ClockSkew(t *testing.T) {
	// Committer-time ordering is exactly what the first-parent walk must NOT
	// rely on: craft a chain where a later first-parent commit carries an
	// EARLIER committer timestamp than its child, and confirm it is still
	// found by strictly following Parent(0), not by any time-based heap.
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	now := time.Now()
	commitAt := func(path, content, message string, when time.Time) string {
		t.Helper()
		if err := writeFile(repo, path, content); err != nil {
			t.Fatalf("writing file: %v", err)
		}
		if _, err := wt.Add(path); err != nil {
			t.Fatalf("git add: %v", err)
		}
		sig := &object.Signature{Name: "test", Email: "test@example.com", When: when}
		hash, err := wt.Commit(message, &git.CommitOptions{Author: sig, Committer: sig})
		if err != nil {
			t.Fatalf("git commit: %v", err)
		}
		return hash.String()
	}

	tagHash := commitAt("a.txt", "1", "fix: release baseline", now.Add(-1*time.Hour))
	if _, err := repo.CreateTag("1.0.0", plumbing.NewHash(tagHash), nil); err != nil {
		t.Fatalf("create tag: %v", err)
	}
	// This commit's committer time is BEFORE the tagged commit's, even
	// though it comes after it on the first-parent chain.
	commitAt("a.txt", "2", "feat: skewed clock commit", now.Add(-2*time.Hour))
	commitAt("a.txt", "3", "fix: normal follow-up", now)

	messages, err := CommitMessagesSince(dir, "1.0.0")
	if err != nil {
		t.Fatalf("CommitMessagesSince: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("got %d messages, want 2: %v", len(messages), messages)
	}
	if got := Classify(messages); got != BumpMinor {
		t.Errorf("Classify = %v, want %v (the skewed-clock feat commit must still be found)", got, BumpMinor)
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
