package scan

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func testHash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

const writeRel = "docs/plans/a.md"

// @covers AC-TASKS-PLAN-FILES-006.1
func TestWriteFile_ReplacesContentKeepsModeNoTemp(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, writeRel)
	mustWrite(t, target, "old")
	if err := os.Chmod(target, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(root, writeRel, []byte("new"), testHash([]byte("old")), testHash); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "new" {
		t.Fatalf("content %q", got)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode %v", info.Mode())
	}
	entries, _ := os.ReadDir(filepath.Dir(target))
	if len(entries) != 1 {
		t.Fatalf("leftover entries: %v", entries)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.1
func TestWriteFile_HashMismatchLeavesFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, writeRel)
	mustWrite(t, target, "disk")
	err := WriteFile(root, writeRel, []byte("new"), testHash([]byte("stale")), testHash)
	if !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("err %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "disk" {
		t.Fatalf("content %q", got)
	}
	entries, _ := os.ReadDir(filepath.Dir(target))
	if len(entries) != 1 {
		t.Fatalf("leftover entries: %v", entries)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.1
func TestWriteFile_SymlinkRefused(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.md")
	mustWrite(t, outside, "secret")
	link := filepath.Join(root, writeRel)
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	err := WriteFile(root, writeRel, []byte("x"), testHash([]byte("secret")), testHash)
	if !errors.Is(err, ErrNotRegularFile) {
		t.Fatalf("err %v", err)
	}
	got, _ := os.ReadFile(outside)
	if string(got) != "secret" {
		t.Fatalf("outside modified: %q", got)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.1
func TestWriteFile_OutsideRootRefused(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	mustWrite(t, filepath.Join(outside, "a.md"), "o")
	if err := os.Symlink(outside, filepath.Join(root, "docs")); err != nil {
		t.Fatal(err)
	}
	err := WriteFile(root, "docs/a.md", []byte("x"), testHash([]byte("o")), testHash)
	if !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("err %v", err)
	}
	if err := WriteFile(root, "../a.md", []byte("x"), "", testHash); err == nil {
		t.Fatal("expected invalid path error")
	}
}
