package service

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/layer87-labs/relctl/internal/app/relctl/ces"
	scmportal "github.com/layer87-labs/relctl/internal/app/relctl/scm-portal"
	"github.com/layer87-labs/relctl/internal/pkg/conventionalcommits"
	"github.com/layer87-labs/relctl/internal/pkg/prerelease"
	"github.com/layer87-labs/relctl/internal/pkg/relctlconfig"
	"github.com/layer87-labs/relctl/internal/pkg/semver"
	"github.com/layer87-labs/relctl/internal/pkg/tools"
	"github.com/layer87-labs/relctl/internal/pkg/uploadasset"
	"github.com/layer87-labs/relctl/internal/pkg/versioning"

	log "github.com/sirupsen/logrus"
)

type ReleaseArgs struct {
	Version        string
	PatchLevel     string
	PrNumber       int
	MergeCommitSHA string
	ReleaseBranch  string
	ReleasePrefix  string
	DryRun         bool
	Hotfix         bool
	Body           string
	// VersionScheme overrides the config-file value ("semver" or "calver").
	// Empty string means "use config / default".
	VersionScheme string
	// ConfigFile is the path to .relctl.yaml; empty means default lookup.
	ConfigFile string
	// Prerelease creates a release candidate (X.Y.Z-rc.N, marked as prerelease)
	// for the computed version instead of the release itself.
	Prerelease bool
}

func ReleaseCreate(args *ReleaseArgs) {
	scmLayer, err := scmportal.LoadSCMPortalLayer()
	if err != nil {
		log.Fatalln(err)
	}

	version, releasePrefix, err := argsToVersion(scmLayer, args)
	if err != nil {
		log.Fatalln(err)
	}

	body, err := tools.ReadFileToString(args.Body)
	if err != nil && !errors.Is(err, tools.ErrEmptyString) {
		log.Fatalln(err)
	}

	base := version
	if args.Prerelease {
		// A candidate for a version that is already released can never be released.
		if released := versionAlreadyReleased(scmLayer, base); released {
			log.Infof("%s is already released, no release candidate is created", base)
			fmt.Printf("%s is already released, no release candidate is created\n", base)
			return
		}
		version, err = nextReleaseCandidate(scmLayer.ListTagAndReleaseNames, base)
		if err != nil {
			log.Fatalln(err)
		}
	}

	if isChangelogScheme(args) {
		if body == "" {
			notes, notesErr := changelogNotes(args, base)
			if notesErr != nil {
				log.Fatalln(notesErr)
			}
			body = notes
		}
		// Idempotent: a version that already has a release (draft or published)
		// or a tag is not released again. Candidates get a new number every time.
		if !args.Prerelease {
			if done := changelogAlreadyReleased(scmLayer, version); done {
				return
			}
		}
	}

	if args.DryRun {
		log.Infof("Would create new release with version: %s\n", version)

		fmt.Println("### Info output:")
		fmt.Printf("Would create new release with version: %s\n", version)

		var envVars = []ces.KeyValue{
			{Name: "RELCTL_VERSION", Value: version},
			{Name: "RELCTL_NEXT_VERSION", Value: version},
		}

		if err := scmLayer.CES.ExportAsEnv(envVars); err != nil {
			log.Fatalf("could not export env variables: %v", err)
		}
	} else {
		log.Infof("Writing new release: %s\n", version)
		fmt.Println("### Info output:")
		fmt.Printf("Writing new release with version: %s\n", version)

		createdRelease, err := scmLayer.CreateRelease(version, releasePrefix, releaseTarget(args), body, args.Prerelease)
		if err != nil {
			log.Fatalln(err)
		}
		log.Infof("Create release successful. ID: %d", createdRelease.ID)
		fmt.Printf("Create release successful. ID: %d", createdRelease.ID)

		var envVars = []ces.KeyValue{
			{Name: "RELCTL_VERSION", Value: version},
			{Name: "RELCTL_NEXT_VERSION", Value: version},
			{Name: "RELCTL_RELEASE_ID", Value: fmt.Sprintf("%d", createdRelease.ID)},
		}

		if err := scmLayer.CES.ExportAsEnv(envVars); err != nil {
			log.Fatalf("could not export env variables: %v", err)
		}
	}
}

func ReleasePublish(args *ReleaseArgs, releaseID int64, assetsStr []string) {
	scmLayer, err := scmportal.LoadSCMPortalLayer()
	if err != nil {
		log.Fatalln(err)
	}

	var version, releasePrefix string

	if releaseID == 0 {
		version, releasePrefix, err = argsToVersion(scmLayer, args)
		if err != nil {
			log.Fatalln(err)
		}

		if isChangelogScheme(args) {
			existing, draft, findErr := scmLayer.FindReleaseByTag(version)
			if findErr != nil {
				log.Fatalln(findErr)
			}
			switch {
			case existing != nil && !draft:
				log.Infof("release %s is already published, nothing to do", version)
				return
			case existing != nil:
				releaseID = existing.ID
			}
		}
	}

	var assets []uploadasset.UploadAsset
	if len(assetsStr) > 0 {
		for _, asset := range assetsStr {
			newAsset, err := uploadasset.GetAsset(asset)
			if err != nil {
				log.Fatalln("not all specified assets available, please check", err)
			}
			assets = append(assets, newAsset)
		}
	}
	log.Infof("will upload %d assets", len(assetsStr))

	body, err := tools.ReadFileToString(args.Body)
	if err != nil {
		log.Warnf("could not process the given body: %v", err)
	}
	if body == "" && version != "" && isChangelogScheme(args) {
		if notes, notesErr := changelogNotes(args, version); notesErr == nil {
			body = notes
		}
	}

	if args.DryRun {
		log.Infof("Would publishing release: %s", version)
	} else {
		log.Infof("Publishing release: %s - %d", version, releaseID)
		_, err := scmLayer.PublishRelease(version, releasePrefix, args.ReleaseBranch, body, releaseID, assets)
		if err != nil {
			log.Fatalln(err)
		}

		/* var envVars []ces.KeyValue
		for i, ra := range relAssets {
			// export Download URL to env. See: #53
			envVars = append(envVars, ces.KeyValue{Name: fmt.Sprintf("RELCTL_ARTIFACT_%d_URL", i+1), Value: *ra.BrowserDownloadURL})
		}

		if len(envVars) > 0 {
			if err := scmLayer.CES.ExportAsEnv(envVars); err != nil {
				log.Fatalln("could not export env variables: %v", err)
			}
		} */
	}
}

// resolveVersionScheme determines the active scheme from flag → config → default.
func resolveVersionScheme(args *ReleaseArgs) (string, error) {
	if args.VersionScheme != "" {
		return args.VersionScheme, nil
	}
	cfgPath := args.ConfigFile
	if cfgPath == "" {
		cfgPath = relctlconfig.DefaultConfigFile
	}
	cfg, err := relctlconfig.Load(cfgPath)
	if err != nil {
		return "", fmt.Errorf("loading relctl config: %w", err)
	}
	return cfg.VersionScheme, nil
}

func argsToVersion(scmLayer *scmportal.SCMLayer, args *ReleaseArgs) (version, releasePrefix string, err error) {
	if args.ReleasePrefix != "" {
		releasePrefix = args.ReleasePrefix
	} else {
		releasePrefix = "Release"
	}

	// --- CalVer path (provider-agnostic, no SCM API required) ---
	scheme, schemeErr := resolveVersionScheme(args)
	if schemeErr != nil {
		return "", "", schemeErr
	}
	if scheme == relctlconfig.SchemeCalVer {
		if args.Version != "" {
			// Explicit --version overrides CalVer calculation.
			return args.Version, releasePrefix, nil
		}
		repoPath, pathErr := tools.RepoRoot()
		if pathErr != nil {
			return "", "", fmt.Errorf("calver: could not determine repo root: %w", pathErr)
		}
		cv := versioning.NewCalVerScheme()
		version, err = cv.NextVersion(versioning.ReleaseContext{RepoPath: repoPath})
		return version, releasePrefix, err
	}
	// --- end CalVer path ---

	// --- Conventional Commits path ---
	if scheme == relctlconfig.SchemeConventionalCommits {
		if args.Version != "" {
			// Explicit --version overrides the conventional-commits calculation.
			return args.Version, releasePrefix, nil
		}
		repoPath, pathErr := tools.RepoRoot()
		if pathErr != nil {
			return "", "", fmt.Errorf("conventional-commits: could not determine repo root: %w", pathErr)
		}
		version, err = conventionalCommitsVersion(scmLayer, conventionalcommits.CommitMessagesSince, repoPath)
		return version, releasePrefix, err
	}
	// --- end Conventional Commits path ---

	// --- Changelog path ---
	if scheme == relctlconfig.SchemeChangelog {
		if args.Version != "" {
			return args.Version, releasePrefix, nil
		}
		if args.Prerelease {
			// A candidate may be built from a heading that is not dated yet.
			version, err = changelogHeadingVersionFromFile(args)
			return version, releasePrefix, err
		}
		version, _, err = changelogReleaseFromFile(args)
		return version, releasePrefix, err
	}
	// --- end Changelog path ---

	if args.Version != "" && args.PatchLevel != "" {
		parsedPatchLevel, err := semver.ParsePatchLevel(args.PatchLevel)
		if err != nil && err != semver.ErrUseMinimalPatchVersion {
			log.Fatalln(err)
		}
		version, err = semver.IncreaseVersion(parsedPatchLevel, args.Version)
		if err != nil {
			log.Fatalln(err)
		}

	} else if args.Version != "" && args.PatchLevel == "" {
		version = args.Version

	} else if args.Hotfix {
		release, err := scmLayer.GetLatestPublishedRelease()
		if err != nil {
			log.Fatalln(err)
		}

		version, err = semver.IncreaseVersion(semver.Bugfix, release.TagName)
		if err != nil {
			log.Fatalln(err)
		}

		if args.ReleasePrefix == "" {
			releasePrefix = "Hotfix"
		}

	} else {
		// if no merge commit sha is provided, the pull request number should either be specified or evaluated from the merge message (fallback)
		if args.MergeCommitSHA == "" {
			log.Infoln("no merge commit sha is given, eval...")
			err := evalPrNumber(&args.PrNumber)
			if err != nil {
				log.Fatalln(err)
			}
		}

		prInfos, err := scmLayer.GetPrInfos(args.PrNumber, args.MergeCommitSHA)
		if err != nil {
			log.Fatalln(err)
		}

		version = prInfos.NextVersion

		if err = prInfosToEnv(scmLayer, prInfos); err != nil {
			log.Fatalln(err)
		}
	}
	return
}

func evalPrNumber(override *int) (err error) {
	if *override != 0 {
		return nil
	}

	*override, err = getPrFromMergeMessage()
	if err != nil {
		return err
	}
	return
}

func getPrFromMergeMessage() (pr int, err error) {
	regex := `.*#([0-9]+).*`
	r := regexp.MustCompile(regex)

	subject, err := tools.GitLogSubject()
	if err != nil {
		return 0, fmt.Errorf("reading git log: %w", err)
	}

	mergeMessage := r.FindStringSubmatch(subject)
	if len(mergeMessage) > 1 {
		return strconv.Atoi(mergeMessage[1])
	}
	return 0, errors.New("No PR found in merge message pls make shure this regex matches: " + regex +
		"\nExample: Merge pull request #3 from some-orga/feature/awesome-feature" +
		"\nAlternativly provide the PR-Number by adding the argument -number <int>")
}

func isChangelogScheme(args *ReleaseArgs) bool {
	scheme, err := resolveVersionScheme(args)
	return err == nil && scheme == relctlconfig.SchemeChangelog
}

// changelogAlreadyReleased reports whether version already has a release or
// tag. In that case it exports the version (and the release id, if any) and
// the caller must not create another release.
func changelogAlreadyReleased(scmLayer *scmportal.SCMLayer, version string) bool {
	envVars := []ces.KeyValue{
		{Name: "RELCTL_VERSION", Value: version},
		{Name: "RELCTL_NEXT_VERSION", Value: version},
	}

	existing, draft, err := scmLayer.FindReleaseByTag(version)
	if err != nil {
		log.Fatalln(err)
	}
	if existing != nil {
		if draft {
			log.Infof("a draft release for %s already exists (id %d), reusing it", version, existing.ID)
		} else {
			log.Infof("release %s already exists, nothing to release", version)
		}
		fmt.Printf("Release %s already exists (id %d), nothing to create\n", version, existing.ID)
		envVars = append(envVars, ces.KeyValue{Name: "RELCTL_RELEASE_ID", Value: fmt.Sprintf("%d", existing.ID)})
		if err := scmLayer.CES.ExportAsEnv(envVars); err != nil {
			log.Fatalf("could not export env variables: %v", err)
		}
		return true
	}

	tagExists, err := scmLayer.TagExists(version)
	if err != nil {
		log.Fatalln(err)
	}
	if tagExists {
		log.Infof("tag %s already exists, nothing to release", version)
		fmt.Printf("Tag %s already exists, nothing to release\n", version)
		if err := scmLayer.CES.ExportAsEnv(envVars); err != nil {
			log.Fatalf("could not export env variables: %v", err)
		}
		return true
	}
	return false
}

// nextReleaseCandidate returns the version of the next release candidate of
// base (base-rc.N), numbered after the existing tags and releases.
func nextReleaseCandidate(list func(prefix string) ([]string, error), base string) (string, error) {
	existing, err := list(prerelease.Prefix(base))
	if err != nil {
		return "", fmt.Errorf("release candidate: listing existing candidates: %w", err)
	}
	return prerelease.Tag(base, prerelease.Next(base, existing)), nil
}

// versionAlreadyReleased reports whether a published release or a tag exists for version.
func versionAlreadyReleased(scmLayer *scmportal.SCMLayer, version string) bool {
	existing, draft, err := scmLayer.FindReleaseByTag(version)
	if err != nil {
		log.Fatalln(err)
	}
	if existing != nil && !draft {
		return true
	}
	tagExists, err := scmLayer.TagExists(version)
	if err != nil {
		log.Fatalln(err)
	}
	return tagExists
}

// releaseTarget returns what the release's tag will point to. GitHub creates the
// tag when the release is published, at the target given when the release was
// created. A branch name would resolve to the head of the branch at publish time,
// which may be a newer commit than the one the version was computed from, so the
// merge commit wins when it is known.
func releaseTarget(args *ReleaseArgs) string {
	if args.MergeCommitSHA != "" {
		return args.MergeCommitSHA
	}
	return args.ReleaseBranch
}
