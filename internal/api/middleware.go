package api

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// Logger middleware logs HTTP requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		log.Printf("%s %s %d %v %s",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
			r.RemoteAddr,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher for SSE support
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Recovery middleware recovers from panics
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v\n%s", err, debug.Stack())
				writeJSON(w, http.StatusInternalServerError, APIError{
					Code:    500,
					Message: "internal server error",
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// APIKeyAuth middleware validates API key authentication
func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health endpoint
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			// Get authorization header
			auth := r.Header.Get("Authorization")
			if auth == "" {
				writeJSON(w, http.StatusUnauthorized, APIError{
					Code:    401,
					Message: "authorization required",
					Details: "provide API key in Authorization header as 'Bearer <key>'",
				})
				return
			}

			// Parse bearer token
			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				writeJSON(w, http.StatusUnauthorized, APIError{
					Code:    401,
					Message: "invalid authorization format",
					Details: "expected 'Bearer <key>'",
				})
				return
			}

			// Validate API key
			if parts[1] != apiKey {
				writeJSON(w, http.StatusUnauthorized, APIError{
					Code:    401,
					Message: "invalid API key",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	// Build origin lookup map
	originMap := make(map[string]bool)
	allowAll := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
		originMap[origin] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" && originMap[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}

			// Set other CORS headers
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestID middleware adds a unique request ID to each request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate a simple request ID from timestamp + counter
		requestID := time.Now().UnixNano()
		w.Header().Set("X-Request-ID", string(rune(requestID)))

		next.ServeHTTP(w, r)
	})
}

// ContentType middleware sets default content type for responses
func ContentType(contentType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			next.ServeHTTP(w, r)
		})
	}
}
