package github

import (
	"fmt"

	"github.com/google/go-github/v86/github"
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
		pullRequests, _, err := ghrc.Client.PullRequests.ListPullRequestsWithCommit(ctx, ghrc.Owner, ghrc.Repository, mergeCommitSha, &github.ListOptions{PerPage: 100})
		if err != nil {
			return nil, fmt.Errorf("could not load pull requests for merge commit %s: %v", mergeCommitSha, err)
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
	newer := func(a, b *github.PullRequest) bool {
		if b == nil {
			return true
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
