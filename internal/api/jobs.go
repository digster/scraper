package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"scraper/internal/crawler"

	"github.com/google/uuid"
)

// CrawlJob represents a single crawl job with its state
type CrawlJob struct {
	ID          string
	Crawler     *crawler.Crawler
	Emitter     *SSEEmitter
	Config      *CrawlRequest
	Status      JobStatus
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Error       error
	cancel      context.CancelFunc
	mu          sync.Mutex
}

// GetStatus returns the current job status (thread-safe)
func (j *CrawlJob) GetStatus() JobStatus {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.Status
}

// SetStatus updates the job status (thread-safe)
func (j *CrawlJob) SetStatus(status JobStatus) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = status
}

// GetMetrics returns current metrics snapshot
func (j *CrawlJob) GetMetrics() *MetricsSnapshot {
	if j.Crawler == nil {
		return nil
	}

	m := j.Crawler.GetMetrics()
	if m == nil {
		return nil
	}

	snapshot := m.GetSnapshot()
	elapsed := time.Since(m.StartTime)

	total := snapshot.URLsProcessed + int64(snapshot.QueueSize)
	var percentage float64
	if total > 0 {
		percentage = float64(snapshot.URLsProcessed) / float64(total) * 100
	}

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
		ElapsedTime:     crawler.FormatDuration(elapsed),
		Percentage:      percentage,
	}
}

// ToSummary converts job to a summary view
func (j *CrawlJob) ToSummary() JobSummary {
	j.mu.Lock()
	defer j.mu.Unlock()

	return JobSummary{
		JobID:     j.ID,
		URL:       j.Config.URL,
		Status:    j.Status,
		CreatedAt: j.CreatedAt,
	}
}

// ToDetails converts job to a detailed view
func (j *CrawlJob) ToDetails() JobDetails {
	j.mu.Lock()
	defer j.mu.Unlock()

	waitingForLogin := false
	if j.Crawler != nil {
		waitingForLogin = j.Crawler.IsWaitingForLogin()
	}

	details := JobDetails{
		JobID:           j.ID,
		URL:             j.Config.URL,
		Status:          j.Status,
		CreatedAt:       j.CreatedAt,
		StartedAt:       j.StartedAt,
		CompletedAt:     j.CompletedAt,
		Config:          j.Config,
		WaitingForLogin: waitingForLogin,
	}

	// Add metrics if crawler exists
	j.mu.Unlock()
	metrics := j.GetMetrics()
	j.mu.Lock()
	details.Metrics = metrics

	return details
}

// JobManager manages multiple concurrent crawl jobs
type JobManager struct {
	jobs          map[string]*CrawlJob
	maxConcurrent int
	mu            sync.RWMutex
}

// NewJobManager creates a new job manager
func NewJobManager(maxConcurrent int) *JobManager {
	return &JobManager{
		jobs:          make(map[string]*CrawlJob),
		maxConcurrent: maxConcurrent,
	}
}

// CreateJob creates a new crawl job from the request
func (m *JobManager) CreateJob(req *CrawlRequest) (*CrawlJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check concurrent job limit
	activeCount := 0
	for _, job := range m.jobs {
		if job.GetStatus() == JobStatusRunning || job.GetStatus() == JobStatusPending || job.GetStatus() == JobStatusWaitingForLogin {
			activeCount++
		}
	}
	if activeCount >= m.maxConcurrent {
		return nil, APIError{
			Code:    429,
			Message: "too many active jobs",
			Details: fmt.Sprintf("maximum %d concurrent jobs allowed", m.maxConcurrent),
		}
	}

	// Generate job ID
	jobID := uuid.New().String()[:8]

	// Create emitter for this job
	emitter := NewSSEEmitter()

	job := &CrawlJob{
		ID:        jobID,
		Emitter:   emitter,
		Config:    req,
		Status:    JobStatusPending,
		CreatedAt: time.Now(),
	}

	m.jobs[jobID] = job
	return job, nil
}

// StartJob starts a crawl job
func (m *JobManager) StartJob(jobID string) error {
	m.mu.RLock()
	job, exists := m.jobs[jobID]
	m.mu.RUnlock()

	if !exists {
		return APIError{Code: 404, Message: "job not found"}
	}

	job.mu.Lock()
	if job.Status != JobStatusPending {
		job.mu.Unlock()
		return APIError{Code: 400, Message: "job already started"}
	}

	// Convert API config to crawler config
	crawlerConfig, err := translateConfig(job.Config)
	if err != nil {
		job.mu.Unlock()
		return err
	}

	// Create context for this job
	ctx, cancel := context.WithCancel(context.Background())
	job.cancel = cancel

	// Create crawler with emitter
	c, err := crawler.NewCrawlerWithEmitter(*crawlerConfig, ctx, job.Emitter)
	if err != nil {
		cancel()
		job.mu.Unlock()
		return APIError{Code: 500, Message: "failed to create crawler", Details: err.Error()}
	}

	job.Crawler = c
	job.Status = JobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	job.mu.Unlock()

	// Start crawling in background
	go func() {
		err := c.Start()

		job.mu.Lock()
		job.Crawler.Close()
		now := time.Now()
		job.CompletedAt = &now

		if err != nil {
			job.Status = JobStatusError
			job.Error = err
		} else {
			job.Status = JobStatusCompleted
		}
		job.mu.Unlock()

		// Close the emitter to signal completion to SSE clients
		job.Emitter.Close()
	}()

	return nil
}

// GetJob returns a job by ID
func (m *JobManager) GetJob(jobID string) (*CrawlJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return nil, APIError{Code: 404, Message: "job not found"}
	}

	return job, nil
}

// ListJobs returns all jobs
func (m *JobManager) ListJobs() []*CrawlJob {
	m.mu.RLock()
	defer m.mu.RUnlock()

	jobs := make([]*CrawlJob, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}

// StopJob stops a running job
func (m *JobManager) StopJob(jobID string) error {
	m.mu.RLock()
	job, exists := m.jobs[jobID]
	m.mu.RUnlock()

	if !exists {
		return APIError{Code: 404, Message: "job not found"}
	}

	job.mu.Lock()
	defer job.mu.Unlock()

	if job.Status != JobStatusRunning && job.Status != JobStatusPaused && job.Status != JobStatusWaitingForLogin {
		return APIError{Code: 400, Message: "job is not active"}
	}

	if job.Crawler != nil {
		job.Crawler.Stop()
	}
	if job.cancel != nil {
		job.cancel()
	}
	job.Status = JobStatusStopped

	return nil
}

// PauseJob pauses a running job
func (m *JobManager) PauseJob(jobID string) error {
	m.mu.RLock()
	job, exists := m.jobs[jobID]
	m.mu.RUnlock()

	if !exists {
		return APIError{Code: 404, Message: "job not found"}
	}

	job.mu.Lock()
	defer job.mu.Unlock()

	if job.Status != JobStatusRunning {
		return APIError{Code: 400, Message: "job is not running"}
	}

	if job.Crawler != nil {
		job.Crawler.Pause()
		job.Status = JobStatusPaused
	}

	return nil
}

// ResumeJob resumes a paused job
func (m *JobManager) ResumeJob(jobID string) error {
	m.mu.RLock()
	job, exists := m.jobs[jobID]
	m.mu.RUnlock()

	if !exists {
		return APIError{Code: 404, Message: "job not found"}
	}

	job.mu.Lock()
	defer job.mu.Unlock()

	if job.Status != JobStatusPaused {
		return APIError{Code: 400, Message: "job is not paused"}
	}

	if job.Crawler != nil {
		job.Crawler.Resume()
		job.Status = JobStatusRunning
	}

	return nil
}

// ConfirmLogin confirms manual login for a waiting job
func (m *JobManager) ConfirmLogin(jobID string) error {
	m.mu.RLock()
	job, exists := m.jobs[jobID]
	m.mu.RUnlock()

	if !exists {
		return APIError{Code: 404, Message: "job not found"}
	}

	if job.Crawler == nil {
		return APIError{Code: 400, Message: "crawler not initialized"}
	}

	if !job.Crawler.IsWaitingForLogin() {
		return APIError{Code: 400, Message: "job is not waiting for login"}
	}

	job.Crawler.ConfirmLogin()
	return nil
}

// DeleteJob removes a job (must be stopped or completed)
func (m *JobManager) DeleteJob(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return APIError{Code: 404, Message: "job not found"}
	}

	status := job.GetStatus()
	if status == JobStatusRunning || status == JobStatusPending || status == JobStatusWaitingForLogin {
		return APIError{Code: 400, Message: "cannot delete active job", Details: "stop the job first"}
	}

	// Cleanup
	job.Emitter.Close()
	delete(m.jobs, jobID)

	return nil
}

// ActiveJobCount returns the number of active jobs
func (m *JobManager) ActiveJobCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, job := range m.jobs {
		status := job.GetStatus()
		if status == JobStatusRunning || status == JobStatusPending || status == JobStatusPaused || status == JobStatusWaitingForLogin {
			count++
		}
	}
	return count
}

// Shutdown stops all jobs and cleans up
func (m *JobManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, job := range m.jobs {
		job.mu.Lock()
		if job.Crawler != nil {
			job.Crawler.Stop()
			job.Crawler.Close()
		}
		if job.cancel != nil {
			job.cancel()
		}
		job.Emitter.Close()
		job.mu.Unlock()
	}
}

// translateConfig converts API CrawlRequest to crawler.Config
func translateConfig(req *CrawlRequest) (*crawler.Config, error) {
	// Parse delay duration
	delay := time.Second // default
	if req.Delay != "" {
		d, err := time.ParseDuration(req.Delay)
		if err != nil {
			return nil, APIError{Code: 400, Message: "invalid delay format", Details: err.Error()}
		}
		delay = d
	}

	// Determine fetch mode
	fetchMode := crawler.FetchModeHTTP
	if req.FetchMode == "browser" {
		fetchMode = crawler.FetchModeBrowser
	}

	// Default values
	maxDepth := req.MaxDepth
	if maxDepth == 0 {
		maxDepth = 10
	}

	minContent := req.MinContentLength
	if minContent == 0 {
		minContent = 100
	}

	headless := true
	if req.Headless != nil {
		headless = *req.Headless
	}

	// Parse page load wait duration
	var pageLoadWait time.Duration
	if req.PageLoadWait != "" {
		waitDuration, err := time.ParseDuration(req.PageLoadWait)
		if err != nil {
			return nil, APIError{Code: 400, Message: "invalid pageLoadWait format", Details: err.Error()}
		}
		pageLoadWait = waitDuration
	}

	// Build anti-bot config
	var antiBotConfig crawler.AntiBotConfig
	if req.AntiBot != nil {
		antiBotConfig = crawler.AntiBotConfig{
			HideWebdriver:        req.AntiBot.HideWebdriver,
			SpoofPlugins:         req.AntiBot.SpoofPlugins,
			SpoofLanguages:       req.AntiBot.SpoofLanguages,
			SpoofWebGL:           req.AntiBot.SpoofWebGL,
			AddCanvasNoise:       req.AntiBot.AddCanvasNoise,
			NaturalMouseMovement: req.AntiBot.NaturalMouseMovement,
			RandomTypingDelays:   req.AntiBot.RandomTypingDelays,
			NaturalScrolling:     req.AntiBot.NaturalScrolling,
			RandomActionDelays:   req.AntiBot.RandomActionDelays,
			RandomClickOffset:    req.AntiBot.RandomClickOffset,
			RotateUserAgent:      req.AntiBot.RotateUserAgent,
			RandomViewport:       req.AntiBot.RandomViewport,
			MatchTimezone:        req.AntiBot.MatchTimezone,
			Timezone:             req.AntiBot.Timezone,
		}
	}

	// URL normalization settings (default to true if not specified)
	normalizeURLs := true
	if req.NormalizeURLs != nil {
		normalizeURLs = *req.NormalizeURLs
	}

	config := &crawler.Config{
		URL:                req.URL,
		Concurrent:         req.Concurrent,
		Delay:              delay,
		MaxDepth:           maxDepth,
		OutputDir:          req.OutputDir,
		StateFile:          req.StateFile,
		PrefixFilterURL:    req.PrefixFilterURL,
		ExcludeExtensions:  req.ExcludeExtensions,
		LinkSelectors:      req.LinkSelectors,
		Verbose:            req.Verbose,
		UserAgent:          req.UserAgent,
		IgnoreRobots:       req.IgnoreRobots,
		MinContentLength:   minContent,
		ShowProgress:       false, // API doesn't need console progress
		DisableContentExtraction: req.DisableContentExtraction || req.DisableReadability,
		FetchMode:          fetchMode,
		Headless:           headless,
		WaitForLogin:       req.WaitForLogin,
		PageLoadWait:       pageLoadWait,
		AntiBot:            antiBotConfig,
		NormalizeURLs:      normalizeURLs,
		LowercasePaths:     req.LowercasePaths,
	}

	// Validate config
	if err := crawler.ValidateConfig(config); err != nil {
		return nil, APIError{Code: 400, Message: "invalid configuration", Details: err.Error()}
	}

	// Set default output directory
	if err := crawler.SetDefaultOutputDir(config); err != nil {
		return nil, APIError{Code: 500, Message: "failed to set output directory", Details: err.Error()}
	}

	// Set default state file
	crawler.SetDefaultStateFile(config)

	// Ensure output directory exists
	if err := crawler.EnsureOutputDir(config); err != nil {
		return nil, APIError{Code: 500, Message: "failed to create output directory", Details: err.Error()}
	}

	return config, nil
}
