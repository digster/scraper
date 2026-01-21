package crawler

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateFilename(t *testing.T) {
	c := &Crawler{config: Config{}}

	tests := []struct {
		name     string
		rawURL   string
		expected string
	}{
		{
			name:     "root path without query",
			rawURL:   "https://example.com/",
			expected: "index.html",
		},
		{
			name:     "root path with query",
			rawURL:   "https://example.com/?page=1",
			expected: "index_page-1.html",
		},
		{
			name:     "simple path without query",
			rawURL:   "https://example.com/articles",
			expected: "articles.html",
		},
		{
			name:     "simple path with query",
			rawURL:   "https://example.com/articles?id=123",
			expected: "articles_id-123.html",
		},
		{
			name:     "path with multiple query params",
			rawURL:   "https://example.com/search?q=golang&page=2",
			expected: "search_q-golang_page-2.html",
		},
		{
			name:     "nested path without query",
			rawURL:   "https://example.com/blog/posts/my-article",
			expected: "blog/posts/my-article.html",
		},
		{
			name:     "nested path with query",
			rawURL:   "https://example.com/blog/posts?category=tech&sort=date",
			expected: "blog/posts_category-tech_sort-date.html",
		},
		{
			name:     "path with existing html extension",
			rawURL:   "https://example.com/page.html",
			expected: "page.html",
		},
		{
			name:     "path with existing html extension and query",
			rawURL:   "https://example.com/page.html?v=2",
			expected: "page_v-2.html",
		},
		{
			name:     "path with non-html extension and query",
			rawURL:   "https://example.com/data.json?format=pretty",
			expected: "data_format-pretty.json",
		},
		{
			name:     "empty path",
			rawURL:   "https://example.com",
			expected: "index.html",
		},
		{
			name:     "query with special characters",
			rawURL:   "https://example.com/search?q=foo:bar&filter=a|b",
			expected: "search_q-foo_bar_filter-a_b.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedURL, err := url.Parse(tt.rawURL)
			if err != nil {
				t.Fatalf("failed to parse URL %s: %v", tt.rawURL, err)
			}

			result := c.generateFilename(parsedURL)
			if result != tt.expected {
				t.Errorf("generateFilename(%s) = %q, want %q", tt.rawURL, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilenameComponent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"with:colon", "with_colon"},
		{"with?question", "with_question"},
		{"with*asterisk", "with_asterisk"},
		{"with<less", "with_less"},
		{"with>greater", "with_greater"},
		{"with|pipe", "with_pipe"},
		{"with\"quote", "with_quote"},
		{"with&ampersand", "with_ampersand"},
		{"key=value", "key-value"},
		{"a=1&b=2", "a-1_b-2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilenameComponent(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilenameComponent(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateFilenameUniqueness(t *testing.T) {
	c := &Crawler{config: Config{}}

	// These URLs should produce different filenames
	urls := []string{
		"https://example.com/articles?id=1",
		"https://example.com/articles?id=2",
		"https://example.com/articles?id=3",
	}

	filenames := make(map[string]string)
	for _, rawURL := range urls {
		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			t.Fatalf("failed to parse URL %s: %v", rawURL, err)
		}

		filename := c.generateFilename(parsedURL)
		if existingURL, exists := filenames[filename]; exists {
			t.Errorf("filename collision: %q and %q both produce %q", existingURL, rawURL, filename)
		}
		filenames[filename] = rawURL
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name         string
		rawURL       string
		prefixFilter string
		excludeExts  []string
		expected     bool
	}{
		{
			name:         "valid http URL without prefix filter",
			rawURL:       "http://example.com/page",
			prefixFilter: "",
			expected:     true,
		},
		{
			name:         "valid https URL without prefix filter",
			rawURL:       "https://example.com/page",
			prefixFilter: "",
			expected:     true,
		},
		{
			name:         "invalid scheme (ftp)",
			rawURL:       "ftp://example.com/file",
			prefixFilter: "",
			expected:     false,
		},
		{
			name:         "invalid scheme (javascript)",
			rawURL:       "javascript:void(0)",
			prefixFilter: "",
			expected:     false,
		},
		{
			name:         "mailto link",
			rawURL:       "mailto:test@example.com",
			prefixFilter: "",
			expected:     false,
		},
		{
			name:         "URL matching prefix filter",
			rawURL:       "https://example.com/docs/api",
			prefixFilter: "https://example.com/docs",
			expected:     true,
		},
		{
			name:         "URL not matching prefix filter",
			rawURL:       "https://example.com/blog/post",
			prefixFilter: "https://example.com/docs",
			expected:     false,
		},
		{
			name:         "different host with prefix filter",
			rawURL:       "https://other.com/docs/page",
			prefixFilter: "https://example.com/docs",
			expected:     false,
		},
		{
			name:         "excluded extension js",
			rawURL:       "https://example.com/script.js",
			prefixFilter: "",
			excludeExts:  []string{"js", "css"},
			expected:     false,
		},
		{
			name:         "excluded extension css",
			rawURL:       "https://example.com/style.css",
			prefixFilter: "",
			excludeExts:  []string{"js", "css"},
			expected:     false,
		},
		{
			name:         "non-excluded extension",
			rawURL:       "https://example.com/page.html",
			prefixFilter: "",
			excludeExts:  []string{"js", "css"},
			expected:     true,
		},
		{
			name:         "prefix filter set to none",
			rawURL:       "https://any-domain.com/page",
			prefixFilter: "none",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				config: Config{
					PrefixFilterURL:   tt.prefixFilter,
					ExcludeExtensions: tt.excludeExts,
				},
			}

			result := c.isValidURL(tt.rawURL)
			if result != tt.expected {
				t.Errorf("isValidURL(%q) = %v, want %v", tt.rawURL, result, tt.expected)
			}
		})
	}
}

func TestShouldExcludeByExtension(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		excludeExts []string
		expected    bool
	}{
		{
			name:        "no exclusions configured",
			path:        "/path/to/script.js",
			excludeExts: []string{},
			expected:    false,
		},
		{
			name:        "excluded js extension",
			path:        "/path/to/script.js",
			excludeExts: []string{"js", "css"},
			expected:    true,
		},
		{
			name:        "excluded css extension",
			path:        "/styles/main.css",
			excludeExts: []string{"js", "css"},
			expected:    true,
		},
		{
			name:        "non-excluded html extension",
			path:        "/page.html",
			excludeExts: []string{"js", "css"},
			expected:    false,
		},
		{
			name:        "path without extension",
			path:        "/api/users",
			excludeExts: []string{"js", "css"},
			expected:    false,
		},
		{
			name:        "case insensitive - uppercase extension",
			path:        "/image.PNG",
			excludeExts: []string{"png", "jpg"},
			expected:    true,
		},
		{
			name:        "excluded image extensions",
			path:        "/images/photo.jpg",
			excludeExts: []string{"png", "jpg", "gif"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				config: Config{
					ExcludeExtensions: tt.excludeExts,
				},
			}

			result := c.shouldExcludeByExtension(tt.path)
			if result != tt.expected {
				t.Errorf("shouldExcludeByExtension(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestShouldExcludeByContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		excludeExts []string
		expected    bool
	}{
		{
			name:        "no exclusions configured",
			contentType: "application/javascript",
			excludeExts: []string{},
			expected:    false,
		},
		{
			name:        "excluded javascript",
			contentType: "application/javascript",
			excludeExts: []string{"js"},
			expected:    true,
		},
		{
			name:        "excluded javascript with charset",
			contentType: "application/javascript; charset=utf-8",
			excludeExts: []string{"js"},
			expected:    true,
		},
		{
			name:        "excluded css",
			contentType: "text/css",
			excludeExts: []string{"css"},
			expected:    true,
		},
		{
			name:        "excluded png image",
			contentType: "image/png",
			excludeExts: []string{"png", "jpg"},
			expected:    true,
		},
		{
			name:        "excluded jpeg image",
			contentType: "image/jpeg",
			excludeExts: []string{"png", "jpg"},
			expected:    true,
		},
		{
			name:        "non-excluded html",
			contentType: "text/html; charset=utf-8",
			excludeExts: []string{"js", "css", "png"},
			expected:    false,
		},
		{
			name:        "excluded pdf",
			contentType: "application/pdf",
			excludeExts: []string{"pdf"},
			expected:    true,
		},
		{
			name:        "excluded json",
			contentType: "application/json",
			excludeExts: []string{"json"},
			expected:    true,
		},
		{
			name:        "excluded woff font",
			contentType: "font/woff2",
			excludeExts: []string{"woff2"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				config: Config{
					ExcludeExtensions: tt.excludeExts,
				},
			}

			result := c.shouldExcludeByContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("shouldExcludeByContentType(%q) = %v, want %v", tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestHasContent(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "page with substantial content",
			html:     "<html><body><p>This is a paragraph with more than one hundred characters of meaningful text content that should pass the content validation check.</p></body></html>",
			expected: true,
		},
		{
			name:     "page with only scripts",
			html:     "<html><body><script>var x = 1; console.log('hello world this is a long script with lots of code');</script></body></html>",
			expected: false,
		},
		{
			name:     "page with only styles",
			html:     "<html><head><style>body { margin: 0; padding: 0; font-family: Arial; background-color: #ffffff; }</style></head><body></body></html>",
			expected: false,
		},
		{
			name:     "page with minimal text",
			html:     "<html><body><p>Short</p></body></html>",
			expected: false,
		},
		{
			name:     "empty page",
			html:     "<html><body></body></html>",
			expected: false,
		},
		{
			name:     "page with content mixed with scripts",
			html:     "<html><body><script>alert('x');</script><p>This paragraph contains more than one hundred characters of real text content that should be counted after removing scripts and styles from the page.</p></body></html>",
			expected: true,
		},
		{
			name:     "malformed html with content",
			html:     "<p>This is a paragraph with more than one hundred characters of meaningful text content that should pass the validation even with malformed HTML structure.</p>",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				config: Config{},
				log:    &Logger{verbose: false},
			}
			result := c.hasContent(tt.html)
			if result != tt.expected {
				t.Errorf("hasContent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStatePersistence(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "crawler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stateFile := filepath.Join(tmpDir, "test_state.json")

	// Create a crawler with some state
	ctx := context.Background()
	config := Config{
		URL:       "https://example.com",
		StateFile: stateFile,
		MaxDepth:  10,
	}
	c, err := NewCrawler(config, ctx)
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}
	defer c.Close()

	// Initialize fresh state
	state, err := LoadState(stateFile, config.URL)
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}
	c.state = state

	// Verify initial state
	if c.state.BaseURL != "https://example.com" {
		t.Errorf("BaseURL = %q, want %q", c.state.BaseURL, "https://example.com")
	}
	if len(c.state.Visited) != 0 {
		t.Errorf("Visited should be empty, got %d entries", len(c.state.Visited))
	}

	// Modify state
	c.state.Visited["https://example.com/page1"] = true
	c.state.Visited["https://example.com/page2"] = true
	c.state.Queue = append(c.state.Queue, URLInfo{URL: "https://example.com/page3", Depth: 1})
	c.state.Queued["https://example.com/page3"] = true
	c.state.Processed = 2
	c.state.URLDepths["https://example.com/page1"] = 0
	c.state.URLDepths["https://example.com/page2"] = 1

	// Save state
	err = SaveState(c.state, stateFile)
	if err != nil {
		t.Fatalf("SaveState() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatal("state file was not created")
	}

	// Load state into new state object
	state2, err := LoadState(stateFile, config.URL)
	if err != nil {
		t.Fatalf("LoadState() for state2 failed: %v", err)
	}

	// Verify loaded state
	if len(state2.Visited) != 2 {
		t.Errorf("Visited count = %d, want 2", len(state2.Visited))
	}
	if !state2.Visited["https://example.com/page1"] {
		t.Error("page1 should be in Visited")
	}
	if !state2.Visited["https://example.com/page2"] {
		t.Error("page2 should be in Visited")
	}
	if len(state2.Queue) != 1 {
		t.Errorf("Queue length = %d, want 1", len(state2.Queue))
	}
	if state2.Queue[0].URL != "https://example.com/page3" {
		t.Errorf("Queue[0].URL = %q, want %q", state2.Queue[0].URL, "https://example.com/page3")
	}
	if state2.Processed != 2 {
		t.Errorf("Processed = %d, want 2", state2.Processed)
	}
}

func TestStateBackwardCompatibility(t *testing.T) {
	// Test that loading old state without Queued map works
	tmpDir, err := os.MkdirTemp("", "crawler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stateFile := filepath.Join(tmpDir, "old_state.json")

	// Create old-format state without Queued field (raw JSON to simulate old format)
	oldStateJSON := `{
		"visited": {"https://example.com/page1": true},
		"queue": [{"url": "https://example.com/page2", "depth": 1}],
		"base_url": "https://example.com",
		"processed": 1,
		"url_depths": {"https://example.com/page1": 0}
	}`

	err = os.WriteFile(stateFile, []byte(oldStateJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write old state: %v", err)
	}

	// Load state
	state, err := LoadState(stateFile, "https://example.com")
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	// Verify Queued map was initialized from queue
	if state.Queued == nil {
		t.Fatal("Queued map should be initialized")
	}
	if len(state.Queue) != 1 {
		t.Fatalf("Queue should have 1 item, got %d", len(state.Queue))
	}
	if state.Queue[0].URL != "https://example.com/page2" {
		t.Errorf("Queue[0].URL = %q, want %q", state.Queue[0].URL, "https://example.com/page2")
	}
	if !state.Queued["https://example.com/page2"] {
		t.Error("Queued should contain page2 from queue")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				URL:      "https://example.com",
				MaxDepth: 10,
				Delay:    time.Second,
			},
			expectError: false,
		},
		{
			name: "empty URL",
			config: Config{
				URL:      "",
				MaxDepth: 10,
			},
			expectError: true,
			errorMsg:    "URL is required",
		},
		{
			name: "invalid URL scheme",
			config: Config{
				URL:      "ftp://example.com",
				MaxDepth: 10,
			},
			expectError: true,
			errorMsg:    "URL must use http or https scheme",
		},
		{
			name: "URL without host",
			config: Config{
				URL:      "https:///path",
				MaxDepth: 10,
			},
			expectError: true,
			errorMsg:    "URL must have a host",
		},
		{
			name: "zero depth",
			config: Config{
				URL:      "https://example.com",
				MaxDepth: 0,
			},
			expectError: true,
			errorMsg:    "depth must be greater than 0",
		},
		{
			name: "negative depth",
			config: Config{
				URL:      "https://example.com",
				MaxDepth: -1,
			},
			expectError: true,
			errorMsg:    "depth must be greater than 0",
		},
		{
			name: "negative delay",
			config: Config{
				URL:      "https://example.com",
				MaxDepth: 10,
				Delay:    -time.Second,
			},
			expectError: true,
			errorMsg:    "delay cannot be negative",
		},
		{
			name: "valid prefix filter URL",
			config: Config{
				URL:             "https://example.com",
				MaxDepth:        10,
				PrefixFilterURL: "https://example.com/docs",
			},
			expectError: false,
		},
		{
			name: "prefix filter set to none",
			config: Config{
				URL:             "https://example.com",
				MaxDepth:        10,
				PrefixFilterURL: "none",
			},
			expectError: false,
		},
		{
			name: "invalid prefix filter URL scheme",
			config: Config{
				URL:             "https://example.com",
				MaxDepth:        10,
				PrefixFilterURL: "ftp://example.com",
			},
			expectError: true,
			errorMsg:    "prefix-filter URL must use http or https scheme",
		},
		{
			name: "prefix filter URL without host",
			config: Config{
				URL:             "https://example.com",
				MaxDepth:        10,
				PrefixFilterURL: "https:///path",
			},
			expectError: true,
			errorMsg:    "prefix-filter URL must have a host",
		},
		{
			name: "http URL is valid",
			config: Config{
				URL:      "http://example.com",
				MaxDepth: 5,
			},
			expectError: false,
		},
		{
			name: "negative min-content",
			config: Config{
				URL:              "https://example.com",
				MaxDepth:         10,
				MinContentLength: -1,
			},
			expectError: true,
			errorMsg:    "min-content cannot be negative",
		},
		{
			name: "zero min-content is valid",
			config: Config{
				URL:              "https://example.com",
				MaxDepth:         10,
				MinContentLength: 0,
			},
			expectError: false,
		},
		{
			name: "custom min-content is valid",
			config: Config{
				URL:              "https://example.com",
				MaxDepth:         10,
				MinContentLength: 500,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(&tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestSetDefaultStateFile(t *testing.T) {
	tests := []struct {
		name          string
		outputDir     string
		existingState string
		expectedState string
	}{
		{
			name:          "state file placed inside output directory",
			outputDir:     "backup/example.com",
			existingState: "",
			expectedState: filepath.Join("backup/example.com", "example.com_state.json"),
		},
		{
			name:          "state file with nested output directory",
			outputDir:     "backup/docs.example.com_api",
			existingState: "",
			expectedState: filepath.Join("backup/docs.example.com_api", "docs.example.com_api_state.json"),
		},
		{
			name:          "existing state file is not overwritten",
			outputDir:     "backup/example.com",
			existingState: "/custom/path/my_state.json",
			expectedState: "/custom/path/my_state.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				OutputDir: tt.outputDir,
				StateFile: tt.existingState,
			}
			SetDefaultStateFile(config)
			if config.StateFile != tt.expectedState {
				t.Errorf("SetDefaultStateFile() = %q, want %q", config.StateFile, tt.expectedState)
			}
		})
	}
}

func TestCrawlerMetrics(t *testing.T) {
	m := NewCrawlerMetrics()

	// Test initial state
	if m.URLsProcessed != 0 {
		t.Errorf("initial URLsProcessed = %d, want 0", m.URLsProcessed)
	}

	// Test IncrementProcessed
	m.IncrementProcessed()
	m.IncrementProcessed()
	if m.URLsProcessed != 2 {
		t.Errorf("URLsProcessed after 2 increments = %d, want 2", m.URLsProcessed)
	}

	// Test IncrementSaved
	m.IncrementSaved(1024)
	m.IncrementSaved(2048)
	if m.URLsSaved != 2 {
		t.Errorf("URLsSaved = %d, want 2", m.URLsSaved)
	}
	if m.BytesDownloaded != 3072 {
		t.Errorf("BytesDownloaded = %d, want 3072", m.BytesDownloaded)
	}

	// Test IncrementErrored
	m.IncrementErrored()
	if m.URLsErrored != 1 {
		t.Errorf("URLsErrored = %d, want 1", m.URLsErrored)
	}

	// Test IncrementSkipped
	m.IncrementSkipped()
	if m.URLsSkipped != 1 {
		t.Errorf("URLsSkipped = %d, want 1", m.URLsSkipped)
	}

	// Test IncrementRobotsBlocked
	m.IncrementRobotsBlocked()
	if m.RobotsBlocked != 1 {
		t.Errorf("RobotsBlocked = %d, want 1", m.RobotsBlocked)
	}

	// Test IncrementDepthLimitHits
	m.IncrementDepthLimitHits()
	if m.DepthLimitHits != 1 {
		t.Errorf("DepthLimitHits = %d, want 1", m.DepthLimitHits)
	}

	// Test IncrementContentFiltered
	m.IncrementContentFiltered()
	if m.ContentFiltered != 1 {
		t.Errorf("ContentFiltered = %d, want 1", m.ContentFiltered)
	}

	// Test SetQueueSize
	m.SetQueueSize(42)
	if m.QueueSize != 42 {
		t.Errorf("QueueSize = %d, want 42", m.QueueSize)
	}

	// Test GetSnapshot
	snapshot := m.GetSnapshot()
	if snapshot.URLsProcessed != 2 {
		t.Errorf("snapshot.URLsProcessed = %d, want 2", snapshot.URLsProcessed)
	}

	// Test Finalize
	m.Finalize()
	if m.EndTime.IsZero() {
		t.Error("EndTime should be set after Finalize")
	}
	if m.Duration <= 0 {
		t.Error("Duration should be positive after Finalize")
	}
}

func TestMetricsJSONOutput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "metrics_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := NewCrawlerMetrics()
	m.IncrementProcessed()
	m.IncrementSaved(1024)
	m.SetQueueSize(5)

	metricsFile := filepath.Join(tmpDir, "metrics.json")
	err = m.WriteJSON(metricsFile)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(metricsFile); os.IsNotExist(err) {
		t.Fatal("metrics file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(metricsFile)
	if err != nil {
		t.Fatalf("failed to read metrics file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "\"urls_processed\": 1") {
		t.Error("metrics JSON should contain urls_processed: 1")
	}
	if !strings.Contains(content, "\"urls_saved\": 1") {
		t.Error("metrics JSON should contain urls_saved: 1")
	}
	if !strings.Contains(content, "\"bytes_downloaded\": 1024") {
		t.Error("metrics JSON should contain bytes_downloaded: 1024")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "00:00:00"},
		{30 * time.Second, "00:00:30"},
		{90 * time.Second, "00:01:30"},
		{3661 * time.Second, "01:01:01"},
		{86400 * time.Second, "24:00:00"},
	}

	for _, tt := range tests {
		result := FormatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
		}
	}
}

func TestSanitizeDirName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal name",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "name with colon",
			input:    "example:8080",
			expected: "example_8080",
		},
		{
			name:     "name with special chars",
			input:    "example<>:\"|?*",
			expected: "example_______",
		},
		{
			name:     "name with slashes",
			input:    "example/path\\to",
			expected: "example_path_to",
		},
		{
			name:     "name with leading dots",
			input:    "...hidden",
			expected: "hidden",
		},
		{
			name:     "name with trailing dots",
			input:    "folder...",
			expected: "folder",
		},
		{
			name:     "empty after sanitization",
			input:    "...",
			expected: "scraped_content",
		},
		{
			name:     "very long name",
			input:    "this_is_a_very_long_directory_name_that_exceeds_one_hundred_characters_and_should_be_truncated_to_avoid_filesystem_issues",
			expected: "this_is_a_very_long_directory_name_that_exceeds_one_hundred_characters_and_should_be_truncated_to_av",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeDirName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeDirName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPauseResume(t *testing.T) {
	ctx := context.Background()
	config := Config{
		URL:      "https://example.com",
		MaxDepth: 10,
	}
	c, err := NewCrawler(config, ctx)
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}
	defer c.Close()

	// Initially not paused
	if c.IsPaused() {
		t.Error("crawler should not be paused initially")
	}

	// Pause
	c.Pause()
	if !c.IsPaused() {
		t.Error("crawler should be paused after Pause()")
	}

	// Resume
	c.Resume()
	if c.IsPaused() {
		t.Error("crawler should not be paused after Resume()")
	}
}

func TestExtractReadableContent(t *testing.T) {
	ctx := context.Background()
	config := Config{
		URL:      "https://example.com",
		MaxDepth: 10,
	}
	c, err := NewCrawler(config, ctx)
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}
	defer c.Close()

	tests := []struct {
		name           string
		html           string
		expectError    bool
		expectNonEmpty bool
	}{
		{
			name: "article with clear content",
			html: `<!DOCTYPE html>
<html>
<head><title>Test Article</title></head>
<body>
<header><nav>Navigation</nav></header>
<article>
<h1>Main Article Title</h1>
<p>This is the main content of the article. It contains meaningful text that should be extracted by the readability algorithm. The content needs to be substantial enough to be considered readable.</p>
<p>Here is another paragraph with more content. This helps ensure that the readability algorithm has enough material to work with and can properly identify this as the main content.</p>
</article>
<footer>Footer content</footer>
</body>
</html>`,
			expectError:    false,
			expectNonEmpty: true,
		},
		{
			name: "page with minimal content",
			html: `<!DOCTYPE html>
<html>
<head><title>Empty Page</title></head>
<body>
<p>Short.</p>
</body>
</html>`,
			expectError:    false,
			expectNonEmpty: false, // May return empty or minimal content
		},
		{
			name:           "malformed HTML",
			html:           "<html><body><p>Some content</p>",
			expectError:    false,
			expectNonEmpty: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := c.extractReadableContent("https://example.com/article", tt.html)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectNonEmpty && content == "" {
				t.Error("expected non-empty content")
			}
		})
	}
}

func TestSaveContentWithReadability(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "crawler_readability_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	html := `<!DOCTYPE html>
<html>
<head><title>Test Article</title></head>
<body>
<article>
<h1>Main Article Title</h1>
<p>This is the main content of the article. It contains meaningful text that should be extracted by the readability algorithm. The content needs to be substantial enough to be considered readable.</p>
<p>Here is another paragraph with more content. This helps ensure that the readability algorithm has enough material to work with and can properly identify this as the main content.</p>
</article>
</body>
</html>`

	t.Run("with readability enabled", func(t *testing.T) {
		config := Config{
			URL:                "https://example.com",
			MaxDepth:           10,
			OutputDir:          filepath.Join(tmpDir, "with_readability"),
			DisableReadability: false,
		}
		c, err := NewCrawler(config, ctx)
		if err != nil {
			t.Fatalf("failed to create crawler: %v", err)
		}
		defer c.Close()
		c.log = &Logger{verbose: false}

		err = os.MkdirAll(config.OutputDir, 0755)
		if err != nil {
			t.Fatalf("failed to create output dir: %v", err)
		}

		err = c.saveContent("https://example.com/article", []byte(html))
		if err != nil {
			t.Fatalf("saveContent failed: %v", err)
		}

		// Check that original HTML file exists
		htmlFile := filepath.Join(config.OutputDir, "article.html")
		if _, err := os.Stat(htmlFile); os.IsNotExist(err) {
			t.Error("HTML file was not created")
		}

		// Check that content file exists (readability extracted content)
		contentFile := filepath.Join(config.OutputDir, "article.content.html")
		if _, err := os.Stat(contentFile); os.IsNotExist(err) {
			t.Error("content file was not created")
		}

		// Check metadata file exists and contains readability_extracted field
		metaFile := filepath.Join(config.OutputDir, "article.meta.json")
		metaData, err := os.ReadFile(metaFile)
		if err != nil {
			t.Fatalf("failed to read meta file: %v", err)
		}
		if !strings.Contains(string(metaData), "\"readability_extracted\": true") {
			t.Error("metadata should contain readability_extracted: true")
		}
	})

	t.Run("with readability disabled", func(t *testing.T) {
		config := Config{
			URL:                "https://example.com",
			MaxDepth:           10,
			OutputDir:          filepath.Join(tmpDir, "without_readability"),
			DisableReadability: true,
		}
		c, err := NewCrawler(config, ctx)
		if err != nil {
			t.Fatalf("failed to create crawler: %v", err)
		}
		defer c.Close()
		c.log = &Logger{verbose: false}

		err = os.MkdirAll(config.OutputDir, 0755)
		if err != nil {
			t.Fatalf("failed to create output dir: %v", err)
		}

		err = c.saveContent("https://example.com/article", []byte(html))
		if err != nil {
			t.Fatalf("saveContent failed: %v", err)
		}

		// Check that original HTML file exists
		htmlFile := filepath.Join(config.OutputDir, "article.html")
		if _, err := os.Stat(htmlFile); os.IsNotExist(err) {
			t.Error("HTML file was not created")
		}

		// Check that content file does NOT exist (readability disabled)
		contentFile := filepath.Join(config.OutputDir, "article.content.html")
		if _, err := os.Stat(contentFile); !os.IsNotExist(err) {
			t.Error("content file should not be created when readability is disabled")
		}

		// Check metadata file contains readability_extracted: false
		metaFile := filepath.Join(config.OutputDir, "article.meta.json")
		metaData, err := os.ReadFile(metaFile)
		if err != nil {
			t.Fatalf("failed to read meta file: %v", err)
		}
		if !strings.Contains(string(metaData), "\"readability_extracted\": false") {
			t.Error("metadata should contain readability_extracted: false")
		}
	})
}
