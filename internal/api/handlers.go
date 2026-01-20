package api

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
)

// Handlers holds dependencies for HTTP handlers
type Handlers struct {
	JobManager *JobManager
	StartTime  time.Time
	Version    string
}

// NewHandlers creates a new Handlers instance
func NewHandlers(jm *JobManager, version string) *Handlers {
	return &Handlers{
		JobManager: jm,
		StartTime:  time.Now(),
		Version:    version,
	}
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// writeError writes an error response
func writeError(w http.ResponseWriter, err error) {
	if apiErr, ok := err.(APIError); ok {
		writeJSON(w, apiErr.Code, apiErr)
		return
	}
	writeJSON(w, http.StatusInternalServerError, APIError{
		Code:    500,
		Message: "internal server error",
		Details: err.Error(),
	})
}

// HealthCheck handles GET /health
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.StartTime)

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:     "healthy",
		Version:    h.Version,
		Uptime:     formatUptime(uptime),
		ActiveJobs: h.JobManager.ActiveJobCount(),
	})
}

// CreateCrawl handles POST /api/v1/crawl
func (h *Handlers) CreateCrawl(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, APIError{Code: 400, Message: "failed to read request body"})
		return
	}
	defer r.Body.Close()

	var req CrawlRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, APIError{Code: 400, Message: "invalid JSON", Details: err.Error()})
		return
	}

	// Validate URL is provided
	if req.URL == "" {
		writeError(w, APIError{Code: 400, Message: "url is required"})
		return
	}

	// Create the job
	job, err := h.JobManager.CreateJob(&req)
	if err != nil {
		writeError(w, err)
		return
	}

	// Start the job
	if err := h.JobManager.StartJob(job.ID); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, CrawlResponse{
		JobID:     job.ID,
		Status:    job.GetStatus(),
		CreatedAt: job.CreatedAt,
	})
}

// ListCrawls handles GET /api/v1/crawl
func (h *Handlers) ListCrawls(w http.ResponseWriter, r *http.Request) {
	jobs := h.JobManager.ListJobs()

	// Sort by creation time (newest first)
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})

	summaries := make([]JobSummary, len(jobs))
	for i, job := range jobs {
		summaries[i] = job.ToSummary()
	}

	writeJSON(w, http.StatusOK, summaries)
}

// GetCrawl handles GET /api/v1/crawl/{jobId}
func (h *Handlers) GetCrawl(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	job, err := h.JobManager.GetJob(jobID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, job.ToDetails())
}

// DeleteCrawl handles DELETE /api/v1/crawl/{jobId}
func (h *Handlers) DeleteCrawl(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	// First try to stop if running
	job, err := h.JobManager.GetJob(jobID)
	if err != nil {
		writeError(w, err)
		return
	}

	status := job.GetStatus()
	if status == JobStatusRunning || status == JobStatusPaused || status == JobStatusWaitingForLogin {
		// Stop the job first
		if err := h.JobManager.StopJob(jobID); err != nil {
			writeError(w, err)
			return
		}
		// Give it a moment to stop
		time.Sleep(100 * time.Millisecond)
	}

	// Now delete
	if err := h.JobManager.DeleteJob(jobID); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PauseCrawl handles POST /api/v1/crawl/{jobId}/pause
func (h *Handlers) PauseCrawl(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	if err := h.JobManager.PauseJob(jobID); err != nil {
		writeError(w, err)
		return
	}

	job, _ := h.JobManager.GetJob(jobID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobId":  jobID,
		"status": job.GetStatus(),
	})
}

// ResumeCrawl handles POST /api/v1/crawl/{jobId}/resume
func (h *Handlers) ResumeCrawl(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	if err := h.JobManager.ResumeJob(jobID); err != nil {
		writeError(w, err)
		return
	}

	job, _ := h.JobManager.GetJob(jobID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobId":  jobID,
		"status": job.GetStatus(),
	})
}

// ConfirmLogin handles POST /api/v1/crawl/{jobId}/confirm-login
func (h *Handlers) ConfirmLogin(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	if err := h.JobManager.ConfirmLogin(jobID); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobId":   jobID,
		"message": "login confirmed",
	})
}

// GetMetrics handles GET /api/v1/crawl/{jobId}/metrics
func (h *Handlers) GetMetrics(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	job, err := h.JobManager.GetJob(jobID)
	if err != nil {
		writeError(w, err)
		return
	}

	metrics := job.GetMetrics()
	if metrics == nil {
		writeError(w, APIError{Code: 404, Message: "metrics not available"})
		return
	}

	writeJSON(w, http.StatusOK, metrics)
}

// formatUptime formats duration as a human-readable string
func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return formatDuration(days, "d") + " " + formatDuration(hours, "h") + " " + formatDuration(minutes, "m")
	}
	if hours > 0 {
		return formatDuration(hours, "h") + " " + formatDuration(minutes, "m") + " " + formatDuration(seconds, "s")
	}
	if minutes > 0 {
		return formatDuration(minutes, "m") + " " + formatDuration(seconds, "s")
	}
	return formatDuration(seconds, "s")
}

func formatDuration(value int, unit string) string {
	return string(rune('0'+value/10)) + string(rune('0'+value%10)) + unit
}
