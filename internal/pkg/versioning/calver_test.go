package versioning_test

import (
	"testing"
	"time"

	"github.com/layer87-labs/relctl/internal/pkg/versioning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedDate returns a time.Time for 2026-06-01 UTC – used across all tests.
func fixedDate() time.Time {
	return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
}

// newTestScheme constructs a CalVerScheme with an injected clock and tag list.
func newTestScheme(tags []string) *versioning.CalVerScheme {
	s := versioning.NewCalVerScheme()
	versioning.SetCalVerClock(s, fixedDate)
	versioning.SetCalVerTagsFunc(s, func(_ string) ([]string, error) {
		return tags, nil
	})
	return s
}

func TestCalVer_NoTagsToday_ReturnsN1(t *testing.T) {
	s := newTestScheme([]string{
		"1.2.3",        // unrelated semver
		"2025.06.01.5", // different year
		"2026.05.31.3", // yesterday
	})
	v, err := s.NextVersion(versioning.ReleaseContext{RepoPath: "."})
	require.NoError(t, err)
	assert.Equal(t, "2026.06.01.1", v)
}

func TestCalVer_ExistingTagsToday_IncrementsN(t *testing.T) {
	s := newTestScheme([]string{
		"2026.06.01.1",
		"2026.06.01.2",
		"2026.06.01.3",
	})
	v, err := s.NextVersion(versioning.ReleaseContext{RepoPath: "."})
	require.NoError(t, err)
	assert.Equal(t, "2026.06.01.4", v)
}

func TestCalVer_OnlyUnrelatedTags_ReturnsN1(t *testing.T) {
	s := newTestScheme([]string{"v1.0.0", "v2.3.4", "2026.06.02.1"})
	v, err := s.NextVersion(versioning.ReleaseContext{RepoPath: "."})
	require.NoError(t, err)
	assert.Equal(t, "2026.06.01.1", v)
}

func TestCalVer_NonNumericSuffix_Ignored(t *testing.T) {
	s := newTestScheme([]string{"2026.06.01.alpha", "2026.06.01.1"})
	v, err := s.NextVersion(versioning.ReleaseContext{RepoPath: "."})
	require.NoError(t, err)
	assert.Equal(t, "2026.06.01.2", v)
}

func TestCalVer_EmptyRepo_ReturnsN1(t *testing.T) {
	s := newTestScheme([]string{})
	v, err := s.NextVersion(versioning.ReleaseContext{RepoPath: "."})
	require.NoError(t, err)
	assert.Equal(t, "2026.06.01.1", v)
}

// --- Validate ---

func TestCalVer_Validate(t *testing.T) {
	s := versioning.NewCalVerScheme()
	cases := []struct {
		in    string
		valid bool
	}{
		{"2026.06.01.1", true},
		{"2026.06.01.42", true},
		{"2026.6.1.1", true},     // strconv.Atoi accepts no leading zeros
		{"2026.06.01.0", false},  // N must be > 0
		{"2026.06.01", false},    // missing N
		{"v2026.06.01.1", false}, // prefix not valid
		{"1.2.3", false},
		{"", false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.valid, s.Validate(tc.in), "Validate(%q)", tc.in)
	}
}
