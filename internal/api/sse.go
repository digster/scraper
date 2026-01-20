package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// SSEHeartbeatInterval is how often to send heartbeat comments
const SSEHeartbeatInterval = 15 * time.Second

// StreamEvents handles GET /api/v1/crawl/{jobId}/events (SSE)
func (h *Handlers) StreamEvents(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	job, err := h.JobManager.GetJob(jobID)
	if err != nil {
		writeError(w, err)
		return
	}

	// Check if client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, APIError{Code: 500, Message: "streaming not supported"})
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Subscribe to events
	eventChan, unsubscribe := job.Emitter.Subscribe()
	defer unsubscribe()

	// Create heartbeat ticker
	heartbeat := time.NewTicker(SSEHeartbeatInterval)
	defer heartbeat.Stop()

	// Send initial connection event
	sendSSEEvent(w, "connected", map[string]interface{}{
		"jobId":  jobID,
		"status": job.GetStatus(),
	})
	flusher.Flush()

	// Stream events
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed, job completed
				sendSSEEvent(w, "disconnected", map[string]string{
					"reason": "job completed",
				})
				flusher.Flush()
				return
			}

			sendSSEEvent(w, event.Type, event)
			flusher.Flush()

		case <-heartbeat.C:
			// Send heartbeat comment to keep connection alive
			fmt.Fprintf(w, ": heartbeat %d\n\n", time.Now().Unix())
			flusher.Flush()

		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}

// sendSSEEvent writes a single SSE event to the response writer
func sendSSEEvent(w http.ResponseWriter, eventType string, data interface{}) {
	// Event type
	fmt.Fprintf(w, "event: %s\n", eventType)

	// Data (JSON encoded)
	jsonData, err := json.Marshal(data)
	if err != nil {
		jsonData = []byte(`{"error": "failed to encode event data"}`)
	}
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
}

// SSEClient is used for testing SSE connections
type SSEClient struct {
	Events chan SSEEvent
	Done   chan struct{}
}

// NewSSEClient creates a new SSE client for testing
func NewSSEClient() *SSEClient {
	return &SSEClient{
		Events: make(chan SSEEvent, 100),
		Done:   make(chan struct{}),
	}
}
