package docs

import (
	"fmt"

	"github.com/layer87-labs/relctl/internal/pkg/clidocs"

	"github.com/spf13/cobra"
)

var out string

// New returns the hidden `docs` command. It needs the root command to document, which is
// why it is created by the root instead of registering itself.
func New(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "docs",
		Short:  "Write the command line reference as Markdown",
		Hidden: true,
		Long: `Write the command line reference of relctl as Markdown pages for a Docusaurus
site: one page per command plus an index. The output is deterministic, so the
documentation can be regenerated on every release and changes only when the CLI does.`,
		Example: `relctl docs --out docs/relctl/cli`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := clidocs.Generate(root, out); err != nil {
				return err
			}
			fmt.Printf("command line reference written to %s\n", out)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "directory for the pages (required)")
	_ = cmd.MarkFlagRequired("out")
	return cmd
}
