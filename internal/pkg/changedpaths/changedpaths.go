// Package changedpaths tells which files changed between a commit and HEAD, and
// which of them belong to a component.
package changedpaths

import (
	"fmt"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Files returns the paths that differ between the commit since and HEAD.
func Files(repo *git.Repository, since plumbing.Hash) ([]string, error) {
	sinceCommit, err := repo.CommitObject(since)
	if err != nil {
		return nil, fmt.Errorf("commit %s is not in the local history (fetch the full history, for example fetch-depth: 0): %w", since, err)
	}
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("reading HEAD: %w", err)
	}
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("reading the commit of HEAD: %w", err)
	}

	sinceTree, err := sinceCommit.Tree()
	if err != nil {
		return nil, err
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, err
	}
	changes, err := object.DiffTree(sinceTree, headTree)
	if err != nil {
		return nil, fmt.Errorf("comparing %s with HEAD: %w", since, err)
	}

	seen := map[string]bool{}
	var files []string
	for _, c := range changes {
		for _, name := range []string{c.From.Name, c.To.Name} {
			if name != "" && !seen[name] {
				seen[name] = true
				files = append(files, name)
			}
		}
	}
	return files, nil
}

// Match returns the files that match at least one pattern. "*" matches within a
// directory, "**" any number of directories; a pattern that names a directory
// (no wildcard, no extension match) matches everything below it.
func Match(files, patterns []string) ([]string, error) {
	var out []string
	for _, f := range files {
		for _, p := range patterns {
			ok, err := doublestar.PathMatch(p, f)
			if err != nil {
				return nil, fmt.Errorf("pattern %q: %w", p, err)
			}
			if !ok {
				// "dir" or "dir/" as shorthand for "dir/**"
				if below, err := doublestar.PathMatch(trimSlash(p)+"/**", f); err == nil && below {
					ok = true
				}
			}
			if ok {
				out = append(out, f)
				break
			}
		}
	}
	return out, nil
}

func trimSlash(p string) string {
	for len(p) > 0 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	return p
}
