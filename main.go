package main

import (
	"fmt"
	"os"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "fetch":
		if err := runFetch(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Printf("hugo-pretalx v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`hugo-pretalx - Pretalx integration for Hugo

Usage:
  hugo-pretalx <command> [flags]

Commands:
  fetch       Fetch data from Pretalx API and generate Hugo content
  version     Print version information
  help        Print this help message

Fetch flags:
  --config PATH    Path to config file (default: pretalx.json)
  --token TOKEN    API token (overrides config; or set PRETALX_TOKEN env var)
  --output DIR     Hugo site root directory (default: current directory)
  --dry-run        Print what would be done without writing files
  --force          Overwrite existing content files (front matter updated, body preserved)
  --data-only      Only write data files, skip content page generation
  --event SLUG     Only fetch this specific event (by its Pretalx slug)

Environment variables:
  PRETALX_TOKEN      API token for authenticated access
  PRETALX_INSTANCE   Pretalx instance URL (overrides config)

Examples:
  hugo-pretalx fetch
  hugo-pretalx fetch --dry-run
  hugo-pretalx fetch --event my-conference-2025
  hugo-pretalx fetch --config pretalx.json --output ./my-site
  PRETALX_TOKEN=abc123 hugo-pretalx fetch

`)
}
