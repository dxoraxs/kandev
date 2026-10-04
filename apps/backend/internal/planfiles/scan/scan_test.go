package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testDir = "docs/plans"

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func relPaths(files []ScannedFile) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.RelPath)
	}
	return out
}

func hasError(errs []FileError, rel string, reason Reason) bool {
	for _, e := range errs {
		if e.RelPath == rel && e.Reason == reason {
			return true
		}
	}
	return false
}

// @covers AC-TASKS-PLAN-FILES-002.1
func TestScanRepository_SortedMarkdownOnly(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, testDir)
	mustWrite(t, filepath.Join(dir, "b.md"), "b")
	mustWrite(t, filepath.Join(dir, "a.md"), "a")
	mustWrite(t, filepath.Join(dir, "notes.txt"), "x")
	mustWrite(t, filepath.Join(dir, "nested", "c.md"), "c")

	files, errs := ScanRepository(root, []string{testDir})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := strings.Join(relPaths(files), ",")
	if got != "docs/plans/a.md,docs/plans/b.md" {
		t.Fatalf("got %s", got)
	}
	if string(files[0].Content) != "a" || files[0].Mode.Perm() != 0o644 {
		t.Fatalf("bad content/mode: %q %v", files[0].Content, files[0].Mode)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.4
func TestScanRepository_MissingDirSkipped(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, testDir, "a.md"), "a")
	files, errs := ScanRepository(root, []string{"nope", testDir})
	if len(errs) != 0 || len(files) != 1 {
		t.Fatalf("files=%v errs=%v", relPaths(files), errs)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.4
func TestScanRepository_MissingRoot(t *testing.T) {
	files, errs := ScanRepository(filepath.Join(t.TempDir(), "gone"), []string{testDir})
	if len(files) != 0 || len(errs) != 1 || errs[0].Reason != ReasonRootMissing {
		t.Fatalf("files=%v errs=%v", files, errs)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.1
func TestScanRepository_SymlinkedFileReported(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, testDir)
	mustWrite(t, filepath.Join(dir, "real.md"), "r")
	outside := filepath.Join(t.TempDir(), "secret.md")
	mustWrite(t, outside, "secret")
	if err := os.Symlink(outside, filepath.Join(dir, "link.md")); err != nil {
		t.Fatal(err)
	}
	files, errs := ScanRepository(root, []string{testDir})
	if len(files) != 1 || files[0].RelPath != "docs/plans/real.md" {
		t.Fatalf("files=%v", relPaths(files))
	}
	if !hasError(errs, "docs/plans/link.md", ReasonNotRegularFile) {
		t.Fatalf("errs=%v", errs)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.1
func TestScanRepository_SymlinkedDirOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	mustWrite(t, filepath.Join(outside, "x.md"), "x")
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, testDir)); err != nil {
		t.Fatal(err)
	}
	files, errs := ScanRepository(root, []string{testDir})
	if len(files) != 0 || !hasError(errs, testDir, ReasonOutsideRoot) {
		t.Fatalf("files=%v errs=%v", relPaths(files), errs)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.2
func TestScanRepository_OversizedNotRead(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, testDir)
	mustWrite(t, filepath.Join(dir, "ok.md"), "ok")
	big := filepath.Join(dir, "big.md")
	mustWrite(t, big, "")
	if err := os.Truncate(big, MaxPlanFileBytes+1); err != nil {
		t.Fatal(err)
	}
	files, errs := ScanRepository(root, []string{testDir})
	if len(files) != 1 || !hasError(errs, "docs/plans/big.md", ReasonTooLarge) {
		t.Fatalf("files=%v errs=%v", relPaths(files), errs)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.2
func TestScanRepository_AtLimitIsAccepted(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, testDir, "edge.md")
	mustWrite(t, p, "")
	if err := os.Truncate(p, MaxPlanFileBytes); err != nil {
		t.Fatal(err)
	}
	files, errs := ScanRepository(root, []string{testDir})
	if len(files) != 1 || len(errs) != 0 {
		t.Fatalf("files=%v errs=%v", relPaths(files), errs)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.2
func TestScanRepository_TruncatesBeyondLimit(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, testDir)
	for i := 0; i < MaxFilesPerDirectory+1; i++ {
		mustWrite(t, filepath.Join(dir, fmt.Sprintf("p%04d.md", i)), "x")
	}
	files, errs := ScanRepository(root, []string{testDir})
	if len(files) != MaxFilesPerDirectory {
		t.Fatalf("got %d files", len(files))
	}
	if !hasError(errs, testDir, ReasonTruncated) {
		t.Fatalf("errs=%v", errs)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.4
func TestScanRepository_UnreadableFileDoesNotStopPass(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	root := t.TempDir()
	dir := filepath.Join(root, testDir)
	mustWrite(t, filepath.Join(dir, "a.md"), "a")
	mustWrite(t, filepath.Join(dir, "b.md"), "b")
	if err := os.Chmod(filepath.Join(dir, "a.md"), 0); err != nil {
		t.Fatal(err)
	}
	files, errs := ScanRepository(root, []string{testDir})
	if len(files) != 1 || files[0].RelPath != "docs/plans/b.md" {
		t.Fatalf("files=%v", relPaths(files))
	}
	if !hasError(errs, "docs/plans/a.md", ReasonUnreadable) {
		t.Fatalf("errs=%v", errs)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.1
func TestValidateDirectory(t *testing.T) {
	good := map[string]string{
		"docs/plans":   "docs/plans",
		"docs//plans/": "docs/plans",
		"./docs/plans": "docs/plans",
		".":            ".",
	}
	for in, want := range good {
		got, err := ValidateDirectory(in)
		if err != nil || got != want {
			t.Errorf("%q: got %q err %v, want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "/etc", "../x", "a/../../x", "a/..", `a\..\b`} {
		if _, err := ValidateDirectory(in); err == nil {
			t.Errorf("%q: expected error", in)
		}
	}
}
