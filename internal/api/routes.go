package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRouter creates and configures the HTTP router with all routes
func NewRouter(handlers *Handlers, config *ServerConfig) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(Recovery)
	r.Use(Logger)

	// CORS middleware (if configured)
	if config.HasCORS() {
		r.Use(CORS(config.CORSOrigins))
	}

	// API key authentication (if configured)
	if config.HasAuth() {
		r.Use(APIKeyAuth(config.APIKey))
	}

	// Health check (always accessible)
	r.Get("/health", handlers.HealthCheck)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Crawl job endpoints
		r.Route("/crawl", func(r chi.Router) {
			r.Post("/", handlers.CreateCrawl)        // Create new crawl
			r.Get("/", handlers.ListCrawls)          // List all crawls

			// Job-specific endpoints
			r.Route("/{jobId}", func(r chi.Router) {
				r.Get("/", handlers.GetCrawl)              // Get job details
				r.Delete("/", handlers.DeleteCrawl)        // Stop and remove job
				r.Post("/pause", handlers.PauseCrawl)      // Pause job
				r.Post("/resume", handlers.ResumeCrawl)    // Resume job
				r.Post("/confirm-login", handlers.ConfirmLogin) // Confirm manual login
				r.Get("/metrics", handlers.GetMetrics)     // Get metrics
				r.Get("/events", handlers.StreamEvents)    // SSE event stream
			})
		})
	})

	// 404 handler
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, APIError{
			Code:    404,
			Message: "not found",
			Details: "endpoint does not exist",
		})
	})

	// 405 handler
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{
			Code:    405,
			Message: "method not allowed",
		})
	})

	return r
}
