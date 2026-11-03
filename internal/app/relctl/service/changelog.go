package service

import (
	"errors"
	"fmt"
	"os"

	"github.com/layer87-labs/relctl/internal/pkg/changelog"
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

// changelogNotes returns the release notes for version from the configured
// changelog. If the topmost heading belongs to a different version (an
// explicit --version was given), no notes are returned.
func changelogNotes(args *ReleaseArgs, version string) (string, error) {
	path, err := changelogFileFor(args)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("changelog: %w", err)
	}
	h, err := changelog.Top(string(data))
	if err != nil {
		return "", err
	}
	if h.Version != version {
		return "", nil
	}
	return changelog.Section(string(data))
}

// changelogFileFor returns the changelog path from the config file.
func changelogFileFor(args *ReleaseArgs) (string, error) {
	eff, err := loadEffective(args.ConfigFile, args.Component)
	if err != nil {
		return "", err
	}
	return eff.ChangelogFile, nil
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

// changelogHeadingVersionFromFile returns the version of the topmost heading,
// dated or not. It is used for release candidates, which are built from a
// heading that has not been dated yet.
func changelogHeadingVersionFromFile(args *ReleaseArgs) (string, error) {
	path, err := changelogFileFor(args)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("changelog: %w", err)
	}
	h, err := changelog.Top(string(data))
	if err != nil {
		return "", err
	}
	return h.Version, nil
}
