package relctlconfig

import (
	"os"

	"gopkg.in/yaml.v2"
)

const (
	DefaultConfigFile         = ".relctl.yaml"
	SchemeCalVer              = "calver"
	SchemeSemVer              = "semver"
	SchemeConventionalCommits = "conventional-commits"
	SchemeChangelog           = "changelog"
	DefaultChangelogFile      = "CHANGELOG.md"
	DefaultVersionScheme      = SchemeSemVer
	DefaultBranch             = "main"
)

// Config holds the contents of a .relctl.yaml project config file.
// All fields are optional – missing values fall back to their defaults.
type Config struct {
	// VersionScheme selects the versioning strategy: "semver" (default), "calver",
	// "conventional-commits" or "changelog".
	VersionScheme string `yaml:"version_scheme"`
	// ChangelogFile is the changelog read by the "changelog" scheme (default CHANGELOG.md).
	ChangelogFile string `yaml:"changelog_file"`
	// ProductionBranch is the branch that receives releases (default "main").
	ProductionBranch string `yaml:"production_branch"`
	// DevelopmentBranch is the branch work lands on before it is released. Empty
	// means the default branch of the repository. Used by `release forward`.
	DevelopmentBranch string `yaml:"development_branch"`
	// DefaultBranch is the former name of ProductionBranch. It is still read and
	// used as ProductionBranch if that is not set.
	DefaultBranch string `yaml:"default_branch"`
}

// Load reads the config file at the given path and returns a populated Config.
// If the file does not exist, a Config with default values is returned without
// an error – absence of a config file is valid and backwards-compatible.
func Load(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		applyDefaults(cfg)
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	applyDefaults(cfg)
	return cfg, nil
}

// defaults returns a Config pre-populated with the relctl defaults.
func defaults() *Config {
	return &Config{
		VersionScheme: DefaultVersionScheme,
		ChangelogFile: DefaultChangelogFile,
	}
}

// applyDefaults fills in any zero-value fields with their defaults so that
// partial config files still produce a fully-populated struct.
func applyDefaults(cfg *Config) {
	if cfg.VersionScheme == "" {
		cfg.VersionScheme = DefaultVersionScheme
	}
	if cfg.ChangelogFile == "" {
		cfg.ChangelogFile = DefaultChangelogFile
	}
	if cfg.ProductionBranch == "" {
		cfg.ProductionBranch = cfg.DefaultBranch
	}
	if cfg.ProductionBranch == "" {
		cfg.ProductionBranch = DefaultBranch
	}
	if cfg.DefaultBranch == "" {
		cfg.DefaultBranch = cfg.ProductionBranch
	}
}
