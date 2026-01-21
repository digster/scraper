package crawler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/temoto/robotstxt"
)

// Crawler configuration constants
const (
	// HTTPTimeout is the timeout for HTTP requests
	HTTPTimeout = 30 * time.Second

	// MaxConcurrentRequests is the maximum number of simultaneous requests in concurrent mode
	MaxConcurrentRequests = 10

	// StateSaveInterval is how often state is saved (every N processed URLs)
	StateSaveInterval = 10

	// QueueEmptyWaitTime is how long to wait when queue is empty but goroutines are active
	QueueEmptyWaitTime = 100 * time.Millisecond

	// MinContentLength is the minimum text length (characters) for a page to be considered having meaningful content
	MinContentLength = 100

	// DefaultUserAgent is the default User-Agent header sent with requests
	DefaultUserAgent = "Mozilla/5.0 (compatible; WebScraper/1.0; +https://github.com/user/scraper)"

	// MaxRedirects is the maximum number of redirects to follow per request
	MaxRedirects = 10
)

// Logger provides leveled logging for the crawler
type Logger struct {
	verbose bool
	emitter EventEmitter
}

// Debug logs a message only if verbose mode is enabled
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbose {
		msg := fmt.Sprintf(format, args...)
		log.Printf("[DEBUG] %s", msg)
		EmitLog(l.emitter, "debug", msg)
	}
}

// Info logs an informational message (always shown)
func (l *Logger) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[INFO] %s", msg)
	EmitLog(l.emitter, "info", msg)
}

// Warn logs a warning message (always shown)
func (l *Logger) Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[WARN] %s", msg)
	EmitLog(l.emitter, "warn", msg)
}

// Error logs an error message (always shown)
func (l *Logger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[ERROR] %s", msg)
	EmitLog(l.emitter, "error", msg)
}

// Crawler handles web crawling operations
type Crawler struct {
	config       Config
	state        *CrawlerState
	fetcher      Fetcher
	robotsClient *http.Client // Separate client for robots.txt (always HTTP)
	mu           sync.RWMutex
	wg           sync.WaitGroup
	semaphore    chan struct{}
	log          *Logger
	robotsCache  map[string]*robotstxt.RobotsData
	robotsMu     sync.RWMutex
	metrics      *CrawlerMetrics
	ctx          context.Context
	cancel       context.CancelFunc
	emitter      EventEmitter
	paused       bool
	pauseMu      sync.Mutex
	pauseCond    *sync.Cond
	loginDone    chan struct{}
	loginWaiting bool
	loginMu      sync.Mutex
	normalizer   *URLNormalizer // URL normalizer for deduplication
}

// NewCrawler creates a new Crawler instance with the given configuration
func NewCrawler(config Config, ctx context.Context) (*Crawler, error) {
	return NewCrawlerWithEmitter(config, ctx, nil)
}

// NewCrawlerWithEmitter creates a new Crawler instance with event emission capability
func NewCrawlerWithEmitter(config Config, ctx context.Context, emitter EventEmitter) (*Crawler, error) {
	// Set default user agent if not provided
	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	// Set default fetch mode
	if config.FetchMode == "" {
		config.FetchMode = FetchModeHTTP
	}

	// Create a child context so we can cancel it independently
	crawlerCtx, cancel := context.WithCancel(ctx)

	// Create the appropriate fetcher based on config
	var fetcher Fetcher
	var err error

	logger := &Logger{verbose: config.Verbose, emitter: emitter}

	switch config.FetchMode {
	case FetchModeBrowser:
		logger.Info("Using browser-based fetching (headless=%v)", config.Headless)
		fetcher, err = NewBrowserFetcherWithPageLoadWait(config.Headless, userAgent, config.AntiBot, config.PageLoadWait)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create browser fetcher: %w", err)
		}
	default:
		logger.Info("Using HTTP-based fetching")
		fetcher = NewHTTPFetcher()
	}

	// Configure HTTP transport for robots.txt fetching (always HTTP)
	robotsTransport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	}

	c := &Crawler{
		config:  config,
		fetcher: fetcher,
		robotsClient: &http.Client{
			Timeout:   HTTPTimeout,
			Transport: robotsTransport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= MaxRedirects {
					return fmt.Errorf("stopped after %d redirects", MaxRedirects)
				}
				req.Header.Set("User-Agent", userAgent)
				return nil
			},
		},
		log:         logger,
		robotsCache: make(map[string]*robotstxt.RobotsData),
		metrics:     NewCrawlerMetrics(),
		ctx:         crawlerCtx,
		cancel:      cancel,
		emitter:     emitter,
	}

	c.pauseCond = sync.NewCond(&c.pauseMu)

	// Initialize URL normalizer if enabled (default: true)
	if config.NormalizeURLs {
		c.normalizer = NewURLNormalizer(config.LowercasePaths)
		logger.Debug("URL normalization enabled (lowercase paths: %v)", config.LowercasePaths)
	}

	if config.Concurrent {
		c.semaphore = make(chan struct{}, MaxConcurrentRequests)
	}

	return c, nil
}

// GetMetrics returns the current metrics
func (c *Crawler) GetMetrics() *CrawlerMetrics {
	return c.metrics
}

// GetState returns the current crawler state
func (c *Crawler) GetState() *CrawlerState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// IsPaused returns whether the crawler is currently paused
func (c *Crawler) IsPaused() bool {
	c.pauseMu.Lock()
	defer c.pauseMu.Unlock()
	return c.paused
}

// Pause pauses the crawler
func (c *Crawler) Pause() {
	c.pauseMu.Lock()
	c.paused = true
	c.pauseMu.Unlock()
	EmitStateChange(c.emitter, EventCrawlPaused)
}

// Resume resumes the crawler
func (c *Crawler) Resume() {
	c.pauseMu.Lock()
	c.paused = false
	c.pauseCond.Broadcast()
	c.pauseMu.Unlock()
	EmitStateChange(c.emitter, EventCrawlResumed)
}

// IsWaitingForLogin returns whether the crawler is waiting for manual login
func (c *Crawler) IsWaitingForLogin() bool {
	c.loginMu.Lock()
	defer c.loginMu.Unlock()
	return c.loginWaiting
}

// ConfirmLogin signals that manual login is complete
func (c *Crawler) ConfirmLogin() {
	c.loginMu.Lock()
	defer c.loginMu.Unlock()
	if c.loginDone != nil && c.loginWaiting {
		close(c.loginDone)
		c.loginWaiting = false
	}
}

// Stop stops the crawler gracefully
func (c *Crawler) Stop() {
	c.cancel()
	// Also resume in case we're paused, so the crawler can exit
	c.Resume()
	EmitStateChange(c.emitter, EventCrawlStopped)
}

// Close releases resources held by the crawler
func (c *Crawler) Close() error {
	if c.fetcher != nil {
		return c.fetcher.Close()
	}
	return nil
}

// checkPaused checks if crawler is paused and waits
func (c *Crawler) checkPaused() {
	c.pauseMu.Lock()
	for c.paused && !c.isShuttingDown() {
		c.pauseCond.Wait()
	}
	c.pauseMu.Unlock()
}

// isShuttingDown checks if the crawler should stop due to context cancellation
func (c *Crawler) isShuttingDown() bool {
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

// normalizeURL normalizes a URL for deduplication if normalization is enabled
func (c *Crawler) normalizeURL(rawURL string) string {
	if c.normalizer == nil {
		return rawURL
	}
	return c.normalizer.Normalize(rawURL)
}

// fetchRobots performs an HTTP GET request for robots.txt
func (c *Crawler) fetchRobots(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}

	// Set User-Agent header
	userAgent := c.config.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	req.Header.Set("User-Agent", userAgent)

	return c.robotsClient.Do(req)
}

// getRobots fetches and caches robots.txt for a given host
func (c *Crawler) getRobots(host string, scheme string) *robotstxt.RobotsData {
	// Check cache first
	c.robotsMu.RLock()
	robots, exists := c.robotsCache[host]
	c.robotsMu.RUnlock()

	if exists {
		return robots
	}

	// Fetch robots.txt
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", scheme, host)
	resp, err := c.fetchRobots(robotsURL)
	if err != nil {
		c.log.Debug("Failed to fetch robots.txt for %s: %v", host, err)
		// Cache nil to avoid repeated failed fetches
		c.robotsMu.Lock()
		c.robotsCache[host] = nil
		c.robotsMu.Unlock()
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Debug("robots.txt returned %d for %s", resp.StatusCode, host)
		c.robotsMu.Lock()
		c.robotsCache[host] = nil
		c.robotsMu.Unlock()
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Debug("Failed to read robots.txt for %s: %v", host, err)
		c.robotsMu.Lock()
		c.robotsCache[host] = nil
		c.robotsMu.Unlock()
		return nil
	}

	robots, err = robotstxt.FromBytes(body)
	if err != nil {
		c.log.Debug("Failed to parse robots.txt for %s: %v", host, err)
		c.robotsMu.Lock()
		c.robotsCache[host] = nil
		c.robotsMu.Unlock()
		return nil
	}

	c.log.Debug("Loaded robots.txt for %s", host)
	c.robotsMu.Lock()
	c.robotsCache[host] = robots
	c.robotsMu.Unlock()
	return robots
}

// isAllowedByRobots checks if a URL is allowed by robots.txt
func (c *Crawler) isAllowedByRobots(rawURL string) bool {
	// Skip check if robots.txt is ignored
	if c.config.IgnoreRobots {
		return true
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return true // Allow if we can't parse
	}

	robots := c.getRobots(parsed.Host, parsed.Scheme)
	if robots == nil {
		return true // Allow if no robots.txt
	}

	// Get the user agent to check against
	userAgent := c.config.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	// Check if the path is allowed
	group := robots.FindGroup(userAgent)
	if group == nil {
		return true
	}

	return group.Test(parsed.Path)
}

// Start begins the crawling process
func (c *Crawler) Start() error {
	// Load state
	state, err := LoadState(c.config.StateFile, c.config.URL)
	if err != nil {
		return fmt.Errorf("failed to load state: %v", err)
	}
	c.state = state

	if err := EnsureOutputDir(&c.config); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	if len(c.state.Queue) == 0 {
		// Normalize the initial URL for consistent deduplication
		initialURL := c.normalizeURL(c.config.URL)
		c.state.Queue = append(c.state.Queue, URLInfo{URL: initialURL, Depth: 0})
		c.state.URLDepths[initialURL] = 0
		c.state.Queued[initialURL] = true
	}

	// Handle login wait for non-headless browser mode
	if c.config.WaitForLogin && c.config.FetchMode == FetchModeBrowser && !c.config.Headless {
		if bf, ok := c.fetcher.(*BrowserFetcher); ok {
			c.log.Info("Opening browser for login at: %s", c.config.URL)

			// Navigate browser to initial URL
			closeTab, err := bf.NavigateForLogin(c.config.URL)
			if err != nil {
				return fmt.Errorf("failed to open browser for login: %w", err)
			}

			// Set up login wait state
			c.loginMu.Lock()
			c.loginDone = make(chan struct{})
			c.loginWaiting = true
			c.loginMu.Unlock()

			// Emit event to notify frontend/CLI
			EmitWaitingForLogin(c.emitter, c.config.URL)
			c.log.Info("Waiting for login confirmation...")

			// Wait for login confirmation or cancellation
			select {
			case <-c.loginDone:
				c.log.Info("Login confirmed, starting crawl...")
			case <-c.ctx.Done():
				closeTab()
				return fmt.Errorf("crawl cancelled while waiting for login")
			}

			// Close the login tab
			closeTab()
		}
	}

	c.log.Info("Starting crawler with %d URLs in queue", len(c.state.Queue))
	c.log.Debug("Max depth set to: %d", c.config.MaxDepth)

	EmitStateChange(c.emitter, EventCrawlStarted)

	if c.config.Concurrent {
		c.crawlConcurrent()
	} else {
		c.crawlSequential()
	}

	// Display final summary if progress is enabled
	if c.config.ShowProgress {
		c.metrics.DisplayFinalSummary()
	}

	// Write metrics to JSON if requested
	if c.config.MetricsFile != "" {
		if err := c.metrics.WriteJSON(c.config.MetricsFile); err != nil {
			c.log.Error("Failed to write metrics: %v", err)
		} else {
			c.log.Info("Metrics written to %s", c.config.MetricsFile)
		}
	}

	EmitStateChange(c.emitter, EventCrawlCompleted)

	// Generate index page
	if err := GenerateIndex(c.config.OutputDir); err != nil {
		c.log.Warn("Failed to generate index: %v", err)
	} else {
		c.log.Info("Generated index page at %s", filepath.Join(c.config.OutputDir, "_index.html"))
	}

	return SaveState(c.state, c.config.StateFile)
}

func (c *Crawler) crawlSequential() {
	for len(c.state.Queue) > 0 {
		// Check for pause
		c.checkPaused()

		// Check for shutdown signal
		if c.isShuttingDown() {
			c.log.Info("Shutdown signal received, stopping crawl...")
			return
		}

		currentURLInfo := c.state.Queue[0]
		c.state.Queue = c.state.Queue[1:]

		// Update metrics queue size
		c.metrics.SetQueueSize(len(c.state.Queue))

		c.log.Debug("Queue length: %d, Processing: %s (depth %d)", len(c.state.Queue), currentURLInfo.URL, currentURLInfo.Depth)

		// Remove from queued map
		delete(c.state.Queued, currentURLInfo.URL)

		if c.state.Visited[currentURLInfo.URL] {
			c.log.Debug("Skipping already visited: %s", currentURLInfo.URL)
			c.metrics.IncrementSkipped()
			continue
		}

		// Check depth constraint
		if currentURLInfo.Depth > c.config.MaxDepth {
			c.log.Debug("Skipping due to depth limit (%d > %d): %s", currentURLInfo.Depth, c.config.MaxDepth, currentURLInfo.URL)
			c.metrics.IncrementDepthLimitHits()
			continue
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					c.log.Error("Recovered from panic while processing %s: %v", currentURLInfo.URL, r)
					c.metrics.IncrementErrored()
				}
			}()
			c.processURL(currentURLInfo.URL, currentURLInfo.Depth)
		}()

		// Emit progress event and display if enabled
		if c.config.ShowProgress && c.metrics.ShouldDisplay() {
			c.metrics.DisplayProgress(c.config.Verbose)
			EmitProgress(c.emitter, c.metrics, currentURLInfo.URL)
		}

		time.Sleep(c.config.Delay)

		// Save state periodically
		if c.state.Processed%StateSaveInterval == 0 {
			c.log.Debug("Saving state at %d processed URLs", c.state.Processed)
			if err := SaveState(c.state, c.config.StateFile); err != nil {
				c.log.Warn("Failed to save state: %v", err)
			}
		}
	}
	c.log.Debug("Crawling completed. Queue is now empty.")
}

func (c *Crawler) crawlConcurrent() {
	var activeGoroutines atomic.Int64

	for {
		// Check for pause
		c.checkPaused()

		// Check for shutdown signal
		if c.isShuttingDown() {
			c.log.Info("Shutdown signal received, waiting for active goroutines to finish...")
			break
		}

		// Check if we have URLs to process
		if len(c.state.Queue) > 0 {
			currentURLInfo := c.state.Queue[0]
			c.state.Queue = c.state.Queue[1:]

			// Update metrics queue size
			c.metrics.SetQueueSize(len(c.state.Queue))

			c.log.Debug("Concurrent - Queue length: %d, Processing: %s (depth %d)", len(c.state.Queue), currentURLInfo.URL, currentURLInfo.Depth)

			// Remove from queued map with proper locking
			c.mu.Lock()
			delete(c.state.Queued, currentURLInfo.URL)
			visited := c.state.Visited[currentURLInfo.URL]
			c.mu.Unlock()

			if visited {
				c.log.Debug("Concurrent - Skipping already visited: %s", currentURLInfo.URL)
				c.metrics.IncrementSkipped()
				continue
			}

			// Check depth constraint
			if currentURLInfo.Depth > c.config.MaxDepth {
				c.log.Debug("Concurrent - Skipping due to depth limit (%d > %d): %s", currentURLInfo.Depth, c.config.MaxDepth, currentURLInfo.URL)
				c.metrics.IncrementDepthLimitHits()
				continue
			}

			c.wg.Add(1)
			activeGoroutines.Add(1)
			c.semaphore <- struct{}{} // Acquire semaphore

			go func(urlInfo URLInfo) {
				defer c.wg.Done()
				defer func() { <-c.semaphore }()            // Release semaphore
				defer func() { activeGoroutines.Add(-1) }() // Decrement counter atomically
				defer func() {
					if r := recover(); r != nil {
						c.log.Error("Recovered from panic while processing %s: %v", urlInfo.URL, r)
						c.metrics.IncrementErrored()
					}
				}()

				c.processURL(urlInfo.URL, urlInfo.Depth)
				time.Sleep(c.config.Delay)
			}(currentURLInfo)

			// Emit progress event and display if enabled
			if c.config.ShowProgress && c.metrics.ShouldDisplay() {
				c.metrics.DisplayProgress(c.config.Verbose)
				EmitProgress(c.emitter, c.metrics, currentURLInfo.URL)
			}

			// Save state periodically
			if c.state.Processed%StateSaveInterval == 0 {
				c.log.Debug("Concurrent - Waiting for goroutines before saving state at %d processed URLs", c.state.Processed)
				c.wg.Wait()
				if err := SaveState(c.state, c.config.StateFile); err != nil {
					c.log.Warn("Failed to save state: %v", err)
				}
			}
		} else {
			// Queue is empty, check if we have active goroutines that might add more URLs
			currentActive := activeGoroutines.Load()

			if currentActive == 0 {
				// No more goroutines running and queue is empty - we're done
				c.log.Debug("Concurrent - No active goroutines and empty queue, crawling completed")
				break
			} else {
				// Wait a bit for goroutines to potentially add more URLs
				c.log.Debug("Concurrent - Queue empty but %d goroutines still active, waiting...", currentActive)
				time.Sleep(QueueEmptyWaitTime)
			}
		}
	}

	c.log.Debug("Concurrent - Main loop finished, waiting for remaining goroutines")
	c.wg.Wait()
	c.log.Debug("Concurrent - All goroutines finished, crawling completed")
}

func (c *Crawler) processURL(rawURL string, currentDepth int) {
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("Panic in processURL for %s: %v", rawURL, r)
			c.metrics.IncrementErrored()
		}
	}()

	c.mu.Lock()
	if c.state.Visited[rawURL] {
		c.mu.Unlock()
		return
	}
	c.state.Visited[rawURL] = true
	c.state.Processed++
	c.mu.Unlock()

	c.metrics.IncrementProcessed()
	c.log.Info("[%d] Processing: %s", c.state.Processed, rawURL)

	// Check robots.txt before fetching
	if !c.isAllowedByRobots(rawURL) {
		c.log.Debug("Blocked by robots.txt: %s", rawURL)
		c.metrics.IncrementRobotsBlocked()
		return
	}

	// Get user agent for fetcher
	userAgent := c.config.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	// Check if pagination is enabled and we're using browser mode
	if c.config.Pagination.Enable && c.config.FetchMode == FetchModeBrowser {
		c.processURLWithPagination(rawURL, currentDepth, userAgent)
		return
	}

	result, err := c.fetcher.Fetch(rawURL, userAgent)
	if err != nil {
		c.log.Error("Error fetching %s: %v", rawURL, err)
		c.metrics.IncrementErrored()
		return
	}

	if result.StatusCode != http.StatusOK {
		c.log.Debug("HTTP %d for %s", result.StatusCode, rawURL)
		c.metrics.IncrementErrored()
		return
	}

	// Check if content type should be excluded
	if c.shouldExcludeByContentType(result.ContentType) {
		c.log.Debug("Skipping %s: excluded content type %s", rawURL, result.ContentType)
		c.metrics.IncrementContentFiltered()
		return
	}

	body := result.Body

	// Check if page has meaningful content
	if !c.hasContent(string(body)) {
		c.log.Debug("Skipping %s: no meaningful content", rawURL)
		c.metrics.IncrementContentFiltered()
		return
	}

	// Save the content
	if err := c.saveContent(rawURL, body); err != nil {
		c.log.Error("Error saving content for %s: %v", rawURL, err)
		c.metrics.IncrementErrored()
		return
	}

	c.metrics.IncrementSaved(int64(len(body)))

	// Extract and queue new URLs - wrap in error handling
	func() {
		defer func() {
			if r := recover(); r != nil {
				c.log.Error("Panic extracting URLs from %s: %v", rawURL, r)
			}
		}()
		c.extractAndQueueURLs(rawURL, string(body), currentDepth)
	}()
}

// processURLWithPagination handles URL processing with click-based pagination
func (c *Crawler) processURLWithPagination(rawURL string, currentDepth int, userAgent string) {
	browserFetcher, ok := c.fetcher.(*BrowserFetcher)
	if !ok {
		c.log.Error("Pagination enabled but fetcher is not a BrowserFetcher")
		c.metrics.IncrementErrored()
		return
	}

	c.log.Debug("Using pagination for %s (selector: %s)", rawURL, c.config.Pagination.Selector)

	// Page callback processes each paginated page
	pageCallback := func(result *FetchResult, pageNumber int, virtualURL string) error {
		// Check if content type should be excluded
		if c.shouldExcludeByContentType(result.ContentType) {
			c.log.Debug("Skipping page %d of %s: excluded content type %s", pageNumber, rawURL, result.ContentType)
			c.metrics.IncrementContentFiltered()
			return nil
		}

		body := result.Body

		// Check if page has meaningful content
		if !c.hasContent(string(body)) {
			c.log.Debug("Skipping page %d of %s: no meaningful content", pageNumber, rawURL)
			c.metrics.IncrementContentFiltered()
			return nil
		}

		// Save the content using the virtual URL for unique filenames
		if err := c.saveContent(virtualURL, body); err != nil {
			c.log.Error("Error saving content for page %d of %s: %v", pageNumber, rawURL, err)
			c.metrics.IncrementErrored()
			return nil // Don't stop pagination on save error
		}

		c.metrics.IncrementSaved(int64(len(body)))
		c.log.Info("[%d] Saved page %d: %s", c.state.Processed, pageNumber, virtualURL)

		// Extract and queue new URLs at the same depth (pagination doesn't increase depth)
		func() {
			defer func() {
				if r := recover(); r != nil {
					c.log.Error("Panic extracting URLs from page %d of %s: %v", pageNumber, rawURL, r)
				}
			}()
			c.extractAndQueueURLs(rawURL, string(body), currentDepth)
		}()

		return nil
	}

	// Execute paginated fetch
	paginationResult, err := browserFetcher.FetchWithPagination(rawURL, userAgent, c.config.Pagination, pageCallback)
	if err != nil {
		c.log.Error("Error during pagination for %s: %v", rawURL, err)
		c.metrics.IncrementErrored()
		return
	}

	c.log.Debug("Pagination completed for %s: %d pages fetched, reason: %s",
		rawURL, paginationResult.TotalPages, paginationResult.ExhaustedReason)

	if paginationResult.LastError != nil {
		c.log.Warn("Pagination had errors for %s: %v", rawURL, paginationResult.LastError)
	}
}

func (c *Crawler) extractAndQueueURLs(baseURL, html string, currentDepth int) {
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("Panic in extractAndQueueURLs for %s: %v", baseURL, r)
		}
	}()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		c.log.Error("Error parsing HTML for %s: %v", baseURL, err)
		return
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		c.log.Error("Error parsing base URL %s: %v", baseURL, err)
		return
	}

	// Determine which selectors to use
	selectors := c.config.LinkSelectors
	if len(selectors) == 0 {
		// Default: use all links with href attribute
		selectors = []string{"a[href]"}
	}

	// Process each selector
	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			defer func() {
				if r := recover(); r != nil {
					c.log.Error("Panic processing link in %s with selector %s: %v", baseURL, selector, r)
				}
			}()

			href, exists := s.Attr("href")
			if !exists {
				return
			}

			absoluteURL, err := base.Parse(href)
			if err != nil {
				// Skip malformed URLs silently
				return
			}

			urlStr := absoluteURL.String()

			if c.isValidURL(urlStr) {
				func() {
					defer func() {
						if r := recover(); r != nil {
							c.log.Error("Panic queuing URL %s: %v", urlStr, r)
						}
					}()

					// Normalize URL for deduplication
					normalizedURL := c.normalizeURL(urlStr)

					c.mu.Lock()
					defer c.mu.Unlock()

					if !c.state.Visited[normalizedURL] && !c.state.Queued[normalizedURL] {
						// Add URL with incremented depth (store normalized URL)
						newDepth := currentDepth + 1
						c.state.Queue = append(c.state.Queue, URLInfo{URL: normalizedURL, Depth: newDepth})
						c.state.URLDepths[normalizedURL] = newDepth
						c.state.Queued[normalizedURL] = true
					}
				}()
			}
		})
	}
}
