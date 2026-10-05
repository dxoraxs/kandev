package scan

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrExists means a file or link already occupies the path to create.
var ErrExists = errors.New("plan file already exists")

const createdFileMode = 0o644

// CreateFile creates the plan file at relPath under root with content. The
// directory is created when missing. The nearest existing ancestor of the
// directory and the directory itself must resolve inside the resolved root
// before anything is created, and the file is opened exclusively without
// following links, so an existing file or link is never replaced or written
// through. A failed write removes the file it created.
func CreateFile(root, relPath string, content []byte) (err error) {
	cleaned, err := ValidateDirectory(relPath)
	if err != nil {
		return err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	dir := filepath.Dir(filepath.Join(resolvedRoot, filepath.FromSlash(cleaned)))
	if err := makeDirectoryInside(resolvedRoot, dir); err != nil {
		return err
	}
	target, err := resolveTarget(root, relPath)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, createdFileMode)
	if errors.Is(err, fs.ErrExist) {
		return ErrExists
	}
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = f.Close()
			_ = os.Remove(target)
		}
	}()
	if _, err = f.Write(content); err != nil {
		return fmt.Errorf("write plan file: %w", err)
	}
	if err = f.Sync(); err != nil {
		return fmt.Errorf("sync plan file: %w", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("close plan file: %w", err)
	}
	return nil
}

// makeDirectoryInside creates dir (below resolvedRoot) after checking that its
// nearest existing ancestor resolves inside resolvedRoot.
func makeDirectoryInside(resolvedRoot, dir string) error {
	existing := dir
	for {
		resolved, err := filepath.EvalSymlinks(existing)
		if err == nil {
			if !inside(resolvedRoot, resolved) {
				return ErrOutsideRoot
			}
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(existing)
		if parent == existing || !inside(resolvedRoot, parent) {
			return ErrOutsideRoot
		}
		existing = parent
	}
	return os.MkdirAll(dir, 0o755)
}
