package github

import (
	"fmt"
	"time"

	"github.com/google/go-github/v86/github"
	"github.com/layer87-labs/relctl/internal/pkg/conventionalcommits"
	"github.com/sirupsen/logrus"
)

// GetPrInfos need the PullRequest-Number
func (ghrc *GitHubRichClient) GetPrInfos(prNumber int, mergeCommitSha string) (prInfos *github.PullRequest, err error) {
	if prNumber != 0 {

		prInfos, _, err = ghrc.Client.PullRequests.Get(ctx, ghrc.Owner, ghrc.Repository, prNumber)
		if err != nil {
			return nil, fmt.Errorf("could not load any information about the given pull request  %d: %v", prNumber, err)
		}
		logrus.Infof("found pull request '%s' with given number %d", *prInfos.Title, prNumber)

		return
	} else if mergeCommitSha != "" {

		logrus.Infof("looking up pull requests associated with merge commit %s", mergeCommitSha)
		// GitHub indexes the commit-to-PR association asynchronously, so the
		// list can be empty right after a merge. Retry a few times.
		var pullRequests []*github.PullRequest
		for attempt := 1; attempt <= 4; attempt++ {
			pullRequests, _, err = ghrc.Client.PullRequests.ListPullRequestsWithCommit(ctx, ghrc.Owner, ghrc.Repository, mergeCommitSha, &github.ListOptions{PerPage: 100})
			if err != nil {
				return nil, fmt.Errorf("could not load pull requests for merge commit %s: %v", mergeCommitSha, err)
			}
			if len(pullRequests) > 0 {
				break
			}
			if attempt < 4 {
				logrus.Infof("no pull request listed for %s yet, retrying (%d/3)", mergeCommitSha, attempt)
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
			}
		}

		prInfos = pickPullRequest(pullRequests, mergeCommitSha)
		if prInfos != nil {
			logrus.Infof("selected pull request %d for merge commit %s (%d candidate(s))", prInfos.GetNumber(), mergeCommitSha, len(pullRequests))
		}
	}

	if prInfos == nil {
		return nil, fmt.Errorf("no pull request found, please check if all resources are specified")
	}
	return
}

// pickPullRequest selects the pull request a merge commit belongs to.
//
// GitHub marks a pull request as merged automatically when its head branch is
// contained in the merge of another pull request, and gives it the same
// mergeCommitSHA. Several merged pull requests can therefore match one commit.
// Preference order: merged pull requests whose mergeCommitSHA equals sha, then
// any merged pull request; within a group the most recently merged wins.
// It returns nil if there is no merged candidate.
func pickPullRequest(candidates []*github.PullRequest, sha string) *github.PullRequest {
	var best, bestExact *github.PullRequest
	// newer orders by merge time; equal times (a PR that GitHub marked as merged
	// together with another one) are decided by the lower pull request number,
	// so the result does not depend on the order of the API response.
	newer := func(a, b *github.PullRequest) bool {
		if b == nil {
			return true
		}
		if a.GetMergedAt().Time.Equal(b.GetMergedAt().Time) {
			return a.GetNumber() < b.GetNumber()
		}
		return a.GetMergedAt().After(b.GetMergedAt().Time)
	}
	for _, pr := range candidates {
		if pr == nil || pr.MergedAt == nil {
			continue
		}
		if newer(pr, best) {
			best = pr
		}
		if pr.GetMergeCommitSHA() == sha && newer(pr, bestExact) {
			bestExact = pr
		}
	}
	if bestExact != nil {
		return bestExact
	}
	return best
}

// maxPullRequestCommits is the most commits GitHub lists for a pull request.
const maxPullRequestCommits = 250

// ErrCommitsCapped means the pull request may have more commits than GitHub
// lists; the commits returned with it are incomplete.
var ErrCommitsCapped = fmt.Errorf("GitHub lists at most %d commits of a pull request, more may exist", maxPullRequestCommits)

// PullRequestCommits returns the commits of a pull request, and only those:
// neither the merge commit nor commits of the target branch. With
// ErrCommitsCapped it returns the commits GitHub listed.
func (ghrc *GitHubRichClient) PullRequestCommits(prNumber int) ([]conventionalcommits.Commit, error) {
	var commits []conventionalcommits.Commit
	opts := &github.ListOptions{PerPage: 100}
	for {
		page, resp, err := ghrc.Client.PullRequests.ListCommits(ctx, ghrc.Owner, ghrc.Repository, prNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("could not list the commits of pull request %d: %v", prNumber, err)
		}
		for _, c := range page {
			commits = append(commits, conventionalcommits.Commit{SHA: c.GetSHA(), Message: c.GetCommit().GetMessage()})
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	if len(commits) >= maxPullRequestCommits {
		return commits, ErrCommitsCapped
	}
	return commits, nil
}
