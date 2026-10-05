package scan

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
)

// ErrNotGenerated means a file that does not start with the generated marker
// occupies the path, so it is not ours to replace.
var ErrNotGenerated = errors.New("file exists and is not generated")

const generatedFileMode = 0o644

// WriteGenerated writes content to relPath under root. The file is created
// when absent and replaced only when its current bytes start with marker;
// anything else is left untouched and reported as ErrNotGenerated. A link or
// other non-regular file at the path is never followed or replaced. The
// directory must already exist and resolve inside the resolved root. The
// replacement is a temporary file in the same directory renamed over the
// target, and nothing is written when the bytes are already equal. It reports
// whether the file was written.
func WriteGenerated(root, relPath string, content []byte, marker string) (bool, error) {
	target, err := resolveTarget(root, relPath)
	if err != nil {
		return false, err
	}
	info, err := os.Lstat(target)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return true, replaceFile(target, content, generatedFileMode)
	case err != nil:
		return false, err
	case !info.Mode().IsRegular():
		return false, ErrNotRegularFile
	}
	current, err := os.ReadFile(target)
	if err != nil {
		return false, err
	}
	if !bytes.HasPrefix(current, []byte(marker)) {
		return false, ErrNotGenerated
	}
	if bytes.Equal(current, content) {
		return false, nil
	}
	return true, replaceFile(target, content, info.Mode().Perm())
}
