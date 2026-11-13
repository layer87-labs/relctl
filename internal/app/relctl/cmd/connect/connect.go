package connect

import (
	"github.com/layer87-labs/relctl/internal/app/relctl/connect"
	"github.com/layer87-labs/relctl/internal/pkg/rcpersist"

	"github.com/spf13/cobra"
)

var (
	Token      string
	Server     string
	Repository string
)

var Cmd = &cobra.Command{
	Use:   "connect",
	Short: "Store server and repository of a GitHub or GitHub Enterprise repository in .relctl-connection.yaml",
	Long: `Verifies the connection to GitHub or GitHub Enterprise and stores server and repository
in .relctl-connection.yaml. The token is never stored: provide it as GITHUB_TOKEN on every call.
The file holds no secret and may be committed.
GitLab is not supported as source control system (open an issue if you need it)
Useful where relctl cannot detect the pipeline (GitLab CI with a GitHub repository, local shell).`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "Connect to a GitHub or GitHub Enterprise instance",
	Run: func(cmd *cobra.Command, args []string) {
		connect.UpdateRcFileForGitHub(Server, Repository, Token)
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "remove the connection file (and a legacy .awesomerc.yaml)",
	Run: func(cmd *cobra.Command, args []string) {
		rcpersist.RemoveRcFile()
	},
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "verify current connection is working",
	Run: func(cmd *cobra.Command, args []string) {
		connect.CheckConnection()
	},
}

func init() {
	// commands
	Cmd.AddCommand(githubCmd)
	Cmd.AddCommand(removeCmd)
	Cmd.AddCommand(checkCmd)

	// Flags
	githubCmd.Flags().StringVarP(&Server, "server", "s", "https://github.com", "github instance to connect, default: https://github.com")
	githubCmd.Flags().StringVarP(&Repository, "repository", "r", "", "(required) repo eg.: octo-org/octo-repo")
	githubCmd.Flags().StringVarP(&Token, "token", "t", "", "token used to verify the connection, not stored (default: $GITHUB_TOKEN)")

	_ = githubCmd.MarkFlagRequired("repository")

	// exclusive Flags
}
