package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PretalxClient handles communication with the Pretalx REST API.
type PretalxClient struct {
	BaseURL    string
	Token      string
	Lang       string
	HTTPClient *http.Client
}

// NewClient creates a new Pretalx API client.
func NewClient(baseURL, token, lang string) *PretalxClient {
	return &PretalxClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		Lang:    lang,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// pagedResponse represents a single page of Pretalx API results.
type pagedResponse struct {
	Count    int           `json:"count"`
	Next     *string       `json:"next"`
	Previous *string       `json:"previous"`
	Results  []interface{} `json:"results"`
}

// httpStatusError is returned for non-200 API responses so callers can branch on status.
type httpStatusError struct {
	StatusCode int
	URL        string
	Body       string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("HTTP %d from %s: %s", e.StatusCode, e.URL, e.Body)
}

// fetchRequest describes a paginated Pretalx API listing to retrieve.
type fetchRequest struct {
	Event    string
	Endpoint string
	Query    string // extra query params, e.g. "expand=speakers"
}

// FetchAll retrieves all results from a paginated Pretalx API endpoint.
// It handles pagination automatically, following "next" links until all results are collected.
func (c *PretalxClient) FetchAll(req fetchRequest) ([]interface{}, error) {
	var all []interface{}

	url := fmt.Sprintf("%s/api/events/%s/%s/", c.BaseURL, req.Event, req.Endpoint)
	sep := "?"
	if c.Lang != "" {
		url += sep + "lang=" + c.Lang
		sep = "&"
	}
	if req.Query != "" {
		url += sep + req.Query
	}

	for url != "" {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		if c.Token != "" {
			req.Header.Set("Authorization", "Token "+c.Token)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request to %s failed: %w", url, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}

		// Handle rate limiting
		if resp.StatusCode == 429 {
			wait := 5 * time.Second
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := strconv.Atoi(ra); err == nil {
					wait = time.Duration(secs) * time.Second
				}
			}
			fmt.Printf(" [rate limited, waiting %s]", wait)
			time.Sleep(wait)
			continue
		}

		if resp.StatusCode != 200 {
			return nil, &httpStatusError{StatusCode: resp.StatusCode, URL: url, Body: truncate(string(body), 200)}
		}

		var page pagedResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parsing response from %s: %w", url, err)
		}

		all = append(all, page.Results...)

		if page.Next != nil && *page.Next != "" {
			url = *page.Next
		} else {
			url = ""
		}

		fmt.Print(".")
	}

	c.upgradeInsecureURLs(all)

	return all, nil
}

// upgradeInsecureURLs rewrites http:// links pointing at the Pretalx host to https://
// when the configured instance is HTTPS. Pretalx can emit absolute http:// media URLs
// even on an HTTPS instance, and browsers block those as mixed content.
func (c *PretalxClient) upgradeInsecureURLs(items []interface{}) {
	host := strings.TrimPrefix(c.BaseURL, "https://")
	if host == c.BaseURL || host == "" {
		return
	}
	insecure := "http://" + host
	rewriteStrings(items, func(s string) string {
		if strings.HasPrefix(s, insecure) {
			return c.BaseURL + strings.TrimPrefix(s, insecure)
		}
		return s
	})
}

// rewriteStrings applies fn to every string value in a decoded JSON structure.
func rewriteStrings(v interface{}, fn func(string) string) {
	switch t := v.(type) {
	case map[string]interface{}:
		for key, val := range t {
			if s, ok := val.(string); ok {
				t[key] = fn(s)
			} else {
				rewriteStrings(val, fn)
			}
		}
	case []interface{}:
		for i, val := range t {
			if s, ok := val.(string); ok {
				t[i] = fn(s)
			} else {
				rewriteStrings(val, fn)
			}
		}
	}
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// talksRequest describes which event's talks to fetch and which states to keep.
type talksRequest struct {
	Event  string
	States []string // allowed submission states; empty means ["confirmed"]
}

// allowedStates builds the state filter set, defaulting to confirmed-only.
// Only confirmed talks are part of the published program; other states
// (accepted, submitted, withdrawn, rejected, …) must be opted into explicitly.
func allowedStates(states []string) map[string]bool {
	if len(states) == 0 {
		return map[string]bool{"confirmed": true}
	}
	allowed := make(map[string]bool, len(states))
	for _, s := range states {
		allowed[s] = true
	}
	return allowed
}

// FetchTalks returns talks for an event, filtered to the allowed states.
// Older Pretalx versions expose /talks/ (confirmed-only by design); newer versions
// dropped that endpoint, so fall back to /submissions/ with expanded relations,
// normalized to the legacy talk shape the Hugo layouts consume.
func (c *PretalxClient) FetchTalks(req talksRequest) ([]interface{}, error) {
	talks, err := c.FetchAll(fetchRequest{Event: req.Event, Endpoint: "talks"})
	if err != nil {
		var httpErr *httpStatusError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != 404 {
			return nil, err
		}
		subs, err := c.FetchAll(fetchRequest{
			Event:    req.Event,
			Endpoint: "submissions",
			Query:    "expand=speakers,slots,slots.room,submission_type,track",
		})
		if err != nil {
			return nil, err
		}
		talks = normalizeSubmissions(subs)
	}
	return filterByState(talks, allowedStates(req.States)), nil
}

// speakersRequest describes which event's speakers to fetch and which Pretalx
// question, if any, supplies speaker.tagline.
type speakersRequest struct {
	Event           string
	TaglineQuestion string
}

// FetchSpeakers returns speakers for an event, normalized to the legacy shape
// (avatar_url from newer Pretalx versions is exposed as avatar). When
// TaglineQuestion is set, the matching per-speaker answer is copied to tagline
// and the raw answers are dropped.
func (c *PretalxClient) FetchSpeakers(req speakersRequest) ([]interface{}, error) {
	query := ""
	if req.TaglineQuestion != "" {
		query = "questions=all&expand=answers.question"
	}
	speakers, err := c.FetchAll(fetchRequest{Event: req.Event, Endpoint: "speakers", Query: query})
	if err != nil {
		return nil, err
	}
	for _, item := range speakers {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		normalizeAvatar(m)
		if req.TaglineQuestion != "" {
			if tagline := speakerTagline(m["answers"], req.TaglineQuestion); tagline != "" {
				m["tagline"] = tagline
			}
			delete(m, "answers")
		}
	}
	return speakers, nil
}

// speakerTagline returns the answer whose question label matches question.
// Question text and answers may be plain strings or {"en": "..."} maps.
func speakerTagline(answers interface{}, question string) string {
	list, ok := answers.([]interface{})
	if !ok {
		return ""
	}
	want := strings.TrimSpace(question)
	for _, item := range list {
		ans, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		q, ok := ans["question"].(map[string]interface{})
		if !ok || localizedText(q["question"]) != want {
			continue
		}
		if text := localizedText(ans["answer"]); text != "" {
			return text
		}
	}
	return ""
}

// localizedText reads a Pretalx string that may already be coerced to one
// language or still be a language map. English wins when several languages exist.
func localizedText(v interface{}) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case map[string]interface{}:
		if en, ok := t["en"].(string); ok && strings.TrimSpace(en) != "" {
			return strings.TrimSpace(en)
		}
		for _, val := range t {
			if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// filterByState keeps only talks whose state is in the allowed set.
// Talks without a state field are kept (the legacy /talks/ endpoint is confirmed-only).
func filterByState(talks []interface{}, allowed map[string]bool) []interface{} {
	filtered := make([]interface{}, 0, len(talks))
	for _, item := range talks {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if state, ok := m["state"].(string); ok && !allowed[state] {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// normalizeSubmissions maps newer /submissions/ items to the legacy /talks/ shape:
// slots[] becomes slot with a plain room name, expanded submission_type/track objects
// collapse to their name, and speaker avatar_url becomes avatar.
func normalizeSubmissions(subs []interface{}) []interface{} {
	for _, item := range subs {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if slots, ok := m["slots"].([]interface{}); ok {
			if len(slots) > 0 {
				if slot, ok := slots[0].(map[string]interface{}); ok {
					if room, ok := slot["room"].(map[string]interface{}); ok {
						if name, exists := room["name"]; exists {
							slot["room"] = name
						}
					}
					m["slot"] = slot
				}
			}
			delete(m, "slots")
		}

		for _, key := range []string{"submission_type", "track"} {
			if obj, ok := m[key].(map[string]interface{}); ok {
				if name, exists := obj["name"]; exists {
					m[key] = name
				}
			}
		}

		if speakers, ok := m["speakers"].([]interface{}); ok {
			for _, s := range speakers {
				if sm, ok := s.(map[string]interface{}); ok {
					normalizeAvatar(sm)
				}
			}
		}
	}
	return subs
}

// normalizeAvatar maps the newer avatar_url field to the legacy avatar field.
func normalizeAvatar(m map[string]interface{}) {
	if _, hasAvatar := m["avatar"]; hasAvatar {
		return
	}
	if url, ok := m["avatar_url"]; ok {
		m["avatar"] = url
	}
}

// scheduleBreak is one non-talk schedule slot (registration, breakfast, lunch, …).
// The shape matches the specials list Hugo schedule layouts already accept.
type scheduleBreak struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Room     string `json:"room"`
	Duration int    `json:"duration"`
	Start    string `json:"start"`
}

// breaksRequest describes which event's released-schedule breaks to fetch.
type breaksRequest struct {
	Event     string
	BreakRoom string // pseudo room; copies of the same break collapse into it
}

// scheduleBreaksRequest is a decoded Pretalx schedule widget plus the pseudo room.
type scheduleBreaksRequest struct {
	Widget    map[string]interface{}
	BreakRoom string
}

// FetchBreaks reads breaks from the released public schedule widget.
// A 404 means nothing has been released yet, so the result is empty.
// Breaks are not submissions, so the talks API never returns them.
func (c *PretalxClient) FetchBreaks(req breaksRequest) ([]scheduleBreak, error) {
	endpoint := fmt.Sprintf("%s/%s/schedule/widget/v2.json", c.BaseURL, url.PathEscape(req.Event))
	body, status, err := c.fetchBody(endpoint)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		fmt.Print(" no released schedule")
		return []scheduleBreak{}, nil
	}
	if status != http.StatusOK {
		return nil, &httpStatusError{StatusCode: status, URL: endpoint, Body: truncate(string(body), 200)}
	}

	var widget map[string]interface{}
	if err := json.Unmarshal(body, &widget); err != nil {
		return nil, fmt.Errorf("parsing schedule widget: %w", err)
	}
	return scheduleBreaks(scheduleBreaksRequest{Widget: widget, BreakRoom: req.BreakRoom}), nil
}

// fetchBody performs one GET, retrying when Pretalx rate-limits the client.
func (c *PretalxClient) fetchBody(rawURL string) ([]byte, int, error) {
	for {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("creating request: %w", err)
		}
		if c.Token != "" {
			req.Header.Set("Authorization", "Token "+c.Token)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("request to %s failed: %w", rawURL, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, 0, fmt.Errorf("reading response body: %w", err)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			wait := 5 * time.Second
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := strconv.Atoi(ra); err == nil {
					wait = time.Duration(secs) * time.Second
				}
			}
			fmt.Printf(" [rate limited, waiting %s]", wait)
			time.Sleep(wait)
			continue
		}
		return body, resp.StatusCode, nil
	}
}

// scheduleBreaks keeps widget entries whose slot_type is "break".
// When BreakRoom is set, entries that share a title and time become one slot in that room.
func scheduleBreaks(req scheduleBreaksRequest) []scheduleBreak {
	rooms := widgetRoomNames(req.Widget["rooms"])
	breakRoom := strings.TrimSpace(req.BreakRoom)
	talks, _ := req.Widget["talks"].([]interface{})

	items := make([]scheduleBreak, 0)
	seen := map[string]int{}
	seenID := map[string]int{}

	for _, item := range talks {
		m, ok := item.(map[string]interface{})
		if !ok || m["slot_type"] != "break" {
			continue
		}
		title := localizedText(m["title"])
		start, _ := m["start"].(string)
		end, _ := m["end"].(string)
		if title == "" || start == "" || end == "" {
			continue
		}
		startAt, err := time.Parse(time.RFC3339, start)
		if err != nil {
			continue
		}
		endAt, err := time.Parse(time.RFC3339, end)
		if err != nil || !endAt.After(startAt) {
			continue
		}
		idNum, ok := jsonInt(m["id"])
		if !ok {
			continue
		}
		room := breakRoom
		if room == "" {
			roomID, ok := jsonInt(m["room"])
			if !ok {
				continue
			}
			room = rooms[roomID]
			if room == "" {
				continue
			}
		}

		entry := scheduleBreak{
			ID:       fmt.Sprintf("break-%d", idNum),
			Title:    title,
			Room:     room,
			Duration: int(endAt.Sub(startAt).Minutes()),
			Start:    start,
		}
		key := title + "\n" + start + "\n" + end
		if breakRoom == "" {
			key += "\n" + entry.Room
		}
		if prev, exists := seen[key]; exists {
			if idNum < seenID[key] {
				items[prev].ID = entry.ID
				seenID[key] = idNum
			}
			continue
		}
		seen[key] = len(items)
		seenID[key] = idNum
		items = append(items, entry)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Start != items[j].Start {
			return items[i].Start < items[j].Start
		}
		if items[i].Title != items[j].Title {
			return items[i].Title < items[j].Title
		}
		return items[i].ID < items[j].ID
	})
	return items
}

// widgetRoomNames maps a schedule-widget room id to its display name.
func widgetRoomNames(rooms interface{}) map[int]string {
	names := map[int]string{}
	list, ok := rooms.([]interface{})
	if !ok {
		return names
	}
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		id, ok := jsonInt(m["id"])
		if !ok {
			continue
		}
		if name := localizedText(m["name"]); name != "" {
			names[id] = name
		}
	}
	return names
}

// jsonInt reads a JSON number decoded into a float64 or int.
func jsonInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	default:
		return 0, false
	}
}

// filterSpeakersByTalks keeps only speakers that appear on at least one of the
// given (accepted) talks, so no pages are generated for rejected/withdrawn submitters.
func filterSpeakersByTalks(speakers, talks []interface{}) []interface{} {
	codes := make(map[string]bool)
	for _, item := range talks {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		talkSpeakers, ok := m["speakers"].([]interface{})
		if !ok {
			continue
		}
		for _, s := range talkSpeakers {
			if sm, ok := s.(map[string]interface{}); ok {
				if code, ok := sm["code"].(string); ok {
					codes[code] = true
				}
			}
		}
	}

	filtered := make([]interface{}, 0, len(speakers))
	for _, item := range speakers {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if code, ok := m["code"].(string); ok && codes[code] {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
