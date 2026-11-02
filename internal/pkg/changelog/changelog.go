// Package changelog reads release information from a CHANGELOG.md file.
//
// The version of a repository state is the topmost level-2 heading
// ("## ...") of the changelog. It has exactly one of two forms:
//
//	## 1.4.0                  upcoming: version only
//	## 1.3.0 — 2026-09-20     released: version, dash, ISO date
//
// The version is MAJOR.MINOR.PATCH without a "v" prefix and without leading
// zeros. A pre-release suffix is not allowed: release candidate numbers are
// computed, they never appear in the file. The date must be a real calendar
// date. The separator is an em dash; an en dash or a hyphen with surrounding
// spaces is accepted as well.
//
// An invalid topmost heading is an error rather than "nothing to do": a
// heading a human takes for a version but the tool does not would otherwise
// only show up as a missing release.
//
// Lines inside fenced code blocks are ignored, and "###" headings are not
// version headings.
package changelog

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	// ErrNoHeading is returned when the changelog has no level-2 heading.
	ErrNoHeading = errors.New("changelog: no '## ' version heading found")
	// ErrInvalidHeading is returned when the topmost level-2 heading is not a
	// valid version heading.
	ErrInvalidHeading = errors.New("changelog: invalid version heading")
)

// Heading is the parsed topmost version heading of a changelog.
type Heading struct {
	// Version is MAJOR.MINOR.PATCH.
	Version string
	// Date is the release date (YYYY-MM-DD); empty for an upcoming version.
	Date string
}

// Released reports whether the heading carries a release date.
func (h Heading) Released() bool { return h.Date != "" }

var headingRe = regexp.MustCompile(
	`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:[ \t]+(?:—|–|-)[ \t]+(\d{4}-\d{2}-\d{2}))?$`)

// Top returns the topmost version heading of the changelog content.
func Top(content string) (Heading, error) {
	_, h, _, err := locate(content)
	return h, err
}

// Section returns the text below the topmost version heading up to the next
// level-2 heading, trimmed of surrounding blank lines. It is the text of the
// release notes for that version.
func Section(content string) (string, error) {
	lines, _, start, err := locate(content)
	if err != nil {
		return "", err
	}
	end := len(lines)
	fence := false
	for i := start + 1; i < len(lines); i++ {
		if isFence(lines[i]) {
			fence = !fence
			continue
		}
		if !fence && strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines[start+1:end], "\n")), nil
}

// locate finds the topmost level-2 heading and returns the split lines, the
// parsed heading and the heading's line index.
func locate(content string) ([]string, Heading, int, error) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	fence := false
	for i, line := range lines {
		if isFence(line) {
			fence = !fence
			continue
		}
		if fence || !strings.HasPrefix(line, "## ") {
			continue
		}
		h, err := parseHeading(strings.TrimSpace(strings.TrimPrefix(line, "## ")))
		return lines, h, i, err
	}
	return nil, Heading{}, 0, ErrNoHeading
}

func isFence(line string) bool {
	t := strings.TrimLeft(line, " ")
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}

func parseHeading(text string) (Heading, error) {
	m := headingRe.FindStringSubmatch(text)
	if m == nil {
		return Heading{}, fmt.Errorf("%w: %q (expected 'X.Y.Z' or 'X.Y.Z — YYYY-MM-DD')", ErrInvalidHeading, text)
	}
	h := Heading{Version: fmt.Sprintf("%s.%s.%s", m[1], m[2], m[3]), Date: m[4]}
	if h.Date != "" {
		if _, err := time.Parse("2006-01-02", h.Date); err != nil {
			return Heading{}, fmt.Errorf("%w: %q is not a calendar date", ErrInvalidHeading, h.Date)
		}
	}
	return h, nil
}
