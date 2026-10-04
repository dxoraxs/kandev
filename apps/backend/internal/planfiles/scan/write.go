package scan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	// ErrHashMismatch means the file on disk no longer matches the expected hash.
	ErrHashMismatch = errors.New("plan file changed on disk")
	// ErrOutsideRoot means the target resolves outside the repository root.
	ErrOutsideRoot = errors.New("path resolves outside repository root")
	// ErrNotRegularFile means the target is a symlink or another non-regular file.
	ErrNotRegularFile = errors.New("target is not a regular file")
)

const tempPattern = ".planfile-*.tmp"

// WriteFile replaces the plan file at relPath under root with content, only if
// the bytes currently on disk hash to expectedHash. hash is the content hash
// function used to produce expectedHash (the format package's ContentHash), so
// this package never defines a second hashing scheme. The replacement is
// written to a temporary file in the same directory, synced, given the
// original mode and renamed over the target; the temp file never survives a
// failure.
func WriteFile(root, relPath string, content []byte, expectedHash string, hash func([]byte) string) error {
	target, err := resolveTarget(root, relPath)
	if err != nil {
		return err
	}
	info, err := os.Lstat(target)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return ErrNotRegularFile
	}
	current, err := os.ReadFile(target)
	if err != nil {
		return err
	}
	if hash(current) != expectedHash {
		return ErrHashMismatch
	}
	return replaceFile(target, content, info.Mode().Perm())
}

// resolveTarget returns the absolute path of relPath with its directory
// symlink-resolved and verified to lie inside the resolved root.
func resolveTarget(root, relPath string) (string, error) {
	cleaned, err := ValidateDirectory(relPath)
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	full := filepath.Join(resolvedRoot, filepath.FromSlash(cleaned))
	resolvedDir, err := filepath.EvalSymlinks(filepath.Dir(full))
	if err != nil {
		return "", err
	}
	if !inside(resolvedRoot, resolvedDir) {
		return "", ErrOutsideRoot
	}
	return filepath.Join(resolvedDir, filepath.Base(full)), nil
}

func replaceFile(target string, content []byte, mode os.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(target), tempPattern)
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()
	if _, err = tmp.Write(content); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err = os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, target)
}
