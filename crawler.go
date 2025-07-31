package main

import (
	// "crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type URLInfo struct {
	URL   string `json:"url"`
	Depth int    `json:"depth"`
}

type CrawlerState struct {
	Visited   map[string]bool `json:"visited"`
	Queue     []URLInfo       `json:"queue"`
	BaseURL   string          `json:"base_url"`
	Processed int             `json:"processed"`
	URLDepths map[string]int  `json:"url_depths"`
	Queued    map[string]bool `json:"queued"`
}

type Crawler struct {
	config    Config
	state     *CrawlerState
	client    *http.Client
	mu        sync.RWMutex
	wg        sync.WaitGroup
	semaphore chan struct{}
}

func NewCrawler(config Config) *Crawler {
	c := &Crawler{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if config.Concurrent {
		c.semaphore = make(chan struct{}, 10) // Limit to 10 concurrent requests
	}

	return c
}

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

	fmt.Printf("Starting crawler with %d URLs in queue\n", len(c.state.Queue))
	fmt.Printf("Debug: Max depth set to: %d\n", c.config.MaxDepth)

	if c.config.Concurrent {
		c.crawlConcurrent()
	} else {
		c.crawlSequential()
	}

	return c.saveState()
}

func (c *Crawler) isValidURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Check if URL extension should be excluded
	if c.shouldExcludeByExtension(parsed.Path) {
		return false
	}

	// Must be HTTP/HTTPS
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	// If prefix filtering is disabled (empty or "none"), allow any HTTP/HTTPS URL discovered through the tree
	if c.config.PrefixFilterURL == "" || c.config.PrefixFilterURL == "none" {
		return true
	}

	// With prefix filtering enabled: check URL prefix constraint using the specified prefix filter URL
	prefixURL, err := url.Parse(c.config.PrefixFilterURL)
	if err != nil {
		return false
	}

	// Check if URL has the prefix URL as prefix
	if parsed.Host != prefixURL.Host {
		return false
	}

	// Check if path starts with prefix path
	prefixPath := strings.TrimSuffix(prefixURL.Path, "/")
	urlPath := strings.TrimSuffix(parsed.Path, "/")

	return strings.HasPrefix(urlPath, prefixPath)
}

func (c *Crawler) shouldExcludeByExtension(path string) bool {
	if len(c.config.ExcludeExtensions) == 0 {
		return false
	}

	// Extract extension from path
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		// Remove the dot from extension
		ext = ext[1:]
	}

	// Check if extension is in exclude list
	for _, excludeExt := range c.config.ExcludeExtensions {
		if ext == excludeExt {
			return true
		}
	}

	return false
}

func (c *Crawler) shouldExcludeByContentType(contentType string) bool {
	if len(c.config.ExcludeExtensions) == 0 {
		return false
	}

	// Convert content type to lowercase for comparison
	contentType = strings.ToLower(contentType)
	
	// Remove charset and other parameters (e.g., "application/json; charset=utf-8" -> "application/json")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	// Comprehensive mapping of content types to extensions
	contentTypeToExt := map[string]string{
		// Common web assets
		"application/json":            "json",
		"text/javascript":             "js", 
		"application/javascript":      "js",
		"text/css":                   "css",
		
		// Images
		"image/png":                  "png",
		"image/jpeg":                 "jpg",
		"image/jpg":                  "jpg",
		"image/gif":                  "gif",
		"image/webp":                 "webp",
		"image/svg+xml":              "svg",
		"image/bmp":                  "bmp",
		"image/tiff":                 "tiff",
		"image/ico":                  "ico",
		
		// Documents
		"application/pdf":            "pdf",
		"application/msword":         "doc",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
		"application/vnd.ms-excel":   "xls",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": "xlsx",
		"application/vnd.ms-powerpoint": "ppt",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",
		
		// Archives
		"application/zip":            "zip",
		"application/x-rar-compressed": "rar",
		"application/x-tar":          "tar",
		"application/gzip":           "gz",
		"application/x-7z-compressed": "7z",
		
		// Data formats
		"application/xml":            "xml",
		"text/xml":                   "xml",
		"text/csv":                   "csv",
		"application/yaml":           "yaml",
		"text/yaml":                  "yaml",
		
		// Media
		"video/mp4":                  "mp4",
		"video/mpeg":                 "mpeg",
		"video/quicktime":            "mov",
		"video/x-msvideo":            "avi",
		"audio/mpeg":                 "mp3",
		"audio/wav":                  "wav",
		"audio/ogg":                  "ogg",
		
		// Fonts
		"font/woff":                  "woff",
		"font/woff2":                 "woff2",
		"application/font-woff":      "woff",
		"application/font-woff2":     "woff2",
		"font/ttf":                   "ttf",
		"font/otf":                   "otf",
	}

	// First check exact mapping
	if ext, exists := contentTypeToExt[contentType]; exists {
		for _, excludeExt := range c.config.ExcludeExtensions {
			if ext == excludeExt {
				return true
			}
		}
	}

	// For unmapped content types, try to infer from the content type string
	// e.g., "application/vnd.company.customformat" might contain the extension
	for _, excludeExt := range c.config.ExcludeExtensions {
		if strings.Contains(contentType, excludeExt) {
			return true
		}
	}

	return false
}

func (c *Crawler) loadState() error {
	c.state = &CrawlerState{
		Visited:   make(map[string]bool),
		Queue:     []URLInfo{},
		BaseURL:   c.config.URL,
		URLDepths: make(map[string]int),
		Queued:    make(map[string]bool),
	}

	if _, err := os.Stat(c.config.StateFile); os.IsNotExist(err) {
		return nil // No existing state file
	}

	data, err := os.ReadFile(c.config.StateFile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, c.state); err != nil {
		return err
	}

	// Initialize Queued map if it doesn't exist (backward compatibility)
	if c.state.Queued == nil {
		c.state.Queued = make(map[string]bool)
		// Populate queued map from existing queue
		for _, urlInfo := range c.state.Queue {
			c.state.Queued[urlInfo.URL] = true
		}
	}

	return nil
}

func (c *Crawler) saveState() error {
	data, err := json.MarshalIndent(c.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.config.StateFile, data, 0644)
}

func (c *Crawler) crawlSequential() {
	for len(c.state.Queue) > 0 {
		currentURLInfo := c.state.Queue[0]
		c.state.Queue = c.state.Queue[1:]

		fmt.Printf("Debug: Queue length: %d, Processing: %s (depth %d)\n", len(c.state.Queue), currentURLInfo.URL, currentURLInfo.Depth)

		// Remove from queued map
		delete(c.state.Queued, currentURLInfo.URL)

		if c.state.Visited[currentURLInfo.URL] {
			fmt.Printf("Debug: Skipping already visited: %s\n", currentURLInfo.URL)
			continue
		}

		// Check depth constraint
		if currentURLInfo.Depth > c.config.MaxDepth {
			fmt.Printf("Debug: Skipping due to depth limit (%d > %d): %s\n", currentURLInfo.Depth, c.config.MaxDepth, currentURLInfo.URL)
			continue
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Recovered from panic while processing %s: %v\n", currentURLInfo.URL, r)
				}
			}()
			c.processURL(currentURLInfo.URL, currentURLInfo.Depth)
		}()

		time.Sleep(c.config.Delay)

		// Save state periodically
		if c.state.Processed%10 == 0 {
			fmt.Printf("Debug: Saving state at %d processed URLs\n", c.state.Processed)
			if err := c.saveState(); err != nil {
				fmt.Printf("Warning: Failed to save state: %v\n", err)
			}
		}
	}
	fmt.Printf("Debug: Crawling completed. Queue is now empty.\n")
}

func (c *Crawler) crawlConcurrent() {
	activeGoroutines := 0
	
	for {
		// Check if we have URLs to process
		if len(c.state.Queue) > 0 {
			currentURLInfo := c.state.Queue[0]
			c.state.Queue = c.state.Queue[1:]

			fmt.Printf("Debug: Concurrent - Queue length: %d, Processing: %s (depth %d)\n", len(c.state.Queue), currentURLInfo.URL, currentURLInfo.Depth)

			// Remove from queued map with proper locking
			c.mu.Lock()
			delete(c.state.Queued, currentURLInfo.URL)
			visited := c.state.Visited[currentURLInfo.URL]
			c.mu.Unlock()

			if visited {
				fmt.Printf("Debug: Concurrent - Skipping already visited: %s\n", currentURLInfo.URL)
				continue
			}

			// Check depth constraint
			if currentURLInfo.Depth > c.config.MaxDepth {
				fmt.Printf("Debug: Concurrent - Skipping due to depth limit (%d > %d): %s\n", currentURLInfo.Depth, c.config.MaxDepth, currentURLInfo.URL)
				continue
			}

			c.wg.Add(1)
			activeGoroutines++
			c.semaphore <- struct{}{} // Acquire semaphore

			go func(urlInfo URLInfo) {
				defer c.wg.Done()
				defer func() { <-c.semaphore }() // Release semaphore
				defer func() {
					c.mu.Lock()
					activeGoroutines--
					c.mu.Unlock()
				}()
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("Recovered from panic while processing %s: %v\n", urlInfo.URL, r)
					}
				}()

				c.processURL(urlInfo.URL, urlInfo.Depth)
				time.Sleep(c.config.Delay)
			}(currentURLInfo)

			// Save state periodically
			if c.state.Processed%10 == 0 {
				fmt.Printf("Debug: Concurrent - Waiting for goroutines before saving state at %d processed URLs\n", c.state.Processed)
				c.wg.Wait()
				if err := c.saveState(); err != nil {
					fmt.Printf("Warning: Failed to save state: %v\n", err)
				}
			}
		} else {
			// Queue is empty, check if we have active goroutines that might add more URLs
			c.mu.RLock()
			currentActiveGoroutines := activeGoroutines
			c.mu.RUnlock()
			
			if currentActiveGoroutines == 0 {
				// No more goroutines running and queue is empty - we're done
				fmt.Printf("Debug: Concurrent - No active goroutines and empty queue, crawling completed\n")
				break
			} else {
				// Wait a bit for goroutines to potentially add more URLs
				fmt.Printf("Debug: Concurrent - Queue empty but %d goroutines still active, waiting...\n", currentActiveGoroutines)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	fmt.Printf("Debug: Concurrent - Main loop finished, waiting for remaining goroutines\n")
	c.wg.Wait()
	fmt.Printf("Debug: Concurrent - All goroutines finished, crawling completed\n")
}

func (c *Crawler) processURL(rawURL string, currentDepth int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in processURL for %s: %v\n", rawURL, r)
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

	fmt.Printf("[%d] Processing: %s\n", c.state.Processed, rawURL)

	resp, err := c.client.Get(rawURL)
	if err != nil {
		fmt.Printf("Error fetching %s: %v\n", rawURL, err)
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("HTTP %d for %s\n", resp.StatusCode, rawURL)
		return
	}

	// Check if content type should be excluded
	if c.shouldExcludeByContentType(resp.Header.Get("Content-Type")) {
		fmt.Printf("Skipping %s: excluded content type %s\n", rawURL, resp.Header.Get("Content-Type"))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading body for %s: %v\n", rawURL, err)
		return
	}

	// Check if page has meaningful content
	if !c.hasContent(string(body)) {
		fmt.Printf("Skipping %s: no meaningful content\n", rawURL)
		return
	}

	// Save the content
	if err := c.saveContent(rawURL, body); err != nil {
		fmt.Printf("Error saving content for %s: %v\n", rawURL, err)
		return
	}

	// Extract and queue new URLs - wrap in error handling
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic extracting URLs from %s: %v\n", rawURL, r)
			}
		}()
		c.extractAndQueueURLs(rawURL, string(body), currentDepth)
	}()
}

func (c *Crawler) hasContent(html string) bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in hasContent: %v\n", r)
		}
	}()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return false
	}

	// Remove script and style elements
	doc.Find("script, style").Remove()

	// Get text content
	text := strings.TrimSpace(doc.Text())

	// Consider page has content if it has more than 100 characters of text
	return len(text) > 100
}

func (c *Crawler) saveContent(rawURL string, content []byte) error {
	// Create filename based on URL structure
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL %s: %v", rawURL, err)
	}

	// Generate filename from URL path
	filename := c.generateFilename(parsedURL)

	// Create subdirectories if needed
	fullPath := filepath.Join(c.config.OutputDir, filename)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	// Create metadata file
	metadata := map[string]interface{}{
		"url":       rawURL,
		"timestamp": time.Now().Unix(),
		"size":      len(content),
	}

	metaData, _ := json.MarshalIndent(metadata, "", "  ")
	metaFile := strings.TrimSuffix(fullPath, ".html") + ".meta.json"

	// Save both files
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return err
	}

	return os.WriteFile(metaFile, metaData, 0644)
}

func (c *Crawler) generateFilename(parsedURL *url.URL) string {
	path := parsedURL.Path

	// Handle root path
	if path == "" || path == "/" {
		return "index.html"
	}

	// Clean up the path
	path = strings.Trim(path, "/")

	// Replace invalid characters for filenames
	path = strings.ReplaceAll(path, ":", "_")
	path = strings.ReplaceAll(path, "?", "_")
	path = strings.ReplaceAll(path, "*", "_")
	path = strings.ReplaceAll(path, "<", "_")
	path = strings.ReplaceAll(path, ">", "_")
	path = strings.ReplaceAll(path, "|", "_")
	path = strings.ReplaceAll(path, "\"", "_")

	// Add .html extension if it doesn't have an extension
	if !strings.Contains(filepath.Base(path), ".") {
		path += ".html"
	}

	return path
}

func (c *Crawler) extractAndQueueURLs(baseURL, html string, currentDepth int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in extractAndQueueURLs for %s: %v\n", baseURL, r)
		}
	}()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		fmt.Printf("Error parsing HTML for %s: %v\n", baseURL, err)
		return
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		fmt.Printf("Error parsing base URL %s: %v\n", baseURL, err)
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
					fmt.Printf("Panic processing link in %s with selector %s: %v\n", baseURL, selector, r)
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
							fmt.Printf("Panic queuing URL %s: %v\n", urlStr, r)
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
