package ces

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithLegacyAliases(t *testing.T) {
	got := withLegacyAliases([]KeyValue{
		{Name: "RELCTL_VERSION", Value: "1.2.3"},
		{Name: "OTHER", Value: "x"},
	})
	require.Equal(t, []KeyValue{
		{Name: "RELCTL_VERSION", Value: "1.2.3"},
		{Name: "OTHER", Value: "x"},
		{Name: "ACI_VERSION", Value: "1.2.3"},
	}, got)
}

func TestLegacyEnabled(t *testing.T) {
	t.Setenv(LegacyEnvVar, "")
	require.False(t, legacyEnabled())
	t.Setenv(LegacyEnvVar, "true")
	require.True(t, legacyEnabled())
	t.Setenv(LegacyEnvVar, "1")
	require.True(t, legacyEnabled())
	t.Setenv(LegacyEnvVar, "no")
	require.False(t, legacyEnabled())
}
