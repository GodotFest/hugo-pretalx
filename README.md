# hugo-pretalx

A [Hugo](https://gohugo.io) module that integrates with [Pretalx](https://pretalx.com) to build conference websites. It fetches talk and speaker data from the Pretalx API and provides Hugo layouts, shortcodes, and partials to render schedules, speaker grids, and talk listings.

## Features

- **CLI tool** — fetches talks and speakers from any Pretalx instance, writes raw JSON data files and generates Hugo content pages
- **Multi-event support** — configure multiple events (e.g. yearly conferences) each mapped to a URL prefix
- **Hugo module** — provides layouts, shortcodes, and partials that read directly from the Pretalx API data
- **Unstyled by design** — semantic HTML with `pretalx-*` CSS classes; you provide the visual identity
- **No runtime dependencies** — pure Go CLI (stdlib only), pure Hugo templates
- **Works offline** — data files can be committed to git; the site builds without API access

## Quick Start

### 1. Add the module to your Hugo site

In your site's `hugo.toml`:

```toml
[module]
  [[module.imports]]
    path = "github.com/GodotFest/hugo-pretalx"
```

Then run:

```bash
hugo mod get github.com/GodotFest/hugo-pretalx
```

### 2. Create a config file

Create `pretalx.json` in your site root:

```json
{
  "instance": "https://pretalx.example.com",
  "lang": "en",
  "events": [
    {
      "event": "my-conference-2025",
      "prefix": "2025",
      "tags": ["2025", "myconf"]
    }
  ]
}
```

### 3. Fetch data

```bash
# Set your API token (optional — unauthenticated access works for public events)
export PRETALX_TOKEN=your-token-here

# Install and run the CLI
go run github.com/GodotFest/hugo-pretalx@latest fetch
```

This creates:
- `data/pretalx/2025/talks.json` — raw Pretalx API data
- `data/pretalx/2025/speakers.json` — raw Pretalx API data
- `content/2025/talks/<slug>/index.md` — one page per talk
- `content/2025/speakers/<slug>/index.md` — one page per speaker
- `content/2025/talks/_index.md`, `content/2025/speakers/_index.md`, `content/2025/schedule/_index.md` — section pages

### 4. Build your site

```bash
hugo
```

That's it. The module's layouts render everything from the data files.

## CLI Reference

```
hugo-pretalx fetch [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `pretalx.json` | Path to config file |
| `--token` | | API token (or set `PRETALX_TOKEN` env var) |
| `--output` | `.` | Hugo site root directory |
| `--dry-run` | | Print what would be done without writing |
| `--force` | | Overwrite existing content (preserves manual body content) |
| `--data-only` | | Only write data files, skip content page generation |
| `--event` | | Only fetch this specific event |

### Install the CLI

```bash
# Run directly (no install needed)
go run github.com/GodotFest/hugo-pretalx@latest fetch

# Or install globally
go install github.com/GodotFest/hugo-pretalx@latest
hugo-pretalx fetch
```

## Configuration

### `pretalx.json`

```json
{
  "instance": "https://pretalx.example.com",
  "lang": "en",
  "events": [
    {
      "event": "my-conf-2025",
      "prefix": "2025",
      "tags": ["2025", "myconf"],
      "speaker_layout": "pretalx-speaker",
      "talk_layout": "pretalx-talk"
    },
    {
      "event": "my-conf-2026",
      "prefix": "2026",
      "tags": ["2026", "myconf"]
    }
  ]
}
```

| Field | Description |
|-------|-------------|
| `instance` | Base URL of the Pretalx instance |
| `lang` | Language code for API requests (default: `"en"`) |
| `events[].event` | Pretalx event slug (as in the URL) |
| `events[].prefix` | Local directory prefix for content and data |
| `events[].tags` | Auto-applied tags on all generated content pages |
| `events[].speaker_layout` | Override layout for speaker pages (default: `pretalx-speaker`) |
| `events[].talk_layout` | Override layout for talk pages (default: `pretalx-talk`) |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `PRETALX_TOKEN` | API token for authenticated access |
| `PRETALX_INSTANCE` | Overrides `instance` from config |

## Generated File Structure

```
your-site/
├── data/pretalx/
│   ├── 2025/
│   │   ├── talks.json          # Raw Pretalx API response
│   │   └── speakers.json       # Raw Pretalx API response
│   └── 2026/
│       ├── talks.json
│       └── speakers.json
└── content/
    ├── 2025/
    │   ├── talks/
    │   │   ├── _index.md       # Talks list page
    │   │   └── my-talk/
    │   │       └── index.md    # Individual talk page
    │   ├── speakers/
    │   │   ├── _index.md       # Speakers grid page
    │   │   └── jane-doe/
    │   │       └── index.md    # Individual speaker page
    │   └── schedule/
    │       └── _index.md       # Schedule page
    └── 2026/
        └── ...
```

## Layouts

The module provides these layouts (set via `layout` in front matter):

| Layout | Used for | Description |
|--------|----------|-------------|
| `pretalx-talk` | Individual talk pages | Shows title, speakers, abstract, metadata |
| `pretalx-talks` | Talks list page | Lists all talks with cards |
| `pretalx-speaker` | Individual speaker pages | Shows name, bio, avatar, and their talks |
| `pretalx-speakers` | Speakers grid page | Grid of all speaker cards |
| `pretalx-schedule` | Schedule page | Timeline grouped by day/time with room filters |

All layouts use `{{ define "main" }}` and require a `baseof.html` from your theme.

## Shortcodes

Embed conference components in any markdown page:

```markdown
<!-- Full speakers grid -->
{{</* pretalx-speakers prefix="2025" */>}}

<!-- Full talks list -->
{{</* pretalx-talks prefix="2025" */>}}

<!-- Filtered talks -->
{{</* pretalx-talks prefix="2025" type="Workshop" */>}}
{{</* pretalx-talks prefix="2025" speaker="SPK001" */>}}
{{</* pretalx-talks prefix="2025" track="Development" */>}}

<!-- Single speaker card -->
{{</* pretalx-speaker code="SPK001" prefix="2025" */>}}

<!-- Single talk card -->
{{</* pretalx-talk code="ABC123" prefix="2025" */>}}

<!-- Full schedule -->
{{</* pretalx-schedule prefix="2025" */>}}
```

## Styling

The module outputs semantic HTML with BEM-style CSS classes. No visual styles are applied — you control the look via your site's CSS.

### Key CSS Classes

```
.pretalx-speakers__grid         — Speaker grid container
.pretalx-speaker-card           — Individual speaker card
.pretalx-speaker-card__avatar   — Speaker avatar wrapper
.pretalx-speaker-card__name     — Speaker name
.pretalx-speaker-card__bio      — Speaker biography excerpt

.pretalx-talks__list            — Talks list container
.pretalx-talk-card              — Individual talk card
.pretalx-talk-card__title       — Talk title
.pretalx-talk-card__meta        — Metadata row (type, duration, room)
.pretalx-talk-card__speakers    — Speakers in a talk card

.pretalx-schedule               — Schedule container
.pretalx-schedule__tabs         — Day tab navigation
.pretalx-schedule__tab          — Individual day tab
.pretalx-schedule__filters      — Room filter buttons
.pretalx-schedule__slot         — Time slot row
.pretalx-schedule__time         — Time label
.pretalx-schedule__entry        — Talk entry within a slot
```

### Optional Base CSS

The module includes a structural-only stylesheet you can import:

```css
@import "css/pretalx/base.css";
```

Or via Hugo Pipes in your layout:

```go-html-template
{{ $pretalxCSS := resources.Get "css/pretalx/base.css" }}
{{ if $pretalxCSS }}
  <link rel="stylesheet" href="{{ ($pretalxCSS | minify | fingerprint).RelPermalink }}">
{{ end }}
```

## Overriding Layouts

Hugo's layout lookup order means your site's templates always take priority. Override at any level:

### CSS only (lightest)
Just add styles targeting `pretalx-*` classes in your CSS.

### Override a partial
Create `layouts/partials/pretalx/speaker-card.html` in your site to change how speaker cards render everywhere.

### Override a full layout
Create `layouts/_default/pretalx-talk.html` in your site to completely replace the talk page layout.

### Per-event layouts
Set `speaker_layout` or `talk_layout` in the event config to use different layouts per year:

```json
{
  "event": "my-conf-2025",
  "prefix": "2025",
  "speaker_layout": "my-custom-speaker-2025"
}
```

### Per-page override
Edit a generated `index.md` and change its `layout` field.

## Data Format

The data files contain the raw Pretalx API response — no transformations applied. Templates work directly with the [Pretalx API schema](https://docs.pretalx.org/api/resources/submissions/).

Key fields available in templates:

```
talk.code               — Unique Pretalx identifier
talk.title              — Talk title
talk.abstract           — Abstract text
talk.description        — Longer description
talk.duration           — Duration in minutes
talk.submission_type    — e.g. "Talk", "Workshop"
talk.track              — e.g. "Development"
talk.content_locale     — Language code
talk.do_not_record      — Boolean
talk.slot.start         — ISO 8601 start time
talk.slot.end           — ISO 8601 end time
talk.slot.room          — Room name
talk.speakers[]         — Array of {name, code, biography, avatar}

speaker.code            — Unique Pretalx identifier
speaker.name            — Speaker name
speaker.biography       — Biography text
speaker.avatar          — Avatar URL (or null)
speaker.submissions[]   — Array of talk codes
```

## CI/CD Integration

```yaml
# GitHub Actions example
steps:
  - uses: actions/checkout@v4

  - uses: actions/setup-go@v5
    with:
      go-version: "1.22"

  - uses: peaceiris/actions-hugo@v3
    with:
      hugo-version: "0.139.0"
      extended: true

  - name: Fetch Pretalx data
    run: go run github.com/GodotFest/hugo-pretalx@latest fetch
    env:
      PRETALX_TOKEN: ${{ secrets.PRETALX_TOKEN }}

  - name: Build site
    run: hugo --minify
```

## Offline / No Token

If `PRETALX_TOKEN` is not set, the CLI fetches publicly available data (confirmed, scheduled talks only). If the API is unreachable, the site builds from whatever data files already exist — commit them to git as a fallback.

## Development

```bash
# Build the CLI
make build

# Run the test site (builds Hugo site with fixture data)
make test

# Clean build artifacts
make clean
```

The test site at `test/site/` uses fixture data and local module resolution — no network access needed.

## License

MIT
