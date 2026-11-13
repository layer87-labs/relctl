package relctlconfig

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	DefaultConfigFile         = ".relctl.yaml"
	SchemeCalVer              = "calver"
	SchemeSemVer              = "semver"
	SchemeConventionalCommits = "conventional-commits"
	SchemeChangelog           = "changelog"
	DefaultChangelogFile      = "CHANGELOG.md"
	// ComponentTagSeparator ends the default tag prefix of a component: "whisper/".
	ComponentTagSeparator = "/"
	DefaultVersionScheme  = SchemeSemVer
	DefaultBranch         = "main"

	// BreakingMarkersMajor raises the SemVer bump of a pull request with a
	// Conventional Commits breaking-change marker to major (default).
	BreakingMarkersMajor = "major"
	// BreakingMarkersWarn keeps the bump and warns in the log and the release notes.
	BreakingMarkersWarn = "warn"
	// BreakingMarkersOff ignores breaking-change markers (the behaviour before 2.12.0).
	BreakingMarkersOff = "off"
	// DefaultBreakingMarkers is used when the config file does not set breaking_markers.
	DefaultBreakingMarkers = BreakingMarkersMajor
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
	// TagPrefix is put in front of every tag and release of the repository, for
	// example "v" for v1.2.3. The version itself stays without it. Default: none.
	TagPrefix string `yaml:"tag_prefix"`
	// Components are independently released parts of the repository (for example
	// one container image each). A component has its own tags, its own latest
	// release and its own counter.
	Components map[string]Component `yaml:"components"`
	// BreakingMarkers decides what the semver scheme does with a Conventional
	// Commits breaking-change marker ("fix!:" in the pull request title or a
	// commit subject, a "BREAKING CHANGE:" footer) on a pull request whose
	// branch prefix gives a patch or minor bump: "major" (default) raises the
	// bump, "warn" keeps it and warns, "off" ignores the marker.
	BreakingMarkers string `yaml:"breaking_markers"`
}

// Component is an independently released part of a repository.
type Component struct {
	// TagPrefix of the component's tags. Default: "<name>/".
	TagPrefix string `yaml:"tag_prefix"`
	// Paths (glob patterns, "**" matches any number of directories) that belong to
	// the component. `relctl release changed` compares them with the last release.
	Paths []string `yaml:"paths"`
	// VersionScheme overrides the repository's scheme for this component.
	VersionScheme string `yaml:"version_scheme"`
	// ChangelogFile overrides the repository's changelog for this component.
	ChangelogFile string `yaml:"changelog_file"`
}

// Effective is what applies to the repository itself (Component == "") or to one
// of its components, with all defaults resolved.
type Effective struct {
	Component     string
	TagPrefix     string
	VersionScheme string
	ChangelogFile string
	Paths         []string
	// BreakingMarkers is a repository-wide setting; components inherit it.
	BreakingMarkers string
}

var componentName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// Effective resolves the settings for a component. An empty name means the
// repository itself. An unknown component is an error that lists the known ones.
func (c *Config) Effective(component string) (*Effective, error) {
	if component == "" {
		return &Effective{
			TagPrefix:       c.TagPrefix,
			VersionScheme:   c.VersionScheme,
			ChangelogFile:   c.ChangelogFile,
			BreakingMarkers: c.BreakingMarkers,
		}, nil
	}
	if !componentName.MatchString(component) {
		return nil, fmt.Errorf("invalid component name %q: use lower case letters, digits, '.', '_' and '-'", component)
	}
	comp, ok := c.Components[component]
	if !ok {
		return nil, fmt.Errorf("unknown component %q: %s", component, c.knownComponents())
	}
	eff := &Effective{
		Component:       component,
		TagPrefix:       comp.TagPrefix,
		VersionScheme:   comp.VersionScheme,
		ChangelogFile:   comp.ChangelogFile,
		Paths:           comp.Paths,
		BreakingMarkers: c.BreakingMarkers,
	}
	if eff.TagPrefix == "" {
		eff.TagPrefix = component + ComponentTagSeparator
	}
	if eff.VersionScheme == "" {
		eff.VersionScheme = c.VersionScheme
	}
	if eff.ChangelogFile == "" {
		eff.ChangelogFile = c.ChangelogFile
	}
	return eff, nil
}

func (c *Config) knownComponents() string {
	if len(c.Components) == 0 {
		return "the config file defines no components (key 'components')"
	}
	names := make([]string, 0, len(c.Components))
	for n := range c.Components {
		names = append(names, n)
	}
	sort.Strings(names)
	return "known components: " + strings.Join(names, ", ")
}

// validate rejects settings that would make tags ambiguous.
func (c *Config) validate() error {
	switch c.BreakingMarkers {
	case BreakingMarkersMajor, BreakingMarkersWarn, BreakingMarkersOff:
	default:
		return fmt.Errorf("breaking_markers %q: use %q, %q or %q", c.BreakingMarkers,
			BreakingMarkersMajor, BreakingMarkersWarn, BreakingMarkersOff)
	}
	seen := map[string]string{}
	if c.TagPrefix != "" {
		seen[c.TagPrefix] = "the repository"
	}
	for name, comp := range c.Components {
		if !componentName.MatchString(name) {
			return fmt.Errorf("invalid component name %q: use lower case letters, digits, '.', '_' and '-'", name)
		}
		prefix := comp.TagPrefix
		if prefix == "" {
			prefix = name + ComponentTagSeparator
		}
		if other, dup := seen[prefix]; dup {
			return fmt.Errorf("component %q and %s use the same tag prefix %q", name, other, prefix)
		}
		seen[prefix] = fmt.Sprintf("component %q", name)
	}
	return nil
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
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// defaults returns a Config pre-populated with the relctl defaults.
func defaults() *Config {
	return &Config{
		VersionScheme:   DefaultVersionScheme,
		ChangelogFile:   DefaultChangelogFile,
		BreakingMarkers: DefaultBreakingMarkers,
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
	if cfg.BreakingMarkers == "" {
		cfg.BreakingMarkers = DefaultBreakingMarkers
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
