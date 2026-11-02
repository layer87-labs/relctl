package service

import (
	"errors"
	"fmt"
	"os"

	"github.com/layer87-labs/relctl/internal/pkg/changelog"
	"github.com/layer87-labs/relctl/internal/pkg/relctlconfig"
)

// ErrChangelogNotReleased is returned when the topmost changelog heading has
// no release date. Only a dated heading may be released.
var ErrChangelogNotReleased = errors.New("changelog: topmost version has no release date, refusing to release")

// changelogRelease derives version and release notes from changelog content.
// The topmost heading must be dated.
func changelogRelease(content string) (version, notes string, err error) {
	h, err := changelog.Top(content)
	if err != nil {
		return "", "", err
	}
	if !h.Released() {
		return "", "", fmt.Errorf("%w (%s)", ErrChangelogNotReleased, h.Version)
	}
	notes, err = changelog.Section(content)
	if err != nil {
		return "", "", err
	}
	return h.Version, notes, nil
}

// changelogFileFor returns the changelog path from the config file.
func changelogFileFor(args *ReleaseArgs) (string, error) {
	cfgPath := args.ConfigFile
	if cfgPath == "" {
		cfgPath = relctlconfig.DefaultConfigFile
	}
	cfg, err := relctlconfig.Load(cfgPath)
	if err != nil {
		return "", fmt.Errorf("loading relctl config: %w", err)
	}
	return cfg.ChangelogFile, nil
}

// changelogReleaseFromFile reads the configured changelog and derives the release.
func changelogReleaseFromFile(args *ReleaseArgs) (version, notes string, err error) {
	path, err := changelogFileFor(args)
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("changelog: %w", err)
	}
	return changelogRelease(string(data))
}
