package main

import (
	"net/url"
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
