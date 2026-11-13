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

func TestChangelogFileDefault(t *testing.T) {
	cfg, err := relctlconfig.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	require.NoError(t, err)
	require.Equal(t, "CHANGELOG.md", cfg.ChangelogFile)

	p := filepath.Join(t.TempDir(), ".relctl.yaml")
	require.NoError(t, os.WriteFile(p, []byte("version_scheme: changelog\nchangelog_file: docs/CHANGES.md\n"), 0o600))
	cfg, err = relctlconfig.Load(p)
	require.NoError(t, err)
	require.Equal(t, relctlconfig.SchemeChangelog, cfg.VersionScheme)
	require.Equal(t, "docs/CHANGES.md", cfg.ChangelogFile)
}

func TestBranchSettings(t *testing.T) {
	write := func(content string) string {
		p := filepath.Join(t.TempDir(), ".relctl.yaml")
		require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
		return p
	}

	cfg, err := relctlconfig.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	require.NoError(t, err)
	require.Equal(t, "main", cfg.ProductionBranch)
	require.Equal(t, "", cfg.DevelopmentBranch, "empty means the repository's default branch")

	cfg, err = relctlconfig.Load(write("production_branch: release\ndevelopment_branch: dev\n"))
	require.NoError(t, err)
	require.Equal(t, "release", cfg.ProductionBranch)
	require.Equal(t, "dev", cfg.DevelopmentBranch)

	// the former name still works
	cfg, err = relctlconfig.Load(write("default_branch: trunk\n"))
	require.NoError(t, err)
	require.Equal(t, "trunk", cfg.ProductionBranch)

	// the new name wins
	cfg, err = relctlconfig.Load(write("default_branch: trunk\nproduction_branch: main\n"))
	require.NoError(t, err)
	require.Equal(t, "main", cfg.ProductionBranch)
}

func TestBreakingMarkers(t *testing.T) {
	write := func(content string) string {
		path := filepath.Join(t.TempDir(), ".relctl.yaml")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
		return path
	}

	cfg, err := relctlconfig.Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "warn", cfg.BreakingMarkers)

	cfg, err = relctlconfig.Load(write("version_scheme: semver\n"))
	require.NoError(t, err)
	assert.Equal(t, "warn", cfg.BreakingMarkers)

	for _, mode := range []string{"major", "warn", "off"} {
		cfg, err = relctlconfig.Load(write("breaking_markers: " + mode + "\n"))
		require.NoError(t, err, mode)
		assert.Equal(t, mode, cfg.BreakingMarkers)
		eff, err := cfg.Effective("")
		require.NoError(t, err)
		assert.Equal(t, mode, eff.BreakingMarkers)
	}

	path := write("breaking_markers: minor\n")
	_, err = relctlconfig.Load(path)
	require.EqualError(t, err, path+`: breaking_markers "minor": use "major", "warn" or "off"`)
}
