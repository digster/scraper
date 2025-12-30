package crawler

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

// NewCrawlerState creates a new empty crawler state
func NewCrawlerState(baseURL string) *CrawlerState {
	return &CrawlerState{
		Visited:   make(map[string]bool),
		Queue:     []URLInfo{},
		BaseURL:   baseURL,
		URLDepths: make(map[string]int),
		Queued:    make(map[string]bool),
	}
}

// LoadState loads crawler state from a file or returns a fresh state
func LoadState(stateFile string, baseURL string) (*CrawlerState, error) {
	state := NewCrawlerState(baseURL)

	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return state, nil // No existing state file
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, state); err != nil {
		return nil, err
	}

	// Initialize Queued map from Queue if empty (backward compatibility with old state files)
	if len(state.Queued) == 0 && len(state.Queue) > 0 {
		for _, urlInfo := range state.Queue {
			state.Queued[urlInfo.URL] = true
		}
	}

	return state, nil
}

// SaveState persists the current crawler state to a file
func SaveState(state *CrawlerState, stateFile string) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0644)
}
