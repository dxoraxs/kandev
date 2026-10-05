package scan

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	genRel    = "docs/plans/INDEX.md"
	genMarker = "<!-- generated -->\n"
)

// @covers AC-TASKS-PLAN-BOARD-OPS-006.2
func TestWriteGenerated_CreatesMissingFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs/plans"), 0o755); err != nil {
		t.Fatal(err)
	}
	wrote, err := WriteGenerated(root, genRel, []byte(genMarker+"one"), genMarker)
	if err != nil || !wrote {
		t.Fatalf("wrote %v err %v", wrote, err)
	}
	got, _ := os.ReadFile(filepath.Join(root, genRel))
	if string(got) != genMarker+"one" {
		t.Fatalf("content %q", got)
	}
	entries, _ := os.ReadDir(filepath.Join(root, "docs/plans"))
	if len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.2
func TestWriteGenerated_ReplacesOwnedFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, genRel)
	mustWrite(t, target, genMarker+"old")
	wrote, err := WriteGenerated(root, genRel, []byte(genMarker+"new"), genMarker)
	if err != nil || !wrote {
		t.Fatalf("wrote %v err %v", wrote, err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != genMarker+"new" {
		t.Fatalf("content %q", got)
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.3
func TestWriteGenerated_EqualContentIsNotWritten(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, genRel)
	mustWrite(t, target, genMarker+"same")
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(target, past, past); err != nil {
		t.Fatal(err)
	}
	wrote, err := WriteGenerated(root, genRel, []byte(genMarker+"same"), genMarker)
	if err != nil || wrote {
		t.Fatalf("wrote %v err %v", wrote, err)
	}
	info, _ := os.Stat(target)
	if info.ModTime().Sub(past).Abs() > time.Second {
		t.Fatalf("file was rewritten: %v", info.ModTime())
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.2
func TestWriteGenerated_ForeignFileIsKept(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, genRel)
	mustWrite(t, target, "# my own index\n")
	wrote, err := WriteGenerated(root, genRel, []byte(genMarker+"new"), genMarker)
	if !errors.Is(err, ErrNotGenerated) || wrote {
		t.Fatalf("wrote %v err %v", wrote, err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "# my own index\n" {
		t.Fatalf("content %q", got)
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.2
func TestWriteGenerated_SymlinkAtTargetIsNotFollowed(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.md")
	mustWrite(t, outside, genMarker+"secret")
	if err := os.MkdirAll(filepath.Join(root, "docs/plans"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, genRel)); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteGenerated(root, genRel, []byte(genMarker+"new"), genMarker); !errors.Is(err, ErrNotRegularFile) {
		t.Fatalf("err %v", err)
	}
	got, _ := os.ReadFile(outside)
	if string(got) != genMarker+"secret" {
		t.Fatalf("outside file changed: %q", got)
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.2
func TestWriteGenerated_DirectoryOutsideRootIsRefused(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "docs/plans")); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteGenerated(root, genRel, []byte(genMarker), genMarker); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("err %v", err)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("wrote outside the root: %v", entries)
	}
}
