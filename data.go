package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// eventMeta is written to data/pretalx/<prefix>/event.json. Its presence tells the
// module's content adapters to build talk and speaker pages for that prefix; prefixes
// without it (e.g. an archived year with hand-written pages) only provide data.
type eventMeta struct {
	Event         string   `json:"event"`
	Tags          []string `json:"tags,omitempty"`
	TalkLayout    string   `json:"talk_layout,omitempty"`
	SpeakerLayout string   `json:"speaker_layout,omitempty"`
}

func newEventMeta(event EventConfig) eventMeta {
	return eventMeta{
		Event:         event.Event,
		Tags:          event.Tags,
		TalkLayout:    event.TalkLayout,
		SpeakerLayout: event.SpeakerLayout,
	}
}

// dataFileRequest is one JSON file to write under the site's data directory.
type dataFileRequest struct {
	Path   string
	Data   interface{}
	DryRun bool
}

// writeDataFile writes req.Data as indented JSON. Data files are always overwritten
// because they mirror the current API state.
func writeDataFile(req dataFileRequest) error {
	if req.DryRun {
		fmt.Printf("  [dry-run] Would write %s\n", req.Path)
		return nil
	}

	dir := filepath.Dir(req.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	content, err := json.MarshalIndent(req.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	content = append(content, '\n')

	if err := os.WriteFile(req.Path, content, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", req.Path, err)
	}

	fmt.Printf("  Wrote %s\n", req.Path)
	return nil
}
