package main

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateFilename(t *testing.T) {
	c := &Crawler{}

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
	c := &Crawler{}

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
		name            string
		rawURL          string
		prefixFilter    string
		excludeExts     []string
		expected        bool
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
			c := &Crawler{}
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
	c := &Crawler{
		config: Config{
			URL:       "https://example.com",
			StateFile: stateFile,
		},
	}

	// Initialize fresh state
	err = c.loadState()
	if err != nil {
		t.Fatalf("loadState() failed: %v", err)
	}

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
	err = c.saveState()
	if err != nil {
		t.Fatalf("saveState() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatal("state file was not created")
	}

	// Create new crawler and load state
	c2 := &Crawler{
		config: Config{
			URL:       "https://example.com",
			StateFile: stateFile,
		},
	}

	err = c2.loadState()
	if err != nil {
		t.Fatalf("loadState() for c2 failed: %v", err)
	}

	// Verify loaded state
	if len(c2.state.Visited) != 2 {
		t.Errorf("Visited count = %d, want 2", len(c2.state.Visited))
	}
	if !c2.state.Visited["https://example.com/page1"] {
		t.Error("page1 should be in Visited")
	}
	if !c2.state.Visited["https://example.com/page2"] {
		t.Error("page2 should be in Visited")
	}
	if len(c2.state.Queue) != 1 {
		t.Errorf("Queue length = %d, want 1", len(c2.state.Queue))
	}
	if c2.state.Queue[0].URL != "https://example.com/page3" {
		t.Errorf("Queue[0].URL = %q, want %q", c2.state.Queue[0].URL, "https://example.com/page3")
	}
	if c2.state.Processed != 2 {
		t.Errorf("Processed = %d, want 2", c2.state.Processed)
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
	c := &Crawler{
		config: Config{
			URL:       "https://example.com",
			StateFile: stateFile,
		},
	}

	err = c.loadState()
	if err != nil {
		t.Fatalf("loadState() failed: %v", err)
	}

	// Verify Queued map was initialized from queue
	if c.state.Queued == nil {
		t.Fatal("Queued map should be initialized")
	}
	if len(c.state.Queue) != 1 {
		t.Fatalf("Queue should have 1 item, got %d", len(c.state.Queue))
	}
	if c.state.Queue[0].URL != "https://example.com/page2" {
		t.Errorf("Queue[0].URL = %q, want %q", c.state.Queue[0].URL, "https://example.com/page2")
	}
	if !c.state.Queued["https://example.com/page2"] {
		t.Error("Queued should contain page2 from queue")
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
			result := sanitizeDirName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeDirName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
