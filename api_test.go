package main

import (
	"reflect"
	"testing"
)

func talk(state string, speakerCodes ...string) map[string]interface{} {
	speakers := make([]interface{}, 0, len(speakerCodes))
	for _, code := range speakerCodes {
		speakers = append(speakers, map[string]interface{}{"code": code})
	}
	return map[string]interface{}{"state": state, "speakers": speakers}
}

func TestFilterByStateDefaultsToConfirmed(t *testing.T) {
	talks := []interface{}{
		talk("submitted"),
		talk("accepted"),
		talk("confirmed"),
		talk("rejected"),
		talk("withdrawn"),
		map[string]interface{}{"title": "no state field"},
	}

	filtered := filterByState(talks, allowedStates(nil))
	if len(filtered) != 2 {
		t.Fatalf("expected 2 talks (confirmed, stateless), got %d", len(filtered))
	}
	for _, item := range filtered {
		m := item.(map[string]interface{})
		if state, ok := m["state"].(string); ok && state != "confirmed" {
			t.Fatalf("unexpected state %q in filtered talks", state)
		}
	}
}

func TestFilterByStateCustomStates(t *testing.T) {
	talks := []interface{}{
		talk("submitted"),
		talk("accepted"),
		talk("confirmed"),
		talk("rejected"),
	}

	filtered := filterByState(talks, allowedStates([]string{"confirmed", "accepted", "submitted"}))
	if len(filtered) != 3 {
		t.Fatalf("expected 3 talks, got %d", len(filtered))
	}
}

func TestFilterSpeakersByTalks(t *testing.T) {
	talks := []interface{}{talk("confirmed", "AAA"), talk("confirmed", "BBB")}
	speakers := []interface{}{
		map[string]interface{}{"code": "AAA"},
		map[string]interface{}{"code": "BBB"},
		map[string]interface{}{"code": "REJECTED_ONLY"},
	}

	filtered := filterSpeakersByTalks(speakers, talks)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 speakers, got %d", len(filtered))
	}
}

func TestNormalizeSubmissions(t *testing.T) {
	subs := []interface{}{
		map[string]interface{}{
			"state": "accepted",
			"slots": []interface{}{
				map[string]interface{}{
					"start": "2026-01-01T10:00:00+01:00",
					"room": map[string]interface{}{
						"id":   float64(2),
						"name": map[string]interface{}{"en": "Ballsaal"},
					},
				},
			},
			"submission_type": map[string]interface{}{"id": float64(1), "name": "Talk (long)"},
			"track":           map[string]interface{}{"id": float64(9), "name": "Panel"},
			"speakers": []interface{}{
				map[string]interface{}{"code": "AAA", "avatar_url": "https://example.com/a.webp"},
			},
		},
	}

	normalized := normalizeSubmissions(subs)
	m := normalized[0].(map[string]interface{})

	slot, ok := m["slot"].(map[string]interface{})
	if !ok {
		t.Fatal("expected slots[0] to become slot")
	}
	if _, exists := m["slots"]; exists {
		t.Fatal("expected slots to be removed")
	}
	room, ok := slot["room"].(map[string]interface{})
	if !ok || room["en"] != "Ballsaal" {
		t.Fatalf("expected room to collapse to its name, got %v", slot["room"])
	}
	if m["submission_type"] != "Talk (long)" {
		t.Fatalf("expected submission_type name, got %v", m["submission_type"])
	}
	if m["track"] != "Panel" {
		t.Fatalf("expected track name, got %v", m["track"])
	}
	speaker := m["speakers"].([]interface{})[0].(map[string]interface{})
	if speaker["avatar"] != "https://example.com/a.webp" {
		t.Fatalf("expected avatar_url mapped to avatar, got %v", speaker["avatar"])
	}
}

func TestScheduleBreaksCollapseIntoPseudoRoom(t *testing.T) {
	widget := map[string]interface{}{
		"rooms": []interface{}{
			map[string]interface{}{"id": float64(2), "name": map[string]interface{}{"en": "Barcelona"}},
			map[string]interface{}{"id": float64(3), "name": map[string]interface{}{"en": "Foyer"}},
		},
		"talks": []interface{}{
			map[string]interface{}{
				"code": "TALK", "title": "A talk", "room": float64(2),
				"start": "2026-11-03T10:00:00Z", "end": "2026-11-03T10:30:00Z",
			},
			map[string]interface{}{
				"id": float64(39), "slot_type": "break", "room": float64(3),
				"title": map[string]interface{}{"en": "Lunch (Foyer)"},
				"start": "2026-11-03T13:00:00Z", "end": "2026-11-03T14:30:00Z",
			},
			map[string]interface{}{
				"id": float64(36), "slot_type": "break", "room": float64(2),
				"title": map[string]interface{}{"en": "Lunch (Foyer)"},
				"start": "2026-11-03T13:00:00Z", "end": "2026-11-03T14:30:00Z",
			},
			map[string]interface{}{
				"id": float64(34), "slot_type": "break", "room": float64(2),
				"title": "Registration and Breakfast",
				"start": "2026-11-03T09:00:00Z", "end": "2026-11-03T10:00:00Z",
			},
		},
	}

	got := scheduleBreaks(scheduleBreaksRequest{Widget: widget, BreakRoom: "General info"})
	want := []scheduleBreak{
		{ID: "break-34", Title: "Registration and Breakfast", Room: "General info", Duration: 60, Start: "2026-11-03T09:00:00Z"},
		{ID: "break-36", Title: "Lunch (Foyer)", Room: "General info", Duration: 90, Start: "2026-11-03T13:00:00Z"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scheduleBreaks = %#v, want %#v", got, want)
	}
}

func TestScheduleBreaksKeepPretalxRooms(t *testing.T) {
	widget := map[string]interface{}{
		"rooms": []interface{}{
			map[string]interface{}{"id": float64(2), "name": "Barcelona"},
			map[string]interface{}{"id": float64(3), "name": "Foyer"},
		},
		"talks": []interface{}{
			map[string]interface{}{
				"id": float64(50), "slot_type": "break", "room": float64(3),
				"title": "Breakfast", "start": "2026-11-04T09:00:00Z", "end": "2026-11-04T10:00:00Z",
			},
			map[string]interface{}{
				"id": float64(51), "slot_type": "break", "room": float64(2),
				"title": "Breakfast", "start": "2026-11-04T09:00:00Z", "end": "2026-11-04T10:00:00Z",
			},
		},
	}

	got := scheduleBreaks(scheduleBreaksRequest{Widget: widget, BreakRoom: ""})
	rooms := map[string]bool{}
	for _, item := range got {
		rooms[item.Room] = true
	}
	if len(got) != 2 || !rooms["Barcelona"] || !rooms["Foyer"] {
		t.Fatalf("expected one break per Pretalx room, got %#v", got)
	}
}

func TestSpeakerTagline(t *testing.T) {
	answers := []interface{}{
		map[string]interface{}{
			"answer": "W4 Games",
			"question": map[string]interface{}{
				"question": map[string]interface{}{"en": "Company"},
			},
		},
		map[string]interface{}{
			"answer": map[string]interface{}{"en": "Engine Developer"},
			"question": map[string]interface{}{
				"question": map[string]interface{}{"en": "Title/Tagline", "de": "Titel"},
			},
		},
	}

	if got := speakerTagline(answers, "Title/Tagline"); got != "Engine Developer" {
		t.Fatalf("speakerTagline = %q", got)
	}
	if got := speakerTagline(answers, "Missing"); got != "" {
		t.Fatalf("expected no tagline, got %q", got)
	}
	if got := speakerTagline([]interface{}{float64(99)}, "Title/Tagline"); got != "" {
		t.Fatalf("unexpanded answers should not yield a tagline, got %q", got)
	}
}
