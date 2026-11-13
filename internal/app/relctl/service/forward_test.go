package service

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/google/go-github/v86/github"
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

	from, to, configured, err := forwardBranches(&ForwardArgs{ConfigFile: cfg}, def("dev", nil))
	require.NoError(t, err)
	require.Equal(t, "main", from)
	require.Equal(t, "dev", to, "the repository's default branch is the development branch")
	require.False(t, configured, "the default branch is not the configured development_branch")

	from, to, configured, err = forwardBranches(&ForwardArgs{ConfigFile: cfg, From: "release", To: "next"}, def("dev", nil))
	require.NoError(t, err)
	require.Equal(t, "release", from)
	require.Equal(t, "next", to, "flags win")
	require.False(t, configured)

	cfgFile := t.TempDir() + "/relctl.yaml"
	require.NoError(t, os.WriteFile(cfgFile, []byte("development_branch: dev\n"), 0o600))
	_, to, configured, err = forwardBranches(&ForwardArgs{ConfigFile: cfgFile}, def("x", nil))
	require.NoError(t, err)
	require.Equal(t, "dev", to)
	require.True(t, configured, "development_branch from the config")
	_, to, configured, err = forwardBranches(&ForwardArgs{ConfigFile: cfgFile, To: "deve"}, def("x", nil))
	require.NoError(t, err)
	require.Equal(t, "deve", to)
	require.False(t, configured, "--to is never trusted")

	_, _, _, err = forwardBranches(&ForwardArgs{ConfigFile: cfg}, def("main", nil))
	require.ErrorContains(t, err, "nothing to forward")

	_, _, _, err = forwardBranches(&ForwardArgs{ConfigFile: cfg}, def("", errors.New("api down")))
	require.ErrorContains(t, err, "api down")
}

type fakeForwardAPI struct {
	compareErr error
	exists     []bool // answers of BranchExists in order; the last one repeats
	existsErr  error
	createErr  error
	created    map[string]string
}

func (f *fakeForwardAPI) CompareStatus(string, string) (string, error) {
	if f.compareErr != nil {
		return "", f.compareErr
	}
	return "ahead", nil
}

func (f *fakeForwardAPI) BranchExists(string) (bool, error) {
	if f.existsErr != nil {
		return false, f.existsErr
	}
	v := f.exists[0]
	if len(f.exists) > 1 {
		f.exists = f.exists[1:]
	}
	return v, nil
}

func (f *fakeForwardAPI) BranchSHA(branch string) (string, error) {
	if branch != "main" {
		return "", fmt.Errorf("asked for the head of %q, want main", branch)
	}
	return "abc1234567", nil
}

func (f *fakeForwardAPI) CreateBranch(b, sha string) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = map[string]string{b: sha}
	return nil
}

func httpErr(code int) error {
	return fmt.Errorf("api: %w", &github.ErrorResponse{Response: &http.Response{StatusCode: code}})
}

func TestCompareOrRecreate(t *testing.T) {
	t.Run("compare works: nothing is created", func(t *testing.T) {
		f := &fakeForwardAPI{}
		status, rec, err := compareOrRecreate(f, "main", "dev", true, false)
		require.NoError(t, err)
		require.Equal(t, "ahead", status)
		require.Empty(t, rec)
		require.Nil(t, f.created)
	})
	t.Run("404 and configured dev missing: dev is created at the main head", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: httpErr(404), exists: []bool{false}}
		_, rec, err := compareOrRecreate(f, "main", "dev", true, false)
		require.NoError(t, err)
		require.Equal(t, "recreated", rec)
		require.Equal(t, map[string]string{"dev": "abc1234567"}, f.created)
	})
	t.Run("--to name that is not the configured branch is never created", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: httpErr(404), exists: []bool{false}}
		_, _, err := compareOrRecreate(f, "main", "deve", false, false)
		require.ErrorContains(t, err, "check --to for a typo")
		require.Nil(t, f.created)
	})
	t.Run("dry run does not create", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: httpErr(404), exists: []bool{false}}
		_, rec, err := compareOrRecreate(f, "main", "dev", true, true)
		require.NoError(t, err)
		require.Equal(t, "would recreate", rec)
		require.Nil(t, f.created)
	})
	t.Run("404 but dev exists: error stays", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: httpErr(404), exists: []bool{true}}
		_, _, err := compareOrRecreate(f, "main", "dev", true, false)
		require.Error(t, err)
		require.Nil(t, f.created)
	})
	t.Run("403 on compare is not recreated", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: httpErr(403), exists: []bool{false}}
		_, _, err := compareOrRecreate(f, "main", "dev", true, false)
		require.Error(t, err)
		require.Nil(t, f.created)
	})
	t.Run("other errors stay", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: errors.New("boom"), exists: []bool{false}}
		_, _, err := compareOrRecreate(f, "main", "dev", true, false)
		require.ErrorContains(t, err, "boom")
	})
	t.Run("existence check failure is reported with the original error", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: httpErr(404), existsErr: errors.New("rate limited")}
		_, _, err := compareOrRecreate(f, "main", "dev", true, false)
		require.ErrorContains(t, err, "rate limited")
		require.ErrorContains(t, err, "404")
	})
	t.Run("403 on create is an error with a hint", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: httpErr(404), exists: []bool{false}, createErr: httpErr(403)}
		_, _, err := compareOrRecreate(f, "main", "dev", true, false)
		require.ErrorContains(t, err, "permission")
	})
	t.Run("422 race: dev appeared meanwhile is success", func(t *testing.T) {
		f := &fakeForwardAPI{compareErr: httpErr(404), exists: []bool{false, true}, createErr: httpErr(422)}
		_, rec, err := compareOrRecreate(f, "main", "dev", true, false)
		require.NoError(t, err)
		require.Equal(t, "recreated", rec)
	})
}
