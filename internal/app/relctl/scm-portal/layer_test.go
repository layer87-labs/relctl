package scmportal

import (
	"testing"

	"github.com/layer87-labs/relctl/internal/pkg/rcpersist"
	"github.com/stretchr/testify/require"
)

func TestUnsupportedSCMError(t *testing.T) {
	err := unsupportedSCMError(rcpersist.SCMPortalTypeGitLab)
	require.ErrorContains(t, err, "GitLab is not supported")
	require.ErrorContains(t, err, "open an issue")
	require.ErrorContains(t, err, IssuesURL)
	require.ErrorContains(t, err, "relctl connect github", "hint for GitLab CI with a GitHub repository")

	err = unsupportedSCMError("")
	require.ErrorContains(t, err, "no CI environment")
	require.ErrorContains(t, err, "relctl connect github")

	err = unsupportedSCMError("bitbucket")
	require.ErrorContains(t, err, "bitbucket")
	require.ErrorContains(t, err, IssuesURL)
}
