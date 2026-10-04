package scan

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadFile_ReturnsContentAndMode(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, writeRel), "hello")

	got, err := ReadFile(root, writeRel)

	if err != nil {
		t.Fatal(err)
	}
	if string(got.Content) != "hello" || got.RelPath != writeRel {
		t.Fatalf("ReadFile = %q at %q", got.Content, got.RelPath)
	}
}

func TestReadFile_RefusesSymlinkAndEscapes(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "x.md")
	mustWrite(t, outside, "secret")
	mustWrite(t, filepath.Join(root, "docs/plans/real.md"), "ok")
	if err := os.Symlink(outside, filepath.Join(root, "docs/plans/link.md")); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadFile(root, "docs/plans/link.md"); !errors.Is(err, ErrNotRegularFile) {
		t.Fatalf("symlink: err = %v, want ErrNotRegularFile", err)
	}
	if _, err := ReadFile(root, "../x.md"); err == nil {
		t.Fatal("a path with .. must be refused")
	}
	if _, err := ReadFile(root, "docs/plans/missing.md"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing: err = %v, want not-exist", err)
	}
}
