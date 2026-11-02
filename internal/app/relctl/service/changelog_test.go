package service

import (
	"errors"
	"testing"

	"github.com/layer87-labs/relctl/internal/pkg/changelog"
	"github.com/stretchr/testify/require"
)

func TestChangelogRelease(t *testing.T) {
	v, notes, err := changelogRelease("# Changelog\n\n## 1.2.0 — 2026-10-01\n\n- feature\n\n## 1.1.0 — 2026-09-01\n\n- old\n")
	require.NoError(t, err)
	require.Equal(t, "1.2.0", v)
	require.Equal(t, "- feature", notes)

	_, _, err = changelogRelease("## 1.3.0\n\n- wip\n")
	require.True(t, errors.Is(err, ErrChangelogNotReleased), "got %v", err)

	_, _, err = changelogRelease("## Unreleased\n")
	require.True(t, errors.Is(err, changelog.ErrInvalidHeading), "got %v", err)

	_, _, err = changelogRelease("nothing here")
	require.True(t, errors.Is(err, changelog.ErrNoHeading), "got %v", err)
}
