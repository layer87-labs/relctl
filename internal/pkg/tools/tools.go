package tools

import (
	"fmt"
	"os/exec"
	"sort"
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

// TaggedVersion is a tag whose name, without the tag prefix, is a version.
type TaggedVersion struct {
	// Tag is the full name of the tag, prefix included.
	Tag     string
	Version *semver.Version
}

// GetGitTagsUpToHead returns the tags reachable from HEAD that are versions,
// ordered by version (lowest first).
func GetGitTagsUpToHead(gitRepo *git.Repository) (tags []*semver.Version, err error) {
	tagged, err := GetGitTagsUpToHeadWithPrefix(gitRepo, "")
	if err != nil {
		return nil, err
	}
	tags = []*semver.Version{}
	for _, t := range tagged {
		tags = append(tags, t.Version)
	}
	return tags, nil
}

// GetGitTagsUpToHeadWithPrefix returns the tags reachable from HEAD whose name
// starts with prefix and continues as a version, ordered by version (lowest
// first). Other tags, for example those of another component, are ignored.
func GetGitTagsUpToHeadWithPrefix(gitRepo *git.Repository, prefix string) (tags []TaggedVersion, err error) {
	tags = []TaggedVersion{}
	commitToTag, _, err := GetGitTagMaps(gitRepo)
	if err != nil {
		return nil, err
	}

	headRef, err := gitRepo.Head()
	if err != nil {
		return nil, err
	}

	tagIter, err := gitRepo.Log(&git.LogOptions{
		From:  headRef.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, err
	}

	_ = tagIter.ForEach(func(r *object.Commit) error {
		for _, tagName := range commitToTag[r.Hash.String()] {
			if !strings.HasPrefix(tagName, prefix) {
				continue
			}
			parsedVersion, err := semver.NewVersion(strings.TrimPrefix(tagName, prefix))
			if err == nil {
				tags = append(tags, TaggedVersion{Tag: tagName, Version: parsedVersion})
			}
		}
		return nil
	})

	sort.SliceStable(tags, func(i, j int) bool { return tags[i].Version.LessThan(*tags[j].Version) })
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
