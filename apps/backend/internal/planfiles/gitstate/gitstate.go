// Package gitstate reads the uncommitted state of plan files and commits a set
// of paths without touching any other change. Every path is passed to git as a
// literal pathspec after a `--` separator.
package gitstate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/kandev/kandev/internal/common/subproc"
)

// ErrNotRepository reports that a directory is not inside a git working tree.
var ErrNotRepository = errors.New("not a git working tree")

// Names inside the git directory that mark an interrupted operation.
var busyMarkers = []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "rebase-merge", "rebase-apply"}

// base holds the options every invocation starts with: the repository root,
// literal pathspecs, and no opportunistic index writes.
func base(root string) []string {
	return []string{"-C", root, "--literal-pathspecs", "--no-optional-locks"}
}

func run(ctx context.Context, class subproc.GitWorkClass, root string, args ...string) ([]byte, error) {
	cmd := subproc.NewGitCommand(ctx, append(base(root), args...)...)
	return subproc.RunGitOutputClass(ctx, class, cmd)
}

func runCombined(ctx context.Context, class subproc.GitWorkClass, root string, args ...string) ([]byte, error) {
	cmd := subproc.NewGitCommand(ctx, append(base(root), args...)...)
	return subproc.RunGitCombinedOutputClass(ctx, class, cmd)
}

// prefix returns the path of root inside its working tree ("" at the top
// level, otherwise with a trailing slash). It reports ErrNotRepository when
// root is not inside a working tree.
func prefix(ctx context.Context, root string) (string, error) {
	out, err := run(ctx, subproc.GitBackground, root, "rev-parse", "--is-inside-work-tree", "--show-prefix")
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", ErrNotRepository
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "true" {
		return "", ErrNotRepository
	}
	if len(lines) > 1 {
		return strings.TrimSpace(lines[1]), nil
	}
	return "", nil
}

// Dirty returns the paths, relative to root, of the files under dirs that are
// modified, staged, or untracked. A directory that holds untracked files
// reports each file. It returns ErrNotRepository when root is not inside a git
// working tree.
func Dirty(ctx context.Context, root string, dirs []string) (map[string]struct{}, error) {
	pre, err := prefix(ctx, root)
	if err != nil {
		return nil, err
	}
	dirty := map[string]struct{}{}
	if len(dirs) == 0 {
		return dirty, nil
	}
	args := append([]string{"status", "--porcelain=v1", "-z", "--untracked-files=all", "--"}, dirs...)
	out, err := run(ctx, subproc.GitBackground, root, args...)
	if err != nil {
		return nil, err
	}
	for _, p := range parseStatus(out) {
		if rel, ok := strings.CutPrefix(p, pre); ok {
			dirty[rel] = struct{}{}
		}
	}
	return dirty, nil
}

// parseStatus returns the current path of every record of `status -z`. A
// rename or copy record is followed by a second record that holds the original
// path; it is skipped.
func parseStatus(out []byte) []string {
	records := strings.Split(string(out), "\x00")
	var paths []string
	for i := 0; i < len(records); i++ {
		rec := records[i]
		if len(rec) < 4 {
			continue
		}
		paths = append(paths, rec[3:])
		if strings.ContainsAny(rec[:2], "RC") {
			i++
		}
	}
	return paths
}

// Busy reports that the repository at root is in the middle of a merge,
// cherry-pick, revert, or rebase. A directory that is not a repository is not
// busy.
func Busy(ctx context.Context, root string) bool {
	out, err := run(ctx, subproc.GitBackground, root, "rev-parse", "--git-dir")
	if err != nil {
		return false
	}
	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}
	for _, name := range busyMarkers {
		if _, err := os.Lstat(filepath.Join(gitDir, name)); err == nil {
			return true
		}
	}
	return false
}

// CommitPaths creates one commit on the current branch that contains exactly
// paths and nothing else: changes that are already staged for other paths stay
// staged and out of the commit. Untracked paths are added first. On failure the
// index is restored and output holds git's combined output.
func CommitPaths(ctx context.Context, root string, paths []string, message string) (sha, output string, err error) {
	if len(paths) == 0 {
		return "", "", errors.New("no paths to commit")
	}
	untracked, err := untrackedPaths(ctx, root, paths)
	if err != nil {
		return "", "", err
	}
	if len(untracked) > 0 {
		out, err := runCombined(ctx, subproc.GitInteractive, root, append([]string{"add", "--"}, untracked...)...)
		if err != nil {
			resetPaths(ctx, root, untracked)
			return "", string(out), err
		}
	}
	args := append([]string{"commit", "--only", "-m", message, "--"}, paths...)
	out, err := runCombined(ctx, subproc.GitInteractive, root, args...)
	if err != nil {
		resetPaths(ctx, root, untracked)
		return "", string(out), err
	}
	head, err := run(ctx, subproc.GitInteractive, root, "rev-parse", "HEAD")
	if err != nil {
		return "", string(out), err
	}
	return strings.TrimSpace(string(head)), string(out), nil
}

// untrackedPaths returns the paths that exist on disk but are not in the index.
func untrackedPaths(ctx context.Context, root string, paths []string) ([]string, error) {
	out, err := run(ctx, subproc.GitInteractive, root, append([]string{"ls-files", "-z", "--"}, paths...)...)
	if err != nil {
		return nil, err
	}
	tracked := map[string]struct{}{}
	for _, p := range strings.Split(string(out), "\x00") {
		tracked[p] = struct{}{}
	}
	var untracked []string
	for _, p := range paths {
		if _, ok := tracked[p]; ok {
			continue
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(p))); err == nil {
			untracked = append(untracked, p)
		}
	}
	return untracked, nil
}

// resetPaths unstages paths that CommitPaths added, so a failed commit leaves
// them untracked again. It is best effort: the commit error is what matters.
func resetPaths(ctx context.Context, root string, paths []string) {
	if len(paths) == 0 {
		return
	}
	_, _ = runCombined(context.WithoutCancel(ctx), subproc.GitInteractive, root, append([]string{"reset", "-q", "--"}, paths...)...)
}
