package crawler

import (
	"net/url"
	"path/filepath"
	"strings"
)

// isValidURL checks if a URL should be crawled based on scheme, extension, and prefix filter
func (c *Crawler) isValidURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Check if URL extension should be excluded
	if c.shouldExcludeByExtension(parsed.Path) {
		return false
	}

	// Must be HTTP/HTTPS
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	// If prefix filtering is disabled (empty or "none"), allow any HTTP/HTTPS URL discovered through the tree
	if c.config.PrefixFilterURL == "" || c.config.PrefixFilterURL == "none" {
		return true
	}

	// With prefix filtering enabled: check URL prefix constraint using the specified prefix filter URL
	prefixURL, err := url.Parse(c.config.PrefixFilterURL)
	if err != nil {
		return false
	}

	// Check if URL has the prefix URL as prefix
	if parsed.Host != prefixURL.Host {
		return false
	}

	// Check if path starts with prefix path
	prefixPath := strings.TrimSuffix(prefixURL.Path, "/")
	urlPath := strings.TrimSuffix(parsed.Path, "/")

	return strings.HasPrefix(urlPath, prefixPath)
}

// shouldExcludeByExtension checks if a URL path has an excluded file extension
func (c *Crawler) shouldExcludeByExtension(path string) bool {
	if len(c.config.ExcludeExtensions) == 0 {
		return false
	}

	// Extract extension from path
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		// Remove the dot from extension
		ext = ext[1:]
	}

	// Check if extension is in exclude list
	for _, excludeExt := range c.config.ExcludeExtensions {
		if ext == excludeExt {
			return true
		}
	}

	return false
}

// shouldExcludeByContentType checks if content should be excluded based on its Content-Type header
func (c *Crawler) shouldExcludeByContentType(contentType string) bool {
	if len(c.config.ExcludeExtensions) == 0 {
		return false
	}

	// Convert content type to lowercase for comparison
	contentType = strings.ToLower(contentType)

	// Remove charset and other parameters (e.g., "application/json; charset=utf-8" -> "application/json")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	// Comprehensive mapping of content types to extensions
	contentTypeToExt := map[string]string{
		// Common web assets
		"application/json":       "json",
		"text/javascript":        "js",
		"application/javascript": "js",
		"text/css":               "css",

		// Images
		"image/png":     "png",
		"image/jpeg":    "jpg",
		"image/jpg":     "jpg",
		"image/gif":     "gif",
		"image/webp":    "webp",
		"image/svg+xml": "svg",
		"image/bmp":     "bmp",
		"image/tiff":    "tiff",
		"image/ico":     "ico",

		// Documents
		"application/pdf":     "pdf",
		"application/msword":  "doc",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   "docx",
		"application/vnd.ms-excel":                                                  "xls",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "xlsx",
		"application/vnd.ms-powerpoint":                                             "ppt",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",

		// Archives
		"application/zip":              "zip",
		"application/x-rar-compressed": "rar",
		"application/x-tar":            "tar",
		"application/gzip":             "gz",
		"application/x-7z-compressed":  "7z",

		// Data formats
		"application/xml": "xml",
		"text/xml":        "xml",
		"text/csv":        "csv",
		"application/yaml": "yaml",
		"text/yaml":        "yaml",

		// Media
		"video/mp4":       "mp4",
		"video/mpeg":      "mpeg",
		"video/quicktime": "mov",
		"video/x-msvideo": "avi",
		"audio/mpeg":      "mp3",
		"audio/wav":       "wav",
		"audio/ogg":       "ogg",

		// Fonts
		"font/woff":            "woff",
		"font/woff2":           "woff2",
		"application/font-woff":  "woff",
		"application/font-woff2": "woff2",
		"font/ttf":             "ttf",
		"font/otf":             "otf",
	}

	// First check exact mapping
	if ext, exists := contentTypeToExt[contentType]; exists {
		for _, excludeExt := range c.config.ExcludeExtensions {
			if ext == excludeExt {
				return true
			}
		}
	}

	// For unmapped content types, try to infer from the content type string
	// e.g., "application/vnd.company.customformat" might contain the extension
	for _, excludeExt := range c.config.ExcludeExtensions {
		if strings.Contains(contentType, excludeExt) {
			return true
		}
	}

	return false
}
