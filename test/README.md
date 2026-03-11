# Test harness

## Mock Pretalx API

CI and local tests use a **mock Pretalx API server** so the CLI and site are generated from fixture data without a real Pretalx instance.

- **Fixtures:** `test/fixtures/talks.json` and `test/fixtures/speakers.json` (Pretalx-shaped JSON).
- **Server:** `test/mock-pretalx-server.js` — Node HTTP server on port 9876 that serves those fixtures at `/api/events/:event/talks/` and `/api/events/:event/speakers/` in paginated format. Accepts any `Authorization: Token` (e.g. `test-token`).
- **Config:** `test/site/pretalx.json` points at `http://127.0.0.1:9876` and event `test-event` → prefix `2025`.

## Run the full test locally

From repo root:

```bash
# Terminal 1: start mock API
node test/mock-pretalx-server.js

# Terminal 2:
npm run build
PRETALX_TOKEN=test-token ./hugo-pretalx fetch --output test/site
npm run test:site
```

CI does the same (start mock in background, fetch, then `hugo --minify` in test/site, then verify generated pages).
