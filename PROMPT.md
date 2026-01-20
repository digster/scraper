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
