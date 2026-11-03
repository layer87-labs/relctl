package scmportal

import "testing"

func TestShortSha(t *testing.T) {
	cases := map[string]string{
		"":                         "",
		"abc":                      "abc",
		"0123456789abcdef":         "01234567",
		"0123456789abcdef0123abcd": "01234567",
	}
	for in, want := range cases {
		if got := shortSha(in); got != want {
			t.Errorf("shortSha(%q) = %q, want %q", in, got, want)
		}
	}
}
