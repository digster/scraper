package crawler

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Query parameter sorting
		{
			name:     "sort query parameters alphabetically",
			input:    "https://example.com/page?b=2&a=1",
			expected: "https://example.com/page?a=1&b=2",
		},
		{
			name:     "sort query parameters with multiple values",
			input:    "https://example.com/page?z=3&a=2&a=1",
			expected: "https://example.com/page?a=1&a=2&z=3",
		},

		// Trailing slash normalization
		{
			name:     "remove trailing slash from path",
			input:    "https://example.com/page/",
			expected: "https://example.com/page",
		},
		{
			name:     "keep single slash for root",
			input:    "https://example.com/",
			expected: "https://example.com/",
		},
		{
			name:     "add root slash if missing",
			input:    "https://example.com",
			expected: "https://example.com/",
		},

		// Default port removal
		{
			name:     "remove default HTTP port",
			input:    "http://example.com:80/page",
			expected: "http://example.com/page",
		},
		{
			name:     "remove default HTTPS port",
			input:    "https://example.com:443/page",
			expected: "https://example.com/page",
		},
		{
			name:     "keep non-default port",
			input:    "https://example.com:8443/page",
			expected: "https://example.com:8443/page",
		},

		// Host case normalization
		{
			name:     "lowercase host",
			input:    "https://EXAMPLE.COM/page",
			expected: "https://example.com/page",
		},
		{
			name:     "lowercase scheme",
			input:    "HTTPS://example.com/page",
			expected: "https://example.com/page",
		},

		// Encoding normalization
		{
			name:     "uppercase percent encoding",
			input:    "https://example.com/pa%2fge",
			expected: "https://example.com/pa%2Fge",
		},
		{
			name:     "normalize space encoding in query",
			input:    "https://example.com/page?q=hello+world",
			expected: "https://example.com/page?q=hello+world",
		},
		{
			name:     "normalize percent-encoded space in query",
			input:    "https://example.com/page?q=hello%20world",
			expected: "https://example.com/page?q=hello+world",
		},

		// Empty query parameter removal
		{
			name:     "remove empty query parameter value",
			input:    "https://example.com/page?a=&b=2",
			expected: "https://example.com/page?b=2",
		},
		{
			name:     "remove all empty query parameters",
			input:    "https://example.com/page?a=&b=",
			expected: "https://example.com/page",
		},

		// Fragment removal
		{
			name:     "remove fragment",
			input:    "https://example.com/page#section",
			expected: "https://example.com/page",
		},
		{
			name:     "remove fragment with query",
			input:    "https://example.com/page?q=1#section",
			expected: "https://example.com/page?q=1",
		},

		// Combined normalizations
		{
			name:     "complex URL with multiple normalizations",
			input:    "HTTPS://EXAMPLE.COM:443/PATH/?z=3&a=1&b=2#fragment",
			expected: "https://example.com/PATH?a=1&b=2&z=3",
		},
		{
			name:     "URL with encoded characters and unsorted params",
			input:    "https://example.com/search?q=hello%20world&sort=desc&page=1",
			expected: "https://example.com/search?page=1&q=hello+world&sort=desc",
		},

		// Edge cases
		{
			name:     "URL with only query string",
			input:    "https://example.com?a=1",
			expected: "https://example.com/?a=1",
		},
		{
			name:     "URL with duplicate query params same value",
			input:    "https://example.com/page?a=1&a=1",
			expected: "https://example.com/page?a=1&a=1",
		},
		{
			name:     "URL with duplicate query params different values",
			input:    "https://example.com/page?a=2&a=1",
			expected: "https://example.com/page?a=1&a=2",
		},
		{
			name:     "URL with special characters in path",
			input:    "https://example.com/path/with spaces/file.html",
			expected: "https://example.com/path/with%20spaces/file.html",
		},
	}

	normalizer := NewURLNormalizer(false)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeURLWithLowercasePaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase path when enabled",
			input:    "https://example.com/Path/To/Page",
			expected: "https://example.com/path/to/page",
		},
		{
			name:     "lowercase path with query",
			input:    "https://example.com/PATH?Query=Value",
			expected: "https://example.com/path?Query=Value",
		},
	}

	normalizer := NewURLNormalizer(true) // Enable lowercase paths

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeURLPreserveCase(t *testing.T) {
	// By default, path case should be preserved
	normalizer := NewURLNormalizer(false)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "preserve path case",
			input:    "https://example.com/Path/To/Page",
			expected: "https://example.com/Path/To/Page",
		},
		{
			name:     "preserve mixed case path",
			input:    "https://example.com/API/v2/Users",
			expected: "https://example.com/API/v2/Users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sort simple params",
			input:    "c=3&b=2&a=1",
			expected: "a=1&b=2&c=3",
		},
		{
			name:     "remove empty values",
			input:    "a=1&b=&c=3",
			expected: "a=1&c=3",
		},
		{
			name:     "all empty values",
			input:    "a=&b=",
			expected: "",
		},
		{
			name:     "duplicate keys sorted",
			input:    "a=2&a=1&a=3",
			expected: "a=1&a=2&a=3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeQuery(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeQuery(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRemoveDefaultPort(t *testing.T) {
	tests := []struct {
		host     string
		scheme   string
		expected string
	}{
		{"example.com:80", "http", "example.com"},
		{"example.com:443", "https", "example.com"},
		{"example.com:8080", "http", "example.com:8080"},
		{"example.com:8443", "https", "example.com:8443"},
		{"example.com", "http", "example.com"},
		{"example.com", "https", "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.host+"_"+tt.scheme, func(t *testing.T) {
			result := removeDefaultPort(tt.host, tt.scheme)
			if result != tt.expected {
				t.Errorf("removeDefaultPort(%q, %q) = %q, want %q", tt.host, tt.scheme, result, tt.expected)
			}
		})
	}
}

func TestUppercasePercentEncoding(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"%2f", "%2F"},
		{"%2F", "%2F"},
		{"hello%20world", "hello%20world"},
		{"hello%2fworld%3a", "hello%2Fworld%3A"},
		{"no-encoding", "no-encoding"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := uppercasePercentEncoding(tt.input)
			if result != tt.expected {
				t.Errorf("uppercasePercentEncoding(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeURLConvenienceFunction(t *testing.T) {
	// Test the convenience function uses conservative defaults
	input := "https://Example.COM:443/Path?b=2&a=1#frag"
	expected := "https://example.com/Path?a=1&b=2"

	result := NormalizeURL(input)
	if result != expected {
		t.Errorf("NormalizeURL(%q) = %q, want %q", input, result, expected)
	}
}

func TestNormalizeURLInvalidURL(t *testing.T) {
	// Invalid URL should return unchanged
	invalid := "://not-a-valid-url"
	result := NormalizeURL(invalid)
	if result != invalid {
		t.Errorf("NormalizeURL(%q) = %q, want %q (unchanged)", invalid, result, invalid)
	}
}

// Benchmark URL normalization
func BenchmarkNormalizeURL(b *testing.B) {
	normalizer := NewURLNormalizer(false)
	url := "https://example.com/path/to/page?z=3&a=1&b=2&c=test%20value#fragment"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(url)
	}
}
