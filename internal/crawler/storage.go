package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"
)

// extractContent uses trafilatura to extract the main article content and metadata from HTML
func (c *Crawler) extractContent(rawURL string, htmlContent string) (string, *trafilatura.ExtractResult, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse URL: %v", err)
	}

	opts := trafilatura.Options{
		OriginalURL:    parsedURL,
		EnableFallback: true,
	}

	result, err := trafilatura.Extract(strings.NewReader(htmlContent), opts)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract content: %v", err)
	}

	if result == nil || result.ContentNode == nil {
		return "", nil, nil
	}

	// Render the content node to HTML
	var buf bytes.Buffer
	if err := html.Render(&buf, result.ContentNode); err != nil {
		return "", nil, fmt.Errorf("failed to render content: %v", err)
	}

	return buf.String(), result, nil
}

// hasContent checks if an HTML page has meaningful text content
func (c *Crawler) hasContent(html string) bool {
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("Panic in hasContent: %v", r)
		}
	}()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return false
	}

	// Remove script and style elements
	doc.Find("script, style").Remove()

	// Get text content
	text := strings.TrimSpace(doc.Text())

	// Use configured minimum content length, fall back to constant if not set
	minLength := c.config.MinContentLength
	if minLength == 0 {
		minLength = MinContentLength
	}

	// Consider page has content if it has more than minLength characters of text
	return len(text) > minLength
}

// saveContent saves HTML content and metadata to the output directory
func (c *Crawler) saveContent(rawURL string, content []byte) error {
	// Create filename based on URL structure
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL %s: %v", rawURL, err)
	}

	// Generate filename from URL path
	filename := c.generateFilename(parsedURL)

	// Create subdirectories if needed
	fullPath := filepath.Join(c.config.OutputDir, filename)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	// Create metadata file
	metadata := map[string]interface{}{
		"url":       rawURL,
		"timestamp": time.Now().Unix(),
		"size":      len(content),
	}

	// Save original HTML file
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return err
	}

	// Extract and save content if enabled
	contentExtracted := false
	if !c.config.DisableContentExtraction {
		extractedHTML, doc, err := c.extractContent(rawURL, string(content))
		if err != nil {
			c.log.Debug("Failed to extract content for %s: %v", rawURL, err)
		} else if extractedHTML != "" {
			// Save extracted content to .content.html file
			contentFile := strings.TrimSuffix(fullPath, ".html") + ".content.html"
			if err := os.WriteFile(contentFile, []byte(extractedHTML), 0644); err != nil {
				c.log.Debug("Failed to save extracted content for %s: %v", rawURL, err)
			} else {
				contentExtracted = true
				metadata["content_file"] = strings.TrimSuffix(filename, ".html") + ".content.html"
				metadata["content_size"] = len(extractedHTML)
			}

			// Add trafilatura metadata when available
			if doc != nil {
				meta := doc.Metadata
				if meta.Title != "" {
					metadata["title"] = meta.Title
				}
				if meta.Author != "" {
					metadata["author"] = meta.Author
				}
				if !meta.Date.IsZero() {
					metadata["date"] = meta.Date.Format(time.RFC3339)
				}
				if meta.Language != "" {
					metadata["language"] = meta.Language
				}
				if meta.Description != "" {
					metadata["description"] = meta.Description
				}
				if meta.Sitename != "" {
					metadata["sitename"] = meta.Sitename
				}
			}
		}
	}
	metadata["content_extracted"] = contentExtracted

	metaData, _ := json.MarshalIndent(metadata, "", "  ")
	metaFile := strings.TrimSuffix(fullPath, ".html") + ".meta.json"

	return os.WriteFile(metaFile, metaData, 0644)
}

// generateFilename creates a filesystem-safe filename from a URL
func (c *Crawler) generateFilename(parsedURL *url.URL) string {
	path := parsedURL.Path
	query := parsedURL.RawQuery

	// Handle root path
	if path == "" || path == "/" {
		if query != "" {
			return "index_" + sanitizeFilenameComponent(query) + ".html"
		}
		return "index.html"
	}

	// Clean up the path
	path = strings.Trim(path, "/")

	// Replace invalid characters for filenames
	path = sanitizeFilenameComponent(path)

	// Append query parameters if present
	if query != "" {
		// Remove extension temporarily if present
		ext := filepath.Ext(path)
		if ext != "" {
			path = strings.TrimSuffix(path, ext)
			path += "_" + sanitizeFilenameComponent(query) + ext
		} else {
			path += "_" + sanitizeFilenameComponent(query)
		}
	}

	// Add .html extension if it doesn't have an extension
	if !strings.Contains(filepath.Base(path), ".") {
		path += ".html"
	}

	return path
}

// sanitizeFilenameComponent replaces characters invalid in filenames
func sanitizeFilenameComponent(s string) string {
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, "*", "_")
	s = strings.ReplaceAll(s, "<", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, "|", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "&", "_")
	s = strings.ReplaceAll(s, "=", "-")
	return s
}
