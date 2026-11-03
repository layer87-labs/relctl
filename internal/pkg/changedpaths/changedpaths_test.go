package changedpaths

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type fixture struct {
	repo *git.Repository
	dir  string
	wt   *git.Worktree
	when time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{repo: repo, dir: dir, wt: wt, when: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

// commit writes files (path -> content; empty content deletes) and commits them.
func (f *fixture) commit(t *testing.T, files map[string]string) plumbing.Hash {
	t.Helper()
	for p, content := range files {
		full := filepath.Join(f.dir, p)
		if content == "" {
			if _, err := f.wt.Remove(p); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := f.wt.Add(p); err != nil {
			t.Fatal(err)
		}
	}
	f.when = f.when.Add(time.Hour)
	h, err := f.wt.Commit("change", &git.CommitOptions{Author: &object.Signature{Name: "t", Email: "t@example.test", When: f.when}})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func sorted(in []string) []string {
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}

func TestFilesBetweenCommitAndHead(t *testing.T) {
	f := newFixture(t)
	base := f.commit(t, map[string]string{"a.txt": "1", "deploy/whisper/server.py": "1", "deploy/router/main.go": "1"})
	f.commit(t, map[string]string{"deploy/whisper/server.py": "2", "docs/new.md": "x"})
	f.commit(t, map[string]string{"deploy/router/main.go": ""}) // deleted

	got, err := Files(f.repo, base)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"deploy/router/main.go", "deploy/whisper/server.py", "docs/new.md"}
	if !reflect.DeepEqual(sorted(got), want) {
		t.Fatalf("got %v, want %v", sorted(got), want)
	}
}

func TestFilesWithoutChanges(t *testing.T) {
	f := newFixture(t)
	head := f.commit(t, map[string]string{"a.txt": "1"})
	got, err := Files(f.repo, head)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestFilesUnknownCommit(t *testing.T) {
	f := newFixture(t)
	f.commit(t, map[string]string{"a.txt": "1"})
	if _, err := Files(f.repo, plumbing.NewHash("1234567890123456789012345678901234567890")); err == nil {
		t.Fatal("a commit that is not in the history must be an error that mentions the full history")
	}
}

func TestMatch(t *testing.T) {
	files := []string{
		"deploy/whisper/server.py",
		"deploy/whisper/tests/test_server.py",
		"deploy/Containerfile.whisper",
		"deploy/Containerfile.router",
		"README.md",
		"docs/guide.md",
	}
	cases := []struct {
		patterns []string
		want     []string
	}{
		{[]string{"deploy/whisper/**"}, []string{"deploy/whisper/server.py", "deploy/whisper/tests/test_server.py"}},
		{[]string{"deploy/whisper/**", "deploy/Containerfile.whisper"}, []string{"deploy/whisper/server.py", "deploy/whisper/tests/test_server.py", "deploy/Containerfile.whisper"}},
		{[]string{"deploy/whisper"}, []string{"deploy/whisper/server.py", "deploy/whisper/tests/test_server.py"}},
		{[]string{"deploy/whisper/"}, []string{"deploy/whisper/server.py", "deploy/whisper/tests/test_server.py"}},
		{[]string{"deploy/Containerfile.*"}, []string{"deploy/Containerfile.whisper", "deploy/Containerfile.router"}},
		{[]string{"**/*.md"}, []string{"README.md", "docs/guide.md"}},
		{[]string{"nothing/**"}, nil},
	}
	for _, c := range cases {
		got, err := Match(files, c.patterns)
		if err != nil {
			t.Fatalf("%v: %v", c.patterns, err)
		}
		if !reflect.DeepEqual(sorted(got), sorted(c.want)) {
			t.Errorf("%v: got %v, want %v", c.patterns, sorted(got), sorted(c.want))
		}
	}
	if _, err := Match(files, []string{"[invalid"}); err == nil {
		t.Error("an invalid pattern must be reported")
	}
}
