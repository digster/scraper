# Architecture

A web scraper application built in Go with dual interfaces: CLI for command-line usage and a desktop GUI powered by Wails with a Svelte frontend.

## High-Level Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Entry Points                                 │
├──────────────────────────────┬──────────────────────────────────────┤
│     CLI (cmd/cli/main.go)    │       GUI (main.go + Wails)          │
│     - Flag parsing           │     - Svelte frontend                │
│     - Signal handling        │     - Event-driven updates           │
└──────────────────────────────┴──────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    pkg/app/app.go (GUI Bridge)                       │
│   - CrawlConfig translation   - Event emission via Wails runtime     │
│   - Lifecycle methods         - Metrics/status queries               │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                internal/crawler/crawler.go (Core Orchestrator)       │
│                                                                      │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────────┐  │
│  │   Config    │ │   State     │ │   Metrics   │ │    Events     │  │
│  │ (config.go) │ │ (state.go)  │ │(metrics.go) │ │  (events.go)  │  │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────────┘  │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    Fetcher Interface                         │    │
│  ├──────────────────────────┬──────────────────────────────────┤    │
│  │   HTTPFetcher            │   BrowserFetcher                 │    │
│  │   (http_fetcher.go)      │   (browser.go)                   │    │
│  │   - net/http client      │   - chromedp automation          │    │
│  │   - Lightweight, fast    │   - JS rendering support         │    │
│  └──────────────────────────┴──────────────────────────────────┘    │
│                                                                      │
│  ┌──────────────┐ ┌──────────────┐ ┌─────────────────────────────┐  │
│  │   Storage    │ │   Filter     │ │   Index Generator           │  │
│  │(storage.go)  │ │ (filter.go)  │ │   (index.go)                │  │
│  └──────────────┘ └──────────────┘ └─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │   Output Directory  │
                         │   - *.html files    │
                         │   - *.content.html  │
                         │   - *.meta.json     │
                         │   - _index.html     │
                         └─────────────────────┘
```

## Directory Structure

```
scraper/
├── main.go                    # GUI entry point (Wails)
├── cmd/cli/main.go            # CLI entry point
├── pkg/app/app.go             # Wails app bridge (Go ↔ Frontend)
├── internal/crawler/          # Core crawler package
│   ├── crawler.go             # Main orchestrator (779 lines)
│   ├── config.go              # Configuration structs and validation
│   ├── state.go               # JSON state persistence for resume
│   ├── metrics.go             # Thread-safe progress tracking
│   ├── events.go              # Event emission for GUI/CLI
│   ├── http_fetcher.go        # Standard HTTP client fetcher
│   ├── browser.go             # Chromedp browser automation
│   ├── storage.go             # Content extraction and file saving
│   ├── filter.go              # URL and content-type filtering
│   └── index.go               # Post-crawl HTML report generator
├── frontend/                  # Svelte frontend
│   ├── src/
│   │   ├── App.svelte         # Main container
│   │   └── lib/components/
│   │       ├── ConfigForm.svelte       # Configuration UI
│   │       ├── ProgressDashboard.svelte # Real-time metrics
│   │       ├── LogViewer.svelte        # Log output display
│   │       ├── ControlButtons.svelte   # Start/Pause/Stop controls
│   │       └── LoginModal.svelte       # Manual login flow UI
│   └── wailsjs/               # Auto-generated Wails bindings
└── backup/                    # Default output directory
```

## Core Components

### Crawler (`internal/crawler/crawler.go`)

The central orchestrator managing the crawl lifecycle:

- **Queue Management**: BFS traversal with `URLInfo` structs tracking URL and depth
- **Concurrency**: Optional concurrent mode with semaphore-based limiting (10 max workers)
- **Pause/Resume**: Condition variable-based pause mechanism
- **Login Flow**: For browser mode, supports waiting for manual authentication
- **robots.txt**: Respects or ignores based on configuration

Key methods:
- `Start()` - Initiates crawl, loads/creates state
- `processURL()` - Fetches, validates, saves, extracts links
- `crawlSequential()` / `crawlConcurrent()` - Processing strategies

### Fetcher Interface

Abstraction allowing pluggable page retrieval strategies:

```go
type Fetcher interface {
    Fetch(rawURL string, userAgent string) (*FetchResult, error)
    Close() error
}
```

**HTTPFetcher** (`http_fetcher.go`):
- Standard `net/http` client with connection pooling
- Handles redirects (max 10)
- Best for static content, faster execution

**BrowserFetcher** (`browser.go`):
- Uses chromedp (Chrome DevTools Protocol)
- Full JavaScript rendering
- Anti-bot bypass options (fingerprint spoofing, human behavior simulation)
- Supports headless or visible mode for login flows

### State Management (`state.go`)

Enables resume functionality via JSON persistence:

```go
type CrawlerState struct {
    Queue     []URLInfo          // URLs pending processing
    Visited   map[string]bool    // Already processed URLs
    Queued    map[string]bool    // URLs in queue (prevents duplicates)
    URLDepths map[string]int     // Depth tracking per URL
    Processed int                // Total count for progress
}
```

State is saved every 10 URLs processed (configurable via `StateSaveInterval`).

### Event System (`events.go`)

Decouples crawler from UI via the `EventEmitter` interface:

```go
type EventEmitter interface {
    Emit(event CrawlerEvent)
}
```

Event types: `progress`, `log`, `state_changed`, `crawl_started`, `crawl_completed`, `waiting_for_login`, etc.

The GUI's `App` struct implements this interface, forwarding events via `runtime.EventsEmit()`.

### Storage (`storage.go`)

Handles content extraction and file persistence:

1. **Readability extraction**: Uses `go-readability` to extract main article content
2. **File naming**: URL path → filesystem-safe path with query parameter encoding
3. **Outputs per page**:
   - `{path}.html` - Original HTML
   - `{path}.content.html` - Extracted readable content (optional)
   - `{path}.meta.json` - URL, timestamp, size metadata

### Filter (`filter.go`)

Controls which URLs are processed:

- **Prefix filtering**: Only follow URLs matching a specified prefix
- **Extension exclusion**: Skip assets like `.js`, `.css`, `.png`
- **Content-type filtering**: Skip non-HTML responses
- **Link selectors**: CSS selectors to limit which `<a>` tags are followed

## GUI-Backend Communication

The Wails framework bridges Go and Svelte:

```
┌─────────────────┐         ┌─────────────────┐
│  Svelte Store   │ ──────► │  Wails Binding  │
│  (TypeScript)   │ calls   │  (Generated)    │
└─────────────────┘         └────────┬────────┘
                                     │ invokes
                                     ▼
                            ┌─────────────────┐
                            │  pkg/app/app.go │
                            │  Go Methods     │
                            └────────┬────────┘
                                     │ emits
                                     ▼
                            ┌─────────────────┐
                            │ runtime.Events  │
                            │ Emit(eventName, │
                            │       data)     │
                            └────────┬────────┘
                                     │
                                     ▼
                            ┌─────────────────┐
                            │  Svelte         │
                            │  runtime.Events │
                            │  On(eventName)  │
                            └─────────────────┘
```

**Bound Go methods** (callable from frontend):
- `StartCrawl(config)` - Begin crawling with given config
- `StopCrawl()` / `PauseCrawl()` / `ResumeCrawl()` - Control flow
- `GetStatus()` - Query current state
- `GetMetrics()` - Get real-time statistics
- `ConfirmLogin()` - Signal login completion
- `BrowseDirectory()` / `BrowseFile()` - Native dialogs

## Configuration

The `Config` struct (`config.go`) supports:

| Option | CLI Flag | Description |
|--------|----------|-------------|
| URL | `-url` | Starting URL (required) |
| MaxDepth | `-depth` | Maximum crawl depth (default: 10) |
| Concurrent | `-concurrent` | Enable parallel fetching |
| Delay | `-delay` | Delay between requests (default: 1s) |
| FetchMode | `-fetch-mode` | `http` or `browser` |
| Headless | `-headless` | Run browser headlessly (default: true) |
| WaitForLogin | `-wait-login` | Pause for manual login |
| PrefixFilterURL | `-prefix-filter` | Only follow URLs with this prefix |
| ExcludeExtensions | `-exclude-extensions` | Skip file extensions (e.g., `js,css,png`) |
| IgnoreRobots | `-ignore-robots` | Bypass robots.txt |
| DisableReadability | `-no-readability` | Skip content extraction |

### Anti-Bot Options (browser mode only)

Fingerprint modifications: `HideWebdriver`, `SpoofPlugins`, `SpoofWebGL`, `AddCanvasNoise`

Behavior simulation: `NaturalMouseMovement`, `RandomTypingDelays`, `NaturalScrolling`

## Build and Run

### Prerequisites

- Go 1.24+
- Node.js (for frontend)
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Chrome/Chromium (for browser fetch mode)

### Development

```bash
# Run GUI in development mode (hot reload)
wails dev

# Run CLI directly
go run cmd/cli/main.go -url https://example.com -depth 2
```

### Production Build

```bash
# Build GUI application
wails build

# Build CLI binary
go build -o scraper-cli cmd/cli/main.go
```

### CLI Examples

```bash
# Basic crawl
./scraper-cli -url https://example.com

# Concurrent with browser rendering
./scraper-cli -url https://spa-site.com -concurrent -fetch-mode browser

# Resume from state with prefix filtering
./scraper-cli -url https://docs.example.com -state docs_state.json \
    -prefix-filter https://docs.example.com/api/

# With anti-bot protections
./scraper-cli -url https://protected-site.com -fetch-mode browser \
    -headless=false -wait-login -hide-webdriver -spoof-plugins
```

## Testing

Tests are located alongside source files:

```bash
# Run all tests
go test ./...

# Run crawler tests with verbose output
go test -v ./internal/crawler/...
```

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/wailsapp/wails/v2` | Desktop GUI framework |
| `github.com/chromedp/chromedp` | Browser automation |
| `github.com/PuerkitoBio/goquery` | HTML parsing and link extraction |
| `github.com/go-shiori/go-readability` | Article content extraction |
| `github.com/temoto/robotstxt` | robots.txt parsing |

## Design Decisions

1. **Fetcher Abstraction**: Allows easy addition of new fetching strategies (e.g., Playwright) without changing crawler core.

2. **Event-Driven GUI**: The crawler doesn't know about the GUI; it just emits events. This keeps the core testable and reusable.

3. **State Persistence**: JSON-based state allows manual inspection and recovery. Saving every N URLs balances performance with durability.

4. **Depth-First vs Breadth-First**: Uses BFS (queue-based) to prioritize shallow pages first, which typically captures site structure before diving deep.

5. **Readability Extraction**: Separates raw HTML (for completeness) from extracted content (for usability), letting users choose based on their needs.
