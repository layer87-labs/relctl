package github

import (
	"fmt"
	"net/http"

	"github.com/google/go-github/v86/github"
)

// RepositoryDefaultBranch returns the default branch of the repository.
func (ghrc *GitHubRichClient) RepositoryDefaultBranch() (string, error) {
	repo, _, err := ghrc.Client.Repositories.Get(ctx, ghrc.Owner, ghrc.Repository)
	if err != nil {
		return "", fmt.Errorf("error at reading the repository: %v", err)
	}
	return repo.GetDefaultBranch(), nil
}

// BranchSHA returns the commit a branch points to.
func (ghrc *GitHubRichClient) BranchSHA(branch string) (string, error) {
	b, _, err := ghrc.Client.Repositories.GetBranch(ctx, ghrc.Owner, ghrc.Repository, branch, 0)
	if err != nil {
		return "", fmt.Errorf("error at reading branch %q: %w", branch, err)
	}
	return b.GetCommit().GetSHA(), nil
}

// BranchExists reports whether a branch exists.
func (ghrc *GitHubRichClient) BranchExists(branch string) (bool, error) {
	_, resp, err := ghrc.Client.Repositories.GetBranch(ctx, ghrc.Owner, ghrc.Repository, branch, 0)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("error at reading branch %q: %w", branch, err)
	}
	return true, nil
}

// CreateBranch creates a branch pointing to sha.
func (ghrc *GitHubRichClient) CreateBranch(branch, sha string) error {
	_, _, err := ghrc.Client.Git.CreateRef(ctx, ghrc.Owner, ghrc.Repository, github.CreateRef{Ref: "refs/heads/" + branch, SHA: sha})
	if err != nil {
		return fmt.Errorf("error at creating branch %q: %w", branch, err)
	}
	return nil
}

// CompareStatus compares head with base and returns GitHub's status: "identical",
// "ahead" (base is an ancestor of head), "behind" (head is an ancestor of base)
// or "diverged".
func (ghrc *GitHubRichClient) CompareStatus(base, head string) (string, error) {
	cmp, _, err := ghrc.Client.Repositories.CompareCommits(ctx, ghrc.Owner, ghrc.Repository, base, head, &github.ListOptions{PerPage: 1})
	if err != nil {
		return "", fmt.Errorf("error at comparing %q with %q: %w", base, head, err)
	}
	return cmp.GetStatus(), nil
}

// FastForwardBranch moves a branch to sha without force. It fails if that is not
// a fast-forward or if the branch rules do not allow the token to push.
func (ghrc *GitHubRichClient) FastForwardBranch(branch, sha string) error {
	force := false
	_, _, err := ghrc.Client.Git.UpdateRef(ctx, ghrc.Owner, ghrc.Repository, "heads/"+branch, github.UpdateRef{SHA: sha, Force: &force})
	if err != nil {
		return fmt.Errorf("error at fast-forwarding %q: %v", branch, err)
	}
	return nil
}

// FindOpenPullRequest returns the number of an open pull request from head into
// base, or 0 if there is none.
func (ghrc *GitHubRichClient) FindOpenPullRequest(head, base string) (int, error) {
	prs, _, err := ghrc.Client.PullRequests.List(ctx, ghrc.Owner, ghrc.Repository, &github.PullRequestListOptions{
		State:       "open",
		Head:        ghrc.Owner + ":" + head,
		Base:        base,
		ListOptions: github.ListOptions{PerPage: 10},
	})
	if err != nil {
		return 0, fmt.Errorf("error at listing pull requests: %v", err)
	}
	if len(prs) == 0 {
		return 0, nil
	}
	return prs[0].GetNumber(), nil
}

// CreatePullRequest opens a pull request from head into base.
func (ghrc *GitHubRichClient) CreatePullRequest(head, base, title, body string) (int, error) {
	pr, _, err := ghrc.Client.PullRequests.Create(ctx, ghrc.Owner, ghrc.Repository, &github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Base:  &base,
		Body:  &body,
	})
	if err != nil {
		return 0, fmt.Errorf("error at creating the pull request: %v", err)
	}
	return pr.GetNumber(), nil
}
