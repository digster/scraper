# Architecture

A web scraper application built in Go with four interfaces: CLI for command-line usage, a desktop GUI powered by Wails with a Svelte frontend, an HTTP API for programmatic control, and an MCP server for LLM agent integration.

## High-Level Overview

```
┌───────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                              Entry Points                                                   │
├─────────────────────────┬─────────────────────────────┬────────────────────────────┬──────────────────────┤
│   CLI (cmd/cli/main.go) │   GUI (main.go + Wails)     │   API (cmd/api/main.go)    │ MCP (cmd/mcp/main.go)│
│   - Flag parsing        │   - Svelte frontend         │   - HTTP server            │ - stdio transport    │
│   - Signal handling     │   - Event-driven updates    │   - RESTful endpoints      │ - 9 MCP tools        │
│                         │                             │   - SSE event streaming    │ - LLM integration    │
└─────────────────────────┴─────────────────────────────┴────────────────────────────┴──────────────────────┘
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
├── cmd/
│   ├── cli/main.go            # CLI entry point
│   ├── api/main.go            # API server entry point
│   └── mcp/main.go            # MCP server entry point
├── pkg/app/
│   ├── app.go                 # Wails app bridge (Go ↔ Frontend)
│   └── presets_test.go        # Preset unit tests
├── internal/
│   ├── crawler/               # Core crawler package
│   │   ├── crawler.go         # Main orchestrator
│   │   ├── config.go          # Configuration structs and validation
│   │   ├── state.go           # JSON state persistence for resume
│   │   ├── metrics.go         # Thread-safe progress tracking
│   │   ├── events.go          # Event emission interface
│   │   ├── http_fetcher.go    # Standard HTTP client fetcher
│   │   ├── browser.go         # Chromedp browser automation
│   │   ├── storage.go         # Content extraction and file saving
│   │   ├── filter.go          # URL and content-type filtering
│   │   ├── url.go             # URL normalization for deduplication
│   │   └── index.go           # Post-crawl HTML report generator
│   ├── api/                   # HTTP API package
│   │   ├── server.go          # HTTP server lifecycle
│   │   ├── routes.go          # Chi router configuration
│   │   ├── handlers.go        # REST endpoint handlers
│   │   ├── jobs.go            # Multi-job management
│   │   ├── emitter.go         # SSE event broadcaster
│   │   ├── sse.go             # Server-Sent Events streaming
│   │   ├── middleware.go      # Auth, CORS, logging middleware
│   │   ├── types.go           # Request/response types
│   │   └── config.go          # Server configuration
│   └── mcp/                   # MCP server package
│       ├── server.go          # MCP server setup and tool registration
│       ├── tools.go           # Tool handler implementations
│       ├── types.go           # MCP input/output types
│       └── server_test.go     # Unit tests
├── frontend/                  # Svelte frontend
│   ├── src/
│   │   ├── App.svelte         # Main container
│   │   └── lib/components/
│   │       ├── ConfigForm.svelte       # Configuration UI
│   │       ├── PresetSelector.svelte   # Save/load configuration presets
│   │       ├── ProgressDashboard.svelte # Real-time metrics
│   │       ├── LogViewer.svelte        # Log output display
│   │       ├── ControlButtons.svelte   # Start/Pause/Stop controls
│   │       └── LoginModal.svelte       # Manual login flow UI
│   └── wailsjs/               # Auto-generated Wails bindings
├── docs/
│   └── skill.md               # Claude Code skill documentation
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
- Click-based pagination support via `FetchWithPagination()`

### Pagination (`pagination.go`)

Handles click-based pagination for SPAs and infinite scroll pages:

```go
type PaginationConfig struct {
    Enable          bool          // Enable click-based pagination
    Selector        string        // CSS selector for pagination element
    MaxClicks       int           // Maximum pagination clicks (default: 100)
    WaitAfterClick  time.Duration // Wait time after each click (default: 2s)
    WaitSelector    string        // Optional: wait for element after click
    StopOnDuplicate bool          // Stop if duplicate content detected
}
```

**Features:**
- Human-like clicking behavior (leverages anti-bot infrastructure)
- Automatic exhaustion detection (element not found, disabled, not visible)
- Content hashing for duplicate detection
- Natural scrolling to pagination elements

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

### URL Normalization (`url.go`)

Transforms URLs to canonical form for better duplicate detection:

```go
type URLNormalizer struct {
    lowercasePaths bool // Optionally lowercase URL paths
}
```

**Normalizations performed:**
- Lowercase scheme and host (`HTTPS://EXAMPLE.COM` → `https://example.com`)
- Remove default ports (`:80` for HTTP, `:443` for HTTPS)
- Sort query parameters alphabetically (`?b=2&a=1` → `?a=1&b=2`)
- Uppercase percent encoding (`%2f` → `%2F`)
- Remove empty query parameters (`?a=&b=2` → `?b=2`)
- Remove trailing slashes (`/page/` → `/page`, except root `/`)
- Remove URL fragments (for deduplication)

**Usage:**
- Enabled by default (`Config.NormalizeURLs = true`)
- Path lowercasing disabled by default (some servers are case-sensitive)
- Normalization happens at queue insertion points (initial URL, extracted URLs)

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

1. **Content extraction**: Uses `go-trafilatura` to extract main article content (with go-readability and go-domdistiller as fallbacks)
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
- `ListPresets()` / `SavePreset(name, config)` / `LoadPreset(name)` / `DeletePreset(name)` - Preset management

## API Server (`internal/api/`)

The HTTP API provides programmatic control over crawl jobs:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         HTTP Request                                 │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Middleware Stack (middleware.go)                  │
│   Recovery → Logger → CORS (optional) → APIKey Auth (optional)      │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Chi Router (routes.go)                            │
│   /health, /api/v1/crawl/*, /api/v1/crawl/{jobId}/*                 │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Handlers (handlers.go, sse.go)                    │
│   CreateCrawl, ListCrawls, GetCrawl, PauseCrawl, StreamEvents, etc. │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    JobManager (jobs.go)                              │
│   - Tracks multiple concurrent CrawlJob instances                    │
│   - Translates API config to crawler.Config                          │
│   - Manages job lifecycle (create, start, pause, stop, delete)       │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
               ┌────────────────┴────────────────┐
               ▼                                 ▼
┌─────────────────────────┐         ┌─────────────────────────┐
│   crawler.Crawler       │         │   SSEEmitter            │
│   (internal/crawler)    │ ─────── │   (emitter.go)          │
│   - Core crawl logic    │ events  │   - Fan-out to clients  │
└─────────────────────────┘         └─────────────────────────┘
```

### Key Components

**JobManager (`jobs.go`)**: Manages multiple concurrent crawl jobs. Each job has its own:
- `CrawlJob` struct tracking ID, status, config, metrics
- `crawler.Crawler` instance for the actual work
- `SSEEmitter` for event broadcasting to connected clients

**SSEEmitter (`emitter.go`)**: Implements `crawler.EventEmitter` interface:
- Channel-based fan-out to multiple SSE clients
- Non-blocking sends prevent slow clients from blocking the crawler
- Automatic cleanup on client disconnect

**Handlers (`handlers.go`)**: RESTful endpoint handlers:
- `POST /api/v1/crawl` - Create and start new job
- `GET /api/v1/crawl/{jobId}` - Get job details
- `POST /api/v1/crawl/{jobId}/pause` - Pause running job
- `GET /api/v1/crawl/{jobId}/events` - SSE event stream

### SSE Event Flow

```
Crawler                   SSEEmitter                  HTTP Clients
   │                          │                            │
   │  EmitProgress(...)       │                            │
   ├─────────────────────────►│                            │
   │                          │  broadcast to all          │
   │                          │  subscribed channels       │
   │                          ├───────────────────────────►│
   │                          │                            │
   │  EmitLog(...)            │                            │
   ├─────────────────────────►│                            │
   │                          ├───────────────────────────►│
```

## MCP Server (`internal/mcp/`)

The MCP (Model Context Protocol) server exposes the scraper as tools for LLM agents:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         MCP Client (Claude Code)                      │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ stdio (JSON-RPC)
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    MCP Server (server.go)                            │
│   - Tool registration (9 tools)                                      │
│   - Request routing to handlers                                      │
│   - JSON schema generation                                           │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Tool Handlers (tools.go)                          │
│   handleStart, handleList, handleGet, handleStop, handlePause,       │
│   handleResume, handleMetrics, handleConfirmLogin, handleWait        │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    JobManager (reused from api/jobs.go)              │
│   - Same job lifecycle management as HTTP API                        │
│   - Config translation, concurrent job limits                        │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    crawler.Crawler (internal/crawler)                │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

1. **Direct JobManager Reuse**: The MCP server uses `api.JobManager` directly rather than making HTTP calls. This:
   - Avoids network overhead
   - Reuses all validation and config translation logic
   - Shares the same job lifecycle semantics

2. **stdio Transport**: MCP uses JSON-RPC over stdin/stdout, which is the standard transport for Claude Code integration.

3. **Async Job Model**: `scraper_start` returns immediately with a job ID. Clients poll with `scraper_metrics` or block with `scraper_wait`.

4. **Wait Polling**: The `scraper_wait` tool implements a polling loop (default 2s interval) that checks for terminal states (completed, stopped, error).

### Available Tools

| Tool | Purpose | Maps to |
|------|---------|---------|
| `scraper_start` | Start crawl job | `JobManager.CreateJob` + `StartJob` |
| `scraper_list` | List all jobs | `JobManager.ListJobs` |
| `scraper_get` | Get job details | `JobManager.GetJob` + `ToDetails` |
| `scraper_stop` | Stop job | `JobManager.StopJob` |
| `scraper_pause` | Pause job | `JobManager.PauseJob` |
| `scraper_resume` | Resume job | `JobManager.ResumeJob` |
| `scraper_metrics` | Get metrics | `CrawlJob.GetMetrics` |
| `scraper_confirm_login` | Confirm login | `JobManager.ConfirmLogin` |
| `scraper_wait` | Poll until done | Custom polling loop |

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
| DisableContentExtraction | `-no-extract` | Skip content extraction |

### Pagination Options (browser mode only)

| Option | CLI Flag | Description |
|--------|----------|-------------|
| Enable | `--enable-pagination` | Enable click-based pagination |
| Selector | `--pagination-selector` | CSS selector for pagination element |
| MaxClicks | `--max-pagination-clicks` | Max clicks per URL (default: 100) |
| WaitAfterClick | `--pagination-wait` | Wait after click (default: 2s) |
| WaitSelector | `--pagination-wait-selector` | Wait for element after click |
| StopOnDuplicate | `--pagination-stop-duplicate` | Stop on duplicate content (default: true) |

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

# Run API server directly
go run cmd/api/main.go --port 8080
```

### Production Build

```bash
# Build GUI application
wails build

# Build CLI binary
go build -o scraper-cli cmd/cli/main.go

# Build API server binary
go build -o scraper-api cmd/api/main.go

# Build MCP server binary
go build -o scraper-mcp cmd/mcp/main.go
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
| `github.com/markusmobius/go-trafilatura` | Article content extraction (with readability + domdistiller fallbacks) |
| `github.com/temoto/robotstxt` | robots.txt parsing |
| `github.com/go-chi/chi/v5` | HTTP router for API server |
| `github.com/google/uuid` | Job ID generation |
| `github.com/mark3labs/mcp-go` | MCP server framework |

## Design Decisions

1. **Fetcher Abstraction**: Allows easy addition of new fetching strategies (e.g., Playwright) without changing crawler core.

2. **Event-Driven GUI**: The crawler doesn't know about the GUI; it just emits events. This keeps the core testable and reusable.

3. **State Persistence**: JSON-based state allows manual inspection and recovery. Saving every N URLs balances performance with durability.

4. **Depth-First vs Breadth-First**: Uses BFS (queue-based) to prioritize shallow pages first, which typically captures site structure before diving deep.

5. **Content Extraction**: Separates raw HTML (for completeness) from extracted content (for usability), letting users choose based on their needs. Uses trafilatura for higher extraction accuracy with go-readability and go-domdistiller as fallbacks.

6. **Configuration Presets**: Stored as individual JSON files in `~/.config/scraper/presets/` (cross-platform via `os.UserConfigDir()`). Presets save all settings except `outputDir` and `stateFile` (job-specific paths), making them reusable across different crawl jobs.
