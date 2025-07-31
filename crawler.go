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

		// Remove from queued map
		delete(c.state.Queued, currentURLInfo.URL)

		if c.state.Visited[currentURLInfo.URL] {
			continue
		}

		// Check depth constraint
		if currentURLInfo.Depth > c.config.MaxDepth {
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
			if err := c.saveState(); err != nil {
				fmt.Printf("Warning: Failed to save state: %v\n", err)
			}
		}
	}
}

func (c *Crawler) crawlConcurrent() {
	for len(c.state.Queue) > 0 {
		currentURLInfo := c.state.Queue[0]
		c.state.Queue = c.state.Queue[1:]

		// Remove from queued map with proper locking
		c.mu.Lock()
		delete(c.state.Queued, currentURLInfo.URL)
		visited := c.state.Visited[currentURLInfo.URL]
		c.mu.Unlock()

		if visited {
			continue
		}

		// Check depth constraint
		if currentURLInfo.Depth > c.config.MaxDepth {
			continue
		}

		c.wg.Add(1)
		c.semaphore <- struct{}{} // Acquire semaphore

		go func(urlInfo URLInfo) {
			defer c.wg.Done()
			defer func() { <-c.semaphore }() // Release semaphore
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
			c.wg.Wait()
			if err := c.saveState(); err != nil {
				fmt.Printf("Warning: Failed to save state: %v\n", err)
			}
		}
	}

	c.wg.Wait()
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
