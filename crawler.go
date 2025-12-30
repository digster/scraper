package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
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
}

// Debug logs a message only if verbose mode is enabled
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbose {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs an informational message (always shown)
func (l *Logger) Info(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

// Warn logs a warning message (always shown)
func (l *Logger) Warn(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

// Error logs an error message (always shown)
func (l *Logger) Error(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// Crawler handles web crawling operations
type Crawler struct {
	config      Config
	state       *CrawlerState
	client      *http.Client
	mu          sync.RWMutex
	wg          sync.WaitGroup
	semaphore   chan struct{}
	log         *Logger
	robotsCache map[string]*robotstxt.RobotsData
	robotsMu    sync.RWMutex
	metrics     *CrawlerMetrics
}

// NewCrawler creates a new Crawler instance with the given configuration
func NewCrawler(config Config) *Crawler {
	// Set default user agent if not provided
	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	// Configure HTTP transport with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	c := &Crawler{
		config: config,
		client: &http.Client{
			Timeout:   HTTPTimeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= MaxRedirects {
					return fmt.Errorf("stopped after %d redirects", MaxRedirects)
				}
				// Preserve User-Agent header across redirects
				req.Header.Set("User-Agent", userAgent)
				return nil
			},
		},
		log:         &Logger{verbose: config.Verbose},
		robotsCache: make(map[string]*robotstxt.RobotsData),
		metrics:     NewCrawlerMetrics(),
	}

	if config.Concurrent {
		c.semaphore = make(chan struct{}, MaxConcurrentRequests)
	}

	return c
}

// fetch performs an HTTP GET request with the configured User-Agent header
func (c *Crawler) fetch(rawURL string) (*http.Response, error) {
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

	return c.client.Do(req)
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
	resp, err := c.fetch(robotsURL)
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
	if err := c.loadState(); err != nil {
		return fmt.Errorf("failed to load state: %v", err)
	}

	if err := os.MkdirAll(c.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	if len(c.state.Queue) == 0 {
		c.state.Queue = append(c.state.Queue, URLInfo{URL: c.config.URL, Depth: 0})
		c.state.URLDepths[c.config.URL] = 0
		c.state.Queued[c.config.URL] = true
	}

	c.log.Info("Starting crawler with %d URLs in queue", len(c.state.Queue))
	c.log.Debug("Max depth set to: %d", c.config.MaxDepth)

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

	return c.saveState()
}

func (c *Crawler) crawlSequential() {
	for len(c.state.Queue) > 0 {
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

		// Display progress if enabled
		if c.config.ShowProgress && c.metrics.ShouldDisplay() {
			c.metrics.DisplayProgress(c.config.Verbose)
		}

		time.Sleep(c.config.Delay)

		// Save state periodically
		if c.state.Processed%StateSaveInterval == 0 {
			c.log.Debug("Saving state at %d processed URLs", c.state.Processed)
			if err := c.saveState(); err != nil {
				c.log.Warn("Failed to save state: %v", err)
			}
		}
	}
	c.log.Debug("Crawling completed. Queue is now empty.")
}

func (c *Crawler) crawlConcurrent() {
	var activeGoroutines atomic.Int64

	for {
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

			// Display progress if enabled
			if c.config.ShowProgress && c.metrics.ShouldDisplay() {
				c.metrics.DisplayProgress(c.config.Verbose)
			}

			// Save state periodically
			if c.state.Processed%StateSaveInterval == 0 {
				c.log.Debug("Concurrent - Waiting for goroutines before saving state at %d processed URLs", c.state.Processed)
				c.wg.Wait()
				if err := c.saveState(); err != nil {
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

	resp, err := c.fetch(rawURL)
	if err != nil {
		c.log.Error("Error fetching %s: %v", rawURL, err)
		c.metrics.IncrementErrored()
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK {
		c.log.Debug("HTTP %d for %s", resp.StatusCode, rawURL)
		c.metrics.IncrementErrored()
		return
	}

	// Check if content type should be excluded
	if c.shouldExcludeByContentType(resp.Header.Get("Content-Type")) {
		c.log.Debug("Skipping %s: excluded content type %s", rawURL, resp.Header.Get("Content-Type"))
		c.metrics.IncrementContentFiltered()
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("Error reading body for %s: %v", rawURL, err)
		c.metrics.IncrementErrored()
		return
	}

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

					c.mu.Lock()
					defer c.mu.Unlock()

					if !c.state.Visited[urlStr] && !c.state.Queued[urlStr] {
						// Add URL with incremented depth
						newDepth := currentDepth + 1
						c.state.Queue = append(c.state.Queue, URLInfo{URL: urlStr, Depth: newDepth})
						c.state.URLDepths[urlStr] = newDepth
						c.state.Queued[urlStr] = true
					}
				}()
			}
		})
	}
}
