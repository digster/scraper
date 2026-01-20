package api

import (
	"os"
	"strconv"
	"strings"
)

// ServerConfig holds all configuration for the API server
type ServerConfig struct {
	// Host is the address to bind to (default: "0.0.0.0")
	Host string

	// Port is the port to listen on (default: 8080)
	Port int

	// MaxConcurrentJobs is the maximum number of concurrent crawl jobs (default: 5)
	MaxConcurrentJobs int

	// APIKey is the optional API key for authentication (empty = no auth)
	APIKey string

	// CORSOrigins is a list of allowed CORS origins (empty = no CORS)
	CORSOrigins []string

	// ReadTimeout is the maximum duration for reading the entire request (seconds)
	ReadTimeout int

	// WriteTimeout is the maximum duration before timing out writes of the response (seconds)
	WriteTimeout int

	// IdleTimeout is the maximum amount of time to wait for the next request (seconds)
	IdleTimeout int
}

// DefaultServerConfig returns a ServerConfig with sensible defaults
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:              "0.0.0.0",
		Port:              8080,
		MaxConcurrentJobs: 5,
		APIKey:            "",
		CORSOrigins:       nil,
		ReadTimeout:       30,
		WriteTimeout:      60,
		IdleTimeout:       120,
	}
}

// Address returns the full address string (host:port)
func (c *ServerConfig) Address() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

// LoadFromEnv loads configuration from environment variables
// Environment variables take precedence over existing values
func (c *ServerConfig) LoadFromEnv() {
	if host := os.Getenv("API_HOST"); host != "" {
		c.Host = host
	}

	if port := os.Getenv("API_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil && p > 0 {
			c.Port = p
		}
	}

	if maxJobs := os.Getenv("API_MAX_CONCURRENT_JOBS"); maxJobs != "" {
		if m, err := strconv.Atoi(maxJobs); err == nil && m > 0 {
			c.MaxConcurrentJobs = m
		}
	}

	if apiKey := os.Getenv("API_KEY"); apiKey != "" {
		c.APIKey = apiKey
	}

	if origins := os.Getenv("API_CORS_ORIGINS"); origins != "" {
		c.CORSOrigins = strings.Split(origins, ",")
		for i, origin := range c.CORSOrigins {
			c.CORSOrigins[i] = strings.TrimSpace(origin)
		}
	}

	if readTimeout := os.Getenv("API_READ_TIMEOUT"); readTimeout != "" {
		if t, err := strconv.Atoi(readTimeout); err == nil && t > 0 {
			c.ReadTimeout = t
		}
	}

	if writeTimeout := os.Getenv("API_WRITE_TIMEOUT"); writeTimeout != "" {
		if t, err := strconv.Atoi(writeTimeout); err == nil && t > 0 {
			c.WriteTimeout = t
		}
	}

	if idleTimeout := os.Getenv("API_IDLE_TIMEOUT"); idleTimeout != "" {
		if t, err := strconv.Atoi(idleTimeout); err == nil && t > 0 {
			c.IdleTimeout = t
		}
	}
}

// Validate checks that the configuration is valid
func (c *ServerConfig) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return APIError{Code: 500, Message: "invalid port", Details: "port must be between 1 and 65535"}
	}

	if c.MaxConcurrentJobs < 1 {
		return APIError{Code: 500, Message: "invalid max concurrent jobs", Details: "must be at least 1"}
	}

	return nil
}

// HasAuth returns true if API key authentication is enabled
func (c *ServerConfig) HasAuth() bool {
	return c.APIKey != ""
}

// HasCORS returns true if CORS is configured
func (c *ServerConfig) HasCORS() bool {
	return len(c.CORSOrigins) > 0
}
