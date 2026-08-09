package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
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
	prune := fs.Bool("prune", false, "Delete generated pages that are no longer in the fetched set")
	eventFilter := fs.String("event", "", "Only fetch this event (by slug)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Load .env from output dir so PRETALX_API_TOKEN / PRETALX_TOKEN are available (fixture fallback if absent)
	loadEnvFromDir(*outputDir)

	// If config path is the default, look for it inside the output dir (e.g. test/site/pretalx.json)
	resolvedConfig := *configPath
	if *configPath == "pretalx.json" {
		resolvedConfig = filepath.Join(*outputDir, "pretalx.json")
	}

	cfg, err := LoadConfig(resolvedConfig, *token)
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

		// Fetch talks, filtered to the event's allowed states (default: confirmed)
		states := event.States
		if len(states) == 0 {
			states = []string{"confirmed"}
		}
		fmt.Printf("  Fetching talks")
		talks, err := client.FetchTalks(talksRequest{Event: event.Event, States: states})
		if err != nil {
			return fmt.Errorf("fetching talks for %s: %w", event.Event, err)
		}
		fmt.Printf(" %d talks (states: %s)\n", len(talks), strings.Join(states, ", "))

		// Fetch speakers, limited to those on included talks
		fmt.Printf("  Fetching speakers")
		speakers, err := client.FetchSpeakers(event.Event)
		if err != nil {
			return fmt.Errorf("fetching speakers for %s: %w", event.Event, err)
		}
		speakers = filterSpeakersByTalks(speakers, talks)
		fmt.Printf(" %d speakers (on included talks)\n", len(speakers))

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
			if err := generateContent(generateRequest{
				OutputDir: *outputDir,
				Event:     event,
				Talks:     talks,
				Speakers:  speakers,
				DryRun:    *dryRun,
				Force:     *force,
				Prune:     *prune,
			}); err != nil {
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
