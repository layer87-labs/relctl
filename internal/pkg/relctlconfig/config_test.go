package relctlconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/layer87-labs/relctl/internal/pkg/relctlconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults_WhenFileAbsent(t *testing.T) {
	cfg, err := relctlconfig.Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	require.NoError(t, err)
	assert.Equal(t, relctlconfig.SchemeSemVer, cfg.VersionScheme)
	assert.Equal(t, relctlconfig.DefaultBranch, cfg.DefaultBranch)
}

func TestLoad_CalVerScheme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".relctl.yaml")
	require.NoError(t, os.WriteFile(path, []byte("version_scheme: calver\n"), 0o644))

	cfg, err := relctlconfig.Load(path)
	require.NoError(t, err)
	assert.Equal(t, relctlconfig.SchemeCalVer, cfg.VersionScheme)
	assert.Equal(t, relctlconfig.DefaultBranch, cfg.DefaultBranch)
}

func TestLoad_FullConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".relctl.yaml")
	require.NoError(t, os.WriteFile(path, []byte("version_scheme: calver\ndefault_branch: develop\n"), 0o644))

	cfg, err := relctlconfig.Load(path)
	require.NoError(t, err)
	assert.Equal(t, relctlconfig.SchemeCalVer, cfg.VersionScheme)
	assert.Equal(t, "develop", cfg.DefaultBranch)
}

func TestLoad_EmptyFile_UsesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".relctl.yaml")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	cfg, err := relctlconfig.Load(path)
	require.NoError(t, err)
	assert.Equal(t, relctlconfig.SchemeSemVer, cfg.VersionScheme)
}
