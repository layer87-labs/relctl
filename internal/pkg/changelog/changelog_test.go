package changelog_test

import (
	"errors"
	"testing"

	"github.com/layer87-labs/relctl/internal/pkg/changelog"
	"github.com/stretchr/testify/require"
)

func TestTop(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    changelog.Heading
		wantErr error
	}{
		{"upcoming", "# Changelog\n\n## 1.4.0\n\n- x\n", changelog.Heading{Version: "1.4.0"}, nil},
		{"released em dash", "## 1.3.0 — 2026-09-20\n", changelog.Heading{Version: "1.3.0", Date: "2026-09-20"}, nil},
		{"released en dash", "## 1.3.0 – 2026-09-20\n", changelog.Heading{Version: "1.3.0", Date: "2026-09-20"}, nil},
		{"released hyphen", "## 1.3.0 - 2026-09-20\n", changelog.Heading{Version: "1.3.0", Date: "2026-09-20"}, nil},
		{"crlf", "## 1.0.0\r\n\r\ntext\r\n", changelog.Heading{Version: "1.0.0"}, nil},
		{"topmost wins", "## 2.0.0\n## 1.0.0 — 2026-01-01\n", changelog.Heading{Version: "2.0.0"}, nil},
		{"ignores code fence", "```\n## 9.9.9\n```\n## 1.0.0\n", changelog.Heading{Version: "1.0.0"}, nil},
		{"ignores level 3", "### 9.9.9\n## 1.0.0\n", changelog.Heading{Version: "1.0.0"}, nil},
		{"no heading", "# Changelog\n", changelog.Heading{}, changelog.ErrNoHeading},
		{"empty", "", changelog.Heading{}, changelog.ErrNoHeading},
		{"v prefix", "## v1.0.0\n", changelog.Heading{}, changelog.ErrInvalidHeading},
		{"unreleased", "## Unreleased\n", changelog.Heading{}, changelog.ErrInvalidHeading},
		{"rc suffix", "## 1.0.0-rc.1\n", changelog.Heading{}, changelog.ErrInvalidHeading},
		{"leading zero", "## 01.0.0\n", changelog.Heading{}, changelog.ErrInvalidHeading},
		{"parenthesised date", "## 1.0.0 (2026-10-02)\n", changelog.Heading{}, changelog.ErrInvalidHeading},
		{"trailing text", "## 1.0.0 — 2026-10-02 hotfix\n", changelog.Heading{}, changelog.ErrInvalidHeading},
		{"not a date", "## 1.0.0 — 2026-02-30\n", changelog.Heading{}, changelog.ErrInvalidHeading},
		{"two components", "## 1.0\n", changelog.Heading{}, changelog.ErrInvalidHeading},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := changelog.Top(tt.in)
			if tt.wantErr != nil {
				require.True(t, errors.Is(err, tt.wantErr), "got %v", err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestReleased(t *testing.T) {
	require.True(t, changelog.Heading{Version: "1.0.0", Date: "2026-01-01"}.Released())
	require.False(t, changelog.Heading{Version: "1.0.0"}.Released())
}

func TestSection(t *testing.T) {
	in := "# Changelog\n\n## 1.1.0 — 2026-02-01\n\n### Fixed\n\n- a\n\n```\n## not a heading\n```\n\n## 1.0.0 — 2026-01-01\n\n- old\n"
	got, err := changelog.Section(in)
	require.NoError(t, err)
	require.Equal(t, "### Fixed\n\n- a\n\n```\n## not a heading\n```", got)

	got, err = changelog.Section("## 1.0.0\n")
	require.NoError(t, err)
	require.Equal(t, "", got)

	_, err = changelog.Section("nothing")
	require.ErrorIs(t, err, changelog.ErrNoHeading)
}
