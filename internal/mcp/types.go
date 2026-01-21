// Package mcp provides an MCP (Model Context Protocol) server for the web scraper.
// It exposes scraper functionality as tools for LLM agents to use.
package mcp

import "time"

// StartCrawlInput is the input for scraper_start tool
type StartCrawlInput struct {
	URL               string           `json:"url" jsonschema:"required,description=Target URL to start crawling from"`
	MaxDepth          int              `json:"maxDepth,omitempty" jsonschema:"description=Maximum link depth to crawl (default: 10)"`
	Concurrent        bool             `json:"concurrent,omitempty" jsonschema:"description=Enable concurrent crawling for faster processing"`
	Delay             string           `json:"delay,omitempty" jsonschema:"description=Delay between requests (e.g. '500ms' or '1s')"`
	OutputDir         string           `json:"outputDir,omitempty" jsonschema:"description=Directory to save crawled content"`
	PrefixFilter      string           `json:"prefixFilter,omitempty" jsonschema:"description=Only crawl URLs starting with this prefix"`
	FetchMode         string           `json:"fetchMode,omitempty" jsonschema:"enum=http,enum=browser,description=Fetch mode: 'http' for fast requests or 'browser' for JavaScript-rendered pages"`
	Headless          *bool            `json:"headless,omitempty" jsonschema:"description=Run browser in headless mode (default: true)"`
	WaitForLogin      bool             `json:"waitForLogin,omitempty" jsonschema:"description=Wait for manual login before starting crawl (browser mode only)"`
	UserAgent         string           `json:"userAgent,omitempty" jsonschema:"description=Custom User-Agent string"`
	IgnoreRobots      bool             `json:"ignoreRobots,omitempty" jsonschema:"description=Ignore robots.txt restrictions"`
	MinContentLength  int              `json:"minContent,omitempty" jsonschema:"description=Minimum content length to save a page (default: 100)"`
	ExcludeExtensions []string         `json:"excludeExtensions,omitempty" jsonschema:"description=File extensions to exclude (e.g. ['.pdf', '.zip'])"`
	LinkSelectors     []string         `json:"linkSelectors,omitempty" jsonschema:"description=CSS selectors to find links (defaults to standard link tags)"`
	Pagination        *PaginationInput `json:"pagination,omitempty" jsonschema:"description=Click-based pagination settings (browser mode only)"`
	AntiBot           *AntiBotInput    `json:"antiBot,omitempty" jsonschema:"description=Anti-bot detection evasion settings (browser mode only)"`
}

// PaginationInput configures click-based pagination for browser mode
type PaginationInput struct {
	Enable          bool   `json:"enable,omitempty" jsonschema:"description=Enable click-based pagination"`
	Selector        string `json:"selector,omitempty" jsonschema:"description=CSS selector for pagination element (e.g. 'a.next' or '.load-more-btn')"`
	MaxClicks       int    `json:"maxClicks,omitempty" jsonschema:"description=Maximum pagination clicks per URL (default: 100)"`
	WaitAfterClick  string `json:"waitAfterClick,omitempty" jsonschema:"description=Time to wait after clicking (e.g. '2s')"`
	WaitSelector    string `json:"waitSelector,omitempty" jsonschema:"description=CSS selector to wait for after click (optional)"`
	StopOnDuplicate bool   `json:"stopOnDuplicate,omitempty" jsonschema:"description=Stop if duplicate content detected (default: true)"`
}

// AntiBotInput configures anti-bot detection measures
type AntiBotInput struct {
	// Browser Fingerprint Modifications
	HideWebdriver  bool `json:"hideWebdriver,omitempty" jsonschema:"description=Hide webdriver property to avoid detection"`
	SpoofPlugins   bool `json:"spoofPlugins,omitempty" jsonschema:"description=Spoof browser plugins"`
	SpoofLanguages bool `json:"spoofLanguages,omitempty" jsonschema:"description=Spoof navigator.languages"`
	SpoofWebGL     bool `json:"spoofWebGL,omitempty" jsonschema:"description=Spoof WebGL renderer/vendor"`
	AddCanvasNoise bool `json:"addCanvasNoise,omitempty" jsonschema:"description=Add noise to canvas fingerprints"`

	// Human Behavior Simulation
	NaturalMouseMovement bool `json:"naturalMouseMovement,omitempty" jsonschema:"description=Simulate natural mouse movements"`
	RandomTypingDelays   bool `json:"randomTypingDelays,omitempty" jsonschema:"description=Add random delays between keystrokes"`
	NaturalScrolling     bool `json:"naturalScrolling,omitempty" jsonschema:"description=Simulate natural scroll behavior"`
	RandomActionDelays   bool `json:"randomActionDelays,omitempty" jsonschema:"description=Add random delays between actions"`
	RandomClickOffset    bool `json:"randomClickOffset,omitempty" jsonschema:"description=Add slight randomness to click coordinates"`

	// Browser Properties
	RotateUserAgent bool   `json:"rotateUserAgent,omitempty" jsonschema:"description=Rotate user agent strings"`
	RandomViewport  bool   `json:"randomViewport,omitempty" jsonschema:"description=Use random viewport sizes"`
	MatchTimezone   bool   `json:"matchTimezone,omitempty" jsonschema:"description=Match timezone to IP location"`
	Timezone        string `json:"timezone,omitempty" jsonschema:"description=Specific timezone to use (e.g. 'America/New_York')"`
}

// JobIDInput is input for tools that operate on a specific job
type JobIDInput struct {
	JobID string `json:"jobId" jsonschema:"required,description=Job ID returned from scraper_start"`
}

// WaitInput is input for scraper_wait tool
type WaitInput struct {
	JobID          string `json:"jobId" jsonschema:"required,description=Job ID to wait for"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty" jsonschema:"description=Maximum seconds to wait (default: 300)"`
	PollInterval   int    `json:"pollIntervalMs,omitempty" jsonschema:"description=Polling interval in milliseconds (default: 2000)"`
}

// StartCrawlOutput is the response from scraper_start
type StartCrawlOutput struct {
	JobID     string `json:"jobId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	OutputDir string `json:"outputDir,omitempty"`
}

// JobListOutput is the response from scraper_list
type JobListOutput struct {
	Jobs  []JobSummary `json:"jobs"`
	Total int          `json:"total"`
}

// JobSummary provides a brief overview of a job
type JobSummary struct {
	JobID     string    `json:"jobId"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// JobDetailsOutput is the response from scraper_get
type JobDetailsOutput struct {
	JobID           string           `json:"jobId"`
	URL             string           `json:"url"`
	Status          string           `json:"status"`
	CreatedAt       time.Time        `json:"createdAt"`
	StartedAt       *time.Time       `json:"startedAt,omitempty"`
	CompletedAt     *time.Time       `json:"completedAt,omitempty"`
	Metrics         *MetricsSnapshot `json:"metrics,omitempty"`
	WaitingForLogin bool             `json:"waitingForLogin,omitempty"`
	OutputDir       string           `json:"outputDir,omitempty"`
	Error           string           `json:"error,omitempty"`
}

// MetricsSnapshot represents crawl progress metrics
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
	ElapsedTime     string  `json:"elapsedTime,omitempty"`
	Percentage      float64 `json:"percentage,omitempty"`
	CurrentURL      string  `json:"currentUrl,omitempty"`
}

// MetricsOutput is the response from scraper_metrics
type MetricsOutput struct {
	JobID   string           `json:"jobId"`
	Status  string           `json:"status"`
	Metrics *MetricsSnapshot `json:"metrics,omitempty"`
}

// StatusOutput is a generic status response
type StatusOutput struct {
	JobID   string `json:"jobId"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// WaitOutput is the response from scraper_wait
type WaitOutput struct {
	JobID         string           `json:"jobId"`
	Status        string           `json:"status"`
	FinalMetrics  *MetricsSnapshot `json:"finalMetrics,omitempty"`
	OutputDir     string           `json:"outputDir,omitempty"`
	Error         string           `json:"error,omitempty"`
	WaitedSeconds int              `json:"waitedSeconds"`
}

// ErrorOutput represents an error response
type ErrorOutput struct {
	Error   string `json:"error"`
	Code    int    `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
