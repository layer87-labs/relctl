// Package prerelease computes release candidate versions of the form
// MAJOR.MINOR.PATCH-rc.N.
//
// The candidate number is never written into a file: it is derived from the
// candidates that already exist for the same base version, so that two runs
// never produce the same number.
package prerelease

import (
	"fmt"
	"regexp"
	"strconv"
)

// Tag returns the release candidate tag for base and candidate number n.
func Tag(base string, n int) string {
	return fmt.Sprintf("%s-rc.%d", base, n)
}

// Prefix returns the common prefix of all release candidate tags of base.
func Prefix(base string) string {
	return base + "-rc."
}

// Next returns the number of the next release candidate of base, given the
// names of all existing tags and releases. Names that are not release
// candidates of exactly this base version are ignored. The first candidate is 1.
func Next(base string, existing []string) int {
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(Prefix(base)) + `(0|[1-9][0-9]*)$`)
	highest := 0
	for _, name := range existing {
		m := re.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		if n, err := strconv.Atoi(m[1]); err == nil && n > highest {
			highest = n
		}
	}
	return highest + 1
}
