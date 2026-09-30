package main

import (
	"reflect"
	"testing"
)

func TestPublicTalksDropsPrivateFields(t *testing.T) {
	talks := []interface{}{map[string]interface{}{
		"code":             "AAA",
		"title":            "Talk",
		"reviews":          []interface{}{"private"},
		"mean_score":       4.5,
		"invitation_token": "secret",
		"speakers": []interface{}{map[string]interface{}{
			"code":  "SPK",
			"name":  "Jane",
			"email": "jane@example.com",
		}},
	}}

	got := publicTalks(talks)

	want := []interface{}{map[string]interface{}{
		"code":     "AAA",
		"title":    "Talk",
		"speakers": []interface{}{map[string]interface{}{"code": "SPK", "name": "Jane"}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("publicTalks = %v, want %v", got, want)
	}
}

func TestPublicSpeakersDropsPrivateFields(t *testing.T) {
	speakers := []interface{}{map[string]interface{}{
		"code":           "SPK",
		"name":           "Jane",
		"avatar":         "https://example.com/a.webp",
		"email":          "jane@example.com",
		"availabilities": []interface{}{},
		"internal_notes": "private",
	}}

	got := publicSpeakers(speakers)

	want := []interface{}{map[string]interface{}{
		"code":   "SPK",
		"name":   "Jane",
		"avatar": "https://example.com/a.webp",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("publicSpeakers = %v, want %v", got, want)
	}
}
