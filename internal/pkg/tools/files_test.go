package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileToString(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(file, []byte("from a file"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("a file is read", func(t *testing.T) {
		got, err := ReadFileToString(file)
		if err != nil || got != "from a file" {
			t.Fatalf("got %q, %v", got, err)
		}
	})

	t.Run("a string that is no file is returned as it is", func(t *testing.T) {
		for _, in := range []string{
			"Hand written notes",
			"## Changes\n\n- one\n- two",
			strings.Repeat("a long text ", 400),
		} {
			got, err := ReadFileToString(in)
			if err != nil || got != in {
				t.Fatalf("input %.20q: got %q, %v", in, got, err)
			}
		}
	})

	t.Run("empty input", func(t *testing.T) {
		if _, err := ReadFileToString(""); err != ErrEmptyString {
			t.Fatalf("want ErrEmptyString, got %v", err)
		}
	})

	t.Run("a directory is refused", func(t *testing.T) {
		if _, err := ReadFileToString(dir); err != ErrExpextedFileNotDir {
			t.Fatalf("want ErrExpextedFileNotDir, got %v", err)
		}
	})
}
