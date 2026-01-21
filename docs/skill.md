# Web Scraper Skill for Claude Code

This skill provides guidance for using the web scraper through three interfaces: MCP (Model Context Protocol), CLI, and HTTP API. Choose the interface that best fits your use case.

## Quick Start

### MCP (for Claude Code integration)
```bash
# Build and configure
go build -o scraper-mcp cmd/mcp/main.go
# Add to ~/.claude/mcp.json, then use scraper_* tools
```

### CLI (for terminal usage)
```bash
go build -o scraper ./cmd/cli
./scraper -url "https://example.com" -depth 3 -output ./output
```

### HTTP API (for programmatic access)
```bash
go build -o scraper-api cmd/api/main.go
./scraper-api --port 8080
curl -X POST http://localhost:8080/api/v1/crawl -d '{"url":"https://example.com"}'
```

---

## MCP Interface

The MCP server exposes scraper functionality as tools for LLM agents.

### Installation

1. Build the MCP server:
   ```bash
   go build -o scraper-mcp cmd/mcp/main.go
   ```

2. Add to Claude Code config (`~/.claude/mcp.json`):
   ```json
   {
     "mcpServers": {
       "scraper": {
         "command": "/path/to/scraper-mcp",
         "args": ["--max-jobs", "5"]
       }
     }
   }
   ```

3. Restart Claude Code to load the new server.

### Available Tools

#### scraper_start
Start a new crawl job. Returns immediately with a job ID.

**Required parameters:**
- `url` - Target URL to crawl

**Optional parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `maxDepth` | int | 10 | Maximum link depth to crawl |
| `concurrent` | bool | false | Enable parallel crawling |
| `delay` | string | "1s" | Delay between requests (e.g., "500ms", "1s") |
| `outputDir` | string | auto | Directory to save crawled content |
| `prefixFilter` | string | - | Only crawl URLs starting with this prefix |
| `excludeExtensions` | []string | - | File extensions to exclude (e.g., [".pdf", ".zip"]) |
| `linkSelectors` | []string | - | CSS selectors to find links |
| `fetchMode` | string | "http" | "http" for fast requests, "browser" for JavaScript |
| `headless` | bool | true | Run browser in headless mode |
| `waitForLogin` | bool | false | Wait for manual login before crawling |
| `pageLoadWait` | string | - | Time to wait after page load (browser mode, e.g., "500ms", "2s") |
| `userAgent` | string | - | Custom User-Agent string |
| `ignoreRobots` | bool | false | Ignore robots.txt restrictions |
| `minContent` | int | 100 | Minimum content length to save a page |
| `disableReadability` | bool | false | Disable readability extraction and save raw HTML |
| `normalizeUrls` | bool | true | Enable URL normalization for better duplicate detection |
| `lowercasePaths` | bool | false | Lowercase URL paths during normalization (use with caution) |
| `pagination` | object | - | Click-based pagination settings (see below) |
| `antiBot` | object | - | Anti-bot detection settings (see below) |

#### scraper_list
List all crawl jobs with their current status. No parameters required.

#### scraper_get
Get detailed information about a specific job including real-time metrics.

**Parameters:**
- `jobId` (required) - Job ID from scraper_start

#### scraper_stop
Stop a running or paused job.

**Parameters:**
- `jobId` (required) - Job ID to stop

#### scraper_pause
Pause a running job (can be resumed later).

**Parameters:**
- `jobId` (required) - Job ID to pause

#### scraper_resume
Resume a paused job.

**Parameters:**
- `jobId` (required) - Job ID to resume

#### scraper_metrics
Get real-time metrics for a job (URLs processed, saved, errors, etc.).

**Parameters:**
- `jobId` (required) - Job ID to get metrics for

#### scraper_confirm_login
Confirm that manual browser login is complete.

**Parameters:**
- `jobId` (required) - Job ID waiting for login confirmation

#### scraper_wait
Wait for a job to complete, returning final metrics.

**Parameters:**
- `jobId` (required) - Job ID to wait for
- `timeoutSeconds` (optional) - Maximum wait time (default: 300)
- `pollIntervalMs` (optional) - Polling interval in milliseconds (default: 2000)

### MCP Workflows

#### Basic Crawl
```
1. scraper_start with url="https://docs.example.com", maxDepth=3
2. scraper_wait with the returned jobId
3. Content saved to outputDir
```

#### Crawl with Anti-Bot Protection
```
scraper_start with:
  url: "https://protected-site.com"
  fetchMode: "browser"
  antiBot: {
    hideWebdriver: true,
    spoofPlugins: true,
    naturalMouseMovement: true,
    randomActionDelays: true
  }
```

#### Authenticated Crawl
```
1. scraper_start with url, fetchMode="browser", headless=false, waitForLogin=true
2. User logs in via browser window
3. scraper_confirm_login with jobId
4. Crawler continues with authenticated session
```

#### Monitor Progress
```
1. scraper_start with url
2. Periodically call scraper_metrics to check progress
3. scraper_stop if needed, or wait for completion
```

#### Crawl with Pagination
```
scraper_start with:
  url: "https://blog.example.com"
  fetchMode: "browser"
  pagination: {
    enable: true,
    selector: "a.next-page",
    maxClicks: 20,
    waitAfterClick: "2s",
    stopOnDuplicate: true
  }
```

### Pagination Object

Click-based pagination for sites that load content via "Load More" buttons or pagination links.

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `enable` | bool | false | Enable click-based pagination |
| `selector` | string | - | CSS selector for pagination element (e.g., `a.next`, `.load-more-btn`) |
| `maxClicks` | int | 100 | Maximum pagination clicks per URL |
| `waitAfterClick` | string | - | Time to wait after clicking (e.g., `2s`) |
| `waitSelector` | string | - | CSS selector to wait for after click (optional) |
| `stopOnDuplicate` | bool | true | Stop if duplicate content is detected |

---

## CLI Interface

The CLI provides direct terminal access to the scraper.

### Build

```bash
go build -o scraper ./cmd/cli
```

### Flags Reference

#### Required
| Flag | Description |
|------|-------------|
| `-url` | Starting URL to scrape |

#### Core Settings
| Flag | Default | Description |
|------|---------|-------------|
| `-concurrent` | false | Run in concurrent mode |
| `-delay` | 1s | Delay between fetches |
| `-depth` | 10 | Maximum crawl depth |
| `-output` | auto | Output directory |
| `-state` | auto | State file for resume functionality |

#### Content Filtering
| Flag | Default | Description |
|------|---------|-------------|
| `-prefix-filter` | - | URL prefix to filter by |
| `-exclude-extensions` | - | Comma-separated extensions to exclude (e.g., js,css,png) |
| `-link-selectors` | - | CSS selectors to filter links (e.g., 'a.internal,.nav-link') |
| `-min-content` | 100 | Minimum text content length for a page to be saved |
| `-no-readability` | false | Disable readability content extraction |
| `-ignore-robots` | false | Ignore robots.txt rules |

#### Display Options
| Flag | Default | Description |
|------|---------|-------------|
| `-user-agent` | WebScraper/1.0 | Custom User-Agent header |
| `-verbose` | false | Enable verbose debug output |
| `-progress` | true | Show progress bar and statistics |
| `-metrics-json` | - | Output final metrics to JSON file |

#### Fetch Mode Settings
| Flag | Default | Description |
|------|---------|-------------|
| `-fetch-mode` | http | 'http' for standard HTTP, 'browser' for chromedp |
| `-headless` | true | Run browser in headless mode |
| `-wait-login` | false | Wait for manual login before crawling |
| `-page-load-wait` | 500ms | Time to wait after page load for dynamic content (e.g., '500ms', '2s') |

#### URL Normalization
| Flag | Default | Description |
|------|---------|-------------|
| `-normalize-urls` | true | Enable URL normalization for better duplicate detection |
| `-lowercase-paths` | false | Lowercase URL paths during normalization (use with caution) |

#### Pagination (browser mode only)
| Flag | Default | Description |
|------|---------|-------------|
| `-enable-pagination` | false | Enable click-based pagination |
| `-pagination-selector` | - | CSS selector for pagination element |
| `-max-pagination-clicks` | 100 | Maximum pagination clicks per URL |
| `-pagination-wait` | 2s | Time to wait after clicking pagination |
| `-pagination-wait-selector` | - | CSS selector to wait for after click |
| `-pagination-stop-duplicate` | true | Stop if duplicate content detected |

#### Anti-Bot Bypass (browser mode only)

**Browser Fingerprint:**
| Flag | Description |
|------|-------------|
| `-hide-webdriver` | Hide navigator.webdriver flag |
| `-spoof-plugins` | Inject realistic browser plugins |
| `-spoof-languages` | Set realistic navigator.languages |
| `-spoof-webgl` | Override WebGL vendor/renderer |
| `-canvas-noise` | Add noise to canvas fingerprint |

**Human Behavior Simulation:**
| Flag | Description |
|------|-------------|
| `-natural-mouse` | Use Bezier curve mouse movements |
| `-typing-delays` | Add random typing delays |
| `-natural-scroll` | Use momentum-based scrolling |
| `-action-delays` | Add jittered action delays |
| `-click-offset` | Randomize click positions |

**Browser Properties:**
| Flag | Description |
|------|-------------|
| `-rotate-ua` | Rotate through user agents |
| `-random-viewport` | Use random viewport sizes |
| `-match-timezone` | Enable timezone override |
| `-timezone` | Timezone to use (e.g., America/New_York) |

### CLI Examples

**Basic crawl:**
```bash
./scraper -url "https://docs.example.com" -depth 3
```

**Concurrent crawl with delay:**
```bash
./scraper -url "https://docs.example.com" -concurrent -delay 500ms -output ./docs
```

**Browser mode with JavaScript rendering:**
```bash
./scraper -url "https://spa-app.com" -fetch-mode browser -headless
```

**Authenticated crawl with manual login:**
```bash
./scraper -url "https://private.example.com" -fetch-mode browser -headless=false -wait-login
# Browser opens, complete login, then press ENTER
```

**Anti-bot bypass:**
```bash
./scraper -url "https://protected-site.com" \
  -fetch-mode browser \
  -headless=false \
  -hide-webdriver \
  -spoof-plugins \
  -natural-mouse \
  -action-delays \
  -random-viewport
```

**Resume interrupted crawl:**
```bash
./scraper -url "https://docs.example.com" -state ./crawl-state.json
# State file automatically created on first run
```

**Export metrics to JSON:**
```bash
./scraper -url "https://docs.example.com" -metrics-json ./metrics.json
```

**With URL normalization options:**
```bash
./scraper -url "https://docs.example.com" -normalize-urls -lowercase-paths=false
```

**Click-based pagination (browser mode):**
```bash
./scraper -url "https://blog.example.com" \
  -fetch-mode browser \
  -enable-pagination \
  -pagination-selector "a.next-page" \
  -max-pagination-clicks 20 \
  -pagination-wait 2s
```

**Browser mode with custom page load wait:**
```bash
./scraper -url "https://spa.example.com" \
  -fetch-mode browser \
  -page-load-wait 3s
```

---

## HTTP API Interface

The HTTP API provides RESTful access with Server-Sent Events for real-time updates.

### Build & Run

```bash
go build -o scraper-api cmd/api/main.go
./scraper-api --port 8080
```

### Server Configuration

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--host` | `HOST` | localhost | Host address to bind to |
| `--port` | `PORT` | 8080 | Port to listen on |
| `--max-concurrent` | `MAX_CONCURRENT_JOBS` | 5 | Maximum concurrent crawl jobs |
| `--api-key` | `API_KEY` | - | API key for authentication (optional) |
| `--cors-origins` | `CORS_ORIGINS` | - | Comma-separated allowed CORS origins |
| `--read-timeout` | - | 30 | Read timeout in seconds |
| `--write-timeout` | - | 30 | Write timeout in seconds |
| `--idle-timeout` | - | 120 | Idle timeout in seconds |

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/crawl` | Start a new crawl job |
| GET | `/api/v1/crawl` | List all jobs |
| GET | `/api/v1/crawl/{jobId}` | Get job details |
| DELETE | `/api/v1/crawl/{jobId}` | Stop and delete job |
| POST | `/api/v1/crawl/{jobId}/pause` | Pause a running job |
| POST | `/api/v1/crawl/{jobId}/resume` | Resume a paused job |
| POST | `/api/v1/crawl/{jobId}/confirm-login` | Confirm manual login complete |
| GET | `/api/v1/crawl/{jobId}/metrics` | Get job metrics |
| GET | `/api/v1/crawl/{jobId}/events` | SSE event stream |

### Request/Response Types

#### Create Crawl Request (POST /api/v1/crawl)

```json
{
  "url": "https://example.com",
  "maxDepth": 10,
  "concurrent": false,
  "delay": "1s",
  "outputDir": "./output",
  "stateFile": "./state.json",
  "prefixFilter": "https://example.com/docs",
  "excludeExtensions": [".pdf", ".zip"],
  "linkSelectors": ["a.nav-link", ".content a"],
  "verbose": false,
  "userAgent": "CustomBot/1.0",
  "ignoreRobots": false,
  "minContent": 100,
  "disableReadability": false,
  "normalizeUrls": true,
  "lowercasePaths": false,
  "fetchMode": "http",
  "headless": true,
  "waitForLogin": false,
  "pageLoadWait": "500ms",
  "pagination": {
    "enable": true,
    "selector": "a.next-page",
    "maxClicks": 20,
    "waitAfterClick": "2s",
    "stopOnDuplicate": true
  },
  "antiBot": {
    "hideWebdriver": true,
    "spoofPlugins": true,
    "spoofLanguages": true,
    "spoofWebGL": true,
    "addCanvasNoise": true,
    "naturalMouseMovement": true,
    "randomTypingDelays": true,
    "naturalScrolling": true,
    "randomActionDelays": true,
    "randomClickOffset": true,
    "rotateUserAgent": true,
    "randomViewport": true,
    "matchTimezone": true,
    "timezone": "America/New_York"
  }
}
```

#### Create Crawl Response

```json
{
  "jobId": "abc123",
  "status": "running",
  "createdAt": "2024-01-15T10:30:00Z"
}
```

#### Job Details Response

```json
{
  "jobId": "abc123",
  "url": "https://example.com",
  "status": "running",
  "createdAt": "2024-01-15T10:30:00Z",
  "startedAt": "2024-01-15T10:30:01Z",
  "completedAt": null,
  "config": { ... },
  "metrics": {
    "urlsProcessed": 150,
    "urlsSaved": 120,
    "urlsSkipped": 20,
    "urlsErrored": 10,
    "bytesDownloaded": 5242880,
    "robotsBlocked": 5,
    "depthLimitHits": 15,
    "contentFiltered": 8,
    "pagesPerSecond": 2.5,
    "queueSize": 45,
    "elapsedTime": "1m30s",
    "percentage": 76.9,
    "currentUrl": "https://example.com/page"
  },
  "waitingForLogin": false
}
```

### Job States

| State | Description |
|-------|-------------|
| `pending` | Job created but not yet started |
| `running` | Job is actively crawling |
| `paused` | Job is paused and can be resumed |
| `completed` | Job finished successfully |
| `stopped` | Job was manually stopped |
| `error` | Job encountered a fatal error |
| `waiting_for_login` | Job waiting for manual login confirmation |

### SSE Event Stream

Connect to `/api/v1/crawl/{jobId}/events` to receive real-time updates.

**Event Types:**

| Event | Description | Data |
|-------|-------------|------|
| `connected` | Initial connection established | `{jobId, status}` |
| `progress` | Crawl progress update | `{urlsProcessed, urlsSaved, percentage, ...}` |
| `log` | Log message | `{level, message}` |
| `url_processed` | Individual URL processed | URL details |
| `state_changed` | Job state changed | New state |
| `crawl_started` | Crawl began | - |
| `crawl_paused` | Crawl paused | - |
| `crawl_resumed` | Crawl resumed | - |
| `crawl_stopped` | Crawl stopped | - |
| `crawl_completed` | Crawl finished | - |
| `waiting_for_login` | Waiting for login | `{url}` |
| `error` | Error occurred | `{level, message}` |
| `disconnected` | Stream ending | `{reason}` |
| `: heartbeat` | Keep-alive (comment) | timestamp |

### API Examples

**Start a crawl:**
```bash
curl -X POST http://localhost:8080/api/v1/crawl \
  -H "Content-Type: application/json" \
  -d '{"url": "https://docs.example.com", "maxDepth": 3}'
```

**List all jobs:**
```bash
curl http://localhost:8080/api/v1/crawl
```

**Get job details:**
```bash
curl http://localhost:8080/api/v1/crawl/abc123
```

**Pause a job:**
```bash
curl -X POST http://localhost:8080/api/v1/crawl/abc123/pause
```

**Resume a job:**
```bash
curl -X POST http://localhost:8080/api/v1/crawl/abc123/resume
```

**Get metrics:**
```bash
curl http://localhost:8080/api/v1/crawl/abc123/metrics
```

**Stream events (SSE):**
```bash
curl -N http://localhost:8080/api/v1/crawl/abc123/events
```

**Delete a job:**
```bash
curl -X DELETE http://localhost:8080/api/v1/crawl/abc123
```

**With API key authentication:**
```bash
curl -H "X-API-Key: your-secret-key" http://localhost:8080/api/v1/crawl
```

---

## Shared Concepts

### Anti-Bot Settings

All interfaces support anti-bot detection evasion when using browser mode. These settings modify browser fingerprints and simulate human behavior.

**Browser Fingerprint Modifications:**
| Setting | Description |
|---------|-------------|
| `hideWebdriver` | Hide webdriver property to avoid detection |
| `spoofPlugins` | Spoof browser plugins |
| `spoofLanguages` | Spoof navigator.languages |
| `spoofWebGL` | Spoof WebGL renderer/vendor |
| `addCanvasNoise` | Add noise to canvas fingerprints |

**Human Behavior Simulation:**
| Setting | Description |
|---------|-------------|
| `naturalMouseMovement` | Simulate natural mouse movements with Bezier curves |
| `randomTypingDelays` | Add random delays between keystrokes |
| `naturalScrolling` | Simulate natural momentum-based scrolling |
| `randomActionDelays` | Add random delays between actions |
| `randomClickOffset` | Add slight randomness to click coordinates |

**Browser Properties:**
| Setting | Description |
|---------|-------------|
| `rotateUserAgent` | Rotate user agent strings |
| `randomViewport` | Use random viewport sizes |
| `matchTimezone` | Match timezone to IP location |
| `timezone` | Specific timezone (e.g., "America/New_York") |

### Output Format

Crawled content is saved as markdown files in the output directory, organized by URL path. Each file contains:
- Page title
- Source URL
- Extracted content (cleaned and formatted using readability by default)

Example structure:
```
output/
  example.com/
    index.md
    docs/
      getting-started.md
      api-reference.md
```

### Fetch Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `http` | Standard HTTP client | Fast crawling of static sites |
| `browser` | Real browser via chromedp | JavaScript-rendered SPAs, sites requiring cookies/sessions |

### Best Practices

1. **Start shallow** - Use maxDepth=2 or 3 first to test
2. **Use prefix filters** - Limit crawling to relevant sections
3. **Add delays** - Be respectful with delay="1s" for external sites
4. **Monitor metrics** - Check progress to track crawl health
5. **Clean up** - Stop or delete jobs you don't need
6. **Use state files** - Enable resume functionality for large crawls
7. **Respect robots.txt** - Only use ignoreRobots for sites you own
