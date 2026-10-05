package scan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanRepository_ReportsTheModificationTime(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, writeRel), "x")
	want := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(root, writeRel), want, want); err != nil {
		t.Fatal(err)
	}

	files, _ := ScanRepository(root, []string{filepath.Dir(writeRel)})

	if len(files) != 1 || !files[0].ModTime.Equal(want) {
		t.Fatalf("files = %+v, want ModTime %v", files, want)
	}
}

func TestReadTrack_FileReturnsContentAndModTime(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "docs/notes.md"), "- [ ] one\n")
	want := time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(root, "docs/notes.md"), want, want); err != nil {
		t.Fatal(err)
	}

	track, err := ReadTrack(root, "docs/notes.md")

	if err != nil {
		t.Fatal(err)
	}
	if track.Dir || len(track.Files) != 1 || string(track.Files[0].Content) != "- [ ] one\n" {
		t.Fatalf("track = %+v", track)
	}
	if !track.ModTime.Equal(want) {
		t.Fatalf("ModTime = %v, want %v", track.ModTime, want)
	}
}

func TestReadTrack_DirectoryListsOnlyTaskFiles(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "docs/wo/task-02-b.md"), "b")
	mustWrite(t, filepath.Join(root, "docs/wo/task-01-a.md"), "a")
	mustWrite(t, filepath.Join(root, "docs/wo/plan.md"), "plan")
	mustWrite(t, filepath.Join(root, "docs/wo/task-03.txt"), "txt")
	mustWrite(t, filepath.Join(root, "docs/wo/nested/task-09-z.md"), "z")

	track, err := ReadTrack(root, "docs/wo")

	if err != nil {
		t.Fatal(err)
	}
	if !track.Dir || len(track.Files) != 2 || track.Files[0].RelPath != "docs/wo/task-01-a.md" ||
		track.Files[1].RelPath != "docs/wo/task-02-b.md" {
		t.Fatalf("track = %+v", track)
	}
}

func TestReadTrack_DirectorySkipsLinkedTaskFiles(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "task-x.md")
	mustWrite(t, outside, "secret")
	mustWrite(t, filepath.Join(root, "docs/wo/task-01.md"), "ok")
	if err := os.Symlink(outside, filepath.Join(root, "docs/wo/task-02.md")); err != nil {
		t.Fatal(err)
	}

	track, err := ReadTrack(root, "docs/wo")

	if err != nil || len(track.Files) != 1 || track.Files[0].RelPath != "docs/wo/task-01.md" {
		t.Fatalf("track = %+v, err = %v", track, err)
	}
}

func TestReadTrack_DirectoryCapsTheFileCount(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < MaxTrackedFiles+3; i++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("wo/task-%04d.md", i)), "x")
	}

	track, err := ReadTrack(root, "wo")

	if err != nil || len(track.Files) != MaxTrackedFiles || !track.Truncated {
		t.Fatalf("files = %d, truncated = %v, err = %v", len(track.Files), track.Truncated, err)
	}
}

func TestReadTrack_RefusesUnsafeEntries(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	mustWrite(t, filepath.Join(outsideDir, "x.md"), "secret")
	mustWrite(t, filepath.Join(root, "docs/real.md"), "ok")
	mustWrite(t, filepath.Join(root, "docs/big.md"), string(make([]byte, MaxPlanFileBytes+1)))
	if err := os.Symlink(outsideDir, filepath.Join(root, "docs/outdir")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outsideDir, "x.md"), filepath.Join(root, "docs/outfile.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "docs/real.md"), filepath.Join(root, "docs/inlink.md")); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(root, "docs/real.md")

	cases := map[string]string{
		"escape":        "../x.md",
		"nested escape": "docs/../../x.md",
		"absolute":      abs,
		"empty":         "",
		"missing":       "docs/missing.md",
		"outside dir":   "docs/outdir",
		"outside file":  "docs/outfile.md",
		"inside link":   "docs/inlink.md",
		"too large":     "docs/big.md",
	}
	for name, rel := range cases {
		if _, err := ReadTrack(root, rel); !errors.Is(err, ErrInvalidTrack) {
			t.Errorf("%s (%q): err = %v, want ErrInvalidTrack", name, rel, err)
		}
	}
}
