package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchPatchLevel(t *testing.T) {
	require.Equal(t, "major", matchPatchLevel("relctl_patch_level: major"))
	require.Equal(t, "major", matchPatchLevel("aci_patch_level: major"), "legacy prefix")
	require.Equal(t, "", matchPatchLevel("please relctl_patch_level: major"), "must start the comment")
	require.Equal(t, "", matchPatchLevel("something else"))
}

func TestMatchVersionOverride(t *testing.T) {
	v := matchVersionOverride("relctl_version_override: 2.1.0")
	require.NotNil(t, v)
	require.Equal(t, "2.1.0", *v)

	v = matchVersionOverride("aci_version_override: 3.0.0-rc.1")
	require.NotNil(t, v)
	require.Equal(t, "3.0.0-rc.1", *v)

	require.Nil(t, matchVersionOverride("relctl_version_override: nope"))
	require.Nil(t, matchVersionOverride("hello"))
}
