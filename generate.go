package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// charMap maps accented/special characters to ASCII equivalents for slug generation.
var charMap = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a",
	'æ': "ae", 'ç': "c", 'è': "e", 'é': "e", 'ê': "e", 'ë': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ð': "d", 'ñ': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ý': "y", 'ÿ': "y",
	'ß': "ss", 'þ': "th",
	// Eastern European
	'ą': "a", 'ć': "c", 'ę': "e", 'ł': "l", 'ń': "n", 'ś': "s",
	'ź': "z", 'ż': "z", 'č': "c", 'ď': "d", 'ě': "e", 'ň': "n",
	'ř': "r", 'š': "s", 'ť': "t", 'ž': "z", 'ů': "u",
	'ă': "a", 'ș': "s", 'ț': "t",
	// Turkish
	'ğ': "g", 'ı': "i", 'ş': "s",
	// Croatian/Serbian
	'đ': "d",
}

// slugify converts a string into a URL-friendly slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevHyphen := false

	for _, r := range s {
		if rep, ok := charMap[r]; ok {
			b.WriteString(rep)
			prevHyphen = false
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
		} else if r == ' ' || r == '-' || r == '_' || r == '/' {
			if !prevHyphen && b.Len() > 0 {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
		// All other characters are silently dropped
	}

	return strings.TrimRight(b.String(), "-")
}

// getStr extracts a string value from a map, handling both simple strings
// and Pretalx multi-language objects (e.g. {"en": "Talk"}).
func getStr(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case map[string]interface{}:
		// Multi-language: prefer "en", then first available
		if en, ok := val["en"]; ok {
			return fmt.Sprintf("%v", en)
		}
		for _, v := range val {
			return fmt.Sprintf("%v", v)
		}
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// writeDataFile writes a JSON data file to the given path.
// Data files are always overwritten (they represent the current API state).
func writeDataFile(path string, data []interface{}, dryRun bool) error {
	if dryRun {
		fmt.Printf("  [dry-run] Would write %s (%d items)\n", path, len(data))
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	content = append(content, '\n')

	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	fmt.Printf("  Wrote %s (%d items)\n", path, len(data))
	return nil
}

// generateContent creates Hugo content pages for talks, speakers, and section indices.
func generateContent(outputDir string, event EventConfig, talks, speakers []interface{}, dryRun, force bool) error {
	prefix := event.Prefix
	talkLayout := event.TalkLayout
	if talkLayout == "" {
		talkLayout = "pretalx-talk"
	}
	speakerLayout := event.SpeakerLayout
	if speakerLayout == "" {
		speakerLayout = "pretalx-speaker"
	}

	// Generate section index pages
	sections := []struct {
		relPath string
		title   string
		layout  string
	}{
		{filepath.Join("content", prefix, "talks", "_index.md"), "Talks", "pretalx-talks"},
		{filepath.Join("content", prefix, "speakers", "_index.md"), "Speakers", "pretalx-speakers"},
		{filepath.Join("content", prefix, "schedule", "_index.md"), "Schedule", "pretalx-schedule"},
	}

	for _, s := range sections {
		path := filepath.Join(outputDir, s.relPath)
		if err := writeSectionIndex(path, s.title, s.layout, prefix, dryRun, force); err != nil {
			return err
		}
	}

	// Generate individual talk pages (unique slugs: base from title, append -code on collision)
	talkCount := 0
	usedTalkSlugs := make(map[string]bool)
	for _, item := range talks {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		title := getStr(m, "title")
		code := getStr(m, "code")
		if title == "" || code == "" {
			continue
		}
		baseSlug := slugify(title)
		if baseSlug == "" {
			baseSlug = strings.ToLower(code)
		}
		slug := baseSlug
		if usedTalkSlugs[slug] {
			slug = baseSlug + "-" + strings.ToLower(code)
		}
		usedTalkSlugs[slug] = true
		path := filepath.Join(outputDir, "content", prefix, "talks", slug, "index.md")
		tags := append(append([]string{}, event.Tags...), "talk")
		if err := writeContentPage(path, title, talkLayout, code, prefix, tags, dryRun, force); err != nil {
			return err
		}
		talkCount++
	}
	if !dryRun {
		fmt.Printf("  Generated %d talk pages\n", talkCount)
	}

	// Generate individual speaker pages
	speakerCount := 0
	for _, item := range speakers {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name := getStr(m, "name")
		code := getStr(m, "code")
		if name == "" || code == "" {
			continue
		}
		slug := slugify(name)
		if slug == "" {
			slug = strings.ToLower(code)
		}
		path := filepath.Join(outputDir, "content", prefix, "speakers", slug, "index.md")
		tags := append(append([]string{}, event.Tags...), "speaker")
		if err := writeContentPage(path, name, speakerLayout, code, prefix, tags, dryRun, force); err != nil {
			return err
		}
		speakerCount++
	}
	if !dryRun {
		fmt.Printf("  Generated %d speaker pages\n", speakerCount)
	}

	return nil
}

// writeSectionIndex creates or updates a section _index.md file.
func writeSectionIndex(path, title, layout, prefix string, dryRun, force bool) error {
	if !force && fileExists(path) {
		return nil
	}

	if dryRun {
		fmt.Printf("  [dry-run] Would write %s\n", path)
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	var buf strings.Builder
	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("title: %q\n", title))
	if layout != "" {
		buf.WriteString(fmt.Sprintf("layout: %q\n", layout))
	}
	buf.WriteString(fmt.Sprintf("pretalx_prefix: %q\n", prefix))
	buf.WriteString("---\n")

	if err := os.WriteFile(path, []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	fmt.Printf("  Created %s\n", path)
	return nil
}

// writeContentPage creates or updates a content page (talk or speaker).
// When force=true and the file already exists, front matter is regenerated
// but any manually-added body content below the front matter is preserved.
func writeContentPage(path, title, layout, code, prefix string, tags []string, dryRun, force bool) error {
	exists := fileExists(path)

	if exists && !force {
		return nil
	}

	if dryRun {
		action := "Would create"
		if exists {
			action = "Would update"
		}
		fmt.Printf("  [dry-run] %s %s\n", action, path)
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Preserve manually-added body content on update
	existingBody := ""
	if exists {
		existingBody = extractBody(path)
	}

	var buf strings.Builder
	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("title: %q\n", title))
	buf.WriteString(fmt.Sprintf("layout: %q\n", layout))
	buf.WriteString(fmt.Sprintf("pretalx_code: %q\n", code))
	buf.WriteString(fmt.Sprintf("pretalx_prefix: %q\n", prefix))
	if len(tags) > 0 {
		buf.WriteString("tags:\n")
		for _, t := range tags {
			buf.WriteString(fmt.Sprintf("  - %q\n", t))
		}
	}
	buf.WriteString("---\n")

	if existingBody != "" {
		buf.WriteString(existingBody)
	}

	if err := os.WriteFile(path, []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// extractBody reads a Hugo content file and returns everything after the
// closing front matter delimiter ("---"). Returns empty string if no body.
func extractBody(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := string(data)

	// Front matter is delimited by "---" at start and end.
	// Split on "---\n" — the third part (if any) is the body.
	const delim = "---\n"
	if !strings.HasPrefix(content, delim) {
		return ""
	}
	rest := content[len(delim):]
	idx := strings.Index(rest, delim)
	if idx < 0 {
		return ""
	}
	body := rest[idx+len(delim):]
	if strings.TrimSpace(body) == "" {
		return ""
	}
	return body
}

// fileExists checks whether a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
