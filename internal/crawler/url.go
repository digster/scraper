package crawler

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// URLNormalizer handles URL normalization for deduplication
type URLNormalizer struct {
	lowercasePaths bool
}

// NewURLNormalizer creates a new URL normalizer
func NewURLNormalizer(lowercasePaths bool) *URLNormalizer {
	return &URLNormalizer{
		lowercasePaths: lowercasePaths,
	}
}

// Normalize transforms a URL into its canonical form for deduplication
// It performs the following normalizations:
// - Lowercase scheme and host
// - Remove default ports (:80 for http, :443 for https)
// - Sort query parameters alphabetically
// - Standardize percent encoding (uppercase hex, decode unreserved chars)
// - Remove empty query parameters
// - Normalize trailing slashes (remove from files, keep consistency)
// - Optionally lowercase path
func (n *URLNormalizer) Normalize(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // Return original if parsing fails
	}

	// Lowercase scheme
	scheme := strings.ToLower(parsed.Scheme)

	// Lowercase host and remove default ports
	host := strings.ToLower(parsed.Host)
	host = removeDefaultPort(host, scheme)

	// Get the escaped path (preserves percent-encoding)
	rawPath := parsed.EscapedPath()
	if rawPath == "" {
		rawPath = "/"
	}

	// Uppercase percent encoding in the path
	path := uppercasePercentEncoding(rawPath)

	// Optionally lowercase the path (but not the percent-encoded parts)
	if n.lowercasePaths {
		path = lowercasePathPreservingEncoding(path)
	}

	// Normalize trailing slash for paths (not for root)
	// Remove trailing slash unless it's the root path
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	// Normalize query string
	query := ""
	if parsed.RawQuery != "" {
		query = normalizeQuery(parsed.RawQuery)
	}

	// Build the normalized URL string manually to preserve path encoding
	var result strings.Builder
	result.WriteString(scheme)
	result.WriteString("://")
	result.WriteString(host)
	result.WriteString(path)
	if query != "" {
		result.WriteString("?")
		result.WriteString(query)
	}
	// Fragment is intentionally omitted for deduplication

	return result.String()
}

// removeDefaultPort removes the default port for HTTP/HTTPS
func removeDefaultPort(host, scheme string) string {
	switch scheme {
	case "http":
		return strings.TrimSuffix(host, ":80")
	case "https":
		return strings.TrimSuffix(host, ":443")
	default:
		return host
	}
}

// uppercasePercentEncoding converts percent-encoded sequences to uppercase
func uppercasePercentEncoding(s string) string {
	// Match percent-encoded sequences like %2f, %3a
	re := regexp.MustCompile(`%[0-9a-fA-F]{2}`)
	return re.ReplaceAllStringFunc(s, strings.ToUpper)
}

// lowercasePathPreservingEncoding lowercases the path but preserves percent-encoded hex as uppercase
func lowercasePathPreservingEncoding(path string) string {
	// First lowercase everything
	lower := strings.ToLower(path)
	// Then uppercase the percent-encoded sequences
	return uppercasePercentEncoding(lower)
}

// normalizeQuery sorts query parameters and removes empty ones
func normalizeQuery(rawQuery string) string {
	// Parse query string
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}

	// Remove empty parameters
	for key, vals := range values {
		// Filter out empty values
		nonEmpty := make([]string, 0, len(vals))
		for _, v := range vals {
			if v != "" {
				nonEmpty = append(nonEmpty, v)
			}
		}
		if len(nonEmpty) == 0 {
			delete(values, key)
		} else {
			values[key] = nonEmpty
		}
	}

	// Also remove parameters with empty keys
	delete(values, "")

	if len(values) == 0 {
		return ""
	}

	// Get sorted keys
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build sorted query string
	parts := make([]string, 0)
	for _, k := range keys {
		vals := values[k]
		sort.Strings(vals) // Sort values for the same key
		for _, v := range vals {
			// Encode key and value with uppercase percent encoding
			encodedKey := uppercasePercentEncoding(url.QueryEscape(k))
			encodedVal := uppercasePercentEncoding(url.QueryEscape(v))
			parts = append(parts, encodedKey+"="+encodedVal)
		}
	}

	return strings.Join(parts, "&")
}

// NormalizeURL is a convenience function that normalizes a URL with default settings
// (lowercasePaths = false for conservative behavior)
func NormalizeURL(rawURL string) string {
	n := NewURLNormalizer(false)
	return n.Normalize(rawURL)
}
