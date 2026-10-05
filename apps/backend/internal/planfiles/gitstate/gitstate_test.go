package gitstate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// git runs a git command in dir for test setup and returns its output.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newRepo creates a repository with one commit that holds the given files.
func newRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "Test")
	git(t, root, "config", "user.email", "test@example.com")
	git(t, root, "config", "commit.gpgsign", "false")
	for rel, content := range files {
		write(t, root, rel, content)
	}
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "initial")
	return root
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func equal(a, b []string) bool {
	return strings.Join(a, "\x00") == strings.Join(b, "\x00")
}

func TestDirtyReportsModifiedStagedAndUntrackedFilesInDirs(t *testing.T) {
	root := newRepo(t, map[string]string{
		"docs/plans/a.md": "a\n", "docs/plans/b.md": "b\n", "docs/plans/c.md": "c\n", "src/x.go": "x\n",
	})
	write(t, root, "docs/plans/a.md", "a2\n")
	write(t, root, "docs/plans/b.md", "b2\n")
	git(t, root, "add", "docs/plans/b.md")
	write(t, root, "docs/plans/new.md", "new\n")
	write(t, root, "src/x.go", "x2\n")

	got, err := Dirty(context.Background(), root, []string{"docs/plans"})
	if err != nil {
		t.Fatalf("Dirty: %v", err)
	}
	want := []string{"docs/plans/a.md", "docs/plans/b.md", "docs/plans/new.md"}
	if !equal(sortedKeys(got), want) {
		t.Fatalf("Dirty = %v, want %v", sortedKeys(got), want)
	}
}

func TestDirtyListsUntrackedFilesInsideNewDirectories(t *testing.T) {
	root := newRepo(t, map[string]string{"README.md": "r\n"})
	write(t, root, "docs/plans/deep/n.md", "n\n")
	got, err := Dirty(context.Background(), root, []string{"docs/plans"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"docs/plans/deep/n.md"}; !equal(sortedKeys(got), want) {
		t.Fatalf("Dirty = %v, want %v", sortedKeys(got), want)
	}
}

func TestDirtyReportsOnlyTheNewPathOfARename(t *testing.T) {
	root := newRepo(t, map[string]string{"docs/plans/old.md": "content that is long enough to detect\n"})
	git(t, root, "mv", "docs/plans/old.md", "docs/plans/renamed.md")
	got, err := Dirty(context.Background(), root, []string{"docs/plans"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"docs/plans/renamed.md"}; !equal(sortedKeys(got), want) {
		t.Fatalf("Dirty = %v, want %v", sortedKeys(got), want)
	}
}

func TestDirtyReturnsPathsRelativeToTheGivenRootInsideALargerRepository(t *testing.T) {
	top := newRepo(t, map[string]string{"sub/docs/plans/a.md": "a\n", "other.md": "o\n"})
	write(t, top, "sub/docs/plans/a.md", "a2\n")
	write(t, top, "other.md", "o2\n")
	got, err := Dirty(context.Background(), filepath.Join(top, "sub"), []string{"docs/plans"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"docs/plans/a.md"}; !equal(sortedKeys(got), want) {
		t.Fatalf("Dirty = %v, want %v", sortedKeys(got), want)
	}
}

func TestDirtyTreatsPathspecMagicAsLiteralText(t *testing.T) {
	root := newRepo(t, map[string]string{"docs/plans/a.md": "a\n"})
	write(t, root, "docs/plans/a.md", "a2\n")
	got, err := Dirty(context.Background(), root, []string{"docs/*"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("a glob matched files: %v", sortedKeys(got))
	}
}

func TestDirtyOnANonRepositoryReturnsErrNotRepository(t *testing.T) {
	_, err := Dirty(context.Background(), t.TempDir(), []string{"docs/plans"})
	if err != ErrNotRepository {
		t.Fatalf("err = %v, want ErrNotRepository", err)
	}
}

func TestDirtyOnAMissingDirectoryReturnsErrNotRepository(t *testing.T) {
	_, err := Dirty(context.Background(), filepath.Join(t.TempDir(), "gone"), []string{"docs/plans"})
	if err != ErrNotRepository {
		t.Fatalf("err = %v, want ErrNotRepository", err)
	}
}

func TestBusyDetectsEveryInterruptedOperation(t *testing.T) {
	ctx := context.Background()
	root := newRepo(t, map[string]string{"a.md": "a\n"})
	if Busy(ctx, root) {
		t.Fatal("a clean repository is busy")
	}
	gitDir := filepath.Join(root, ".git")
	for _, name := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "rebase-merge", "rebase-apply"} {
		marker := filepath.Join(gitDir, name)
		if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if !Busy(ctx, root) {
			t.Fatalf("%s did not make the repository busy", name)
		}
		if err := os.Remove(marker); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBusyIsFalseOutsideARepository(t *testing.T) {
	if Busy(context.Background(), t.TempDir()) {
		t.Fatal("a non-repository is busy")
	}
}

func statusPorcelain(t *testing.T, root string) string {
	t.Helper()
	return git(t, root, "status", "--porcelain")
}

func committedFiles(t *testing.T, root, rev string) []string {
	t.Helper()
	out := git(t, root, "show", "--name-only", "--format=", "-z", rev)
	var files []string
	for _, f := range strings.Split(out, "\x00") {
		if f = strings.TrimSpace(f); f != "" {
			files = append(files, f)
		}
	}
	sort.Strings(files)
	return files
}

func TestCommitPathsCommitsOnlyTheGivenPathsAndKeepsOtherStagedChangesStaged(t *testing.T) {
	root := newRepo(t, map[string]string{
		"docs/plans/modified.md": "m\n", "unrelated.txt": "u\n", "staged.txt": "s\n",
	})
	write(t, root, "docs/plans/modified.md", "m2\n")
	write(t, root, "docs/plans/untracked.md", "n\n")
	write(t, root, "unrelated.txt", "u2\n")
	write(t, root, "staged.txt", "s2\n")
	git(t, root, "add", "staged.txt")

	sha, output, err := CommitPaths(context.Background(), root,
		[]string{"docs/plans/modified.md", "docs/plans/untracked.md"}, "docs(plans): update plan files")
	if err != nil {
		t.Fatalf("CommitPaths: %v\n%s", err, output)
	}
	if head := strings.TrimSpace(git(t, root, "rev-parse", "HEAD")); sha != head {
		t.Fatalf("sha = %q, HEAD = %q", sha, head)
	}
	want := []string{"docs/plans/modified.md", "docs/plans/untracked.md"}
	if got := committedFiles(t, root, "HEAD"); !equal(got, want) {
		t.Fatalf("commit files = %v, want %v", got, want)
	}
	if msg := strings.TrimSpace(git(t, root, "log", "-1", "--format=%s")); msg != "docs(plans): update plan files" {
		t.Fatalf("message = %q", msg)
	}
	if status := statusPorcelain(t, root); status != "M  staged.txt\n M unrelated.txt\n" {
		t.Fatalf("status after commit = %q", status)
	}
}

func TestCommitPathsHandlesNamesWithSpacesAndLeadingDashes(t *testing.T) {
	root := newRepo(t, map[string]string{"docs/plans/my plan.md": "p\n", "docs/plans/-dash.md": "d\n"})
	write(t, root, "docs/plans/my plan.md", "p2\n")
	write(t, root, "docs/plans/-dash.md", "d2\n")
	write(t, root, "docs/plans/new plan [1].md", "n\n")

	paths := []string{"docs/plans/my plan.md", "docs/plans/-dash.md", "docs/plans/new plan [1].md"}
	if _, output, err := CommitPaths(context.Background(), root, paths, "msg"); err != nil {
		t.Fatalf("CommitPaths: %v\n%s", err, output)
	}
	want := []string{"docs/plans/-dash.md", "docs/plans/my plan.md", "docs/plans/new plan [1].md"}
	if got := committedFiles(t, root, "HEAD"); !equal(got, want) {
		t.Fatalf("commit files = %v, want %v", got, want)
	}
	if status := statusPorcelain(t, root); status != "" {
		t.Fatalf("status after commit = %q", status)
	}
}

func TestCommitPathsDoesNotExpandGlobsInPaths(t *testing.T) {
	root := newRepo(t, map[string]string{"docs/plans/a.md": "a\n"})
	write(t, root, "docs/plans/a.md", "a2\n")
	write(t, root, "docs/plans/b.md", "b\n")
	_, _, err := CommitPaths(context.Background(), root, []string{"docs/plans/*.md"}, "msg")
	if err == nil {
		t.Fatal("a glob path committed")
	}
	if status := statusPorcelain(t, root); status != " M docs/plans/a.md\n?? docs/plans/b.md\n" {
		t.Fatalf("status = %q", status)
	}
}

func TestCommitPathsCommitsAStagedDeletion(t *testing.T) {
	root := newRepo(t, map[string]string{"docs/plans/gone.md": "g\n", "docs/plans/keep.md": "k\n"})
	git(t, root, "rm", "-q", "docs/plans/gone.md")
	write(t, root, "docs/plans/keep.md", "k2\n")
	paths := []string{"docs/plans/gone.md", "docs/plans/keep.md"}
	if _, output, err := CommitPaths(context.Background(), root, paths, "msg"); err != nil {
		t.Fatalf("CommitPaths: %v\n%s", err, output)
	}
	if got := committedFiles(t, root, "HEAD"); !equal(got, paths) {
		t.Fatalf("commit files = %v, want %v", got, paths)
	}
}

func TestCommitPathsRejectedByAHookLeavesTheRepositoryUntouched(t *testing.T) {
	root := newRepo(t, map[string]string{"docs/plans/modified.md": "m\n", "staged.txt": "s\n"})
	write(t, root, "docs/plans/modified.md", "m2\n")
	write(t, root, "docs/plans/untracked.md", "n\n")
	write(t, root, "staged.txt", "s2\n")
	git(t, root, "add", "staged.txt")
	hook := "#!/bin/sh\necho 'lint failed: bad plan' >&2\nexit 1\n"
	if err := os.MkdirAll(filepath.Join(root, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "hooks", "pre-commit"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	headBefore := git(t, root, "rev-parse", "HEAD")
	before := statusPorcelain(t, root)

	sha, output, err := CommitPaths(context.Background(), root,
		[]string{"docs/plans/modified.md", "docs/plans/untracked.md"}, "msg")
	if err == nil {
		t.Fatal("a rejected commit returned no error")
	}
	if sha != "" {
		t.Fatalf("sha = %q on failure", sha)
	}
	if !strings.Contains(output, "lint failed: bad plan") {
		t.Fatalf("output lacks the hook message: %q", output)
	}
	if after := statusPorcelain(t, root); after != before {
		t.Fatalf("status changed:\nbefore %q\nafter  %q", before, after)
	}
	if headAfter := git(t, root, "rev-parse", "HEAD"); headAfter != headBefore {
		t.Fatal("HEAD moved")
	}
}
