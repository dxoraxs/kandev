package scan

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const createRel = "docs/plans/new.md"

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreateFile_CreatesMissingDirectoryAndFile(t *testing.T) {
	root := t.TempDir()
	if err := CreateFile(root, createRel, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, createRel))
	if err != nil || string(got) != "hello" {
		t.Fatalf("content %q err %v", got, err)
	}
	info, _ := os.Stat(filepath.Join(root, createRel))
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("mode %v", info.Mode())
	}
	entries, _ := os.ReadDir(filepath.Join(root, "docs/plans"))
	if len(entries) != 1 {
		t.Fatalf("unexpected entries %v", entries)
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreateFile_ExistingFileIsKeptAndReported(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, createRel)
	mustWrite(t, target, "first")
	err := CreateFile(root, createRel, []byte("second"))
	if !errors.Is(err, ErrExists) {
		t.Fatalf("err %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "first" {
		t.Fatalf("content %q", got)
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreateFile_ExistingSymlinkIsNotFollowed(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.md")
	mustWrite(t, outside, "secret")
	link := filepath.Join(root, createRel)
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := CreateFile(root, createRel, []byte("x")); !errors.Is(err, ErrExists) {
		t.Fatalf("err %v", err)
	}
	if err := os.Remove(outside); err != nil {
		t.Fatal(err)
	}
	if err := CreateFile(root, createRel, []byte("x")); !errors.Is(err, ErrExists) {
		t.Fatalf("dangling link: err %v", err)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("a file was created through the link: %v", err)
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreateFile_RefusesPathsThatLeaveTheRoot(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"../escape.md", "docs/../../escape.md", "/abs/escape.md", ""} {
		if err := CreateFile(root, rel, []byte("x")); err == nil {
			t.Fatalf("%q accepted", rel)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "escape.md")); !os.IsNotExist(err) {
		t.Fatalf("escape file created: %v", err)
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreateFile_SymlinkedDirectoryOutsideRootIsRefused(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "docs", "plans")); err != nil {
		t.Fatal(err)
	}
	if err := CreateFile(root, createRel, []byte("x")); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("err %v", err)
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatalf("files created outside: %v", entries)
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreateFile_SymlinkedAncestorOutsideRootIsRefusedBeforeCreatingDirectories(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "docs")); err != nil {
		t.Fatal(err)
	}
	if err := CreateFile(root, createRel, []byte("x")); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("err %v", err)
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatalf("directories created outside: %v", entries)
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreateFile_FailedCreateLeavesNoFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "docs/plans")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if err := CreateFile(root, createRel, []byte("x")); err == nil || errors.Is(err, ErrExists) {
		t.Fatalf("err %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("leftover entries: %v", entries)
	}
}
