package github

import (
	"strings"
	"testing"
)

func TestHelpCommentBody(t *testing.T) {
	for _, scheme := range []string{"", "semver", "calver", "conventional-commits", "changelog"} {
		body := HelpCommentBody(scheme)
		if !strings.HasPrefix(body, detailsOpening) {
			t.Errorf("%q: an existing comment is found by its opening, the body must start with it", scheme)
		}
		if !strings.Contains(body, "RELCTL_SILENT") {
			t.Errorf("%q: the comment must say how to switch it off", scheme)
		}
	}

	for _, scheme := range []string{"", "semver"} {
		body := HelpCommentBody(scheme)
		if !strings.Contains(body, "relctl_patch_level: major") || !strings.Contains(body, "relctl_version_override") {
			t.Errorf("%q: the SemVer comment lists the comment commands", scheme)
		}
	}

	for _, scheme := range []string{"calver", "conventional-commits", "changelog"} {
		body := HelpCommentBody(scheme)
		if strings.Contains(body, "relctl_patch_level:") || strings.Contains(body, "relctl_version_override:") {
			t.Errorf("%q: comment commands have no effect in this scheme and must not be offered", scheme)
		}
		if !strings.Contains(body, "**"+scheme+"**") {
			t.Errorf("%q: the comment names the scheme", scheme)
		}
	}
}
