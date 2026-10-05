package scan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// MaxTrackedFiles is the number of task-*.md files read per tracked
	// directory.
	MaxTrackedFiles = 200

	trackedPrefix = "task-"
)

// ErrInvalidTrack reports a `tracks` entry that leaves the repository, is a
// symbolic link, does not exist, or cannot be read.
var ErrInvalidTrack = errors.New("invalid track")

// Track is what one `tracks` entry resolves to.
type Track struct {
	// Dir reports a tracked directory; Files then holds the task-*.md files
	// directly inside it, in name order. For a tracked file, Files holds that
	// one file.
	Dir   bool
	Files []ScannedFile
	// ModTime is the modification time of the entry itself.
	ModTime time.Time
	// Truncated reports that the directory held more than MaxTrackedFiles
	// task files and the rest were not read.
	Truncated bool
}

// ReadTrack reads the file or directory a `tracks` entry names, relative to
// root. The entry must be relative, free of ".." segments, and neither it nor
// anything it resolves to may be a symbolic link that leaves the resolved
// root. Task files that are links or not regular files are skipped.
func ReadTrack(root, relPath string) (Track, error) {
	cleaned, err := ValidateDirectory(relPath)
	if err != nil {
		return Track{}, ErrInvalidTrack
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return Track{}, ErrInvalidTrack
	}
	full := filepath.Join(resolvedRoot, filepath.FromSlash(cleaned))
	info, err := os.Lstat(full)
	if err != nil || info.Mode()&fs.ModeSymlink != 0 {
		return Track{}, ErrInvalidTrack
	}
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil || !inside(resolvedRoot, resolved) {
		return Track{}, ErrInvalidTrack
	}
	if info.IsDir() {
		return readTrackedDirectory(resolved, cleaned, info.ModTime())
	}
	file, reason := readPlanFile(resolved, cleaned)
	if reason != "" {
		return Track{}, ErrInvalidTrack
	}
	return Track{Files: []ScannedFile{file}, ModTime: info.ModTime()}, nil
}

func readTrackedDirectory(resolvedDir, rel string, modTime time.Time) (Track, error) {
	entries, err := os.ReadDir(resolvedDir)
	if err != nil {
		return Track{}, ErrInvalidTrack
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), trackedPrefix) && strings.HasSuffix(e.Name(), markdownExt) && e.Type().IsRegular() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	track := Track{Dir: true, ModTime: modTime}
	if len(names) > MaxTrackedFiles {
		names = names[:MaxTrackedFiles]
		track.Truncated = true
	}
	for _, name := range names {
		file, reason := readPlanFile(filepath.Join(resolvedDir, name), filepath.ToSlash(filepath.Join(rel, name)))
		if reason == "" {
			track.Files = append(track.Files, file)
		}
	}
	return track, nil
}
