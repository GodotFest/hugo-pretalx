package main

// Only these fields are written to data files. An authenticated fetch also returns
// private data (emails, reviews, scores, notes, invitation and access tokens) that
// must never reach the site repository or its build output.
var (
	talkFields = []string{
		"code", "title", "abstract", "description", "duration", "submission_type", "track",
		"content_locale", "do_not_record", "slot", "speakers", "image", "resources", "state",
		"recording",
	}
	talkSpeakerFields = []string{"code", "name", "biography", "avatar"}
	speakerFields     = []string{"code", "name", "biography", "avatar", "submissions"}
)

// pickRequest selects the allowed fields from each map item.
type pickRequest struct {
	Items  []interface{}
	Fields []string
}

func pickFields(req pickRequest) []interface{} {
	picked := make([]interface{}, 0, len(req.Items))
	for _, item := range req.Items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		out := make(map[string]interface{}, len(req.Fields))
		for _, field := range req.Fields {
			if v, exists := m[field]; exists {
				out[field] = v
			}
		}
		picked = append(picked, out)
	}
	return picked
}

// publicTalks strips talks, and the speakers embedded in them, to their public fields.
func publicTalks(talks []interface{}) []interface{} {
	picked := pickFields(pickRequest{Items: talks, Fields: talkFields})
	for _, item := range picked {
		m := item.(map[string]interface{})
		if speakers, ok := m["speakers"].([]interface{}); ok {
			m["speakers"] = pickFields(pickRequest{Items: speakers, Fields: talkSpeakerFields})
		}
	}
	return picked
}

// publicSpeakers strips speakers to their public fields.
func publicSpeakers(speakers []interface{}) []interface{} {
	return pickFields(pickRequest{Items: speakers, Fields: speakerFields})
}
