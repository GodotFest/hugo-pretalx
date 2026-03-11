package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// loadEnvFromDir reads a .env file from dir and sets each KEY=VALUE
// as an environment variable (only if not already set).
// Skips empty lines and lines starting with #.
// No-op if the file does not exist.
func loadEnvFromDir(dir string) {
	path := filepath.Join(dir, ".env")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		// Remove optional surrounding quotes
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
		// Only set if not already in environment (env wins over .env)
		if os.Getenv(key) == "" && key != "" {
			os.Setenv(key, value)
		}
	}
}
