package main

import (
	"crypto/md5"
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

type CrawlerState struct {
	Visited   map[string]bool `json:"visited"`
	Queue     []string        `json:"queue"`
	BaseURL   string          `json:"base_url"`
	Processed int             `json:"processed"`
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
		c.state.Queue = append(c.state.Queue, c.config.URL)
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
	
	// If prefix filtering is disabled, only check for valid HTTP/HTTPS URLs
	if c.config.DisablePrefixFilter {
		return parsed.Scheme == "http" || parsed.Scheme == "https"
	}
	
	baseURL, err := url.Parse(c.state.BaseURL)
	if err != nil {
		return false
	}
	
	// Check if URL has the base URL as prefix
	if parsed.Host != baseURL.Host {
		return false
	}
	
	// Check if path starts with base path
	basePath := strings.TrimSuffix(baseURL.Path, "/")
	urlPath := strings.TrimSuffix(parsed.Path, "/")
	
	return strings.HasPrefix(urlPath, basePath)
}

func (c *Crawler) loadState() error {
	c.state = &CrawlerState{
		Visited: make(map[string]bool),
		Queue:   []string{},
		BaseURL: c.config.URL,
	}
	
	if _, err := os.Stat(c.config.StateFile); os.IsNotExist(err) {
		return nil // No existing state file
	}
	
	data, err := os.ReadFile(c.config.StateFile)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, c.state)
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
		currentURL := c.state.Queue[0]
		c.state.Queue = c.state.Queue[1:]
		
		if c.state.Visited[currentURL] {
			continue
		}
		
		c.processURL(currentURL)
		time.Sleep(c.config.Delay)
		
		// Save state periodically
		if c.state.Processed%10 == 0 {
			c.saveState()
		}
	}
}

func (c *Crawler) crawlConcurrent() {
	for len(c.state.Queue) > 0 {
		currentURL := c.state.Queue[0]
		c.state.Queue = c.state.Queue[1:]
		
		if c.state.Visited[currentURL] {
			continue
		}
		
		c.wg.Add(1)
		c.semaphore <- struct{}{} // Acquire semaphore
		
		go func(url string) {
			defer c.wg.Done()
			defer func() { <-c.semaphore }() // Release semaphore
			
			c.processURL(url)
			time.Sleep(c.config.Delay)
		}(currentURL)
		
		// Save state periodically
		if c.state.Processed%10 == 0 {
			c.wg.Wait()
			c.saveState()
		}
	}
	
	c.wg.Wait()
}

func (c *Crawler) processURL(rawURL string) {
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
	defer resp.Body.Close()
	
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
	
	// Extract and queue new URLs
	c.extractAndQueueURLs(rawURL, string(body))
}

func (c *Crawler) hasContent(html string) bool {
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
	// Create filename based on URL hash
	hash := fmt.Sprintf("%x", md5.Sum([]byte(rawURL)))
	filename := fmt.Sprintf("%s.html", hash)
	filePath := filepath.Join(c.config.OutputDir, filename)
	
	// Create metadata file
	metadata := map[string]interface{}{
		"url":       rawURL,
		"timestamp": time.Now().Unix(),
		"size":      len(content),
	}
	
	metaData, _ := json.MarshalIndent(metadata, "", "  ")
	metaFile := filepath.Join(c.config.OutputDir, fmt.Sprintf("%s.meta.json", hash))
	
	// Save both files
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return err
	}
	
	return os.WriteFile(metaFile, metaData, 0644)
}

func (c *Crawler) extractAndQueueURLs(baseURL, html string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return
	}
	
	base, err := url.Parse(baseURL)
	if err != nil {
		return
	}
	
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		
		absoluteURL, err := base.Parse(href)
		if err != nil {
			return
		}
		
		urlStr := absoluteURL.String()
		
		if c.isValidURL(urlStr) {
			c.mu.Lock()
			if !c.state.Visited[urlStr] {
				// Check if already in queue
				inQueue := false
				for _, queuedURL := range c.state.Queue {
					if queuedURL == urlStr {
						inQueue = true
						break
					}
				}
				if !inQueue {
					c.state.Queue = append(c.state.Queue, urlStr)
				}
			}
			c.mu.Unlock()
		}
	})
}