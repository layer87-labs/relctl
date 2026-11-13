package service

import (
	"fmt"
	"net/http"

	"errors"
	"github.com/google/go-github/v86/github"
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

type fakeForwardAPI struct {
	compareErr error
	exists     bool
	existsErr  error
	created    map[string]string
	createErr  error
}

func (f *fakeForwardAPI) CompareStatus(base, head string) (string, error) {
	if f.compareErr != nil {
		return "", f.compareErr
	}
	return "ahead", nil
}
func (f *fakeForwardAPI) BranchExists(string) (bool, error) { return f.exists, f.existsErr }
func (f *fakeForwardAPI) BranchSHA(string) (string, error)  { return "abc1234567", nil }
func (f *fakeForwardAPI) CreateBranch(b, sha string) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = map[string]string{b: sha}
	return nil
}

func notFound() error {
	return fmt.Errorf("compare: %w", &github.ErrorResponse{Response: &http.Response{StatusCode: http.StatusNotFound}})
}

func TestCompareOrRecreate(t *testing.T) {
	t.Run("compare works: nothing is created", func(t *testing.T) {
		f := &fakeForwardAPI{}
		status, recreated, err := compareOrRecreate(f, "main", "dev", false)
		require.NoError(t, err)
		require.Equal(t, "ahead", status)
		require.False(t, recreated)
		require.Nil(t, f.created)
	})
	t.Run("404 and dev missing: dev is created at the main head", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: notFound()}
		status, recreated, err := compareOrRecreate(f, "main", "dev", false)
		require.NoError(t, err)
		require.Equal(t, "identical", status)
		require.True(t, recreated)
		require.Equal(t, map[string]string{"dev": "abc1234567"}, f.created)
	})
	t.Run("dry run does not create", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: notFound()}
		_, recreated, err := compareOrRecreate(f, "main", "dev", true)
		require.NoError(t, err)
		require.False(t, recreated)
		require.Nil(t, f.created)
	})
	t.Run("404 but dev exists: error stays", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: notFound(), exists: true}
		_, _, err := compareOrRecreate(f, "main", "dev", false)
		require.Error(t, err)
		require.Nil(t, f.created)
	})
	t.Run("other errors stay", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: fmt.Errorf("boom")}
		_, _, err := compareOrRecreate(f, "main", "dev", false)
		require.ErrorContains(t, err, "boom")
		require.Nil(t, f.created)
	})
	t.Run("create failure is returned", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: notFound(), createErr: errors.New("denied")}
		_, _, err := compareOrRecreate(f, "main", "dev", false)
		require.ErrorContains(t, err, "denied")
	})
}
