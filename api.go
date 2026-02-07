package main

import (
	"encoding/json"
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

// FetchAll retrieves all results from a paginated Pretalx API endpoint.
// It handles pagination automatically, following "next" links until all results are collected.
func (c *PretalxClient) FetchAll(event, endpoint string) ([]interface{}, error) {
	var all []interface{}

	url := fmt.Sprintf("%s/api/events/%s/%s/", c.BaseURL, event, endpoint)
	if c.Lang != "" {
		url += "?lang=" + c.Lang
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
			return nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, url, truncate(string(body), 200))
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

	return all, nil
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
