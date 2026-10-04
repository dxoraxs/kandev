package format

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	keyTracks = "tracks"
	// MaxTracks is the maximum number of entries of the `tracks` key.
	MaxTracks = 20
)

// decodeTracks reads the optional `tracks` key. A value that is not a
// sequence of at most MaxTracks non-empty scalars is reported as
// invalid_tracks and leaves Tracks empty. Path safety is the caller's concern.
func (pf *PlanFile) decodeTracks(fields map[string]yaml.Node) {
	n, ok := fields[keyTracks]
	if !ok || isNull(n) {
		return
	}
	if n.Kind != yaml.SequenceNode {
		pf.fail(keyTracks, fmt.Errorf("invalid_tracks: must be a list of paths"))
		return
	}
	if len(n.Content) > MaxTracks {
		pf.fail(keyTracks, fmt.Errorf("invalid_tracks: at most %d entries", MaxTracks))
		return
	}
	tracks := make([]string, 0, len(n.Content))
	for _, item := range n.Content {
		s, err := scalarString(*item)
		s = strings.TrimSpace(s)
		if err != nil || s == "" {
			pf.fail(keyTracks, fmt.Errorf("invalid_tracks: entries must be non-empty paths"))
			return
		}
		tracks = append(tracks, s)
	}
	pf.Tracks = tracks
}
