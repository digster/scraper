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
- **Skill Documentation** (`docs/skill.md`): Tool descriptions, workflows, best practices
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
- `docs/skill.md`

## 2026-01-20: Update Skill Documentation for All Interfaces

Update `/Users/ishan/lab/scraper/docs/skill.md` to document MCP, CLI, and HTTP API interfaces (originally only documented MCP).

### Summary
Expanded skill.md from 170 lines to 564 lines with comprehensive documentation:
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
