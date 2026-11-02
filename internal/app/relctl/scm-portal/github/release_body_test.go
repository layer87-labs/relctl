package github

import (
	"strings"
	"testing"
)

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{
		0:               "0 B",
		1023:            "1023 B",
		1024:            "1.0 KiB",
		1536:            "1.5 KiB",
		5 * 1024 * 1024: "5.0 MiB",
	}
	for in, want := range cases {
		if got := humanSize(in); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderAssetsTable(t *testing.T) {
	if got := renderAssetsTable(nil); got != "" {
		t.Fatalf("no assets must render nothing, got %q", got)
	}
	got := renderAssetsTable([]assetRow{
		{Name: "relctl_linux", URL: "https://example.test/relctl_linux", Size: 2048, Hash: []byte{0xab, 0xcd}},
		{Name: "sha256sum.txt", URL: "https://example.test/sha256sum.txt", Size: 10, Hash: []byte{0x01}},
	})
	want := "### Assets\n\n" +
		"| File | Size | SHA-256 |\n" +
		"| --- | ---: | --- |\n" +
		"| [relctl_linux](https://example.test/relctl_linux) | 2.0 KiB | `abcd` |\n" +
		"| [sha256sum.txt](https://example.test/sha256sum.txt) | 10 B | `01` |"
	if got != want {
		t.Fatalf("unexpected table:\n%s\n--- want ---\n%s", got, want)
	}
}

func TestComposeReleaseBody(t *testing.T) {
	assets := []assetRow{{Name: "a", URL: "u", Size: 1, Hash: []byte{1}}}
	if got := composeReleaseBody("  notes\n", nil); got != "notes" {
		t.Errorf("text only: %q", got)
	}
	if got := composeReleaseBody("", assets); !strings.HasPrefix(got, "### Assets") {
		t.Errorf("assets only: %q", got)
	}
	got := composeReleaseBody("## What's Changed\n* a", assets)
	if !strings.HasPrefix(got, "## What's Changed\n* a\n\n### Assets\n") {
		t.Errorf("text and assets: %q", got)
	}
}
