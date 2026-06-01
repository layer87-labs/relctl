package tools

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/coreos/go-semver/semver"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// RepoRoot returns the absolute path to the root of the current Git repository.
func RepoRoot() (string, error) {
	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return "", fmt.Errorf("could not find git repository: %w", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("could not get worktree: %w", err)
	}
	return wt.Filesystem.Root(), nil
}

func GetDefaultBranch() string {
	out, err := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err != nil {
		return ""
	}
	ref := strings.TrimSpace(string(out))
	return strings.TrimPrefix(ref, "refs/remotes/origin/")
}

// GitLogSubject returns the subject line of the most recent commit.
func GitLogSubject() (string, error) {
	out, err := exec.Command("git", "log", "-1", "--pretty=format:%s").Output()
	if err != nil {
		return "", fmt.Errorf("git log: %w", err)
	}
	return string(out), nil
}

func DevideOwnerAndRepo(fullRepo string) (owner string, repo string) {
	owner = strings.ToLower(strings.Split(fullRepo, "/")[0])
	repo = strings.ToLower(strings.Split(fullRepo, "/")[1])
	return
}

func GetGitTagsUpToHead(gitRepo *git.Repository) (tags []*semver.Version, err error) {

	tags = []*semver.Version{}
	commitToTag, _, err := GetGitTagMaps(gitRepo)

	if err != nil {
		return nil, err
	}

	headRef, _ := gitRepo.Head()

	tagIter, _ := gitRepo.Log(&git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})

	_ = tagIter.ForEach(func(r *object.Commit) error {

		if tagNames, exists := commitToTag[r.Hash.String()]; exists {

			for _, tagName := range tagNames {
				parsedVersion, err := semver.NewVersion(tagName)
				if err == nil {
					tags = append(tags, parsedVersion)
				}
			}
		}
		return nil
	})

	semver.Sort(tags)

	return tags, nil
}

func GetGitTagMaps(gitRepo *git.Repository) (commitToTagMap map[string][]string, tagToCommitMap map[string]string, err error) {
	tagToCommitMap = make(map[string]string)
	commitToTagMap = make(map[string][]string)

	tags, err := gitRepo.Tags()

	if err != nil {
		return nil, nil, err
	}

	_ = tags.ForEach(func(r *plumbing.Reference) error {

		tagList, exists := commitToTagMap[r.Hash().String()]

		if !exists {
			tagList = make([]string, 0)
			commitToTagMap[r.Hash().String()] = tagList
		}

		commitToTagMap[r.Hash().String()] = append(tagList, r.Name().Short())
		tagToCommitMap[r.Name().Short()] = r.Hash().String()
		return nil
	})

	return commitToTagMap, tagToCommitMap, nil
}
