# Web Scraper Skill for Claude Code

This skill provides guidance for using the web scraper MCP tools to crawl websites and extract content.

## Available Tools

### scraper_start
Start a new crawl job. Returns immediately with a job ID.

**Required parameters:**
- `url` - Target URL to crawl

**Optional parameters:**
- `maxDepth` - Maximum link depth (default: 10)
- `concurrent` - Enable parallel crawling (faster but more resource-intensive)
- `delay` - Delay between requests (e.g., "500ms", "1s")
- `outputDir` - Where to save crawled content
- `prefixFilter` - Only crawl URLs starting with this prefix
- `fetchMode` - "http" (fast) or "browser" (JavaScript support)
- `headless` - Run browser in headless mode (default: true)
- `waitForLogin` - Wait for manual browser login first
- `userAgent` - Custom User-Agent string
- `ignoreRobots` - Ignore robots.txt restrictions
- `minContent` - Minimum content length to save (default: 100)
- `antiBot` - Anti-bot detection settings (see below)

### scraper_list
List all crawl jobs with their current status.

### scraper_get
Get detailed information about a specific job including real-time metrics.

**Parameters:**
- `jobId` - Job ID from scraper_start

### scraper_stop
Stop a running or paused job.

**Parameters:**
- `jobId` - Job ID to stop

### scraper_pause
Pause a running job (can be resumed later).

**Parameters:**
- `jobId` - Job ID to pause

### scraper_resume
Resume a paused job.

**Parameters:**
- `jobId` - Job ID to resume

### scraper_metrics
Get real-time metrics for a job (URLs processed, saved, errors, etc.).

**Parameters:**
- `jobId` - Job ID to get metrics for

### scraper_confirm_login
Confirm that manual browser login is complete.

**Parameters:**
- `jobId` - Job ID waiting for login confirmation

### scraper_wait
Wait for a job to complete, returning final metrics.

**Parameters:**
- `jobId` - Job ID to wait for
- `timeoutSeconds` - Maximum wait time (default: 300)
- `pollIntervalMs` - Polling interval (default: 2000)

## Common Workflows

### Basic Crawl
```
1. scraper_start with url="https://docs.example.com", maxDepth=3
2. scraper_wait with the returned jobId
3. Content saved to outputDir
```

### Crawl with Anti-Bot Protection
For sites with bot detection, use browser mode with anti-bot settings:
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

### Authenticated Crawl
For sites requiring login:
```
1. scraper_start with url, fetchMode="browser", headless=false, waitForLogin=true
2. User logs in via browser window
3. scraper_confirm_login with jobId
4. Crawler continues with authenticated session
```

### Monitor Progress
```
1. scraper_start with url
2. Periodically call scraper_metrics to check progress
3. scraper_stop if needed, or wait for completion
```

## Anti-Bot Settings

Browser fingerprint modifications:
- `hideWebdriver` - Hide webdriver property
- `spoofPlugins` - Spoof browser plugins
- `spoofLanguages` - Spoof navigator.languages
- `spoofWebGL` - Spoof WebGL renderer/vendor
- `addCanvasNoise` - Add noise to canvas fingerprints

Human behavior simulation:
- `naturalMouseMovement` - Simulate natural mouse movements
- `randomTypingDelays` - Add random delays between keystrokes
- `naturalScrolling` - Simulate natural scroll behavior
- `randomActionDelays` - Add random delays between actions
- `randomClickOffset` - Add randomness to click coordinates

Browser properties:
- `rotateUserAgent` - Rotate user agent strings
- `randomViewport` - Use random viewport sizes
- `matchTimezone` - Match timezone to IP location
- `timezone` - Specific timezone (e.g., "America/New_York")

## Best Practices

1. **Start shallow** - Use maxDepth=2 or 3 first to test
2. **Use prefix filters** - Limit crawling to relevant sections
3. **Add delays** - Be respectful with delay="1s" for external sites
4. **Monitor metrics** - Check scraper_metrics to track progress
5. **Clean up** - Use scraper_stop for jobs you don't need

## Output

Crawled content is saved as markdown files in the output directory, organized by URL path. Each file contains:
- Page title
- Source URL
- Extracted content (cleaned and formatted)

## Installation

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
