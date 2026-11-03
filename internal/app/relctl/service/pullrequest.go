package service

import (
	"fmt"
	"strconv"

	"github.com/layer87-labs/relctl/internal/app/relctl/ces"
	scmportal "github.com/layer87-labs/relctl/internal/app/relctl/scm-portal"
	"github.com/layer87-labs/relctl/internal/pkg/relctlconfig"

	log "github.com/sirupsen/logrus"
)

// PrInfoArgs are the arguments of `relctl pr info`.
type PrInfoArgs struct {
	Number         int
	MergeCommitSHA string
	// VersionScheme overrides the config-file value; empty means "use config / default".
	VersionScheme string
	// ConfigFile is the path to .relctl.yaml; empty means default lookup.
	ConfigFile string
}

func PrintPRInfos(args *PrInfoArgs) {
	scmLayer, err := scmportal.LoadSCMPortalLayer()
	if err != nil {
		log.Fatalln(err)
	}

	log.Infof("detected ces type: %s", scmLayer.CES.Type)

	scheme, err := resolveVersionScheme(&ReleaseArgs{VersionScheme: args.VersionScheme, ConfigFile: args.ConfigFile})
	if err != nil {
		log.Fatalln(err)
	}

	number := args.Number
	if args.MergeCommitSHA == "" {
		if err = evalPrNumber(&number); err != nil {
			log.Fatalln(err)
		}
		log.Infof("evaluated pull request number %d", number)
	}

	prInfos, err := scmLayer.GetPrInfos(number, args.MergeCommitSHA, scheme)
	if err != nil {
		log.Fatalln(err)
	}

	// SemVer takes the version from the branch prefix and the comment commands
	// (done by GetPrInfos). Every other scheme computes it from the repository.
	if scheme != relctlconfig.SchemeSemVer {
		next, errNext := previewVersion(scmLayer, scheme, args)
		if errNext != nil {
			log.Fatalln(errNext)
		}
		prInfos.NextVersion = next
	}

	if err := prInfosToEnv(scmLayer, prInfos, scheme); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("### Info output:")
	fmt.Printf("Pull Request: %d\n", prInfos.Number)
	fmt.Printf("Version scheme: %s\n", scheme)
	fmt.Printf("Latest release version: %s\n", prInfos.LatestVersion)
	if scheme == relctlconfig.SchemeSemVer {
		fmt.Printf("Patch level: %s\n", prInfos.PatchLevel)
	}
	fmt.Printf("Possible new release version: %s\n", prInfos.NextVersion)
}

// previewVersion returns the version a pull request would release under a scheme
// other than SemVer. The version does not depend on the pull request itself but on
// the repository state the pipeline checked out.
func previewVersion(scmLayer *scmportal.SCMLayer, scheme string, args *PrInfoArgs) (string, error) {
	releaseArgs := &ReleaseArgs{
		VersionScheme:  scheme,
		ConfigFile:     args.ConfigFile,
		PrNumber:       args.Number,
		MergeCommitSHA: args.MergeCommitSHA,
	}
	if scheme == relctlconfig.SchemeChangelog {
		// A pull request usually carries a heading that is not dated yet.
		return changelogHeadingVersionFromFile(releaseArgs)
	}
	version, _, err := argsToVersion(scmLayer, releaseArgs)
	return version, err
}

func prInfosToEnv(scmLayer *scmportal.SCMLayer, prInfos *scmportal.PrMrRequestInfos, scheme string) error {
	var envVars = []ces.KeyValue{
		{Name: "RELCTL_PR", Value: strconv.Itoa(prInfos.Number)},
		{Name: "RELCTL_PR_SHA", Value: prInfos.Sha},
		{Name: "RELCTL_PR_SHA_SHORT", Value: prInfos.ShaShort},
		{Name: "RELCTL_PR_BRANCH", Value: prInfos.BranchName},
		{Name: "RELCTL_MERGE_COMMIT_SHA", Value: prInfos.MergeCommitSha},
		{Name: "RELCTL_OWNER", Value: prInfos.Owner},
		{Name: "RELCTL_REPO", Value: prInfos.Repo},
		{Name: "RELCTL_PATCH_LEVEL", Value: string(prInfos.PatchLevel)},
		{Name: "RELCTL_VERSION", Value: prInfos.NextVersion},
		{Name: "RELCTL_NEXT_VERSION", Value: prInfos.NextVersion},
		{Name: "RELCTL_LATEST_VERSION", Value: prInfos.LatestVersion},
		{Name: "RELCTL_VERSION_SCHEME", Value: scheme},
	}

	if err := scmLayer.CES.ExportAsEnv(envVars); err != nil {
		return fmt.Errorf("could not export env variables: %v", err)
	}
	return nil
}
