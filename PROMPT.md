# Prompts Log

## 2026-01-20: Create ARCHITECTURE.md

Create an ARCHITECTURE.md file documenting the codebase architecture as required by the project instructions in CLAUDE.md.

### Summary
Created a comprehensive ARCHITECTURE.md documenting:
- High-level architecture diagram (text-based)
- Directory structure
- Core components (Crawler, Fetcher, State, Events, Storage, Filter)
- Data flow explanation
- GUI-Backend communication via Wails bridge
- Configuration options overview
- Build and run commands
- Testing structure
- Key dependencies
- Design decisions rationale

## 2026-01-20: Implement API Mode

Add an HTTP API mode to the scraper, enabling programmatic control and real-time event streaming. This creates a third entry point alongside CLI and GUI, reusing the existing core crawler package.

### Summary
Implemented a complete HTTP API server with:
- **Core Types** (`internal/api/types.go`, `config.go`): Request/response types, server configuration
- **Job Management** (`jobs.go`, `emitter.go`): Multi-job support with SSEEmitter implementing EventEmitter
- **HTTP Handlers** (`handlers.go`, `sse.go`): RESTful endpoints and Server-Sent Events streaming
- **Server Infrastructure** (`middleware.go`, `routes.go`, `server.go`): Chi router, auth, CORS, logging
- **Entry Point** (`cmd/api/main.go`): CLI flags, signal handling, graceful shutdown
- **Tests** (`api_test.go`): 25+ tests covering handlers, auth, CORS, SSE emitter
- **Documentation**: Updated README.md and ARCHITECTURE.md

### API Endpoints
- `POST /api/v1/crawl` - Start new crawl
- `GET /api/v1/crawl` - List jobs
- `GET /api/v1/crawl/{jobId}` - Get job details
- `DELETE /api/v1/crawl/{jobId}` - Stop/remove job
- `POST /api/v1/crawl/{jobId}/pause` - Pause
- `POST /api/v1/crawl/{jobId}/resume` - Resume
- `GET /api/v1/crawl/{jobId}/events` - SSE stream
- `GET /health` - Health check

## 2026-01-20: Implement MCP Server

Create an MCP (Model Context Protocol) server exposing the web scraper as tools for LLM agents, plus a Claude Code skill for common workflows.

### Summary
Implemented a complete MCP server with:
- **MCP Package** (`internal/mcp/`): Server setup, tool handlers, and types
- **Entry Point** (`cmd/mcp/main.go`): CLI with `--max-jobs` flag, signal handling
- **9 MCP Tools**: scraper_start, scraper_list, scraper_get, scraper_stop, scraper_pause, scraper_resume, scraper_metrics, scraper_confirm_login, scraper_wait
- **Skill Documentation** (`docs/SKILL.md`): Tool descriptions, workflows, best practices
- **Tests** (`internal/mcp/server_test.go`): 15 tests covering tool handlers
- **Documentation**: Updated README.md and ARCHITECTURE.md

### Architecture Decision
Built the MCP server by directly using `JobManager` from `internal/api/` rather than HTTP calls. This reuses existing job lifecycle management, config translation, and event handling without network overhead.

### MCP Tools
| Tool | Description |
|------|-------------|
| `scraper_start` | Start a new crawl job |
| `scraper_list` | List all jobs |
| `scraper_get` | Get job details and metrics |
| `scraper_stop` | Stop a running job |
| `scraper_pause` | Pause a running job |
| `scraper_resume` | Resume a paused job |
| `scraper_metrics` | Get real-time metrics |
| `scraper_confirm_login` | Confirm browser login |
| `scraper_wait` | Wait for job completion |

### Files Created
- `cmd/mcp/main.go`
- `internal/mcp/server.go`
- `internal/mcp/tools.go`
- `internal/mcp/types.go`
- `internal/mcp/server_test.go`
- `docs/SKILL.md`

## 2026-01-20: Update Skill Documentation for All Interfaces

Update `/Users/ishan/lab/scraper/docs/SKILL.md` to document MCP, CLI, and HTTP API interfaces (originally only documented MCP).

### Summary
Expanded SKILL.md from 170 lines to 564 lines with comprehensive documentation:
- **Quick Start** section with examples for all three interfaces
- **MCP Interface**: All 9 tools with parameters, workflows, and examples
- **CLI Interface**: All 33 flags organized by category (core, filtering, display, fetch mode, anti-bot)
- **HTTP API Interface**: Server config, all 10 endpoints, request/response types, SSE events
- **Shared Concepts**: Anti-bot settings, output format, fetch modes, best practices

### Verification
- All 33 CLI flags from `cmd/cli/main.go` documented
- All 10 API endpoints from `internal/api/routes.go` documented
- All 11 SSE event types from `internal/crawler/events.go` documented
- Copy-paste ready examples for curl commands and CLI usage

## 2026-01-20: Implement Click-Based Pagination for Browser Mode

Add a new option to browser mode that enables clicking pagination elements (Next, Load More buttons) that don't have href attributes. The feature mimics human-like clicking behavior using the existing anti-bot infrastructure.

### Summary
Implemented click-based pagination for browser mode with full parity across all interfaces:

**Core Implementation:**
- `PaginationConfig` struct in `config.go` with validation (requires browser mode, selector required)
- New `pagination.go` with `PaginationState`, `ClickPagination()`, and helper functions
- `FetchWithPagination()` method in `browser.go` with callback-based page processing
- Integration in `crawler.go:processURL()` with new `processURLWithPagination()` method

**Features:**
- Human-like clicking via existing `HumanBehavior` helper (scrolling, click offsets, delays)
- Exhaustion detection (element not found, disabled, not visible, max clicks)
- Duplicate content detection via SHA-256 hashing
- Virtual URLs for unique filenames (`?_page=N`)
- Links extracted at same depth level

**Interface Support:**
- GUI: New "Click-Based Pagination" section in ConfigForm.svelte (visible when browser mode selected)
- CLI: 6 new flags (`--enable-pagination`, `--pagination-selector`, etc.)
- API: New `PaginationConfig` in `CrawlRequest`
- MCP: New `PaginationInput` in `StartCrawlInput`

**Files Created:**
- `internal/crawler/pagination.go`
- `internal/crawler/pagination_test.go`

**Files Modified:**
- `internal/crawler/config.go` (PaginationConfig struct + validation)
- `internal/crawler/browser.go` (FetchWithPagination method)
- `internal/crawler/crawler.go` (processURLWithPagination integration)
- `pkg/app/app.go` (CrawlConfig pagination fields)
- `cmd/cli/main.go` (pagination flags)
- `internal/api/types.go` (PaginationConfig type)
- `internal/mcp/types.go` (PaginationInput type)
- `frontend/src/lib/stores/crawler.js` (pagination settings)
- `frontend/src/lib/components/ConfigForm.svelte` (pagination UI section)
- `ARCHITECTURE.md` (pagination documentation)
- `README.md` (pagination usage examples)

## 2026-01-20: Implement Save/Load Site Settings (Presets)

Add the ability to save and load form configuration presets so users can reuse settings for specific sites.

### Summary
Implemented a presets system for saving and loading crawler configurations in the GUI:

**Backend (Go):**
- `PresetConfig` struct with all saveable fields (excludes `outputDir`, `stateFile`)
- `PresetInfo` struct for lightweight preset listing
- 5 new methods: `GetPresetsDir()`, `ListPresets()`, `SavePreset()`, `LoadPreset()`, `DeletePreset()`
- Presets stored as individual JSON files in `~/.config/scraper/presets/`
- Security: Regex validation prevents path traversal attacks in preset names

**Frontend (Svelte):**
- New `presetsStore` for managing preset state and backend calls
- New `PresetSelector.svelte` component with dropdown, Load/Save/Delete buttons
- Modified `configStore` with `getPresetConfig()` and `applyPreset()` helpers
- Modal dialogs for save (name input) and delete (confirmation)

**Files Created:**
- `pkg/app/presets_test.go` (unit tests for preset operations)
- `frontend/src/lib/stores/presets.js` (presets store)
- `frontend/src/lib/components/PresetSelector.svelte` (UI component)

**Files Modified:**
- `pkg/app/app.go` (preset management methods)
- `frontend/src/lib/stores/crawler.js` (helper methods, refactored to use defaultConfig)
- `frontend/src/lib/components/ConfigForm.svelte` (integrated PresetSelector)
- `ARCHITECTURE.md` (documented presets feature)
- `README.md` (documented presets feature)

**Design Decisions:**
- Storage: `~/.config/scraper/presets/` via `os.UserConfigDir()` (cross-platform)
- Format: Individual JSON files (human-readable, easy to backup/share)
- Fields saved: All except `outputDir` and `stateFile` (job-specific)
- Name validation: `^[a-zA-Z0-9][a-zA-Z0-9_-]*$` (security)

## 2026-01-20: Implement URL Normalization for Better Duplicate Detection

Add URL normalization to transform URLs into a canonical form before storing/checking in Visited/Queued maps. This prevents logically identical URLs with different formatting from being treated as different.

### Summary
Implemented URL normalization with full interface support:

**Core Implementation (`internal/crawler/url.go`):**
- `URLNormalizer` struct with configurable `lowercasePaths` option
- `Normalize()` method performs the following normalizations:
  - Lowercase scheme and host
  - Remove default ports (:80 for HTTP, :443 for HTTPS)
  - Sort query parameters alphabetically
  - Uppercase percent encoding (standardize %2f to %2F)
  - Remove empty query parameters
  - Remove trailing slashes (except root `/`)
  - Remove fragments for deduplication
  - Optionally lowercase paths (disabled by default for server compatibility)

**Configuration Options:**
- `NormalizeURLs`: Enable URL normalization (default: true)
- `LowercasePaths`: Lowercase URL paths during normalization (default: false)

**Interface Support:**
- CLI: `--normalize-urls` (default true), `--lowercase-paths` flags
- GUI: "Normalize URLs" checkbox + "Lowercase Paths" in advanced settings
- API: `normalizeUrls` and `lowercasePaths` fields in `CrawlRequest`
- MCP: `normalizeUrls` and `lowercasePaths` parameters in `scraper_start`

**Files Created:**
- `internal/crawler/url.go` (normalization logic)
- `internal/crawler/url_test.go` (23 test cases covering all normalizations)

**Files Modified:**
- `internal/crawler/config.go` (new config fields)
- `internal/crawler/crawler.go` (normalizer integration, normalizeURL method)
- `cmd/cli/main.go` (new flags)
- `pkg/app/app.go` (CrawlConfig and PresetConfig fields)
- `internal/api/types.go` (CrawlRequest fields)
- `internal/api/jobs.go` (translateConfig update)
- `internal/mcp/server.go` (tool parameters)
- `internal/mcp/tools.go` (parameter handling)
- `frontend/src/lib/stores/crawler.js` (default config values)
- `frontend/src/lib/components/ConfigForm.svelte` (UI toggles)

**Example Normalizations:**
| Input | Normalized |
|-------|------------|
| `?b=2&a=1` | `?a=1&b=2` |
| `/page/` | `/page` |
| `http://example.com:80/` | `http://example.com/` |
| `HTTPS://EXAMPLE.COM/` | `https://example.com/` |
| `?a=&b=2` | `?b=2` |
| `%2f` | `%2F` |

## 2026-01-20: Fix Missing disableReadability Field in Frontend

Add the missing `disableReadability` field to the frontend to achieve full parity with the backend configuration.

### Summary
The `disableReadability` field existed in the backend (`CrawlConfig` and `PresetConfig`) but was completely missing from the frontend, causing presets to not save/load this setting and users having no GUI control.

**Changes Made:**
- `frontend/src/lib/stores/crawler.js`: Added `disableReadability: false` to `defaultConfig`
- `frontend/src/lib/components/ConfigForm.svelte`: Added tooltip and checkbox in the main checkbox group

**Files Modified:**
- `frontend/src/lib/stores/crawler.js` (line 49)
- `frontend/src/lib/components/ConfigForm.svelte` (lines 62, 419-421)

## 2026-01-20: Add Configurable Page Load Wait Time for Browser Mode

Add a new `PageLoadWait` configuration option (duration) that allows users to specify how long to wait after page navigation for dynamic content to load in browser mode.

### Summary
The browser fetcher had a hardcoded 500ms sleep after page navigation. This change makes it configurable so users can adjust it for slow-loading pages or heavy JavaScript sites.

**Core Implementation:**
- Added `PageLoadWait time.Duration` field to `Config` struct
- Added `pageLoadWait time.Duration` field to `BrowserFetcher` struct
- Created `NewBrowserFetcherWithPageLoadWait()` constructor accepting page load wait parameter
- Updated `NewBrowserFetcherWithAntiBot()` to call new constructor with default 500ms
- Replaced hardcoded `500*time.Millisecond` in `browser.go` (lines 182 and 315) with `f.pageLoadWait`

**Interface Support:**
- CLI: `--page-load-wait` flag (default: "500ms")
- GUI: "Page Load Wait" input field (visible only when fetch mode is "browser")
- API: `pageLoadWait` field in `CrawlRequest`
- Presets: `pageLoadWait` saved/loaded with configuration presets

**Files Modified:**
- `internal/crawler/config.go` (PageLoadWait field)
- `internal/crawler/browser.go` (pageLoadWait field, new constructor, configurable sleep)
- `internal/crawler/crawler.go` (use NewBrowserFetcherWithPageLoadWait)
- `cmd/cli/main.go` (--page-load-wait flag)
- `pkg/app/app.go` (CrawlConfig and PresetConfig fields, parsing)
- `frontend/src/lib/stores/crawler.js` (pageLoadWait default)
- `frontend/src/lib/components/ConfigForm.svelte` (tooltip and input field)
- `internal/api/types.go` (pageLoadWait field in CrawlRequest)
- `internal/api/jobs.go` (pageLoadWait parsing in translateConfig)

## 2026-01-20: Fix State File "Read-Only File System" Error

Fix the state file location issue where it was being written to the current working directory instead of a guaranteed-writable location. When running as a packaged Wails app on macOS, the CWD can be read-only (inside the `.app` bundle), causing "read-only file system" errors.

### Summary
The `SetDefaultStateFile` function was setting the state file to just a filename without a directory path, causing it to be written to CWD. Fixed by placing the state file inside the output directory which is always writable.

**Root Cause:**
```go
// Before: just filename, written to CWD
config.StateFile = folderName + "_state.json"
```

**Fix:**
```go
// After: inside output directory
config.StateFile = filepath.Join(config.OutputDir, folderName+"_state.json")
```

**Files Modified:**
- `internal/crawler/config.go` (SetDefaultStateFile function)
- `internal/crawler/crawler_test.go` (added TestSetDefaultStateFile)

**Example:**
| Before | After |
|--------|-------|
| `example.com_state.json` (in CWD) | `backup/example.com/example.com_state.json` |

## 2026-01-21: Update MCP Tools, HTTP API, CLI, and Skill File for Recent Features

Ensure all interfaces (MCP, HTTP API, CLI, skill documentation) have parity for recently added features:
- `pageLoadWait` - Configurable wait time after page load (browser mode)
- `disableReadability` - Skip readability content extraction
- `normalizeUrls` / `lowercasePaths` - URL normalization settings
- `pagination` - Click-based pagination (browser mode)

### Summary
Updated MCP tools to support all recently added features that were missing from the MCP interface:

**MCP server.go Changes:**
- Added `pageLoadWait` string parameter
- Added `disableReadability` boolean parameter
- Added `pagination` object parameter
- Added `excludeExtensions` array parameter
- Added `linkSelectors` array parameter

**MCP tools.go Changes:**
- Added parameter handlers for `pageLoadWait`, `disableReadability`, `pagination`, `excludeExtensions`, `linkSelectors`
- Added `parsePaginationConfig()` helper function to parse pagination object
- Added `toStringSlice()` helper function to convert interface arrays to string slices

**MCP types.go Changes:**
- Added `PageLoadWait`, `DisableReadability`, `NormalizeURLs`, `LowercasePaths` fields to `StartCrawlInput`

**docs/SKILL.md Changes:**
- Updated MCP optional parameters table with all new parameters
- Added Pagination Object documentation section
- Added "Crawl with Pagination" workflow example
- Added URL Normalization CLI flags section
- Added Pagination CLI flags section
- Added CLI examples for URL normalization, pagination, and page load wait
- Updated HTTP API JSON example with new fields

### Files Modified
- `internal/mcp/server.go` (new tool parameter definitions)
- `internal/mcp/tools.go` (parameter handlers and helper functions)
- `internal/mcp/types.go` (updated type definitions)
- `docs/SKILL.md` (comprehensive documentation updates)
