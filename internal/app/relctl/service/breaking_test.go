package service

import "testing"

func TestWithBreakingWarning(t *testing.T) {
	cases := []struct{ body, warning, want string }{
		{body: "## Changes\n- x", warning: "", want: "## Changes\n- x"},
		{body: "## Changes\n- x", warning: "The pull request title marks a breaking change.",
			want: "> [!WARNING]\n> **Breaking change.** The pull request title marks a breaking change.\n\n## Changes\n- x"},
		{body: "  ", warning: "w.", want: "> [!WARNING]\n> **Breaking change.** w.\n"},
	}
	for _, tc := range cases {
		if got := withBreakingWarning(tc.body, tc.warning); got != tc.want {
			t.Errorf("withBreakingWarning(%q, %q) =\n%q\nwant\n%q", tc.body, tc.warning, got, tc.want)
		}
	}
}
