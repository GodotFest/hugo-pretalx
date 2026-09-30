package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds the hugo-pretalx configuration.
type Config struct {
	// Instance is the base URL of the Pretalx instance (e.g. "https://pretalx.example.com").
	Instance string `json:"instance"`

	// Token is the optional API token for authenticated access.
	// Prefer setting this via PRETALX_TOKEN env var instead of the config file.
	Token string `json:"token,omitempty"`

	// Lang is the language code to request from the API (default: "en").
	// This coerces multi-language fields to a single language.
	Lang string `json:"lang,omitempty"`

	// Events is the list of Pretalx events to fetch.
	Events []EventConfig `json:"events"`
}

// EventConfig defines a single Pretalx event to fetch and how to map it into Hugo.
type EventConfig struct {
	// Event is the Pretalx event slug (used in the API URL).
	Event string `json:"event"`

	// Prefix is the data directory key and front-matter pretalx_prefix value.
	// For example, "2025" produces data/pretalx/2025/; the content adapters build
	// pages flat under /talks/ and /speakers/ with tags for year filtering.
	Prefix string `json:"prefix"`

	// Tags are automatically applied to all generated content pages.
	Tags []string `json:"tags,omitempty"`

	// States lists the submission states to include (default: ["confirmed"]).
	// Useful for testing, e.g. ["confirmed", "accepted", "submitted"] to preview
	// the program before talks are confirmed.
	States []string `json:"states,omitempty"`

	// SpeakerLayout overrides the default layout for speaker pages (default: "pretalx-speaker").
	SpeakerLayout string `json:"speaker_layout,omitempty"`

	// TalkLayout overrides the default layout for talk pages (default: "pretalx-talk").
	TalkLayout string `json:"talk_layout,omitempty"`
}

// LoadConfig reads configuration from the given JSON file path,
// then applies overrides from environment variables and CLI flags.
func LoadConfig(path, flagToken string) (*Config, error) {
	cfg := &Config{
		Lang: "en",
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("reading config %s: %w", path, err)
		}
		// File doesn't exist — that's OK if we have env vars
	} else {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config %s: %w", path, err)
		}
	}

	// Override with environment variables (support both PRETALX_TOKEN and PRETALX_API_TOKEN)
	if v := os.Getenv("PRETALX_INSTANCE"); v != "" {
		cfg.Instance = v
	}
	if v := os.Getenv("PRETALX_TOKEN"); v != "" {
		cfg.Token = v
	}
	if cfg.Token == "" {
		if v := os.Getenv("PRETALX_API_TOKEN"); v != "" {
			cfg.Token = v
		}
	}

	// Override with CLI flag
	if flagToken != "" {
		cfg.Token = flagToken
	}

	return cfg, nil
}
