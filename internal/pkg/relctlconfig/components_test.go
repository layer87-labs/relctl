package relctlconfig_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/layer87-labs/relctl/internal/pkg/relctlconfig"
)

func TestEffectiveRepository(t *testing.T) {
	cfg := &relctlconfig.Config{VersionScheme: relctlconfig.SchemeCalVer, ChangelogFile: "CHANGELOG.md", TagPrefix: "v"}
	eff, err := cfg.Effective("")
	if err != nil {
		t.Fatal(err)
	}
	if eff.Component != "" || eff.TagPrefix != "v" || eff.VersionScheme != relctlconfig.SchemeCalVer {
		t.Fatalf("unexpected %+v", eff)
	}
}

func TestEffectiveComponent(t *testing.T) {
	cfg := &relctlconfig.Config{
		VersionScheme: relctlconfig.SchemeSemVer,
		ChangelogFile: "CHANGELOG.md",
		Components: map[string]relctlconfig.Component{
			"whisper": {Paths: []string{"deploy/whisper/**"}, VersionScheme: relctlconfig.SchemeCalVer},
			"router":  {TagPrefix: "router-v", ChangelogFile: "router/CHANGELOG.md"},
		},
	}

	whisper, err := cfg.Effective("whisper")
	if err != nil {
		t.Fatal(err)
	}
	if whisper.TagPrefix != "whisper/" {
		t.Errorf("default prefix is <name>/, got %q", whisper.TagPrefix)
	}
	if whisper.VersionScheme != relctlconfig.SchemeCalVer || whisper.ChangelogFile != "CHANGELOG.md" {
		t.Errorf("scheme is the component's, the changelog the repository's: %+v", whisper)
	}
	if len(whisper.Paths) != 1 {
		t.Errorf("paths: %+v", whisper.Paths)
	}

	router, err := cfg.Effective("router")
	if err != nil {
		t.Fatal(err)
	}
	if router.TagPrefix != "router-v" || router.VersionScheme != relctlconfig.SchemeSemVer || router.ChangelogFile != "router/CHANGELOG.md" {
		t.Errorf("unexpected %+v", router)
	}
}

func TestEffectiveUnknownComponent(t *testing.T) {
	cfg := &relctlconfig.Config{Components: map[string]relctlconfig.Component{"b": {}, "a": {}}}
	_, err := cfg.Effective("c")
	if err == nil || !strings.Contains(err.Error(), "known components: a, b") {
		t.Fatalf("want an error that lists the components, got %v", err)
	}
	_, err = (&relctlconfig.Config{}).Effective("c")
	if err == nil || !strings.Contains(err.Error(), "defines no components") {
		t.Fatalf("want an error that says there are none, got %v", err)
	}
	if _, err := cfg.Effective("Bad Name"); err == nil {
		t.Fatal("an invalid name must be refused")
	}
}

func TestLoadRejectsAmbiguousTagPrefixes(t *testing.T) {
	dir := t.TempDir()
	write := func(content string) string {
		p := filepath.Join(dir, ".relctl.yaml")
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	if _, err := relctlconfig.Load(write("components:\n  a: {tag_prefix: x/}\n  b: {tag_prefix: x/}\n")); err == nil {
		t.Error("two components with the same prefix must be refused")
	}
	if _, err := relctlconfig.Load(write("tag_prefix: v\ncomponents:\n  a: {tag_prefix: v}\n")); err == nil {
		t.Error("a component with the prefix of the repository must be refused")
	}
	cfg, err := relctlconfig.Load(write("tag_prefix: v\ncomponents:\n  a: {}\n  b: {}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TagPrefix != "v" || len(cfg.Components) != 2 {
		t.Errorf("unexpected %+v", cfg)
	}
}
