package relctlconfig

import (
	"os"

	"gopkg.in/yaml.v2"
)

const (
	DefaultConfigFile    = ".relctl.yaml"
	SchemeCalVer         = "calver"
	SchemeSemVer         = "semver"
	DefaultVersionScheme = SchemeSemVer
	DefaultBranch        = "main"
)

// Config holds the contents of a .relctl.yaml project config file.
// All fields are optional – missing values fall back to their defaults.
type Config struct {
	// VersionScheme selects the versioning strategy: "semver" (default) or "calver".
	VersionScheme string `yaml:"version_scheme"`
	// DefaultBranch is the repository's main integration branch.
	DefaultBranch string `yaml:"default_branch"`
}

// Load reads the config file at the given path and returns a populated Config.
// If the file does not exist, a Config with default values is returned without
// an error – absence of a config file is valid and backwards-compatible.
func Load(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
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
		DefaultBranch: DefaultBranch,
	}
}

// applyDefaults fills in any zero-value fields with their defaults so that
// partial config files still produce a fully-populated struct.
func applyDefaults(cfg *Config) {
	if cfg.VersionScheme == "" {
		cfg.VersionScheme = DefaultVersionScheme
	}
	if cfg.DefaultBranch == "" {
		cfg.DefaultBranch = DefaultBranch
	}
}
