package scan

import (
	"errors"
	"os"
)

// ErrUnreadable means the target could not be read as a plan file: it is
// missing, too large, or not a regular file.
var ErrUnreadable = errors.New("plan file is unreadable")

// ReadFile reads the single plan file at relPath under root with the same path
// and size rules as a repository scan.
func ReadFile(root, relPath string) (ScannedFile, error) {
	target, err := resolveTarget(root, relPath)
	if err != nil {
		return ScannedFile{}, err
	}
	file, reason := readPlanFile(target, relPath)
	switch reason {
	case "":
		return file, nil
	case ReasonNotRegularFile:
		return ScannedFile{}, ErrNotRegularFile
	}
	if _, statErr := os.Lstat(target); statErr != nil {
		return ScannedFile{}, statErr
	}
	return ScannedFile{}, ErrUnreadable
}
