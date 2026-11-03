package versioning

import (
	"testing"
	"time"
)

func TestCalVerNextVersionWithTagPrefix(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC) }
	tags := []string{
		"202611.3.1",         // the repository itself
		"whisper/202611.3.1", // two releases of a component today
		"whisper/202611.3.2",
		"router/202611.3.7",  // another component
		"v202611.3.9",        // another prefix
		"whisper/202611.2.5", // another day
	}
	cv := &CalVerScheme{now: now, tagsForRepo: func(string) ([]string, error) { return tags, nil }}

	cases := map[string]string{
		"whisper/": "202611.3.3",
		"router/":  "202611.3.8",
		"v":        "202611.3.10",
		"":         "202611.3.2", // plain tags only count the plain ones
		"other/":   "202611.3.1", // a component without a release today starts at 1
	}
	for prefix, want := range cases {
		got, err := cv.NextVersion(ReleaseContext{RepoPath: ".", TagPrefix: prefix})
		if err != nil {
			t.Fatalf("prefix %q: %v", prefix, err)
		}
		if got != want {
			t.Errorf("prefix %q: got %s, want %s", prefix, got, want)
		}
	}
}
