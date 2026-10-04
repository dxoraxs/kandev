package planfiles

import (
	"strings"

	"github.com/kandev/kandev/internal/planfiles/format"
)

// defaultNotesHeading is the owner-notes heading when the config sets none.
const defaultNotesHeading = "Owner notes"

// notesHeading returns the heading of the owner-notes section.
func notesHeading(cfg *Config) string {
	if heading := strings.TrimSpace(cfg.NotesHeading); heading != "" {
		return heading
	}
	return defaultNotesHeading
}

// composeEdit sets keys and appends note to the notes section in one pass, so
// the result is the single byte slice one compare-and-swap write stores.
func composeEdit(content []byte, keys map[string]string, heading, note string) ([]byte, error) {
	withKeys, err := format.SetKeys(content, keys)
	if err != nil {
		return nil, err
	}
	return format.AppendNote(withKeys, heading, note)
}
