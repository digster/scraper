package api

import (
	"context"
	"log"
	"net/http"
	"time"
)

// Server represents the API server
type Server struct {
	httpServer *http.Server
	jobManager *JobManager
	config     *ServerConfig
}

// NewServer creates a new API server
func NewServer(config *ServerConfig) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	jobManager := NewJobManager(config.MaxConcurrentJobs)
	handlers := NewHandlers(jobManager, "1.0.0")
	router := NewRouter(handlers, config)

	httpServer := &http.Server{
		Addr:         config.Address(),
		Handler:      router,
		ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(config.IdleTimeout) * time.Second,
	}

	return &Server{
		httpServer: httpServer,
		jobManager: jobManager,
		config:     config,
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("API server starting on %s", s.config.Address())
	if s.config.HasAuth() {
		log.Printf("API key authentication enabled")
	}
	if s.config.HasCORS() {
		log.Printf("CORS enabled for origins: %v", s.config.CORSOrigins)
	}
	log.Printf("Max concurrent jobs: %d", s.config.MaxConcurrentJobs)

	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down API server...")

	// Stop all active jobs
	s.jobManager.Shutdown()

	// Shutdown HTTP server
	return s.httpServer.Shutdown(ctx)
}

// JobManager returns the server's job manager
func (s *Server) JobManager() *JobManager {
	return s.jobManager
}

// Address returns the server's address
func (s *Server) Address() string {
	return s.config.Address()
}
