package github

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/layer87-labs/relctl/internal/pkg/semver"
	"github.com/sirupsen/logrus"

	"github.com/google/go-github/v86/github"
)

var (
	direction, sort = "asc", "created"
	detailsOpening  = `<details><summary>Possible relctl commands for this Pull Request</summary>`

	// https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
	semVerRegex = `(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
)

func (ghrc *GitHubRichClient) GetIssueComments(issueNumber int) (issueComments []*github.IssueComment, err error) {

	//FIXME: currently the GitHub API ignores Direction and Sort
	// so, we are using the default settings for the query and sorting the response afterwards
	var commentOpts = &github.IssueListCommentsOptions{
		Direction:   &direction,
		Sort:        &sort,
		ListOptions: standardListOptions,
	}
	issueComments, _, err = ghrc.Client.Issues.ListComments(ctx, ghrc.Owner, ghrc.Repository, issueNumber, commentOpts)

	if err == nil {
		var issueCommentsFiltered []*github.IssueComment
		for _, issueComment := range issueComments {
			if !strings.HasPrefix(*issueComment.Body, detailsOpening) {
				issueCommentsFiltered = append(issueCommentsFiltered, issueComment)
			}
		}

		icCount := len(issueCommentsFiltered)
		issueCommentsSorted := make([]*github.IssueComment, icCount)

		for i, issueComment := range issueCommentsFiltered {
			j := icCount - i - 1
			issueCommentsSorted[j] = issueComment
		}

		return issueCommentsSorted, err
	}

	return nil, err
}

func (ghrc *GitHubRichClient) CommentHelpToPullRequest(number int, scheme string) (err error) {
	var commentOpts = &github.IssueListCommentsOptions{
		Direction:   &direction,
		Sort:        &sort,
		ListOptions: standardListOptions,
	}
	comments, _, err := ghrc.Client.Issues.ListComments(ctx, ghrc.Owner, ghrc.Repository, number, commentOpts)
	if err != nil {
		return fmt.Errorf("unable to list comments on pull request %d: %w", number, err)
	}

	body := HelpCommentBody(scheme)

	var prComment *github.IssueComment
	for _, prc := range comments {
		if strings.HasPrefix(*prc.Body, detailsOpening) {
			prComment = prc
		}
	}

	if prComment == nil {
		var prComment = &github.IssueComment{
			Body: &body,
		}

		_, _, err = ghrc.Client.Issues.CreateComment(ctx, ghrc.Owner, ghrc.Repository, number, prComment)
		if err != nil {
			err = fmt.Errorf("unable to create help comment on pull request %d (the token needs the 'pull-requests: write' permission; read-only tokens, e.g. on fork PRs, cannot comment): %w", number, err)
		}
	} else {
		// edit command if newer version deployed and commands changed
		if *prComment.Body != body {
			editedComment := &github.IssueComment{
				Body: &body,
			}
			_, _, err = ghrc.Client.Issues.EditComment(ctx, ghrc.Owner, ghrc.Repository, *prComment.ID, editedComment)
			if err != nil {
				err = fmt.Errorf("unable to update help comment on pull request %d (the token needs the 'pull-requests: write' permission): %w", number, err)
			}
		}
	}
	return
}

func (ghrc *GitHubRichClient) SearchIssuesForOverrides(prNumber int) (nextVersion *string, patchLevel *semver.PatchLevel, err error) {
	var pLevel semver.PatchLevel
	// if an comment exists with relctl_patch_level=major, make a major version!
	issueComments, err := ghrc.GetIssueComments(prNumber)
	if err != nil {
		return
	}

	for _, comment := range issueComments {
		// FIXME: access must be restricted but GITHUB_TOKEN doesn't get informations.
		// Refs: https://docs.github.com/en/rest/collaborators/collaborators#list-repository-collaborators
		// Refs: https://docs.github.com/en/actions/security-guides/automatic-token-authentication#permissions-for-the-github_token

		// Must be a collaborator to have permission to create an override
		// isCollaborator, resp, err := GithubClient.Repositories.IsCollaborator(ctx, owner, repo, *comment.User.Login)
		// if err != nil {
		// 	return nil, nil, err
		// }
		// fmt.Println(resp.StatusCode)

		// if isCollaborator {
		if true {
			// The legacy awesome-ci prefix "aci_" is still accepted.
			if strings.HasPrefix(*comment.Body, "aci_") {
				logrus.Warnln("the comment command prefix 'aci_' is deprecated, please use 'relctl_'")
			}

			if v := matchVersionOverride(*comment.Body); v != nil {
				nextVersion = v
				logrus.Infof("found version override in comment form %s at %s", comment.User.GetLogin(), comment.GetCreatedAt())
				break
			}

			if level := matchPatchLevel(*comment.Body); level != "" {
				pLevel, err = semver.ParsePatchLevel(level)
				if err != nil && err != semver.ErrUseMinimalPatchVersion {
					logrus.Warnln(err)
				}
				logrus.Infof("found patchLevel override in comment form %s at %s", comment.User.GetLogin(), comment.GetCreatedAt())
				patchLevel = &pLevel
				break
			}
		}
	}
	return
}

/* func (ghrc *GitHubRichClient) CommentPullRequest(number int, comment *github.IssueComment) (err error) {
	_, _, err = ghrc.Client.Issues.CreateComment(ctx, ghrc.Owner, ghrc.Repository, number, comment)
	return
} */

var (
	versionOverrideRe = regexp.MustCompile(`^(?:relctl|aci)_version_override: (` + semVerRegex + `)`)
	patchLevelRe      = regexp.MustCompile(`^(?:relctl|aci)_patch_level: ([a-zA-Z]+)`)
)

// matchVersionOverride returns the version of a "relctl_version_override:" (or
// legacy "aci_version_override:") command at the start of body, or nil.
func matchVersionOverride(body string) *string {
	m := versionOverrideRe.FindStringSubmatch(body)
	if m == nil {
		return nil
	}
	return &m[1]
}

// matchPatchLevel returns the argument of a "relctl_patch_level:" (or legacy
// "aci_patch_level:") command at the start of body, or "".
func matchPatchLevel(body string) string {
	m := patchLevelRe.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return m[1]
}

// HelpCommentBody returns the text of the help comment for a pull request.
// What a pull request can do depends on the version scheme: only SemVer reads
// the branch prefix and the comment commands.
func HelpCommentBody(scheme string) string {
	const footer = `

Need more help?

Have a look at [my repo](https://github.com/layer87-labs/relctl)

This message was created by relctl and can be disabled by the env variable <code>RELCTL_SILENT=true</code>
</details>`

	switch scheme {
	case "calver":
		return detailsOpening + `

Version scheme: **calver**. The version of the release is the date of the release (` + "`YYYYMM.D.N`" + `, N counts the releases of that day). Branch prefixes and comment commands have no effect.` + footer
	case "conventional-commits":
		return detailsOpening + `

Version scheme: **conventional-commits**. The version is derived from the commit messages since the last release: a breaking change (` + "`feat!:`" + ` or a ` + "`BREAKING CHANGE`" + ` footer) is a major, ` + "`feat:`" + ` a minor, anything else a patch bump. With squash merges the pull request title is the commit message, so it must be a Conventional Commit. Branch prefixes and comment commands have no effect.` + footer
	case "changelog":
		return detailsOpening + `

Version scheme: **changelog**. The version is the topmost ` + "`## X.Y.Z`" + ` heading of the changelog; the release is made when the heading is dated and the change reaches the production branch. Branch prefixes and comment commands have no effect.` + footer
	default:
		return detailsOpening + `

| command | description |
| --- | --- |
| relctl_patch_level: major | create a major version bump (a branch prefix like feat!/ does the same) |
| relctl_version_override: 2.1.0 | set the version to 2.1.0 using only semver compatible syntax! |` + footer
	}
}
