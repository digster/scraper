package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"scraper/internal/crawler"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct holds the Wails application state
type App struct {
	ctx     context.Context
	crawler *crawler.Crawler
	cancel  context.CancelFunc
	mu      sync.Mutex
	running bool
}

// NewApp creates a new App instance
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// Emit implements EventEmitter interface
func (a *App) Emit(event crawler.CrawlerEvent) {
	runtime.EventsEmit(a.ctx, string(event.Type), event)
}

// CrawlConfig is the configuration passed from the frontend
type CrawlConfig struct {
	URL                string `json:"url"`
	Concurrent         bool   `json:"concurrent"`
	Delay              string `json:"delay"`
	MaxDepth           int    `json:"maxDepth"`
	OutputDir          string `json:"outputDir"`
	StateFile          string `json:"stateFile"`
	PrefixFilterURL    string `json:"prefixFilter"`
	ExcludeExtensions  string `json:"excludeExtensions"`
	LinkSelectors      string `json:"linkSelectors"`
	Verbose            bool   `json:"verbose"`
	UserAgent          string `json:"userAgent"`
	IgnoreRobots       bool   `json:"ignoreRobots"`
	MinContentLength   int    `json:"minContent"`
	DisableReadability bool   `json:"disableReadability"`
	FetchMode          string `json:"fetchMode"`
	Headless           bool   `json:"headless"`
	WaitForLogin       bool   `json:"waitForLogin"`
	// Pagination settings
	EnablePagination          bool   `json:"enablePagination"`
	PaginationSelector        string `json:"paginationSelector"`
	MaxPaginationClicks       int    `json:"maxPaginationClicks"`
	PaginationWait            string `json:"paginationWait"`
	PaginationWaitSelector    string `json:"paginationWaitSelector"`
	PaginationStopOnDuplicate bool   `json:"paginationStopOnDuplicate"`
	// Anti-bot settings
	HideWebdriver        bool   `json:"hideWebdriver"`
	SpoofPlugins         bool   `json:"spoofPlugins"`
	SpoofLanguages       bool   `json:"spoofLanguages"`
	SpoofWebGL           bool   `json:"spoofWebGL"`
	AddCanvasNoise       bool   `json:"addCanvasNoise"`
	NaturalMouseMovement bool   `json:"naturalMouseMovement"`
	RandomTypingDelays   bool   `json:"randomTypingDelays"`
	NaturalScrolling     bool   `json:"naturalScrolling"`
	RandomActionDelays   bool   `json:"randomActionDelays"`
	RandomClickOffset    bool   `json:"randomClickOffset"`
	RotateUserAgent      bool   `json:"rotateUserAgent"`
	RandomViewport       bool   `json:"randomViewport"`
	MatchTimezone        bool   `json:"matchTimezone"`
	Timezone             string `json:"timezone"`
	// URL normalization settings
	NormalizeURLs  bool `json:"normalizeUrls"`
	LowercasePaths bool `json:"lowercasePaths"`
}

// StartCrawl starts the crawler with the given configuration
func (a *App) StartCrawl(cfg CrawlConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("crawler is already running")
	}

	// Parse delay duration
	delay, err := time.ParseDuration(cfg.Delay)
	if err != nil {
		delay = time.Second // Default to 1 second
	}

	// Determine fetch mode
	fetchMode := crawler.FetchModeHTTP
	if cfg.FetchMode == "browser" {
		fetchMode = crawler.FetchModeBrowser
	}

	// Build anti-bot config
	antiBotConfig := crawler.AntiBotConfig{
		HideWebdriver:        cfg.HideWebdriver,
		SpoofPlugins:         cfg.SpoofPlugins,
		SpoofLanguages:       cfg.SpoofLanguages,
		SpoofWebGL:           cfg.SpoofWebGL,
		AddCanvasNoise:       cfg.AddCanvasNoise,
		NaturalMouseMovement: cfg.NaturalMouseMovement,
		RandomTypingDelays:   cfg.RandomTypingDelays,
		NaturalScrolling:     cfg.NaturalScrolling,
		RandomActionDelays:   cfg.RandomActionDelays,
		RandomClickOffset:    cfg.RandomClickOffset,
		RotateUserAgent:      cfg.RotateUserAgent,
		RandomViewport:       cfg.RandomViewport,
		MatchTimezone:        cfg.MatchTimezone,
		Timezone:             cfg.Timezone,
	}

	// Build pagination config if enabled
	var paginationConfig crawler.PaginationConfig
	if cfg.EnablePagination {
		paginationWait, err := time.ParseDuration(cfg.PaginationWait)
		if err != nil {
			paginationWait = 2 * time.Second // Default to 2 seconds
		}
		paginationConfig = crawler.PaginationConfig{
			Enable:          true,
			Selector:        cfg.PaginationSelector,
			MaxClicks:       cfg.MaxPaginationClicks,
			WaitAfterClick:  paginationWait,
			WaitSelector:    cfg.PaginationWaitSelector,
			StopOnDuplicate: cfg.PaginationStopOnDuplicate,
		}
		// Set defaults if not specified
		if paginationConfig.MaxClicks <= 0 {
			paginationConfig.MaxClicks = 100
		}
	}

	// Build config
	config := crawler.Config{
		URL:                cfg.URL,
		Concurrent:         cfg.Concurrent,
		Delay:              delay,
		MaxDepth:           cfg.MaxDepth,
		OutputDir:          cfg.OutputDir,
		StateFile:          cfg.StateFile,
		PrefixFilterURL:    cfg.PrefixFilterURL,
		Verbose:            cfg.Verbose,
		UserAgent:          cfg.UserAgent,
		IgnoreRobots:       cfg.IgnoreRobots,
		MinContentLength:   cfg.MinContentLength,
		ShowProgress:       false, // GUI handles progress display
		DisableReadability: cfg.DisableReadability,
		FetchMode:          fetchMode,
		Headless:           cfg.Headless,
		WaitForLogin:       cfg.WaitForLogin,
		AntiBot:            antiBotConfig,
		Pagination:         paginationConfig,
		NormalizeURLs:      cfg.NormalizeURLs,
		LowercasePaths:     cfg.LowercasePaths,
	}

	// Parse exclude extensions
	if cfg.ExcludeExtensions != "" {
		exts := splitAndTrim(cfg.ExcludeExtensions, ",")
		config.ExcludeExtensions = exts
	}

	// Parse link selectors
	if cfg.LinkSelectors != "" {
		selectors := splitAndTrim(cfg.LinkSelectors, ",")
		config.LinkSelectors = selectors
	}

	// Set defaults for optional fields (but not MaxDepth - let validation catch invalid values)
	if config.MinContentLength == 0 {
		config.MinContentLength = 100
	}

	// Validate config
	if err := crawler.ValidateConfig(&config); err != nil {
		return err
	}

	// Set default output directory
	if err := crawler.SetDefaultOutputDir(&config); err != nil {
		return err
	}

	// Set default state file
	crawler.SetDefaultStateFile(&config)

	// Create context for this crawl
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	// Create crawler with event emitter
	c, err := crawler.NewCrawlerWithEmitter(config, ctx, a)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create crawler: %w", err)
	}
	a.crawler = c
	a.running = true

	// Start crawling in background
	go func() {
		defer func() {
			a.mu.Lock()
			a.crawler.Close()
			a.running = false
			a.crawler = nil
			a.mu.Unlock()
		}()

		if err := a.crawler.Start(); err != nil {
			crawler.EmitError(a, err.Error())
		}
	}()

	return nil
}

// StopCrawl stops the current crawl
func (a *App) StopCrawl() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running || a.crawler == nil {
		return fmt.Errorf("no crawler running")
	}

	a.crawler.Stop()
	return nil
}

// PauseCrawl pauses the current crawl
func (a *App) PauseCrawl() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running || a.crawler == nil {
		return fmt.Errorf("no crawler running")
	}

	a.crawler.Pause()
	return nil
}

// ResumeCrawl resumes a paused crawl
func (a *App) ResumeCrawl() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running || a.crawler == nil {
		return fmt.Errorf("no crawler running")
	}

	a.crawler.Resume()
	return nil
}

// GetStatus returns the current crawler status
type CrawlerStatus struct {
	Running         bool   `json:"running"`
	Paused          bool   `json:"paused"`
	WaitingForLogin bool   `json:"waitingForLogin"`
	Status          string `json:"status"`
}

func (a *App) GetStatus() CrawlerStatus {
	a.mu.Lock()
	defer a.mu.Unlock()

	status := CrawlerStatus{
		Running: a.running,
		Status:  "stopped",
	}

	if a.running && a.crawler != nil {
		status.Paused = a.crawler.IsPaused()
		status.WaitingForLogin = a.crawler.IsWaitingForLogin()
		if status.WaitingForLogin {
			status.Status = "waiting_for_login"
		} else if status.Paused {
			status.Status = "paused"
		} else {
			status.Status = "running"
		}
	}

	return status
}

// ConfirmLogin signals that manual login is complete
func (a *App) ConfirmLogin() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.crawler == nil {
		return fmt.Errorf("no crawler running")
	}

	if !a.crawler.IsWaitingForLogin() {
		return fmt.Errorf("crawler is not waiting for login")
	}

	a.crawler.ConfirmLogin()
	return nil
}

// MetricsSnapshot is the metrics data sent to the frontend
type MetricsSnapshot struct {
	URLsProcessed   int64   `json:"urlsProcessed"`
	URLsSaved       int64   `json:"urlsSaved"`
	URLsSkipped     int64   `json:"urlsSkipped"`
	URLsErrored     int64   `json:"urlsErrored"`
	BytesDownloaded int64   `json:"bytesDownloaded"`
	RobotsBlocked   int64   `json:"robotsBlocked"`
	DepthLimitHits  int64   `json:"depthLimitHits"`
	ContentFiltered int64   `json:"contentFiltered"`
	PagesPerSecond  float64 `json:"pagesPerSecond"`
	QueueSize       int     `json:"queueSize"`
}

// GetMetrics returns current crawler metrics
func (a *App) GetMetrics() (*MetricsSnapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.crawler == nil {
		return nil, fmt.Errorf("no crawler instance")
	}

	m := a.crawler.GetMetrics()
	snapshot := m.GetSnapshot()

	return &MetricsSnapshot{
		URLsProcessed:   snapshot.URLsProcessed,
		URLsSaved:       snapshot.URLsSaved,
		URLsSkipped:     snapshot.URLsSkipped,
		URLsErrored:     snapshot.URLsErrored,
		BytesDownloaded: snapshot.BytesDownloaded,
		RobotsBlocked:   snapshot.RobotsBlocked,
		DepthLimitHits:  snapshot.DepthLimitHits,
		ContentFiltered: snapshot.ContentFiltered,
		PagesPerSecond:  snapshot.PagesPerSecond,
		QueueSize:       snapshot.QueueSize,
	}, nil
}

// BrowseDirectory opens a directory picker dialog
func (a *App) BrowseDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Output Directory",
	})
}

// BrowseFile opens a file picker dialog for state files
func (a *App) BrowseFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select State File",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
}

// Helper function to split and trim strings
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range splitString(s, sep) {
		trimmed := trimString(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	result := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimString(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// PresetConfig contains all saveable configuration fields (excludes outputDir, stateFile)
type PresetConfig struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	// Core settings
	URL             string `json:"url"`
	Concurrent      bool   `json:"concurrent"`
	Delay           string `json:"delay"`
	MaxDepth        int    `json:"maxDepth"`
	PrefixFilterURL string `json:"prefixFilter"`
	// Content settings
	ExcludeExtensions  string `json:"excludeExtensions"`
	LinkSelectors      string `json:"linkSelectors"`
	Verbose            bool   `json:"verbose"`
	UserAgent          string `json:"userAgent"`
	IgnoreRobots       bool   `json:"ignoreRobots"`
	MinContentLength   int    `json:"minContent"`
	DisableReadability bool   `json:"disableReadability"`
	// Browser settings
	FetchMode    string `json:"fetchMode"`
	Headless     bool   `json:"headless"`
	WaitForLogin bool   `json:"waitForLogin"`
	// Pagination settings
	EnablePagination          bool   `json:"enablePagination"`
	PaginationSelector        string `json:"paginationSelector"`
	MaxPaginationClicks       int    `json:"maxPaginationClicks"`
	PaginationWait            string `json:"paginationWait"`
	PaginationWaitSelector    string `json:"paginationWaitSelector"`
	PaginationStopOnDuplicate bool   `json:"paginationStopOnDuplicate"`
	// Anti-bot settings
	HideWebdriver        bool   `json:"hideWebdriver"`
	SpoofPlugins         bool   `json:"spoofPlugins"`
	SpoofLanguages       bool   `json:"spoofLanguages"`
	SpoofWebGL           bool   `json:"spoofWebGL"`
	AddCanvasNoise       bool   `json:"addCanvasNoise"`
	NaturalMouseMovement bool   `json:"naturalMouseMovement"`
	RandomTypingDelays   bool   `json:"randomTypingDelays"`
	NaturalScrolling     bool   `json:"naturalScrolling"`
	RandomActionDelays   bool   `json:"randomActionDelays"`
	RandomClickOffset    bool   `json:"randomClickOffset"`
	RotateUserAgent      bool   `json:"rotateUserAgent"`
	RandomViewport       bool   `json:"randomViewport"`
	MatchTimezone        bool   `json:"matchTimezone"`
	Timezone             string `json:"timezone"`
	// URL normalization settings
	NormalizeURLs  bool `json:"normalizeUrls"`
	LowercasePaths bool `json:"lowercasePaths"`
}

// PresetInfo contains lightweight metadata for listing presets
type PresetInfo struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// validPresetName validates preset names (alphanumeric, dashes, underscores only)
var validPresetName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// GetPresetsDir returns the presets directory path, creating it if needed
func (a *App) GetPresetsDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}

	presetsDir := filepath.Join(configDir, "scraper", "presets")
	if err := os.MkdirAll(presetsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create presets dir: %w", err)
	}

	return presetsDir, nil
}

// ListPresets returns a list of all available presets
func (a *App) ListPresets() ([]PresetInfo, error) {
	presetsDir, err := a.GetPresetsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(presetsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read presets dir: %w", err)
	}

	presets := make([]PresetInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		name := entry.Name()[:len(entry.Name())-5] // Remove .json extension

		// Read the preset to get CreatedAt
		preset, err := a.LoadPreset(name)
		if err != nil {
			continue // Skip invalid presets
		}

		presets = append(presets, PresetInfo{
			Name:      name,
			CreatedAt: preset.CreatedAt,
		})
	}

	// Sort by creation time (newest first)
	sort.Slice(presets, func(i, j int) bool {
		return presets[i].CreatedAt.After(presets[j].CreatedAt)
	})

	return presets, nil
}

// SavePreset saves a configuration preset with the given name
func (a *App) SavePreset(name string, config PresetConfig) error {
	// Validate preset name
	if name == "" {
		return fmt.Errorf("preset name cannot be empty")
	}
	if len(name) > 50 {
		return fmt.Errorf("preset name too long (max 50 characters)")
	}
	if !validPresetName.MatchString(name) {
		return fmt.Errorf("preset name can only contain letters, numbers, dashes, and underscores")
	}

	presetsDir, err := a.GetPresetsDir()
	if err != nil {
		return err
	}

	// Set metadata
	config.Name = name
	config.CreatedAt = time.Now()

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal preset: %w", err)
	}

	// Write to file
	filePath := filepath.Join(presetsDir, name+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write preset file: %w", err)
	}

	return nil
}

// LoadPreset loads a configuration preset by name
func (a *App) LoadPreset(name string) (*PresetConfig, error) {
	// Validate name to prevent path traversal
	if !validPresetName.MatchString(name) {
		return nil, fmt.Errorf("invalid preset name")
	}

	presetsDir, err := a.GetPresetsDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(presetsDir, name+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("preset '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to read preset file: %w", err)
	}

	var config PresetConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse preset file: %w", err)
	}

	return &config, nil
}

// DeletePreset deletes a preset by name
func (a *App) DeletePreset(name string) error {
	// Validate name to prevent path traversal
	if !validPresetName.MatchString(name) {
		return fmt.Errorf("invalid preset name")
	}

	presetsDir, err := a.GetPresetsDir()
	if err != nil {
		return err
	}

	filePath := filepath.Join(presetsDir, name+".json")
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("preset '%s' not found", name)
		}
		return fmt.Errorf("failed to delete preset: %w", err)
	}

	return nil
}
