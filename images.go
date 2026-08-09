package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// processableExts are the formats Hugo's image pipeline can decode. Anything else
// (notably SVG) is left on Pretalx, since a featured.svg in a page bundle would be
// picked up by the templates and then fail to resize.
var processableExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".tif":  true,
	".tiff": true,
	".bmp":  true,
}

// imageDownloader mirrors remote Pretalx images into Hugo page bundles, so the site
// serves resized, converted and fingerprinted copies instead of hot-linking Pretalx
// at full resolution.
type imageDownloader struct {
	client     *http.Client
	dryRun     bool
	downloaded int
	skipped    int
	failed     int
}

// imageRequest is a single remote image to mirror into a page bundle directory.
type imageRequest struct {
	URL string
	Dir string
}

func newImageDownloader(dryRun bool) *imageDownloader {
	return &imageDownloader{
		client: &http.Client{Timeout: 60 * time.Second},
		dryRun: dryRun,
	}
}

// Mirror downloads req.URL into req.Dir as "featured<ext>". Pages that already carry
// an image keep it, so hand-curated artwork is never overwritten.
func (d *imageDownloader) Mirror(req imageRequest) {
	if req.URL == "" {
		return
	}

	ext := imageExt(req.URL)
	if ext == "" {
		d.skipped++
		return
	}

	if bundleHasImage(req.Dir) {
		d.skipped++
		return
	}

	dest := filepath.Join(req.Dir, "featured"+ext)
	if d.dryRun {
		fmt.Printf("  [dry-run] Would download %s -> %s\n", req.URL, dest)
		return
	}

	if err := d.download(req.URL, dest); err != nil {
		fmt.Printf("  Warning: %v\n", err)
		d.failed++
		return
	}
	d.downloaded++
	fmt.Print(".")
}

// download fetches url and writes it to dest, retrying while Pretalx rate-limits us.
func (d *imageDownloader) download(url, dest string) error {
	var body []byte

	for attempt := 0; ; attempt++ {
		resp, err := d.get(url)
		if err != nil {
			return fmt.Errorf("downloading %s: %w", url, err)
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			resp.Body.Close()
			if attempt >= 3 {
				return fmt.Errorf("downloading %s: giving up after HTTP %d", url, resp.StatusCode)
			}
			time.Sleep(retryDelay(resp, attempt))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("downloading %s: HTTP %d", url, resp.StatusCode)
		}

		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("reading %s: %w", url, err)
		}
		break
	}

	if len(body) == 0 {
		return fmt.Errorf("downloading %s: empty response", url)
	}
	if ct := http.DetectContentType(body); !strings.HasPrefix(ct, "image/") {
		return fmt.Errorf("downloading %s: unexpected content type %s", url, ct)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", dest, err)
	}
	if err := os.WriteFile(dest, body, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	return nil
}

func (d *imageDownloader) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "image/*")
	return d.client.Do(req)
}

// retryDelay honours Retry-After when present, otherwise backs off exponentially.
func retryDelay(resp *http.Response, attempt int) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(1<<attempt) * time.Second
}

// Summary reports what the downloader did, for the fetch output.
func (d *imageDownloader) Summary() string {
	summary := fmt.Sprintf("%d downloaded, %d already present", d.downloaded, d.skipped)
	if d.failed > 0 {
		summary += fmt.Sprintf(", %d failed", d.failed)
	}
	return summary
}

// imageExt returns the lowercased extension of a URL's path when Hugo can process
// that format, and "" otherwise.
func imageExt(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	ext := strings.ToLower(path.Ext(parsed.Path))
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	if !processableExts[ext] {
		return ""
	}
	return ext
}

// bundleHasImage reports whether a page bundle already contains an image resource.
func bundleHasImage(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if processableExts[strings.ToLower(filepath.Ext(entry.Name()))] {
			return true
		}
	}
	return false
}
