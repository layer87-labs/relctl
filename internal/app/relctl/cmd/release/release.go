package release

import (
	"os"
	"strconv"

	"github.com/layer87-labs/relctl/internal/app/relctl/service"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var (
	releaseArgs service.ReleaseArgs
	releaseID   int64
	assets      []string
)

var Cmd = &cobra.Command{
	Use:   "release",
	Short: "Manage GitHub releases with ease",
	Long:  `The release command is used to manage GitHub releases. It provides subcommands to create and publish releases, allowing you to automate the release process and integrate it into CI/CD workflows. Use this command to streamline the release tagging and deployment of your software projects.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new GitHub release",
	Run: func(cmd *cobra.Command, args []string) {
		service.ReleaseCreate(&releaseArgs)
	},
}

var gateArgs service.GateArgs

var gateCmd = &cobra.Command{
	Use:   "gate",
	Short: "Check that the changelog may be released",
	Long: `Check that the topmost changelog heading may be released.

The gate passes only if
  1. the topmost heading is a valid, dated version with release notes,
  2. that version has not been released yet (no published release, no tag), and
  3. (with --require-candidate, the default) a release candidate tag
     X.Y.Z-rc.N points to the commit given with --sha (default: HEAD).

All checks are reported even if an earlier one fails. The command exits with a
non-zero status if any check fails, so it can be used as a required check on the
pull request that releases.

    relctl release gate --sha "$PR_HEAD_SHA"
    relctl release gate --require-candidate=false`,
	Run: func(cmd *cobra.Command, args []string) {
		gateArgs.ConfigFile = releaseArgs.ConfigFile // persistent --config flag
		gateArgs.Component = releaseArgs.Component   // persistent --component flag
		service.ReleaseGate(&gateArgs)
	},
}

var changedArgs service.ChangedArgs

var changedCmd = &cobra.Command{
	Use:   "changed",
	Short: "Report whether a component changed since its latest release",
	Long: `Report whether a file that belongs to a component changed since the component's
latest release.

The paths of a component are listed under components.<name>.paths in .relctl.yaml
(glob patterns; "**" matches any number of directories). The latest release is the
highest release tag of the component that is reachable from HEAD; release candidates
do not count. A component without a release counts as changed.

The command needs the full history and the tags (fetch-depth: 0) and no access to
GitHub. It exports RELCTL_CHANGED (true or false), RELCTL_CHANGED_FILES,
RELCTL_COMPONENT and RELCTL_LATEST_VERSION, so a pipeline builds only what changed:

    relctl release changed --component whisper
    relctl release changed --component whisper --since 202611.2.4`,
	Run: func(cmd *cobra.Command, args []string) {
		changedArgs.ConfigFile = releaseArgs.ConfigFile // persistent --config flag
		changedArgs.Component = releaseArgs.Component   // persistent --component flag
		service.ReleaseChanged(&changedArgs)
	},
}

var forwardArgs service.ForwardArgs

var forwardCmd = &cobra.Command{
	Use:   "forward",
	Short: "Bring the production branch back into the development branch after a release",
	Long: `Bring the released state of the production branch back into the development branch.

The branches come from .relctl.yaml (production_branch, default "main", and
development_branch, default: the default branch of the repository) or from the
flags --from and --to.

relctl fast-forwards the development branch if that is possible. If the
development branch has moved on, or the token may not push to it (branch rules),
relctl opens a pull request instead. Merge that pull request with a merge commit,
do not squash it. The command is idempotent: nothing happens if the development
branch already contains the production branch or if the pull request is open.

    relctl release forward
    relctl release forward --from main --to dev --dry-run`,
	Run: func(cmd *cobra.Command, args []string) {
		forwardArgs.ConfigFile = releaseArgs.ConfigFile // persistent --config flag
		forwardArgs.DryRun = releaseArgs.DryRun         // persistent --dry-run flag
		service.ReleaseForward(&forwardArgs)
	},
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a recently created GitHub release",
	Long: `Publish a recently created GitHub release

Publish will publish a previously created release to GitHub. The release body
can be written in markdown and will be rendered as such on GitHub. Additionally,
any number of assets can be uploaded, including but not limited to zip and tgz
files. The assets will be uploaded to GitHub and linked to the release.

The assets can be specified as a list of local file paths.

The zip and tgz mechanisms can be used by specifying the path to a directory
containing the files to be uploaded. The directory will be zipped or tarred and
gzipped and uploaded to GitHub as a single asset. The name of the asset will be
the name of the directory with the appropriate extension appended. For example,
if the directory is named "myfiles", the asset will be named "myfiles.zip" or "myfiles.tgz".

Example of using the release publish command

To publish a release with a specific release ID and assets, use the following command:

    relctl release publish --release-id 12345 \
        --asset "file=out/relctl_v1.0.0_amd64" \
        --asset "file=out/relctl_v1.0.0_arm64"

If the release ID is not provided, the command will look for the 'RELCTL_RELEASE_ID' environment variable:

    export RELCTL_RELEASE_ID=12345
    relctl release publish \
        --asset "file=out/relctl_v1.0.0_amd64" \
        --asset "file=out/relctl_v1.0.0_arm64"

You can also publish a release with a directory as a zip asset:

    relctl release publish --release-id 12345 \
        --asset "zip=out/myfiles"

The assets should be specified as local file paths to be uploaded to the GitHub release.`,
	Run: func(cmd *cobra.Command, args []string) {

		if releaseID == 0 {
			log.Traceln("looking of env variable 'RELCTL_RELEASE_ID' since flag is not set")
			releaseIDStr, releaseIDBool := os.LookupEnv("RELCTL_RELEASE_ID")
			if releaseIDBool {
				releaseIDTmp, err := strconv.ParseInt(releaseIDStr, 10, 64)
				if err != nil {
					log.Fatalln(err)
				} else {
					releaseID = releaseIDTmp
				}
			} else {
				log.Fatalln("no release ID provided")
			}
		}

		service.ReleasePublish(&releaseArgs, releaseID, assets)
	},
}

func init() {
	// commands
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(publishCmd)
	Cmd.AddCommand(gateCmd)
	Cmd.AddCommand(forwardCmd)
	Cmd.AddCommand(changedCmd)

	// Flags
	Cmd.PersistentFlags().IntVar(&releaseArgs.PrNumber, "prnumber", 0, "overwrite the issue number")
	Cmd.PersistentFlags().BoolVarP(&releaseArgs.DryRun, "dry-run", "", false, "make dry-run before writing version to Git by calling it")
	Cmd.PersistentFlags().BoolVarP(&releaseArgs.Hotfix, "hotfix", "", false, "create a hotfix release")
	Cmd.PersistentFlags().StringVar(&releaseArgs.MergeCommitSHA, "merge-sha", "", "set the merge sha")
	Cmd.PersistentFlags().StringVar(&releaseArgs.ReleaseBranch, "release-branch", "", "set release branch (default: git default)")
	Cmd.PersistentFlags().StringVar(&releaseArgs.ReleasePrefix, "release-prefix", "", "set a custom release prefix (default -> Release or Hotfix)")
	Cmd.PersistentFlags().StringVar(&releaseArgs.Version, "version", "", "override version to Update")
	Cmd.PersistentFlags().StringVarP(&releaseArgs.Body, "body", "b", "", "custom release message (markdown string or file)")
	Cmd.PersistentFlags().StringVarP(&releaseArgs.PatchLevel, "patch-level", "l", "", "predefine patch level of version to Update")
	Cmd.PersistentFlags().StringVar(&releaseArgs.VersionScheme, "version-scheme", "", "versioning scheme to use: semver (default), calver, conventional-commits or changelog (overrides .relctl.yaml)")
	Cmd.PersistentFlags().BoolVar(&releaseArgs.Prerelease, "prerelease", false, "create a release candidate (X.Y.Z-rc.N, marked as prerelease) for the computed version")
	Cmd.PersistentFlags().StringVar(&releaseArgs.ConfigFile, "config", "", "path to .relctl.yaml config file")
	Cmd.PersistentFlags().StringVar(&releaseArgs.Component, "component", "", "release one component of the repository (key components in .relctl.yaml): own tags, own latest release, own counter")

	changedCmd.Flags().StringVar(&changedArgs.Since, "since", "", "compare with this commit or tag instead of the latest release of the component")

	gateCmd.Flags().StringVar(&gateArgs.Sha, "sha", "", "commit that must have a release candidate (default: HEAD)")
	gateCmd.Flags().BoolVar(&gateArgs.RequireCandidate, "require-candidate", true, "require a release candidate tag for the commit")

	forwardCmd.Flags().StringVar(&forwardArgs.From, "from", "", "production branch (default: production_branch of .relctl.yaml, or main)")
	forwardCmd.Flags().StringVar(&forwardArgs.To, "to", "", "development branch (default: development_branch of .relctl.yaml, or the repository's default branch)")

	// exclusive Flags
	publishCmd.Flags().Int64VarP(&releaseID, "release-id", "", 0, "publish an early defined release (also looking for env RELCTL_RELEASE_ID)")
	publishCmd.Flags().StringArrayVarP(&assets, "asset", "a", []string{}, "add an asset to the release, can be specified multiple times.")
}
