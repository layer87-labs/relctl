package prerelease_test

import (
	"testing"

	"github.com/layer87-labs/relctl/internal/pkg/prerelease"
	"github.com/stretchr/testify/require"
)

func TestNext(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		existing []string
		want     int
	}{
		{"none", "1.0.0", nil, 1},
		{"first exists", "1.0.0", []string{"1.0.0-rc.1"}, 2},
		{"takes the highest, not the count", "1.0.0", []string{"1.0.0-rc.1", "1.0.0-rc.7", "1.0.0-rc.3"}, 8},
		{"numeric not lexical order", "1.0.0", []string{"1.0.0-rc.9", "1.0.0-rc.10"}, 11},
		{"other base is ignored", "1.0.0", []string{"1.1.0-rc.4", "0.9.0-rc.2"}, 1},
		{"released tag is ignored", "1.0.0", []string{"1.0.0", "0.1.0"}, 1},
		{"base is not a regex", "1.0.0", []string{"1x0y0-rc.5"}, 1},
		{"longer base is not matched", "1.0.0", []string{"11.0.0-rc.5", "1.0.01-rc.3"}, 1},
		{"other suffix is ignored", "1.0.0", []string{"1.0.0-beta.2", "1.0.0-rc.x", "1.0.0-rc.1-extra"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, prerelease.Next(tt.base, tt.existing))
		})
	}
}

func TestTag(t *testing.T) {
	require.Equal(t, "2.1.0-rc.3", prerelease.Tag("2.1.0", 3))
	require.Equal(t, "2.1.0-rc.", prerelease.Prefix("2.1.0"))
}
