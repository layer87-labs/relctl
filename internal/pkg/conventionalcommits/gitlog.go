package conventionalcommits

import (
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"

	"github.com/layer87-labs/relctl/internal/pkg/tools"
)

// ErrTagNotFound is returned by CommitMessagesSince when the given tag does
// not exist in the local repository, or is not reachable via the
// first-parent chain from HEAD.
var ErrTagNotFound = errors.New("tag not found in local repository")

// ErrMergeCommitInRange is returned by CommitMessagesSince when a commit
// between HEAD and sinceTag has more than one parent. The
// conventional-commits scheme only understands a squash-merge workflow,
// where every commit on the target branch's first-parent chain is itself
// the unit to classify; a real merge commit means that assumption no
// longer holds, and guessing at the "right" set of commits to inspect
// would risk a silently wrong (too low) version bump. Failing loudly here
// is deliberate — see relctl#31 / relctl#32.
var ErrMergeCommitInRange = errors.New("merge commit found between the last published release and HEAD; the conventional-commits scheme only supports a squash-merge workflow")

// CommitMessagesSince returns the full commit messages (header + body) of
// every commit on the first-parent chain from HEAD, down to (but excluding)
// the commit tagged sinceTag, in walk order (HEAD first).
//
// It relies purely on local Git state (the repository must be checked out
// with full history/tags, the same prerequisite relctl already documents
// for the CalVer scheme) — no SCM API call.
//
// Only the first-parent chain is walked, by following object.Commit.Parent(0)
// directly rather than go-git's repo.Log with LogOrderCommitterTime (which
// visits ALL parents ordered by committer time, not just the first parent —
// using it here would silently stop the walk at sinceTag on the main line
// while never visiting commits that only exist on a side branch of a real
// merge commit, understating the bump). This is exactly the commit sequence
// a squash-merge workflow produces on the default branch, where each commit
// on that chain is one squashed PR. If a commit with more than one parent is
// encountered before reaching sinceTag, that assumption no longer holds and
// CommitMessagesSince returns ErrMergeCommitInRange rather than silently
// walking (or not walking) a side branch.
func CommitMessagesSince(repoPath, sinceTag string) ([]string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("conventionalcommits: opening repository at %q: %w", repoPath, err)
	}

	_, tagToCommit, err := tools.GetGitTagMaps(repo)
	if err != nil {
		return nil, fmt.Errorf("conventionalcommits: reading git tags: %w", err)
	}
	sinceHash, ok := tagToCommit[sinceTag]
	if !ok {
		return nil, fmt.Errorf("conventionalcommits: %w: %q", ErrTagNotFound, sinceTag)
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("conventionalcommits: resolving HEAD: %w", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("conventionalcommits: resolving HEAD commit: %w", err)
	}

	var messages []string
	for {
		if commit.Hash.String() == sinceHash {
			return messages, nil
		}
		if commit.NumParents() > 1 {
			return nil, fmt.Errorf("conventionalcommits: %w: %s %q", ErrMergeCommitInRange, commit.Hash.String()[:12], firstLine(commit.Message))
		}
		messages = append(messages, commit.Message)

		if commit.NumParents() == 0 {
			// Reached the root commit without finding sinceTag on the
			// first-parent chain: sinceTag is not an ancestor of HEAD via
			// first-parent (e.g. a stale or unrelated tag).
			return nil, fmt.Errorf("conventionalcommits: %w: %q is not reachable from HEAD via the first-parent chain", ErrTagNotFound, sinceTag)
		}

		commit, err = commit.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("conventionalcommits: walking first-parent chain: %w", err)
		}
	}
}

func firstLine(msg string) string {
	for i, c := range msg {
		if c == '\n' {
			return msg[:i]
		}
	}
	return msg
}
