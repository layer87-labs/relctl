package service

import (
	"fmt"
	"strings"

	"github.com/layer87-labs/relctl/internal/app/relctl/ces"
	"github.com/layer87-labs/relctl/internal/pkg/rcpersist"
)

// reportBreakingWarning makes a breaking_markers warning visible: on stdout, and
// as a warning annotation in the run summary on GitHub Actions.
func reportBreakingWarning(c ces.CES, warning string) {
	if warning == "" {
		return
	}
	fmt.Printf("Warning: %s\n", warning)
	if c.Type == rcpersist.CESTypeGitHubRunner {
		fmt.Printf("::warning title=Breaking change without a major bump::%s\n", warning)
	}
}

// withBreakingWarning puts a breaking_markers warning at the top of the release notes.
func withBreakingWarning(body, warning string) string {
	if warning == "" {
		return body
	}
	note := "> [!WARNING]\n> **Breaking change.** " + warning + "\n"
	if strings.TrimSpace(body) == "" {
		return note
	}
	return note + "\n" + body
}
