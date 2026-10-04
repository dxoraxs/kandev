// Package scan lists, reads and atomically rewrites Markdown plan files
// directly inside configured directories of one repository root. Every path it
// touches resolves, after symlink evaluation, to a regular file inside the
// resolved root.
package scan

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// MaxPlanFileBytes is the largest plan file that is read.
	MaxPlanFileBytes = 1 << 20
	// MaxFilesPerDirectory is the number of Markdown files read per directory.
	MaxFilesPerDirectory = 1000

	markdownExt = ".md"
	parentDir   = ".."
)

// Reason classifies why a path was reported instead of read. The set is closed.
type Reason string

const (
	ReasonRootMissing    Reason = "root_missing"
	ReasonOutsideRoot    Reason = "outside_root"
	ReasonTruncated      Reason = "truncated"
	ReasonNotRegularFile Reason = "not_regular_file"
	ReasonTooLarge       Reason = "too_large"
	ReasonUnreadable     Reason = "unreadable"
)

// ScannedFile is a plan file candidate read from disk. RelPath is
// slash-separated and relative to the repository root.
type ScannedFile struct {
	RelPath string
	Content []byte
	Mode    fs.FileMode
}

// FileError reports a path that was skipped. RelPath is empty for a
// repository-level error and names the directory for a directory-level one.
type FileError struct {
	RelPath string
	Reason  Reason
}

// ValidateDirectory returns the cleaned form of a configured directory. It
// must be a non-empty relative path without any ".." segment.
func ValidateDirectory(dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", errors.New("directory is empty")
	}
	slashed := strings.ReplaceAll(dir, `\`, "/")
	if filepath.IsAbs(dir) || strings.HasPrefix(slashed, "/") {
		return "", fmt.Errorf("directory %q must be relative", dir)
	}
	for _, seg := range strings.Split(slashed, "/") {
		if seg == parentDir {
			return "", fmt.Errorf("directory %q must not contain %q", dir, parentDir)
		}
	}
	return filepath.ToSlash(filepath.Clean(dir)), nil
}

// inside reports whether path equals root or lies below it. Both arguments
// must already be symlink-resolved.
func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != parentDir && !strings.HasPrefix(rel, parentDir+string(filepath.Separator))
}

// ScanRepository reads the Markdown files directly inside each directory of
// dirs under root. Problems are returned per path and never stop the scan.
func ScanRepository(root string, dirs []string) ([]ScannedFile, []FileError) {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, []FileError{{Reason: ReasonRootMissing}}
	}
	var files []ScannedFile
	var errs []FileError
	for _, dir := range dirs {
		cleaned, err := ValidateDirectory(dir)
		if err != nil {
			errs = append(errs, FileError{RelPath: dir, Reason: ReasonOutsideRoot})
			continue
		}
		f, e := scanDirectory(resolvedRoot, cleaned)
		files = append(files, f...)
		errs = append(errs, e...)
	}
	return files, errs
}

func scanDirectory(resolvedRoot, dir string) ([]ScannedFile, []FileError) {
	resolvedDir, err := filepath.EvalSymlinks(filepath.Join(resolvedRoot, filepath.FromSlash(dir)))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, []FileError{{RelPath: dir, Reason: ReasonUnreadable}}
	}
	if !inside(resolvedRoot, resolvedDir) {
		return nil, []FileError{{RelPath: dir, Reason: ReasonOutsideRoot}}
	}
	entries, err := os.ReadDir(resolvedDir)
	if err != nil {
		return nil, []FileError{{RelPath: dir, Reason: ReasonUnreadable}}
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), markdownExt) && !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var files []ScannedFile
	var errs []FileError
	if len(names) > MaxFilesPerDirectory {
		names = names[:MaxFilesPerDirectory]
		errs = append(errs, FileError{RelPath: dir, Reason: ReasonTruncated})
	}
	for _, name := range names {
		rel := filepath.ToSlash(filepath.Join(dir, name))
		file, reason := readPlanFile(filepath.Join(resolvedDir, name), rel)
		if reason != "" {
			errs = append(errs, FileError{RelPath: rel, Reason: reason})
			continue
		}
		files = append(files, file)
	}
	return files, errs
}

// readPlanFile applies the regular-file and size checks, then reads the file.
// A non-empty reason means the file was skipped.
func readPlanFile(path, rel string) (ScannedFile, Reason) {
	info, err := os.Lstat(path)
	if err != nil {
		return ScannedFile{}, ReasonUnreadable
	}
	if !info.Mode().IsRegular() {
		return ScannedFile{}, ReasonNotRegularFile
	}
	if info.Size() > MaxPlanFileBytes {
		return ScannedFile{}, ReasonTooLarge
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ScannedFile{}, ReasonUnreadable
	}
	if len(content) > MaxPlanFileBytes {
		return ScannedFile{}, ReasonTooLarge
	}
	return ScannedFile{RelPath: rel, Content: content, Mode: info.Mode()}, ""
}
