package versioning

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// CalVerScheme implements VersionScheme using the YYYY.MM.DD.N format.
// N is monotonically increasing per day, 1-based, derived exclusively from
// local Git tags – no network calls, no SCM API.
type CalVerScheme struct {
	// now is the time source used for date calculation.
	// Defaults to time.Now().UTC(); overridable in tests.
	now func() time.Time

	// tagsForRepo opens a Git repository and returns its tag names.
	// Overridable in tests to inject a mock repository.
	tagsForRepo func(path string) ([]string, error)
}

// NewCalVerScheme creates a CalVerScheme with production defaults.
func NewCalVerScheme() *CalVerScheme {
	return &CalVerScheme{
		now:         func() time.Time { return time.Now().UTC() },
		tagsForRepo: listTagNames,
	}
}

// NextVersion calculates the next CalVer string for today.
// Format: YYYY.MM.DD.N  (e.g. 2026.06.01.3)
func (c *CalVerScheme) NextVersion(ctx ReleaseContext) (string, error) {
	today := c.now()
	prefix := calverDatePrefix(today)

	tags, err := c.tagsForRepo(ctx.RepoPath)
	if err != nil {
		return "", fmt.Errorf("calver: listing git tags: %w", err)
	}

	n := nextN(prefix, tags)
	return fmt.Sprintf("%s.%d", prefix, n), nil
}

// Validate reports whether version is a well-formed CalVer string (YYYY.MM.DD.N).
func (c *CalVerScheme) Validate(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 4 {
		return false
	}
	// Validate YYYY, MM, DD as integers
	for _, p := range parts[:3] {
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	// N must be a positive integer
	n, err := strconv.Atoi(parts[3])
	return err == nil && n > 0
}

// calverDatePrefix formats the date part of a CalVer version: YYYY.MM.DD
func calverDatePrefix(t time.Time) string {
	return fmt.Sprintf("%04d.%02d.%02d", t.Year(), int(t.Month()), t.Day())
}

// nextN scans tags for the given date prefix and returns max(N)+1.
// Returns 1 if no tags exist for today.
func nextN(prefix string, tags []string) int {
	max := 0
	for _, tag := range tags {
		if !strings.HasPrefix(tag, prefix+".") {
			continue
		}
		rest := tag[len(prefix)+1:]
		n, err := strconv.Atoi(rest)
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return max + 1
}

// listTagNames opens the repository at path and returns all tag names.
func listTagNames(path string) ([]string, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("opening repository at %q: %w", path, err)
	}

	iter, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("reading tags: %w", err)
	}

	var names []string
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		names = append(names, name)
		return nil
	})
	return names, err
}
