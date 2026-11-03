package pullrequest

import (
	"github.com/layer87-labs/relctl/internal/app/relctl/service"

	"github.com/spf13/cobra"
)

var (
	number         int
	formatOut      string
	mergeCommitSha string
	versionScheme  string
	configFile     string
)

var Cmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage GitHub pull requests",
	Long:  `The pull request command is used to manage GitHub pull requests. It provides subcommands to get pull request info, allowing you to get all infos about a pull request in GitHub.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get pull request info and the version the pull request would release",
	Long: `Prints the pull request and the version it would release, and exports them as
RELCTL_* variables. What decides the version depends on the version scheme:

  semver                the branch prefix and the comment commands of the pull request
  calver                the date of the release
  conventional-commits  the commit messages since the last release
  changelog             the topmost heading of the changelog (it may not be dated yet)`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = formatOut // reserved: output formats are not implemented yet
		service.PrintPRInfos(&service.PrInfoArgs{
			Number:         number,
			MergeCommitSHA: mergeCommitSha,
			VersionScheme:  versionScheme,
			ConfigFile:     configFile,
		})
	},
}

func init() {
	// commands
	Cmd.AddCommand(infoCmd)

	// Flags
	infoCmd.Flags().IntVarP(&number, "number", "n", 0, "overwrite the issue number")
	infoCmd.Flags().StringVarP(&mergeCommitSha, "merge-commit-sha", "c", "", "send a given merge commit sha")
	infoCmd.Flags().StringVar(&versionScheme, "version-scheme", "", "versioning scheme: semver, calver, conventional-commits or changelog (overrides .relctl.yaml)")
	infoCmd.Flags().StringVar(&configFile, "config", "", "path to .relctl.yaml config file")
	// needs to be implemented
	// infoCmd.Flags().StringVarP(&formatOut, "output", "o", "", "define output available: [json, yaml]")
}
