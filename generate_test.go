package main

import (
	"os"
	"path/filepath"
	"testing"
)

// page is a content page to lay out in a temporary site for pruning tests.
type page struct {
	Slug   string
	Code   string
	Prefix string
	Body   string
}

// writePages creates a temporary Hugo site root containing the given talk pages.
func writePages(t *testing.T, pages []page) string {
	t.Helper()

	root := t.TempDir()
	for _, p := range pages {
		dir := filepath.Join(root, "content", "talks", p.Slug)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
		content := "---\ntitle: \"" + p.Slug + "\"\n"
		if p.Code != "" {
			content += "pretalx_code: \"" + p.Code + "\"\npretalx_prefix: \"" + p.Prefix + "\"\n"
		}
		content += "---\n" + p.Body
		if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(content), 0644); err != nil {
			t.Fatalf("writing page %s: %v", p.Slug, err)
		}
	}
	return root
}

func pageExists(root, slug string) bool {
	return fileExists(filepath.Join(root, "content", "talks", slug, "index.md"))
}

func TestPruneSectionRemovesOnlyStaleGeneratedPages(t *testing.T) {
	root := writePages(t, []page{
		{Slug: "current", Code: "AAA", Prefix: "2026"},
		{Slug: "withdrawn", Code: "BBB", Prefix: "2026"},
		{Slug: "other-year", Code: "CCC", Prefix: "2025"},
		{Slug: "hand-written"},
		{Slug: "has-manual-body", Code: "DDD", Prefix: "2026", Body: "\nA manually added video embed.\n"},
	})

	err := pruneSection(pruneRequest{
		OutputDir: root,
		Section:   "talks",
		Prefix:    "2026",
		Codes:     map[string]bool{"AAA": true},
	})
	if err != nil {
		t.Fatalf("pruneSection: %v", err)
	}

	if pageExists(root, "withdrawn") {
		t.Error("expected stale page of the fetched prefix to be removed")
	}
	for _, slug := range []string{"current", "other-year", "hand-written", "has-manual-body"} {
		if !pageExists(root, slug) {
			t.Errorf("expected %s to be kept", slug)
		}
	}
}

func TestPruneSectionRemovesWholeBundle(t *testing.T) {
	root := writePages(t, []page{{Slug: "withdrawn", Code: "BBB", Prefix: "2026"}})
	bundle := filepath.Join(root, "content", "talks", "withdrawn")
	if err := os.WriteFile(filepath.Join(bundle, "featured.webp"), []byte("x"), 0644); err != nil {
		t.Fatalf("writing image: %v", err)
	}

	if err := pruneSection(pruneRequest{OutputDir: root, Section: "talks", Prefix: "2026"}); err != nil {
		t.Fatalf("pruneSection: %v", err)
	}

	if _, err := os.Stat(bundle); !os.IsNotExist(err) {
		t.Error("expected the whole page bundle including mirrored images to be removed")
	}
}

func TestPruneSectionDryRunKeepsEverything(t *testing.T) {
	root := writePages(t, []page{{Slug: "withdrawn", Code: "BBB", Prefix: "2026"}})

	err := pruneSection(pruneRequest{OutputDir: root, Section: "talks", Prefix: "2026", DryRun: true})
	if err != nil {
		t.Fatalf("pruneSection: %v", err)
	}

	if !pageExists(root, "withdrawn") {
		t.Error("expected dry run to leave files untouched")
	}
}

func TestPruneSectionMissingSection(t *testing.T) {
	err := pruneSection(pruneRequest{OutputDir: t.TempDir(), Section: "talks", Prefix: "2026"})
	if err != nil {
		t.Fatalf("expected a missing section directory to be a no-op, got %v", err)
	}
}

func TestGenerateContentPrunesAfterWriting(t *testing.T) {
	root := writePages(t, []page{
		{Slug: "withdrawn", Code: "BBB", Prefix: "2026"},
	})

	err := generateContent(generateRequest{
		OutputDir: root,
		Event:     EventConfig{Event: "conf-26", Prefix: "2026", Tags: []string{"2026"}},
		Talks:     []interface{}{map[string]interface{}{"title": "Kept Talk", "code": "AAA"}},
		Prune:     true,
	})
	if err != nil {
		t.Fatalf("generateContent: %v", err)
	}

	if !pageExists(root, "kept-talk") {
		t.Error("expected the fetched talk page to be generated")
	}
	if pageExists(root, "withdrawn") {
		t.Error("expected the stale talk page to be pruned")
	}
}

func TestGenerateContentWithoutPruneKeepsStalePages(t *testing.T) {
	root := writePages(t, []page{{Slug: "withdrawn", Code: "BBB", Prefix: "2026"}})

	err := generateContent(generateRequest{
		OutputDir: root,
		Event:     EventConfig{Event: "conf-26", Prefix: "2026"},
	})
	if err != nil {
		t.Fatalf("generateContent: %v", err)
	}

	if !pageExists(root, "withdrawn") {
		t.Error("expected stale pages to survive without --prune")
	}
}
