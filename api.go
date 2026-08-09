package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	Count    int             `json:"count"`
	Next     *string         `json:"next"`
	Previous *string         `json:"previous"`
	Results  []interface{}   `json:"results"`
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

// FetchSpeakers returns speakers for an event, normalized to the legacy shape
// (avatar_url from newer Pretalx versions is exposed as avatar).
func (c *PretalxClient) FetchSpeakers(event string) ([]interface{}, error) {
	speakers, err := c.FetchAll(fetchRequest{Event: event, Endpoint: "speakers"})
	if err != nil {
		return nil, err
	}
	for _, item := range speakers {
		if m, ok := item.(map[string]interface{}); ok {
			normalizeAvatar(m)
		}
	}
	return speakers, nil
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
