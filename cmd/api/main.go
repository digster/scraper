package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"scraper/internal/api"
)

func main() {
	// Parse command line flags
	config := api.DefaultServerConfig()

	flag.StringVar(&config.Host, "host", config.Host, "Host address to bind to")
	flag.IntVar(&config.Port, "port", config.Port, "Port to listen on")
	flag.IntVar(&config.MaxConcurrentJobs, "max-concurrent", config.MaxConcurrentJobs, "Maximum concurrent crawl jobs")
	flag.StringVar(&config.APIKey, "api-key", config.APIKey, "API key for authentication (optional)")

	var corsOrigins string
	flag.StringVar(&corsOrigins, "cors-origins", "", "Comma-separated list of allowed CORS origins")

	flag.IntVar(&config.ReadTimeout, "read-timeout", config.ReadTimeout, "Read timeout in seconds")
	flag.IntVar(&config.WriteTimeout, "write-timeout", config.WriteTimeout, "Write timeout in seconds")
	flag.IntVar(&config.IdleTimeout, "idle-timeout", config.IdleTimeout, "Idle timeout in seconds")

	flag.Parse()

	// Parse CORS origins
	if corsOrigins != "" {
		config.CORSOrigins = strings.Split(corsOrigins, ",")
		for i, origin := range config.CORSOrigins {
			config.CORSOrigins[i] = strings.TrimSpace(origin)
		}
	}

	// Load environment variables (override flags)
	config.LoadFromEnv()

	// Create and start server
	server, err := api.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case sig := <-shutdown:
		fmt.Println() // New line after ^C
		log.Printf("Received signal %v, shutting down...", sig)

		// Give outstanding requests time to complete
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}

	log.Println("Server stopped")
}
