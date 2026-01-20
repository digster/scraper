package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"scraper/internal/api"
)

// handleStart handles the scraper_start tool
func (s *Server) handleStart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Required: url
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return mcp.NewToolResultError("url is required"), nil
	}

	// Build CrawlRequest from arguments
	crawlReq := &api.CrawlRequest{
		URL: url,
	}

	// Optional parameters
	if maxDepth, ok := args["maxDepth"].(float64); ok {
		crawlReq.MaxDepth = int(maxDepth)
	}
	if concurrent, ok := args["concurrent"].(bool); ok {
		crawlReq.Concurrent = concurrent
	}
	if delay, ok := args["delay"].(string); ok {
		crawlReq.Delay = delay
	}
	if outputDir, ok := args["outputDir"].(string); ok {
		crawlReq.OutputDir = outputDir
	}
	if prefixFilter, ok := args["prefixFilter"].(string); ok {
		crawlReq.PrefixFilterURL = prefixFilter
	}
	if fetchMode, ok := args["fetchMode"].(string); ok {
		crawlReq.FetchMode = fetchMode
	}
	if headless, ok := args["headless"].(bool); ok {
		crawlReq.Headless = &headless
	}
	if waitForLogin, ok := args["waitForLogin"].(bool); ok {
		crawlReq.WaitForLogin = waitForLogin
	}
	if userAgent, ok := args["userAgent"].(string); ok {
		crawlReq.UserAgent = userAgent
	}
	if ignoreRobots, ok := args["ignoreRobots"].(bool); ok {
		crawlReq.IgnoreRobots = ignoreRobots
	}
	if minContent, ok := args["minContent"].(float64); ok {
		crawlReq.MinContentLength = int(minContent)
	}

	// Handle antiBot settings
	if antiBotRaw, ok := args["antiBot"].(map[string]interface{}); ok {
		crawlReq.AntiBot = parseAntiBotConfig(antiBotRaw)
	}

	// Create job
	job, err := s.jobManager.CreateJob(crawlReq)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Start job
	if err := s.jobManager.StartJob(job.ID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get output directory from job config
	outputDir := crawlReq.OutputDir
	if outputDir == "" {
		outputDir = "(auto-generated based on URL)"
	}

	output := StartCrawlOutput{
		JobID:     job.ID,
		Status:    string(job.GetStatus()),
		Message:   fmt.Sprintf("Crawl job started for %s", url),
		OutputDir: outputDir,
	}

	return resultJSON(output)
}

// handleList handles the scraper_list tool
func (s *Server) handleList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobs := s.jobManager.ListJobs()

	summaries := make([]JobSummary, 0, len(jobs))
	for _, job := range jobs {
		summary := job.ToSummary()
		summaries = append(summaries, JobSummary{
			JobID:     summary.JobID,
			URL:       summary.URL,
			Status:    string(summary.Status),
			CreatedAt: summary.CreatedAt,
		})
	}

	output := JobListOutput{
		Jobs:  summaries,
		Total: len(summaries),
	}

	return resultJSON(output)
}

// handleGet handles the scraper_get tool
func (s *Server) handleGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID, err := req.RequireString("jobId")
	if err != nil {
		return mcp.NewToolResultError("jobId is required"), nil
	}

	job, err := s.jobManager.GetJob(jobID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	details := job.ToDetails()
	output := JobDetailsOutput{
		JobID:           details.JobID,
		URL:             details.URL,
		Status:          string(details.Status),
		CreatedAt:       details.CreatedAt,
		StartedAt:       details.StartedAt,
		CompletedAt:     details.CompletedAt,
		WaitingForLogin: details.WaitingForLogin,
	}

	if details.Config != nil {
		output.OutputDir = details.Config.OutputDir
	}

	if details.Metrics != nil {
		output.Metrics = convertMetrics(details.Metrics)
	}

	if job.Error != nil {
		output.Error = job.Error.Error()
	}

	return resultJSON(output)
}

// handleStop handles the scraper_stop tool
func (s *Server) handleStop(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID, err := req.RequireString("jobId")
	if err != nil {
		return mcp.NewToolResultError("jobId is required"), nil
	}

	if err := s.jobManager.StopJob(jobID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output := StatusOutput{
		JobID:   jobID,
		Status:  "stopped",
		Message: "Job stopped successfully",
	}

	return resultJSON(output)
}

// handlePause handles the scraper_pause tool
func (s *Server) handlePause(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID, err := req.RequireString("jobId")
	if err != nil {
		return mcp.NewToolResultError("jobId is required"), nil
	}

	if err := s.jobManager.PauseJob(jobID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output := StatusOutput{
		JobID:   jobID,
		Status:  "paused",
		Message: "Job paused successfully",
	}

	return resultJSON(output)
}

// handleResume handles the scraper_resume tool
func (s *Server) handleResume(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID, err := req.RequireString("jobId")
	if err != nil {
		return mcp.NewToolResultError("jobId is required"), nil
	}

	if err := s.jobManager.ResumeJob(jobID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output := StatusOutput{
		JobID:   jobID,
		Status:  "running",
		Message: "Job resumed successfully",
	}

	return resultJSON(output)
}

// handleMetrics handles the scraper_metrics tool
func (s *Server) handleMetrics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID, err := req.RequireString("jobId")
	if err != nil {
		return mcp.NewToolResultError("jobId is required"), nil
	}

	job, err := s.jobManager.GetJob(jobID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output := MetricsOutput{
		JobID:  jobID,
		Status: string(job.GetStatus()),
	}

	metrics := job.GetMetrics()
	if metrics != nil {
		output.Metrics = convertMetrics(metrics)
	}

	return resultJSON(output)
}

// handleConfirmLogin handles the scraper_confirm_login tool
func (s *Server) handleConfirmLogin(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID, err := req.RequireString("jobId")
	if err != nil {
		return mcp.NewToolResultError("jobId is required"), nil
	}

	if err := s.jobManager.ConfirmLogin(jobID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output := StatusOutput{
		JobID:   jobID,
		Status:  "running",
		Message: "Login confirmed, crawl continuing",
	}

	return resultJSON(output)
}

// handleWait handles the scraper_wait tool
func (s *Server) handleWait(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID, err := req.RequireString("jobId")
	if err != nil {
		return mcp.NewToolResultError("jobId is required"), nil
	}

	// Parse optional parameters
	timeoutSeconds := 300 // default 5 minutes
	if timeout, ok := req.GetArguments()["timeoutSeconds"].(float64); ok {
		timeoutSeconds = int(timeout)
	}

	pollIntervalMs := 2000 // default 2 seconds
	if interval, ok := req.GetArguments()["pollIntervalMs"].(float64); ok {
		pollIntervalMs = int(interval)
	}

	pollInterval := time.Duration(pollIntervalMs) * time.Millisecond
	timeout := time.Duration(timeoutSeconds) * time.Second
	deadline := time.Now().Add(timeout)
	startTime := time.Now()

	// Poll until completion or timeout
	for {
		job, err := s.jobManager.GetJob(jobID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		status := job.GetStatus()

		// Check for terminal states
		if isTerminalStatus(status) {
			details := job.ToDetails()
			output := WaitOutput{
				JobID:         jobID,
				Status:        string(status),
				WaitedSeconds: int(time.Since(startTime).Seconds()),
			}

			if details.Config != nil {
				output.OutputDir = details.Config.OutputDir
			}

			if details.Metrics != nil {
				output.FinalMetrics = convertMetrics(details.Metrics)
			}

			if job.Error != nil {
				output.Error = job.Error.Error()
			}

			return resultJSON(output)
		}

		// Check timeout
		if time.Now().After(deadline) {
			output := WaitOutput{
				JobID:         jobID,
				Status:        string(status),
				Error:         "timeout waiting for job completion",
				WaitedSeconds: int(time.Since(startTime).Seconds()),
			}
			return resultJSON(output)
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			output := WaitOutput{
				JobID:         jobID,
				Status:        string(status),
				Error:         "wait cancelled",
				WaitedSeconds: int(time.Since(startTime).Seconds()),
			}
			return resultJSON(output)
		case <-time.After(pollInterval):
			// Continue polling
		}
	}
}

// Helper functions

// isTerminalStatus checks if a job status is terminal (completed, stopped, or error)
func isTerminalStatus(status api.JobStatus) bool {
	return status == api.JobStatusCompleted ||
		status == api.JobStatusStopped ||
		status == api.JobStatusError
}

// parseAntiBotConfig parses anti-bot settings from a map
func parseAntiBotConfig(raw map[string]interface{}) *api.AntiBotConfig {
	config := &api.AntiBotConfig{}

	// Browser Fingerprint Modifications
	if v, ok := raw["hideWebdriver"].(bool); ok {
		config.HideWebdriver = v
	}
	if v, ok := raw["spoofPlugins"].(bool); ok {
		config.SpoofPlugins = v
	}
	if v, ok := raw["spoofLanguages"].(bool); ok {
		config.SpoofLanguages = v
	}
	if v, ok := raw["spoofWebGL"].(bool); ok {
		config.SpoofWebGL = v
	}
	if v, ok := raw["addCanvasNoise"].(bool); ok {
		config.AddCanvasNoise = v
	}

	// Human Behavior Simulation
	if v, ok := raw["naturalMouseMovement"].(bool); ok {
		config.NaturalMouseMovement = v
	}
	if v, ok := raw["randomTypingDelays"].(bool); ok {
		config.RandomTypingDelays = v
	}
	if v, ok := raw["naturalScrolling"].(bool); ok {
		config.NaturalScrolling = v
	}
	if v, ok := raw["randomActionDelays"].(bool); ok {
		config.RandomActionDelays = v
	}
	if v, ok := raw["randomClickOffset"].(bool); ok {
		config.RandomClickOffset = v
	}

	// Browser Properties
	if v, ok := raw["rotateUserAgent"].(bool); ok {
		config.RotateUserAgent = v
	}
	if v, ok := raw["randomViewport"].(bool); ok {
		config.RandomViewport = v
	}
	if v, ok := raw["matchTimezone"].(bool); ok {
		config.MatchTimezone = v
	}
	if v, ok := raw["timezone"].(string); ok {
		config.Timezone = v
	}

	return config
}

// convertMetrics converts api.MetricsSnapshot to mcp.MetricsSnapshot
func convertMetrics(m *api.MetricsSnapshot) *MetricsSnapshot {
	if m == nil {
		return nil
	}
	return &MetricsSnapshot{
		URLsProcessed:   m.URLsProcessed,
		URLsSaved:       m.URLsSaved,
		URLsSkipped:     m.URLsSkipped,
		URLsErrored:     m.URLsErrored,
		BytesDownloaded: m.BytesDownloaded,
		RobotsBlocked:   m.RobotsBlocked,
		DepthLimitHits:  m.DepthLimitHits,
		ContentFiltered: m.ContentFiltered,
		PagesPerSecond:  m.PagesPerSecond,
		QueueSize:       m.QueueSize,
		ElapsedTime:     m.ElapsedTime,
		Percentage:      m.Percentage,
		CurrentURL:      m.CurrentURL,
	}
}

// resultJSON creates a JSON tool result
func resultJSON(v interface{}) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
