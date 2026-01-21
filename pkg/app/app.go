package app

import (
	"context"
	"fmt"
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
