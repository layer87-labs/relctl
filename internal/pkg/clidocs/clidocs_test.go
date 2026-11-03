package clidocs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func tree() *cobra.Command {
	root := &cobra.Command{Use: "tool", Short: "tool", Long: "A tool with <angle brackets> and {braces}."}
	root.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	release := &cobra.Command{Use: "release", Short: "Manage releases", Long: "Manage releases.\n\nSee components.<name>.paths."}
	release.PersistentFlags().String("config", "", "path to the config file")
	create := &cobra.Command{Use: "create", Short: "Create a release", Run: func(*cobra.Command, []string) {}}
	create.Flags().Bool("dry-run", false, "do not write")
	release.AddCommand(create)

	hidden := &cobra.Command{Use: "internal", Short: "hidden", Hidden: true, Run: func(*cobra.Command, []string) {}}
	pr := &cobra.Command{Use: "pr", Short: "Pull requests | with a pipe", Run: func(*cobra.Command, []string) {}}
	root.AddCommand(release, hidden, pr)
	return root
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestGenerate(t *testing.T) {
	dir := t.TempDir()
	if err := Generate(tree(), dir); err != nil {
		t.Fatal(err)
	}

	index := read(t, filepath.Join(dir, "index.md"))
	for _, want := range []string{"sidebar_position: 1", "[`relctl pr`](./pr.md)", "Pull requests \\| with a pipe", "--verbose"} {
		if !strings.Contains(index, want) {
			t.Errorf("index.md lacks %q:\n%s", want, index)
		}
	}
	if strings.Contains(index, "internal") {
		t.Error("a hidden command must not be listed")
	}

	release := read(t, filepath.Join(dir, "release.md"))
	for _, want := range []string{"title: release", "# tool release", "## tool release create", "--dry-run", "--config", "components.<name>.paths"} {
		if !strings.Contains(release, want) {
			t.Errorf("release.md lacks %q:\n%s", want, release)
		}
	}
	// text with markup characters is inside a code block, never loose in the page
	loose := strings.Split(release, "```")
	for i := 0; i < len(loose); i += 2 { // even parts are outside of code blocks
		if strings.Contains(loose[i], "<name>") {
			t.Errorf("markup characters outside of a code block: %q", loose[i])
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "internal.md")); err == nil {
		t.Error("a hidden command must not get a page")
	}
}

func TestGenerateIsDeterministicAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	if err := Generate(tree(), dir); err != nil {
		t.Fatal(err)
	}
	first := read(t, filepath.Join(dir, "release.md"))

	// a stale page goes, a foreign file stays
	if err := os.WriteFile(filepath.Join(dir, "gone.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_category_.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Generate(tree(), dir); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(dir, "release.md")) != first {
		t.Error("a second run must write the same content")
	}
	if _, err := os.Stat(filepath.Join(dir, "gone.md")); err == nil {
		t.Error("a page of a command that no longer exists must be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "_category_.json")); err != nil {
		t.Error("files that are no pages must stay")
	}
}

func TestPagesAreNotedAsGeneratedAndHaveNoTabs(t *testing.T) {
	root := tree()
	root.Commands()[0].Long = "Example:\n\tindented with a tab"
	dir := t.TempDir()
	if err := Generate(root, dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.md", "release.md", "pr.md"} {
		page := read(t, filepath.Join(dir, name))
		if !strings.Contains(page, ":::note\nThis page is generated") {
			t.Errorf("%s lacks the note that it is generated", name)
		}
		if strings.Contains(page, "\t") {
			t.Errorf("%s contains a tab", name)
		}
		if strings.Contains(page, "{/*") {
			t.Errorf("%s contains a JSX comment, which markdown lint takes for emphasis", name)
		}
	}
}

func TestFenceKeepsLongerMarkers(t *testing.T) {
	got := fence("text with ``` inside")
	if !strings.HasPrefix(got, "````text") || !strings.HasSuffix(got, "````") {
		t.Errorf("the marker must be longer than any run inside: %q", got)
	}
}

func TestGenerateRefusesEmptyDirectory(t *testing.T) {
	if err := Generate(tree(), ""); err == nil {
		t.Fatal("an empty output directory must be refused")
	}
}
