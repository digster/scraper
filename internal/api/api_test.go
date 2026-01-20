package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"scraper/internal/crawler"
)

func TestHealthCheck(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", resp.Status)
	}
	if resp.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", resp.Version)
	}
}

func TestListCrawls_Empty(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	req := httptest.NewRequest("GET", "/api/v1/crawl", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var summaries []JobSummary
	if err := json.Unmarshal(w.Body.Bytes(), &summaries); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(summaries) != 0 {
		t.Errorf("expected empty list, got %d items", len(summaries))
	}
}

func TestCreateCrawl_MissingURL(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/crawl", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var apiErr APIError
	if err := json.Unmarshal(w.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("failed to unmarshal error: %v", err)
	}

	if apiErr.Message != "url is required" {
		t.Errorf("expected 'url is required', got '%s'", apiErr.Message)
	}
}

func TestCreateCrawl_InvalidURL(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	body := `{"url": "not-a-valid-url"}`
	req := httptest.NewRequest("POST", "/api/v1/crawl", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateCrawl_InvalidJSON(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/api/v1/crawl", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetCrawl_NotFound(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	req := httptest.NewRequest("GET", "/api/v1/crawl/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestPauseCrawl_NotFound(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	req := httptest.NewRequest("POST", "/api/v1/crawl/nonexistent/pause", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestResumeCrawl_NotFound(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	req := httptest.NewRequest("POST", "/api/v1/crawl/nonexistent/resume", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestDeleteCrawl_NotFound(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	req := httptest.NewRequest("DELETE", "/api/v1/crawl/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGetMetrics_NotFound(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	req := httptest.NewRequest("GET", "/api/v1/crawl/nonexistent/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestAPIKeyAuth(t *testing.T) {
	config := DefaultServerConfig()
	config.APIKey = "test-secret-key"
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	tests := []struct {
		name           string
		path           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "health endpoint without auth",
			path:           "/health",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "crawl endpoint without auth",
			path:           "/api/v1/crawl",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "crawl endpoint with wrong auth",
			path:           "/api/v1/crawl",
			authHeader:     "Bearer wrong-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "crawl endpoint with correct auth",
			path:           "/api/v1/crawl",
			authHeader:     "Bearer test-secret-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "crawl endpoint with invalid auth format",
			path:           "/api/v1/crawl",
			authHeader:     "Basic test-secret-key",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	config := DefaultServerConfig()
	config.CORSOrigins = []string{"http://localhost:3000", "http://example.com"}
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{
			name:           "allowed origin",
			origin:         "http://localhost:3000",
			expectedOrigin: "http://localhost:3000",
		},
		{
			name:           "another allowed origin",
			origin:         "http://example.com",
			expectedOrigin: "http://example.com",
		},
		{
			name:           "disallowed origin",
			origin:         "http://evil.com",
			expectedOrigin: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/health", nil)
			req.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			gotOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if gotOrigin != tc.expectedOrigin {
				t.Errorf("expected CORS origin '%s', got '%s'", tc.expectedOrigin, gotOrigin)
			}
		})
	}
}

func TestSSEEmitter(t *testing.T) {
	emitter := NewSSEEmitter()

	// Subscribe two clients
	ch1, unsub1 := emitter.Subscribe()
	ch2, unsub2 := emitter.Subscribe()
	defer unsub1()
	defer unsub2()

	if emitter.ClientCount() != 2 {
		t.Errorf("expected 2 clients, got %d", emitter.ClientCount())
	}

	// Emit an event
	testEvent := crawler.CrawlerEvent{
		Type:      crawler.EventProgress,
		Timestamp: time.Now(),
		Data:      "test data",
	}
	emitter.Emit(testEvent)

	// Both clients should receive it
	select {
	case event := <-ch1:
		if event.Type != string(crawler.EventProgress) {
			t.Errorf("expected event type 'progress', got '%s'", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client 1 did not receive event")
	}

	select {
	case event := <-ch2:
		if event.Type != string(crawler.EventProgress) {
			t.Errorf("expected event type 'progress', got '%s'", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client 2 did not receive event")
	}

	// Unsubscribe client 1
	unsub1()

	if emitter.ClientCount() != 1 {
		t.Errorf("expected 1 client after unsubscribe, got %d", emitter.ClientCount())
	}
}

func TestSSEEmitter_Close(t *testing.T) {
	emitter := NewSSEEmitter()

	ch, unsub := emitter.Subscribe()
	defer unsub()

	emitter.Close()

	if !emitter.IsClosed() {
		t.Error("emitter should be closed")
	}

	// Channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("channel read should not block")
	}

	// Emit should not panic after close
	emitter.Emit(crawler.CrawlerEvent{})
}

func TestSSEEmitter_Concurrent(t *testing.T) {
	emitter := NewSSEEmitter()
	const numClients = 10
	const numEvents = 100

	var wg sync.WaitGroup
	channels := make([]<-chan SSEEvent, numClients)
	unsubFuncs := make([]func(), numClients)

	// Subscribe clients
	for i := 0; i < numClients; i++ {
		ch, unsub := emitter.Subscribe()
		channels[i] = ch
		unsubFuncs[i] = unsub
	}

	// Emit events concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numEvents; i++ {
			emitter.Emit(crawler.CrawlerEvent{
				Type:      crawler.EventLogMessage,
				Timestamp: time.Now(),
				Data:      i,
			})
		}
	}()

	// Read events concurrently
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(ch <-chan SSEEvent) {
			defer wg.Done()
			count := 0
			for range ch {
				count++
				if count >= numEvents {
					return
				}
			}
		}(channels[i])
	}

	// Close emitter after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		emitter.Close()
	}()

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("test timed out")
	}
}

func TestServerConfig_Defaults(t *testing.T) {
	config := DefaultServerConfig()

	if config.Host != "0.0.0.0" {
		t.Errorf("expected default host '0.0.0.0', got '%s'", config.Host)
	}
	if config.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", config.Port)
	}
	if config.MaxConcurrentJobs != 5 {
		t.Errorf("expected default max concurrent jobs 5, got %d", config.MaxConcurrentJobs)
	}
}

func TestServerConfig_Address(t *testing.T) {
	config := DefaultServerConfig()
	config.Host = "127.0.0.1"
	config.Port = 9000

	expected := "127.0.0.1:9000"
	if config.Address() != expected {
		t.Errorf("expected address '%s', got '%s'", expected, config.Address())
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*ServerConfig)
		expectError bool
	}{
		{
			name:        "valid default config",
			modify:      func(c *ServerConfig) {},
			expectError: false,
		},
		{
			name:        "invalid port (0)",
			modify:      func(c *ServerConfig) { c.Port = 0 },
			expectError: true,
		},
		{
			name:        "invalid port (negative)",
			modify:      func(c *ServerConfig) { c.Port = -1 },
			expectError: true,
		},
		{
			name:        "invalid port (too high)",
			modify:      func(c *ServerConfig) { c.Port = 70000 },
			expectError: true,
		},
		{
			name:        "invalid max concurrent jobs",
			modify:      func(c *ServerConfig) { c.MaxConcurrentJobs = 0 },
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultServerConfig()
			tc.modify(config)
			err := config.Validate()
			hasError := err != nil
			if hasError != tc.expectError {
				t.Errorf("expected error=%v, got error=%v (%v)", tc.expectError, hasError, err)
			}
		})
	}
}

func TestJobManager_MaxConcurrentJobs(t *testing.T) {
	jm := NewJobManager(2) // Only allow 2 concurrent jobs

	// Create first job
	job1, err := jm.CreateJob(&CrawlRequest{URL: "https://example1.com"})
	if err != nil {
		t.Fatalf("failed to create job 1: %v", err)
	}
	job1.SetStatus(JobStatusRunning)

	// Create second job
	job2, err := jm.CreateJob(&CrawlRequest{URL: "https://example2.com"})
	if err != nil {
		t.Fatalf("failed to create job 2: %v", err)
	}
	job2.SetStatus(JobStatusRunning)

	// Third job should fail
	_, err = jm.CreateJob(&CrawlRequest{URL: "https://example3.com"})
	if err == nil {
		t.Error("expected error when exceeding max concurrent jobs")
	}

	apiErr, ok := err.(APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != 429 {
		t.Errorf("expected code 429, got %d", apiErr.Code)
	}
}

func TestAPIError(t *testing.T) {
	err := APIError{Code: 404, Message: "not found", Details: "job xyz"}

	if err.Error() != "not found: job xyz" {
		t.Errorf("unexpected error string: %s", err.Error())
	}

	err2 := APIError{Code: 500, Message: "internal error"}
	if err2.Error() != "internal error" {
		t.Errorf("unexpected error string: %s", err2.Error())
	}
}

func TestFromCrawlerEvent(t *testing.T) {
	crawlerEvent := crawler.CrawlerEvent{
		Type:      crawler.EventProgress,
		Timestamp: time.Now(),
		Data:      map[string]int{"count": 42},
	}

	sseEvent := FromCrawlerEvent(crawlerEvent)

	if sseEvent.Type != string(crawler.EventProgress) {
		t.Errorf("expected type 'progress', got '%s'", sseEvent.Type)
	}
	if !sseEvent.Timestamp.Equal(crawlerEvent.Timestamp) {
		t.Error("timestamps don't match")
	}
}

func TestNotFound(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	req := httptest.NewRequest("GET", "/nonexistent/path", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	config := DefaultServerConfig()
	jm := NewJobManager(5)
	handlers := NewHandlers(jm, "1.0.0")
	router := NewRouter(handlers, config)

	// PUT is not allowed on /api/v1/crawl
	req := httptest.NewRequest("PUT", "/api/v1/crawl", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
