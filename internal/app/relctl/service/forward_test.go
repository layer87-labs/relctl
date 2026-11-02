package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecideForward(t *testing.T) {
	require.Equal(t, forwardNothing, decideForward("identical"))
	require.Equal(t, forwardNothing, decideForward("behind"), "development already has everything")
	require.Equal(t, forwardFastForward, decideForward("ahead"))
	require.Equal(t, forwardPullRequest, decideForward("diverged"))
	require.Equal(t, forwardPullRequest, decideForward("something unknown"), "unknown means: do not touch the branch")
}

func TestForwardAction(t *testing.T) {
	require.Equal(t, "nothing", forwardNothing.String())
	require.Equal(t, "fast-forward", forwardFastForward.String())
	require.Equal(t, "pull-request", forwardPullRequest.String())
}

func TestForwardBranches(t *testing.T) {
	def := func(b string, err error) func() (string, error) {
		return func() (string, error) { return b, err }
	}
	cfg := t.TempDir() + "/missing.yaml" // no config: defaults

	from, to, err := forwardBranches(&ForwardArgs{ConfigFile: cfg}, def("dev", nil))
	require.NoError(t, err)
	require.Equal(t, "main", from)
	require.Equal(t, "dev", to, "the repository's default branch is the development branch")

	from, to, err = forwardBranches(&ForwardArgs{ConfigFile: cfg, From: "release", To: "next"}, def("dev", nil))
	require.NoError(t, err)
	require.Equal(t, "release", from)
	require.Equal(t, "next", to, "flags win")

	_, _, err = forwardBranches(&ForwardArgs{ConfigFile: cfg}, def("main", nil))
	require.ErrorContains(t, err, "nothing to forward")

	_, _, err = forwardBranches(&ForwardArgs{ConfigFile: cfg}, def("", errors.New("api down")))
	require.ErrorContains(t, err, "api down")
}
