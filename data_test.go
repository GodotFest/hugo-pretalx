package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteDataFileWritesEventMeta(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "pretalx", "2026", "event.json")
	event := EventConfig{Event: "conf-26", Prefix: "2026", Tags: []string{"2026", "conf26"}, TalkLayout: "custom-talk"}

	if err := writeDataFile(dataFileRequest{Path: path, Data: newEventMeta(event)}); err != nil {
		t.Fatalf("writeDataFile: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("parsing event.json: %v", err)
	}
	want := map[string]interface{}{
		"event":       "conf-26",
		"tags":        []interface{}{"2026", "conf26"},
		"talk_layout": "custom-talk",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("event.json = %v, want %v", got, want)
	}
}

func TestWriteDataFileDryRunWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "talks.json")

	if err := writeDataFile(dataFileRequest{Path: path, Data: []interface{}{}, DryRun: true}); err != nil {
		t.Fatalf("writeDataFile: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected dry run to leave the file unwritten")
	}
}
