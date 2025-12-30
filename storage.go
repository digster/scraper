package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

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

	metaData, _ := json.MarshalIndent(metadata, "", "  ")
	metaFile := strings.TrimSuffix(fullPath, ".html") + ".meta.json"

	// Save both files
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return err
	}

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
