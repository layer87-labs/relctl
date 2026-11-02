package service

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"

	scmportal "github.com/layer87-labs/relctl/internal/app/relctl/scm-portal"
	"github.com/layer87-labs/relctl/internal/pkg/changelog"
	"github.com/layer87-labs/relctl/internal/pkg/prerelease"
	"github.com/layer87-labs/relctl/internal/pkg/tools"

	log "github.com/sirupsen/logrus"
)

// GateArgs configures `relctl release gate`.
type GateArgs struct {
	// ConfigFile is the path to .relctl.yaml; empty means default lookup.
	ConfigFile string
	// Sha is the commit that must have a release candidate; empty means HEAD.
	Sha string
	// RequireCandidate additionally demands a release candidate for Sha.
	RequireCandidate bool
}

// GateCheck is the result of one check of the release gate.
type GateCheck struct {
	Name   string
	OK     bool
	Detail string
}

// gateLookups are the repository queries the gate needs. They are injected so
// the decision logic can be tested without a repository.
type gateLookups struct {
	// versionReleased reports whether version already has a published release or a tag.
	versionReleased func(version string) (bool, error)
	// candidateCommits maps the tag of every release candidate of version to its commit.
	candidateCommits func(version string) (map[string]string, error)
}

// evaluateGate decides whether the changelog content may be released from sha.
// All checks are always evaluated and reported, so that one run shows everything
// that is missing.
func evaluateGate(content, sha string, requireCandidate bool, l gateLookups) []GateCheck {
	var checks []GateCheck

	// (a) the topmost heading is valid and dated, and has release notes
	heading, headingErr := changelog.Top(content)
	switch {
	case headingErr != nil:
		checks = append(checks, GateCheck{"changelog heading", false, headingErr.Error()})
	case !heading.Released():
		checks = append(checks, GateCheck{"changelog heading", false,
			fmt.Sprintf("version %s has no release date", heading.Version)})
	default:
		notes, err := changelog.Section(content)
		if err != nil || strings.TrimSpace(notes) == "" {
			checks = append(checks, GateCheck{"changelog heading", false,
				fmt.Sprintf("version %s has no release notes", heading.Version)})
		} else {
			checks = append(checks, GateCheck{"changelog heading", true,
				fmt.Sprintf("%s — %s", heading.Version, heading.Date)})
		}
	}

	if headingErr != nil {
		checks = append(checks,
			GateCheck{"version not released yet", false, "skipped: no valid version"})
		if requireCandidate {
			checks = append(checks,
				GateCheck{"release candidate for commit", false, "skipped: no valid version"})
		}
		return checks
	}

	// (b) the version has not been released yet
	released, err := l.versionReleased(heading.Version)
	switch {
	case err != nil:
		checks = append(checks, GateCheck{"version not released yet", false, err.Error()})
	case released:
		checks = append(checks, GateCheck{"version not released yet", false,
			fmt.Sprintf("%s is already released", heading.Version)})
	default:
		checks = append(checks, GateCheck{"version not released yet", true, heading.Version})
	}

	// (c) a release candidate exists for exactly this commit
	if requireCandidate {
		commits, err := l.candidateCommits(heading.Version)
		if err != nil {
			checks = append(checks, GateCheck{"release candidate for commit", false, err.Error()})
		} else {
			found := ""
			for tag, c := range commits {
				if c == sha {
					found = tag
					break
				}
			}
			if found != "" {
				checks = append(checks, GateCheck{"release candidate for commit", true,
					fmt.Sprintf("%s at %s", found, shortSha(sha))})
			} else {
				checks = append(checks, GateCheck{"release candidate for commit", false,
					fmt.Sprintf("no %s* tag points to %s", prerelease.Prefix(heading.Version), shortSha(sha))})
			}
		}
	}
	return checks
}

func shortSha(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// gatePassed reports whether every check passed.
func gatePassed(checks []GateCheck) bool {
	for _, c := range checks {
		if !c.OK {
			return false
		}
	}
	return true
}

// ReleaseGate checks whether the changelog of the current checkout may be
// released and exits with a non-zero status if not.
func ReleaseGate(args *GateArgs) {
	scmLayer, err := scmportal.LoadSCMPortalLayer()
	if err != nil {
		log.Fatalln(err)
	}

	path, err := changelogFileFor(&ReleaseArgs{ConfigFile: args.ConfigFile})
	if err != nil {
		log.Fatalln(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("changelog: %v", err)
	}

	sha := args.Sha
	if sha == "" && args.RequireCandidate {
		sha, err = headSha()
		if err != nil {
			log.Fatalln(err)
		}
	}

	checks := evaluateGate(string(data), sha, args.RequireCandidate, gateLookups{
		versionReleased: func(version string) (bool, error) {
			return releasedOrTagged(scmLayer, version)
		},
		candidateCommits: func(version string) (map[string]string, error) {
			return scmLayer.CandidateCommits(version)
		},
	})

	fmt.Println("### Release gate:")
	for _, c := range checks {
		mark := "ok  "
		if !c.OK {
			mark = "FAIL"
		}
		fmt.Printf("%s %s: %s\n", mark, c.Name, c.Detail)
	}
	if !gatePassed(checks) {
		log.Fatalln("release gate not passed")
	}
	fmt.Println("release gate passed")
}

// releasedOrTagged reports whether version has a published release or a tag.
func releasedOrTagged(scmLayer *scmportal.SCMLayer, version string) (bool, error) {
	existing, draft, err := scmLayer.FindReleaseByTag(version)
	if err != nil {
		return false, err
	}
	if existing != nil && !draft {
		return true, nil
	}
	return scmLayer.TagExists(version)
}

// headSha returns the commit of HEAD of the repository in the working directory.
func headSha() (string, error) {
	root, err := tools.RepoRoot()
	if err != nil {
		return "", fmt.Errorf("could not determine repo root: %w", err)
	}
	repo, err := git.PlainOpen(root)
	if err != nil {
		return "", err
	}
	head, err := repo.Head()
	if err != nil {
		return "", err
	}
	return head.Hash().String(), nil
}
