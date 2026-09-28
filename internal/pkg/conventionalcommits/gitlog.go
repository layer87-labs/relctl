package conventionalcommits

import (
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"

	"github.com/layer87-labs/relctl/internal/pkg/tools"
)

// ErrTagNotFound is returned by CommitMessagesSince when the given tag does
// not exist in the local repository.
var ErrTagNotFound = errors.New("tag not found in local repository")

// CommitMessagesSince returns the full commit messages (header + body) of
// every commit reachable from HEAD, down to (but excluding) the commit
// tagged sinceTag. The order is unspecified.
//
// It relies purely on local Git state (the repository must be checked out
// with full history/tags, the same prerequisite relctl already documents
// for the CalVer scheme) — no SCM API call.
//
// Only the first-parent chain is walked, which is exactly the commit
// sequence a squash-merge workflow produces on the default branch. A merge
// commit's own subject ("Merge pull request #123 …") is included as one of
// the returned messages but classifies as BumpNone; the individual commits
// squashed into it are not inspected separately. Repositories that use
// merge commits (rather than squash merges) to bring in multi-commit PRs
// are not supported by this scheme.
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

	logIter, err := repo.Log(&git.LogOptions{
		From:  head.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("conventionalcommits: reading commit log: %w", err)
	}

	var messages []string
	err = logIter.ForEach(func(c *object.Commit) error {
		if c.Hash.String() == sinceHash {
			return storer.ErrStop
		}
		messages = append(messages, c.Message)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("conventionalcommits: walking commit log: %w", err)
	}

	return messages, nil
}
