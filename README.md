# hugo-pretalx

A [Hugo](https://gohugo.io) module that integrates with [Pretalx](https://pretalx.com) to build conference websites. It fetches talk and speaker data from the Pretalx API and provides Hugo layouts, shortcodes, and partials to render schedules, speaker grids, and talk listings.

## Features

- **CLI tool** — fetches talks and speakers from any Pretalx instance and writes raw JSON data files
- **Build-time pages** — [content adapters](https://gohugo.io/content-management/content-adapters/) create talk and speaker pages (and their images) from the data when Hugo builds, so nothing generated needs to be committed
- **Multi-event support** — configure multiple events (e.g. yearly conferences); data is keyed by prefix and pages share flat `/talks` / `/speakers` URLs, distinguished by tags
- **Hugo module** — provides layouts, shortcodes, and partials that read directly from the Pretalx API data
- **Unstyled by design** — semantic HTML with `pretalx-*` CSS classes; you provide the visual identity
- **No runtime dependencies** — pure Go CLI (stdlib only), pure Hugo templates (Hugo 0.141+)

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
- `data/pretalx/2025/event.json` — the event's tags and layouts; marks the prefix for page generation

Add `data/pretalx/` to `.gitignore` and run the fetch before every build.

### 4. Add section pages

Create `content/talks/_index.md`, `content/speakers/_index.md` and `content/schedule/_index.md` with the matching `pretalx-talks`, `pretalx-speakers` and `pretalx-schedule` layouts and a `pretalx_prefix`.

### 5. Build your site

```bash
hugo
```

That's it. The module's content adapters create one page per talk and speaker, and its layouts render everything from the data files.

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
| `--event` | | Only fetch this specific event |

### Install the CLI

```bash
# Run directly (no install needed)
go run github.com/GodotFest/hugo-pretalx@latest fetch

# Or install globally
go install github.com/GodotFest/hugo-pretalx@latest
hugo-pretalx fetch
```

## Using the module from a private repository

*Optional.* Only needed when the hugo-pretalx repo is **private** and you use it from another repo (e.g. same GitHub org).

- **When this applies:** The module at e.g. `github.com/yourorg/hugo-pretalx` is private; Go and Hugo need to fetch it via Git, which requires authentication.
- **Local:** Use SSH or HTTPS with a credential helper so `git clone` of the module works. For HTTPS, set `GOPRIVATE=github.com/yourorg/*` and use a [Personal Access Token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) (or fine-grained token) with read access to the module repo so `hugo mod get` can authenticate.
- **CI (GitHub Actions):** The default `GITHUB_TOKEN` only has access to the repo running the workflow, so it cannot fetch another private repo. Use a PAT (or fine-grained token) with read access to the org/repos that contain the module, store it as a secret (e.g. `MODULE_ACCESS_TOKEN`), then set `GOPRIVATE=github.com/yourorg/*` and configure Git to use the token for GitHub before running `hugo mod get` or `hugo`. See [docs/example-ci-private-module.yml](docs/example-ci-private-module.yml) for a full GitHub Actions example.

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
| `events[].prefix` | Data directory key (`data/pretalx/<prefix>/`) and `pretalx_prefix` front matter value |
| `events[].tags` | Auto-applied tags on all generated content pages (include a year tag such as `"2025"` for filtering) |
| `events[].states` | Submission states to include (default: `["confirmed"]`). For testing you can widen this, e.g. `["confirmed", "accepted", "submitted"]`, to preview the program before talks are confirmed |
| `events[].speaker_layout` | Override layout for speaker pages (default: `pretalx-speaker`) |
| `events[].talk_layout` | Override layout for talk pages (default: `pretalx-talk`) |

### Site parameters (hugo.toml or params.toml)

| Param | Default | Description |
|-------|---------|-------------|
| `pretalxRecordingField` | `"recording"` | Front matter / data key used for the talk recording URL. When set on a talk (in data or page params), the talk single page shows a YouTube embed or "Watch recording" link, and talk cards (including schedule) show a "Recording available" badge. |
| `pretalxSpecialRooms` | *(none)* | Slice of room names that are treated as "special" events (e.g. registration, breaks, lunch). Talks scheduled in these rooms appear in the schedule timeline as special slots (title, room, duration) instead of full talk cards. |
| `pretalxSpeakerLinksInNewTab` | `false` | When `true`, speaker name links (grid cards and `pretalx-speaker-link` in talk cards) use `target="_blank"`. Default is same-tab navigation for on-site speaker pages. External URLs (e.g. recording links on talk pages) are unchanged. |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `PRETALX_TOKEN` | API token for authenticated access |
| `PRETALX_INSTANCE` | Overrides `instance` from config |

## File Structure

```
your-site/
├── data/pretalx/               # Written by `hugo-pretalx fetch` (gitignore it)
│   └── 2026/
│       ├── talks.json          # Raw Pretalx API response
│       ├── speakers.json       # Raw Pretalx API response
│       └── event.json          # Tags and layouts; enables page generation for this prefix
└── content/                    # Only hand-written files
    ├── talks/
    │   ├── _index.md           # Talks list
    │   └── my-talk/            # Optional hand-written override (see below)
    │       └── index.md
    ├── speakers/
    │   └── _index.md
    └── schedule/
        └── _index.md           # Schedule (pretalx_prefix in front matter)
```

**Generated pages:** The module's `content/talks/_content.gotmpl` and `content/speakers/_content.gotmpl` adapters add one page per talk and speaker of every prefix that has an `event.json`. Each page gets `layout`, `pretalx_code`, `pretalx_prefix` and `tags` params, plus the Pretalx talk image or speaker avatar as a `featured.<ext>` page resource (downloaded by Hugo; a failed download logs a warning and the fallback image is used). Pages always match the latest fetch, so withdrawn talks disappear on the next build.

**Slugs:** Page paths are derived from title/name (lowercased, accents transliterated). Talk slugs are shortened to the title before its subtitle (`:`, ` – `, ` — `, ` - `), capped at 5 words, so `Console Development Unlocked: Bringing Your Godot Game to PS5` becomes `console-development-unlocked`. Items are processed in `code` order. A talk slug already taken by a file in `content/talks/` or by an earlier item grows by one word of the title at a time; once the full title is taken, it gets `-1`, `-2`, and so on (speaker slugs go straight to the numbered suffix). Link to talk pages by `pretalx_code` (for example via `pretalx/talk-slugs.html`), not by re-slugifying the title.

**Hand-written overrides:** A page in `content/talks/` or `content/speakers/` whose YAML front matter has the same `pretalx_code` and `pretalx_prefix` replaces the generated page. Use it for extra body content or curated images.

**Archived events:** A prefix whose data files are committed without an `event.json` provides data only. Its pages must be hand-written.

## Layouts

The module provides these layouts (set via `layout` in front matter):

| Layout | Used for | Description |
|--------|----------|-------------|
| `pretalx-talk` | Individual talk pages | Shows title, speaker links (same tab by default; see `pretalxSpeakerLinksInNewTab`), optional recording embed (see `pretalxRecordingField`), abstract, metadata |
| `pretalx-talks` | Talks list page | Lists all talks with cards |
| `pretalx-speaker` | Individual speaker pages | Shows name, bio, avatar, and their talks |
| `pretalx-speakers` | Speakers grid page | Grid of all speaker cards |
| `pretalx-schedule` | Schedule page | Timeline grouped by day/time with room filters; special-room slots (see `pretalxSpecialRooms`); Download ICS button; Favorites toggle when any talk is favorited |

The schedule page includes **ICS export** (button in header), **favorites** (star on each talk card; "Favorites only" filter when at least one talk is favorited), and **day/room filtering**. The talks list supports **infinite scroll** (first batch visible, more load on scroll) and the same **favorites** toggle. Favorites are stored in `localStorage` and namespaced by prefix when `#pretalx-schedule-data` is present.

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

The module outputs semantic HTML with BEM-style CSS classes. **Branding and colors** come from your site CSS. The optional `base.css` file only provides minimal layout defaults (see below).

### Key CSS Classes

```
.pretalx-speakers__grid              — Speaker grid container
.pretalx-speaker-card                — Individual speaker card (row: avatar + info)
.pretalx-speaker-card__avatar        — Speaker avatar wrapper
.pretalx-speaker-card__name         — Speaker name heading (contains link when linked)
.pretalx-speaker-card__name-link     — Link to speaker page (only the name is linked)
.pretalx-speaker-card__bio           — Speaker biography excerpt

.pretalx-talks__list                 — Talks list container
.pretalx-talks__item                 — Wrapper around each card (favorites filter, spacing)
.pretalx-talk-card                   — Talk card (hero background, grid content)
.pretalx-talk-card__fav              — Favorite control wrapper (absolute top-right)
.pretalx-talk-card__bg               — Background image layer; __bg-overlay = gradient
.pretalx-talk-card__content          — Main grid (top / middle / speakers rows)
.pretalx-talk-card__title            — Title (link when `linked` is true)
.pretalx-talk-card__meta             — Meta row: room badge, kind badge, duration, language, recording
.pretalx-talk-card__room-badge       — Pill room label; modifier `--{urlized-room}`
.pretalx-talk-card__kind-badge-inner — Talk / Workshop / Panel / Event pill
.pretalx-talk-card__meta-item        — Row with icon + text (duration, language, recording)
.pretalx-talk-card__summary          — Abstract excerpt (replaces legacy summary)
.pretalx-talk-card__speakers         — Speakers row on a talk card
.pretalx-speaker-link                — Speaker link (e.g. in talk cards; optional new tab via `pretalxSpeakerLinksInNewTab`)
.pretalx-talk__recording             — Recording section on talk single page (embed or link)

.pretalx-schedule               — Schedule container
.pretalx-schedule__tabs         — Day tab navigation
.pretalx-schedule__tab          — Individual day tab
.pretalx-schedule__filters      — Room filter buttons
.pretalx-schedule__slot         — Time slot row
.pretalx-schedule__time         — Time label
.pretalx-schedule__entry        — Talk entry within a slot
```

### Optional Base CSS

The module includes a **minimal structural** stylesheet (flex/grid gaps, default speaker card row layout, talk card borders, schedule tabs). It avoids brand colors on speaker placeholders and does not set site-specific grid column counts — import it if you want a quick starting layout, or skip it and style all `pretalx-*` classes yourself.

You can import it as:

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
Add a hand-written page with the same `pretalx_code` and `pretalx_prefix` (see [File Structure](#file-structure)) and set its `layout` field.

## Data Format

The data files contain the Pretalx API response in the [Pretalx API schema](https://docs.pretalx.org/api/resources/submissions/), reduced to the public fields listed below (see `fields.go`). Private data an authenticated fetch returns — emails, reviews, scores, notes, invitation and access tokens — is never written. A talk's `recording` field is kept for `pretalxRecordingField`.

Fields available in templates:

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
talk.image              — Talk image URL (or null)
talk.resources[]        — Array of {resource, description}
talk.state              — Submission state, e.g. "confirmed"
talk.speakers[]         — Array of {name, code, biography, avatar}

speaker.code            — Unique Pretalx identifier
speaker.name            — Speaker name
speaker.biography       — Biography text
speaker.avatar          — Avatar URL (or null)
speaker.submissions[]   — Array of talk codes
```

## CI/CD Integration

If your site repo and hugo-pretalx are both private (e.g. same org), see [Using the module from a private repository](#using-the-module-from-a-private-repository) for token setup.

```yaml
# GitHub Actions example
steps:
  - uses: actions/checkout@v4

  - uses: actions/setup-go@v5
    with:
      go-version: "1.22"

  - uses: peaceiris/actions-hugo@v3
    with:
      hugo-version: "0.147.8"
      extended: true

  - name: Fetch Pretalx data
    run: go run github.com/GodotFest/hugo-pretalx@latest fetch
    env:
      PRETALX_TOKEN: ${{ secrets.PRETALX_TOKEN }}

  - name: Build site
    run: hugo --minify
```

## Offline / No Token

If `PRETALX_TOKEN` is not set, the CLI fetches publicly available data (confirmed, scheduled talks only). Without a successful fetch there are no data files, so the site builds without talk and speaker pages; make the fetch a required CI step.

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
