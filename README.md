# Web Scraper

A Go-based web scraper that creates offline backups of websites by crawling and downloading content. Available as both a CLI tool and a desktop GUI application.

## Initial Prompt
I want to create a go program to create an offline backup of the url provided.
When a url is provided, go through all the links like a crawler and fetch the content from the pages.
Make sure only pages with content are scraped.
The program should take an argument whether to run concurrent or not.
There should be an argument to specify the delay between the fetches.
For the url provided, make sure only the pages with the url having the input url as prefix are fetched, example -  if www.a.com/a is provided, www.a.com/a/c should be parsed but not www.a.com/c
If the task is interrupted, it should resume with the remaining workload, not from the start.
Ask any clarifying questions if needed.

## Features

- **Hierarchical Crawling**: Only crawls URLs discovered from the input URL and its children (tree-based discovery)
- **Depth Control**: Respects maximum crawl depth based on discovery hierarchy
- **Multiple Fetch Modes**: Choose between HTTP client or real browser (Chrome/Chromium) for fetching
- **Browser Mode**: Use a real browser via chromedp to bypass anti-bot protection (headless or visible)
- **Asset Filtering**: Exclude specific file extensions (js, css, images, etc.) from being downloaded
- **Concurrent/Sequential Mode**: Choose between concurrent or sequential crawling
- **Configurable Delays**: Set delays between fetches to be respectful to servers
- **Content Validation**: Only saves pages with meaningful content (>100 characters of text)
- **Readability Extraction**: Automatically extracts main article content using Mozilla's Readability algorithm
- **Resume Functionality**: Automatically resumes from where it left off if interrupted
- **State Persistence**: Saves crawling state to JSON file for resumption
- **Progress Display**: Real-time progress bar with statistics (pages/second, queue size, etc.)
- **Metrics Export**: Optional JSON export of crawl statistics
- **Graceful Shutdown**: Handle SIGINT/SIGTERM signals and save state before exiting
- **Index Page Generation**: Automatically creates a searchable `_index.html` report of all downloaded pages
- **Desktop GUI**: Native desktop application with real-time progress, pause/resume controls, and log viewer

## GUI Features

The desktop GUI provides a user-friendly interface with:

- **Configuration Panel**: All CLI options available as form inputs
- **Real-time Progress Dashboard**: Progress bar, metrics, and current URL display
- **Control Buttons**: Start, Pause/Resume, and Stop controls
- **Live Log Viewer**: Color-coded, scrollable log output
- **Native Dialogs**: File and directory pickers for output and state files

## API Mode

The scraper also provides an HTTP API for programmatic control and integration:

- **RESTful Endpoints**: Create, monitor, pause/resume, and stop crawl jobs
- **Real-time Events**: Server-Sent Events (SSE) for live progress updates
- **Multi-job Support**: Run multiple concurrent crawl jobs
- **Authentication**: Optional API key authentication
- **CORS Support**: Configurable CORS for browser clients

## Installation

### CLI

```bash
go mod tidy
go build -o scraper ./cmd/cli
```

### GUI (Desktop Application)

The GUI requires [Wails](https://wails.io/) to be installed:

```bash
# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Add Go bin to PATH (add to ~/.zshrc for permanence)
export PATH="$HOME/go/bin:$PATH"

# Install frontend dependencies
cd frontend && npm install && cd ..

# Development mode (hot reload)
wails dev

# Build for production
wails build
```

The built application will be in `build/bin/`.

### API Server

```bash
go build -o scraper-api cmd/api/main.go
```

## Usage

### API Server

Start the API server:

```bash
# Basic usage
./scraper-api --port 8080

# With authentication
./scraper-api --port 8080 --api-key "your-secret-key"

# With CORS for browser clients
./scraper-api --port 8080 --cors-origins "http://localhost:3000,http://localhost:5173"

# Full configuration
./scraper-api --host 0.0.0.0 --port 8080 --max-concurrent 10 --api-key "secret"
```

#### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/api/v1/crawl` | Start a new crawl |
| `GET` | `/api/v1/crawl` | List all jobs |
| `GET` | `/api/v1/crawl/{jobId}` | Get job details |
| `DELETE` | `/api/v1/crawl/{jobId}` | Stop and remove job |
| `POST` | `/api/v1/crawl/{jobId}/pause` | Pause crawl |
| `POST` | `/api/v1/crawl/{jobId}/resume` | Resume crawl |
| `POST` | `/api/v1/crawl/{jobId}/confirm-login` | Confirm manual login |
| `GET` | `/api/v1/crawl/{jobId}/metrics` | Get metrics |
| `GET` | `/api/v1/crawl/{jobId}/events` | SSE event stream |

#### API Examples

```bash
# Start a crawl
curl -X POST http://localhost:8080/api/v1/crawl \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "maxDepth": 5,
    "concurrent": true,
    "delay": "500ms"
  }'

# With anti-bot options
curl -X POST http://localhost:8080/api/v1/crawl \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "fetchMode": "browser",
    "headless": false,
    "antiBot": {
      "hideWebdriver": true,
      "spoofPlugins": true
    }
  }'

# Stream events (SSE)
curl -N http://localhost:8080/api/v1/crawl/{jobId}/events

# Get metrics
curl http://localhost:8080/api/v1/crawl/{jobId}/metrics

# Pause/Resume
curl -X POST http://localhost:8080/api/v1/crawl/{jobId}/pause
curl -X POST http://localhost:8080/api/v1/crawl/{jobId}/resume

# Stop and remove job
curl -X DELETE http://localhost:8080/api/v1/crawl/{jobId}
```

#### API Server Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `0.0.0.0` | Host address to bind |
| `--port` | `8080` | Port to listen on |
| `--max-concurrent` | `5` | Maximum concurrent jobs |
| `--api-key` | *(none)* | API key for authentication |
| `--cors-origins` | *(none)* | Allowed CORS origins (comma-separated) |
| `--read-timeout` | `30` | Read timeout (seconds) |
| `--write-timeout` | `60` | Write timeout (seconds) |

Environment variables: `API_HOST`, `API_PORT`, `API_MAX_CONCURRENT_JOBS`, `API_KEY`, `API_CORS_ORIGINS`

### GUI

Simply run the built application or use `wails dev` for development. All options are available in the configuration panel.

### CLI Basic Usage
```bash
./scraper -url https://example.com/docs
```

### Advanced Options
```bash
./scraper -url https://example.com/docs \
          -concurrent \
          -delay 2s \
          -depth 15 \
          -output my_backup \
          -state crawler.json \
          -prefix-filter https://example.com/api \
          -exclude-extensions js,css,png,jpg \
          -link-selectors "a.internal,.nav-link"
```

### Command Line Arguments

- `-url`: Starting URL to scrape (required)
- `-concurrent`: Run in concurrent mode (default: false)
- `-delay`: Delay between fetches (default: 1s)
- `-depth`: Maximum crawl depth based on discovery hierarchy (default: 10)
- `-output`: Output directory for scraped content (default: "scraped_content")
- `-state`: State file for resume functionality (default: "crawler_state.json")
- `-prefix-filter`: URL prefix to filter by (if not specified, no prefix filtering is applied)
- `-exclude-extensions`: Comma-separated list of asset extensions to exclude (e.g., js,css,png)
- `-link-selectors`: Comma-separated list of CSS selectors to filter links (e.g., 'a.internal,.nav-link')
- `-verbose`: Enable verbose debug output (default: false)
- `-user-agent`: Custom User-Agent header for HTTP requests (default: WebScraper/1.0)
- `-ignore-robots`: Ignore robots.txt rules (default: false)
- `-min-content`: Minimum text content length (characters) for a page to be saved (default: 100)
- `-no-readability`: Disable readability content extraction (enabled by default)
- `-progress`: Show progress bar and statistics (default: true)
- `-metrics-json`: Output final metrics to JSON file (optional)
- `-fetch-mode`: Fetch mode - 'http' for standard HTTP client, 'browser' for real Chrome browser (default: http)
- `-headless`: Run browser in headless mode when using browser fetch mode (default: true)
- `-wait-login`: Wait for manual login before crawling; only applies when using browser mode with headless=false (default: false)
- `-hide-webdriver`: Hide navigator.webdriver flag (anti-bot)
- `-spoof-plugins`: Inject realistic browser plugins (anti-bot)
- `-spoof-languages`: Set realistic navigator.languages (anti-bot)
- `-spoof-webgl`: Override WebGL vendor/renderer (anti-bot)
- `-canvas-noise`: Add noise to canvas fingerprint (anti-bot)
- `-natural-mouse`: Use Bezier curve mouse movements (anti-bot)
- `-typing-delays`: Add random typing delays (anti-bot)
- `-natural-scroll`: Use momentum-based scrolling (anti-bot)
- `-action-delays`: Add jittered action delays (anti-bot)
- `-click-offset`: Randomize click positions (anti-bot)
- `-rotate-ua`: Rotate through user agents (anti-bot)
- `-random-viewport`: Use random viewport sizes (anti-bot)
- `-match-timezone`: Enable timezone override (anti-bot)
- `-timezone`: Timezone to use, e.g., America/New_York (anti-bot)

## How It Works

1. **Hierarchical Discovery**: Only processes URLs discovered through the crawling tree starting from the input URL
   - Input: `https://a.com/a`
   - If `b.com` is linked from `a.com/a`, it will be crawled
   - If `c.com` is linked from `b.com`, it will also be crawled  
   - But if `d.com` is linked from `a.com/e` (not discovered through our tree), it won't be crawled
   - Depth is tracked based on discovery steps, not URL structure

2. **URL Filtering Modes**:
   - **Default (No Prefix Filtering)**: Crawls any HTTP/HTTPS URL discovered through the tree, regardless of domain
     - Input: `https://example.com/docs` → Will crawl any domain linked from the discovery tree
   - **With Prefix Filtering**: Use `-prefix-filter <url>` to only crawl URLs matching a specific prefix
     - Example: `-prefix-filter https://example.com/api` → Only crawls URLs starting with `https://example.com/api`
     - Even if other URLs are discovered through the tree, they'll be skipped if they don't match the prefix

3. **Content Filtering**: Pages are only saved if they contain meaningful content (>100 characters of text after removing scripts and styles)

4. **Asset Filtering**: URLs with excluded extensions (specified via `-exclude-extensions`) are skipped

5. **Link Selector Filtering**: Only processes links that match specified CSS selectors
   - **Default**: Processes all links with `href` attributes (`a[href]`)
   - **With `-link-selectors`**: Only processes links matching the specified selectors
   - Examples: `a.internal` (links with class 'internal'), `.nav-link` (any element with class 'nav-link'), `#menu a` (links inside element with id 'menu')

6. **Readability Extraction**: By default, extracts main article content using Mozilla's Readability algorithm
   - Removes navigation, ads, sidebars, and other clutter
   - Preserves article structure (headings, paragraphs, lists)
   - Can be disabled with `-no-readability` flag

7. **File Storage**: Each page is saved as:
   - `{path}.html`: The original HTML content
   - `{path}.content.html`: The extracted readable content (if readability enabled)
   - `{path}.meta.json`: Metadata including original URL, timestamp, size, and readability extraction status
   - Query parameters are included in filenames to avoid collisions (e.g., `/articles?id=1` → `articles_id-1.html`)

8. **Resume Capability**: State is saved periodically and can be resumed by running the same command again

## Output Structure

```
scraped_content/
├── _index.html                   # Generated index page with links to all content
├── index.html                    # Original HTML (root page)
├── index.content.html            # Extracted readable content
├── index.meta.json               # Metadata with readability status
├── articles.html                 # /articles
├── articles.content.html
├── articles.meta.json
├── articles_id-1.html            # /articles?id=1
├── articles_id-1.content.html
├── articles_id-1.meta.json
├── articles_id-2.html            # /articles?id=2
├── articles_id-2.content.html
├── articles_id-2.meta.json
├── blog/
│   ├── posts_page-1.html         # /blog/posts?page=1
│   ├── posts_page-1.content.html
│   └── posts_page-1.meta.json
└── ...
```

### Index Page

After crawling completes, an `_index.html` file is automatically generated in the output directory. This index page provides:

- **Searchable list**: Filter pages by URL or content in real-time
- **Quick navigation**: Links to both raw HTML and extracted content for each page
- **Content preview**: Expandable excerpts from each page's extracted content
- **Metadata**: File sizes and timestamps for each downloaded page
- **Dark/light mode**: Automatically adapts to your system theme

## Examples

### Sequential crawling with 2-second delays
```bash
./scraper -url https://docs.example.com -delay 2s
```

### Concurrent crawling (faster but more resource intensive)
```bash
./scraper -url https://docs.example.com -concurrent -delay 500ms
```

### Resume interrupted crawling
Simply run the same command again - it will automatically resume from the state file.

### Exclude specific asset types
```bash
./scraper -url https://example.com -exclude-extensions js,css,png,jpg,gif
```

### Limit crawl depth
```bash
./scraper -url https://example.com -depth 3
```

### Use prefix filtering to limit to specific URLs
```bash
./scraper -url https://example.com -prefix-filter https://api.example.com
```

### Only follow specific link types
```bash
./scraper -url https://example.com -link-selectors "a.internal,.nav-link,#menu a"
```

### Export metrics to JSON
```bash
./scraper -url https://example.com -metrics-json crawl_metrics.json
```

### Disable readability extraction (save only raw HTML)
```bash
./scraper -url https://example.com -no-readability
```

### Run without progress display
```bash
./scraper -url https://example.com -progress=false
```

### Use browser-based fetching (for anti-bot protected sites)
```bash
# Headless browser mode (default)
./scraper -url https://example.com -fetch-mode browser

# Visible browser window (useful for debugging or CAPTCHA solving)
./scraper -url https://example.com -fetch-mode browser -headless=false
```

## Fetch Modes

The scraper supports two fetching modes:

### HTTP Mode (Default)
Uses Go's standard HTTP client for fetching pages. This is fast and lightweight but may be blocked by sites with anti-bot protection.

```bash
./scraper -url https://example.com -fetch-mode http
```

### Browser Mode
Uses a real Chrome/Chromium browser via chromedp. This renders JavaScript and behaves like a real browser, helping bypass anti-bot measures.

```bash
# Headless (no visible window)
./scraper -url https://example.com -fetch-mode browser

# With visible browser window
./scraper -url https://example.com -fetch-mode browser -headless=false
```

**Requirements for Browser Mode:**
- Chrome or Chromium must be installed on the system
- More resource-intensive than HTTP mode
- Useful when sites block non-browser user agents

### Wait for Login

When crawling sites that require authentication, you can use the "Wait for Login" feature to manually log in before the crawl begins:

1. Select **Browser (Chrome)** as the Fetch Mode
2. Uncheck **Headless** to show the browser window
3. Check **Wait for Login**
4. Click **Start** - the browser will open to your target URL
5. Complete the login process in the browser window
6. Click "Login Complete" in the app (GUI) or press Enter (CLI)
7. Crawling will begin with your authenticated session

**CLI Usage:**
```bash
./scraper -url https://example.com -fetch-mode browser -headless=false -wait-login
```

**GUI Usage:**
When using browser mode with headless disabled, a "Wait for Login" checkbox will appear. Enable it, then start the crawl. A modal dialog will appear prompting you to complete login in the browser window. Click "Login Complete - Start Crawling" when ready.

**Note:** Session cookies are preserved in the browser context, so all subsequent page fetches during the crawl will use your authenticated session.

### Anti-Bot Bypass (Browser Mode Only)

When using browser mode with a visible window (headless=false), additional anti-bot bypass options are available to help evade detection by anti-bot systems.

#### Browser Fingerprint Options
- `--hide-webdriver`: Removes the `navigator.webdriver` flag that identifies automated browsers
- `--spoof-plugins`: Injects realistic browser plugins to match a normal Chrome profile
- `--spoof-languages`: Sets `navigator.languages` to common browser values (en-US, en)
- `--spoof-webgl`: Overrides WebGL vendor/renderer strings to avoid GPU fingerprinting
- `--canvas-noise`: Adds subtle noise to canvas fingerprinting attempts

#### Human Behavior Simulation
- `--natural-mouse`: Moves mouse with natural Bezier curves instead of teleporting
- `--typing-delays`: Types with human-like variable delays between keystrokes
- `--natural-scroll`: Scrolls gradually with momentum simulation (ease-out effect)
- `--action-delays`: Adds random delays (100-500ms) between page interactions
- `--click-offset`: Clicks with small random offset from exact element center

#### Browser Properties
- `--rotate-ua`: Cycles through realistic Chrome user agent strings
- `--random-viewport`: Uses common screen resolutions (1920x1080, 1366x768, etc.) randomly
- `--match-timezone`: Enables browser timezone override
- `--timezone <tz>`: Explicit timezone to use (e.g., `America/New_York`, `Europe/London`)

**CLI Example with anti-bot options:**
```bash
./scraper -url https://example.com \
  -fetch-mode browser \
  -headless=false \
  -hide-webdriver \
  -spoof-plugins \
  -spoof-webgl \
  -natural-mouse \
  -action-delays \
  -rotate-ua
```

**GUI Usage:**
When browser mode is selected with headless disabled, an "Anti-Bot Bypass Options" section appears in the configuration panel. This section is organized into three categories:
- **Browser Fingerprint**: Options to modify browser fingerprint characteristics
- **Human Behavior**: Options to simulate human-like interactions
- **Browser Properties**: Options to randomize browser properties

Each option can be individually enabled or disabled to customize your stealth configuration.

**Sources & References:**
- [ZenRows: Bypass Bot Detection](https://www.zenrows.com/blog/bypass-bot-detection) - Overview of anti-bot bypass methods
- [Puppeteer Stealth Plugin (npm)](https://www.npmjs.com/package/puppeteer-extra-plugin-stealth) - Stealth techniques for browser automation
- [ZenRows: Ghost Cursor](https://www.zenrows.com/blog/ghost-cursor) - Human-like mouse movement patterns
- [ZenRows: WebGL Fingerprinting](https://www.zenrows.com/blog/webgl-fingerprinting) - WebGL fingerprint evasion techniques
- [Intoli: Making Chrome Headless Undetectable](https://intoli.com/blog/making-chrome-headless-undetectable/) - Hiding automation indicators
- [chromedp Documentation](https://pkg.go.dev/github.com/chromedp/chromedp) - Go browser automation library

## Notes

- The scraper uses hierarchical discovery - only URLs found through the crawling tree are processed
- By default, no prefix filtering is applied - any domain discovered through the tree will be crawled
- Use `-prefix-filter <url>` to limit crawling to URLs matching a specific prefix
- Use `-link-selectors` to only follow links matching specific CSS selectors (default: all links with href)
- Depth is measured by discovery steps, not URL path depth
- Only HTML pages with substantial content are saved
- Use `-exclude-extensions` to skip downloading specific asset types (js, css, images, etc.)
- Concurrent mode limits to 10 simultaneous requests to avoid overwhelming servers
- State is saved every 10 processed URLs for resilience
- Press Ctrl+C to gracefully stop crawling - state will be saved automatically for resumption