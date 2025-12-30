package main

import (
	"encoding/json"
	"os"
)

// URLInfo represents a URL with its discovery depth
type URLInfo struct {
	URL   string `json:"url"`
	Depth int    `json:"depth"`
}

// CrawlerState tracks the current state of the crawler for persistence and resumption
type CrawlerState struct {
	Visited   map[string]bool `json:"visited"`
	Queue     []URLInfo       `json:"queue"`
	BaseURL   string          `json:"base_url"`
	Processed int             `json:"processed"`
	URLDepths map[string]int  `json:"url_depths"`
	Queued    map[string]bool `json:"queued"`
}

// loadState loads crawler state from a file or initializes a fresh state
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

	// Initialize Queued map from Queue if empty (backward compatibility with old state files)
	if len(c.state.Queued) == 0 && len(c.state.Queue) > 0 {
		for _, urlInfo := range c.state.Queue {
			c.state.Queued[urlInfo.URL] = true
		}
	}

	return nil
}

// saveState persists the current crawler state to a file
func (c *Crawler) saveState() error {
	data, err := json.MarshalIndent(c.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.config.StateFile, data, 0644)
}
