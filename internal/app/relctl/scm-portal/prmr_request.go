package scmportal

import (
	"strings"

	"github.com/layer87-labs/relctl/internal/app/relctl/scm-portal/github"
	"github.com/layer87-labs/relctl/internal/pkg/conventionalcommits"
	"github.com/layer87-labs/relctl/internal/pkg/relctlconfig"
	"github.com/layer87-labs/relctl/internal/pkg/semver"

	log "github.com/sirupsen/logrus"
)

// GetPrInfos retrieves detailed information about a pull request or merge request
// from the version control system. It uses the provided pull request number or
// merge commit SHA to fetch the relevant information. The function populates a
// PrMrRequestInfos struct with details such as the PR number, SHA, branch name,
// owner, repository, and merge commit SHA. It also parses the patch level from
// the branch name and checks for version or patch level overrides in the issue
// comments. If no version override is found, it determines the latest version
// and calculates the next version based on the patch level.

func (lay SCMLayer) GetPrInfos(number int, mergeCommitSha string, scheme string) (infos *PrMrRequestInfos, err error) {
	// Initialize the standard information
	infos = &PrMrRequestInfos{}

	// Get the pull request or merge request information
	// from the GitHub or GitLab API

	grc := lay.Grc.(*github.GitHubRichClient)
	prInfos, err := grc.GetPrInfos(number, mergeCommitSha)
	if err != nil {
		return nil, err
	}
	log.Traceln("got PR information from scm-portal-layer")

	// GitHub computes the merge commit of an open pull request asynchronously; right after
	// the pull request was created it can be missing. That is not an error.
	infos.Number = prInfos.GetNumber()
	infos.Sha = prInfos.GetHead().GetSHA()
	infos.ShaShort = shortSha(infos.Sha)
	infos.BranchName = prInfos.GetHead().GetRef()
	infos.Owner = grc.Owner
	infos.Repo = grc.Repository
	infos.MergeCommitSha = prInfos.GetMergeCommitSHA()

	// Comment help instructions to the issue. What a pull request can influence
	// depends on the version scheme.
	log.Traceln("comment help instructions to issue")
	if errCommHelp := lay.CommentHelpToPullRequest(infos.Number, scheme); errCommHelp != nil {
		log.Warnln(errCommHelp)
	}
	log.Traceln("commented help instructions to issue")

	// Only SemVer reads the branch prefix and the comment commands. For every
	// other scheme the caller computes the version; the latest release is
	// reported for reference.
	if scheme != relctlconfig.SchemeSemVer && scheme != "" {
		if repositoryRelease, errLatest := lay.GetLatestPublishedRelease(); errLatest == nil {
			infos.LatestVersion = repositoryRelease.BaseVersion()
		}
		return infos, nil
	}

	// Read comments from the pull request or merge request
	// and look for overrides
	log.Traceln("read comments from pr/mr and looking for overrides")
	version, patchLevel, err := lay.SearchIssuesForOverrides(infos.Number)
	if err != nil {
		return nil, err
	}

	// The branch prefix, a relctl_patch_level comment and a breaking-change
	// marker decide the bump. A version override pins the version: then the
	// markers are not checked.
	in := levelInput{
		Branch:   infos.BranchName,
		Override: patchLevel,
		Title:    prInfos.GetTitle(),
		Body:     prInfos.GetBody(),
		Mode:     lay.BreakingMarkers,
	}
	if version != nil {
		in.Mode = relctlconfig.BreakingMarkersOff
	}
	var fetchWarning string
	if in.Mode != relctlconfig.BreakingMarkersOff && basePatchLevel(in.Branch, in.Override) != semver.Major {
		in.Commits, fetchWarning, err = collectCommits(func() ([]conventionalcommits.Commit, error) {
			return grc.PullRequestCommits(infos.Number)
		}, in.Mode)
		if err != nil {
			return nil, err
		}
	}
	decision := decidePatchLevel(in)
	infos.PatchLevel = decision.Level
	infos.BreakingMarker = decision.Marker
	infos.BreakingWarning = strings.TrimSpace(fetchWarning + " " + decision.Warning)
	if patchLevel != nil {
		log.Infof("detected a patch level override to %s", *patchLevel)
	}
	if decision.Marker != "" {
		log.Infof("breaking change marker found in %s", decision.Marker)
	}
	if infos.BreakingWarning != "" {
		log.Warnln(infos.BreakingWarning)
	}

	if version != nil {
		log.Tracef("read comments from pr/mr complete conclusions, found version override version: %s", *version)
	}

	// Check if version override (3)
	if version == nil {
		log.Traceln("no version override found")
		// Get the latest release, if any
		log.Traceln("get latest release, if any")
		// Prereleases (release candidates) are never the base of a bump.
		repositoryRelease, err := lay.GetLatestPublishedRelease()
		if err != nil {
			log.Infoln("no github release found -> writing default 0.0.0")
			infos.LatestVersion = "0.0.0"
		} else {
			log.Infoln("found latest release", repositoryRelease.TagName)
			infos.LatestVersion = repositoryRelease.BaseVersion()
		}

		// Increment the version by the decided patch level
		if infos.NextVersion, err = semver.IncreaseVersion(infos.PatchLevel, infos.LatestVersion); err != nil {
			return nil, err
		}

		return infos, nil

	} else {
		log.Infoln("version override via pr comments specified")
		infos.NextVersion = *version
	}

	return
}

// shortSha returns the first eight characters of a commit sha.
func shortSha(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
