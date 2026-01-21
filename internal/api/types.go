package api

import (
	"time"

	"scraper/internal/crawler"
)

// JobStatus represents the current state of a crawl job
type JobStatus string

const (
	JobStatusPending        JobStatus = "pending"
	JobStatusRunning        JobStatus = "running"
	JobStatusPaused         JobStatus = "paused"
	JobStatusCompleted      JobStatus = "completed"
	JobStatusStopped        JobStatus = "stopped"
	JobStatusError          JobStatus = "error"
	JobStatusWaitingForLogin JobStatus = "waiting_for_login"
)

// CrawlRequest represents the request body for starting a new crawl
type CrawlRequest struct {
	URL                string            `json:"url"`
	MaxDepth           int               `json:"maxDepth,omitempty"`
	Concurrent         bool              `json:"concurrent,omitempty"`
	Delay              string            `json:"delay,omitempty"`
	OutputDir          string            `json:"outputDir,omitempty"`
	StateFile          string            `json:"stateFile,omitempty"`
	PrefixFilterURL    string            `json:"prefixFilter,omitempty"`
	ExcludeExtensions  []string          `json:"excludeExtensions,omitempty"`
	LinkSelectors      []string          `json:"linkSelectors,omitempty"`
	Verbose            bool              `json:"verbose,omitempty"`
	UserAgent          string            `json:"userAgent,omitempty"`
	IgnoreRobots       bool              `json:"ignoreRobots,omitempty"`
	MinContentLength   int               `json:"minContent,omitempty"`
	DisableReadability bool              `json:"disableReadability,omitempty"`
	FetchMode          string            `json:"fetchMode,omitempty"`
	Headless           *bool             `json:"headless,omitempty"`
	WaitForLogin       bool              `json:"waitForLogin,omitempty"`
	PageLoadWait       string            `json:"pageLoadWait,omitempty"`
	Pagination         *PaginationConfig `json:"pagination,omitempty"`
	AntiBot            *AntiBotConfig    `json:"antiBot,omitempty"`
	// URL normalization settings
	NormalizeURLs  *bool `json:"normalizeUrls,omitempty"`
	LowercasePaths bool  `json:"lowercasePaths,omitempty"`
}

// PaginationConfig holds click-based pagination settings
type PaginationConfig struct {
	Enable          bool   `json:"enable,omitempty"`
	Selector        string `json:"selector,omitempty"`
	MaxClicks       int    `json:"maxClicks,omitempty"`
	WaitAfterClick  string `json:"waitAfterClick,omitempty"`
	WaitSelector    string `json:"waitSelector,omitempty"`
	StopOnDuplicate bool   `json:"stopOnDuplicate,omitempty"`
}

// AntiBotConfig mirrors crawler.AntiBotConfig for API requests
type AntiBotConfig struct {
	// Browser Fingerprint Modifications
	HideWebdriver  bool `json:"hideWebdriver,omitempty"`
	SpoofPlugins   bool `json:"spoofPlugins,omitempty"`
	SpoofLanguages bool `json:"spoofLanguages,omitempty"`
	SpoofWebGL     bool `json:"spoofWebGL,omitempty"`
	AddCanvasNoise bool `json:"addCanvasNoise,omitempty"`

	// Human Behavior Simulation
	NaturalMouseMovement bool `json:"naturalMouseMovement,omitempty"`
	RandomTypingDelays   bool `json:"randomTypingDelays,omitempty"`
	NaturalScrolling     bool `json:"naturalScrolling,omitempty"`
	RandomActionDelays   bool `json:"randomActionDelays,omitempty"`
	RandomClickOffset    bool `json:"randomClickOffset,omitempty"`

	// Browser Properties
	RotateUserAgent bool   `json:"rotateUserAgent,omitempty"`
	RandomViewport  bool   `json:"randomViewport,omitempty"`
	MatchTimezone   bool   `json:"matchTimezone,omitempty"`
	Timezone        string `json:"timezone,omitempty"`
}

// CrawlResponse is returned when a crawl job is created
type CrawlResponse struct {
	JobID     string    `json:"jobId"`
	Status    JobStatus `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// JobSummary provides a brief overview of a job (for listing)
type JobSummary struct {
	JobID     string    `json:"jobId"`
	URL       string    `json:"url"`
	Status    JobStatus `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// JobDetails provides full information about a job
type JobDetails struct {
	JobID           string           `json:"jobId"`
	URL             string           `json:"url"`
	Status          JobStatus        `json:"status"`
	CreatedAt       time.Time        `json:"createdAt"`
	StartedAt       *time.Time       `json:"startedAt,omitempty"`
	CompletedAt     *time.Time       `json:"completedAt,omitempty"`
	Config          *CrawlRequest    `json:"config,omitempty"`
	Metrics         *MetricsSnapshot `json:"metrics,omitempty"`
	WaitingForLogin bool             `json:"waitingForLogin,omitempty"`
}

// MetricsSnapshot represents a point-in-time snapshot of crawl metrics
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

// APIError represents a standardized error response
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e APIError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// FromCrawlerEvent converts a crawler event to an SSE event
func FromCrawlerEvent(event crawler.CrawlerEvent) SSEEvent {
	return SSEEvent{
		Type:      string(event.Type),
		Timestamp: event.Timestamp,
		Data:      event.Data,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	ActiveJobs int   `json:"activeJobs"`
}
