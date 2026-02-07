package main

import (
	"flag"
	"fmt"
	"path/filepath"
)

// runFetch implements the "fetch" command.
func runFetch(args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	configPath := fs.String("config", "pretalx.json", "Path to config file")
	token := fs.String("token", "", "API token (overrides config and env)")
	outputDir := fs.String("output", ".", "Hugo site root directory")
	dryRun := fs.Bool("dry-run", false, "Print actions without writing files")
	force := fs.Bool("force", false, "Overwrite existing content files")
	dataOnly := fs.Bool("data-only", false, "Only write data files, skip content pages")
	eventFilter := fs.String("event", "", "Only fetch this event (by slug)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := LoadConfig(*configPath, *token)
	if err != nil {
		return err
	}

	if cfg.Instance == "" {
		return fmt.Errorf("no Pretalx instance URL configured.\n  Set 'instance' in %s or the PRETALX_INSTANCE env var", *configPath)
	}
	if len(cfg.Events) == 0 {
		return fmt.Errorf("no events configured.\n  Add at least one event to the 'events' array in %s", *configPath)
	}

	client := NewClient(cfg.Instance, cfg.Token, cfg.Lang)

	fetched := 0
	for _, event := range cfg.Events {
		if *eventFilter != "" && event.Event != *eventFilter {
			continue
		}

		fmt.Printf("\n=> Event: %s (prefix: %s)\n", event.Event, event.Prefix)

		// Fetch talks
		fmt.Printf("  Fetching talks")
		talks, err := client.FetchAll(event.Event, "talks")
		if err != nil {
			return fmt.Errorf("fetching talks for %s: %w", event.Event, err)
		}
		fmt.Printf(" %d talks\n", len(talks))

		// Fetch speakers
		fmt.Printf("  Fetching speakers")
		speakers, err := client.FetchAll(event.Event, "speakers")
		if err != nil {
			return fmt.Errorf("fetching speakers for %s: %w", event.Event, err)
		}
		fmt.Printf(" %d speakers\n", len(speakers))

		// Write data files (always overwritten — they mirror the API)
		talksPath := filepath.Join(*outputDir, "data", "pretalx", event.Prefix, "talks.json")
		if err := writeDataFile(talksPath, talks, *dryRun); err != nil {
			return err
		}

		speakersPath := filepath.Join(*outputDir, "data", "pretalx", event.Prefix, "speakers.json")
		if err := writeDataFile(speakersPath, speakers, *dryRun); err != nil {
			return err
		}

		// Generate content pages
		if !*dataOnly {
			if err := generateContent(*outputDir, event, talks, speakers, *dryRun, *force); err != nil {
				return err
			}
		}

		fetched++
	}

	if fetched == 0 && *eventFilter != "" {
		return fmt.Errorf("no event matching slug %q found in config", *eventFilter)
	}

	fmt.Printf("\nDone! Fetched %d event(s).\n", fetched)
	return nil
}
