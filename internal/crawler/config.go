package crawler

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// MaxDirNameLength is the maximum length for directory names to avoid filesystem issues
const MaxDirNameLength = 100

// Config holds all configuration options for the crawler
type Config struct {
	URL               string
	Concurrent        bool
	Delay             time.Duration
	MaxDepth          int
	OutputDir         string
	StateFile         string
	PrefixFilterURL   string
	ExcludeExtensions []string
	LinkSelectors     []string
	Verbose           bool
	UserAgent         string
	IgnoreRobots      bool
	MinContentLength  int
	ShowProgress      bool
	MetricsFile       string
}

// ValidateConfig checks that configuration values are valid
func ValidateConfig(config *Config) error {
	// Validate URL
	if config.URL == "" {
		return fmt.Errorf("URL is required")
	}

	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme, got: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	// Validate MaxDepth
	if config.MaxDepth <= 0 {
		return fmt.Errorf("depth must be greater than 0, got: %d", config.MaxDepth)
	}

	// Validate Delay (duration can't be negative from flag parsing, but check anyway)
	if config.Delay < 0 {
		return fmt.Errorf("delay cannot be negative, got: %v", config.Delay)
	}

	// Validate MinContentLength
	if config.MinContentLength < 0 {
		return fmt.Errorf("min-content cannot be negative, got: %d", config.MinContentLength)
	}

	// Validate PrefixFilterURL if provided
	if config.PrefixFilterURL != "" && config.PrefixFilterURL != "none" {
		prefixURL, err := url.Parse(config.PrefixFilterURL)
		if err != nil {
			return fmt.Errorf("invalid prefix-filter URL: %v", err)
		}

		if prefixURL.Scheme != "http" && prefixURL.Scheme != "https" {
			return fmt.Errorf("prefix-filter URL must use http or https scheme, got: %s", prefixURL.Scheme)
		}

		if prefixURL.Host == "" {
			return fmt.Errorf("prefix-filter URL must have a host")
		}
	}

	return nil
}

// SanitizeDirName creates a safe directory name from input
func SanitizeDirName(name string) string {
	// Replace invalid characters for directory names with underscores
	// Invalid characters: < > : " | ? * \ / and control characters
	invalidChars := regexp.MustCompile(`[<>:"|?*\\/\x00-\x1f\x7f]`)
	sanitized := invalidChars.ReplaceAllString(name, "_")

	// Remove leading/trailing dots and spaces (Windows restrictions)
	sanitized = strings.Trim(sanitized, ". ")

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "scraped_content"
	}

	// Limit length to avoid filesystem issues
	if len(sanitized) > MaxDirNameLength {
		sanitized = sanitized[:MaxDirNameLength]
	}

	return sanitized
}

// SetDefaultOutputDir sets the output directory based on URL if not provided
func SetDefaultOutputDir(config *Config) error {
	if config.OutputDir != "" {
		return nil
	}

	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	// Create directory name from domain and path
	dirName := parsedURL.Host
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		pathPart := strings.Trim(parsedURL.Path, "/")
		pathPart = strings.ReplaceAll(pathPart, "/", "_")
		if pathPart != "" {
			dirName += "_" + pathPart
		}
	}

	// Sanitize directory name by removing/replacing invalid characters
	dirName = SanitizeDirName(dirName)
	config.OutputDir = filepath.Join("backup", dirName)

	return nil
}

// SetDefaultStateFile sets the state file name if not provided
func SetDefaultStateFile(config *Config) {
	if config.StateFile != "" {
		return
	}

	// Use just the folder name (without backup path) for state file
	folderName := filepath.Base(config.OutputDir)
	config.StateFile = folderName + "_state.json"
}

// EnsureOutputDir creates the output directory if it doesn't exist
func EnsureOutputDir(config *Config) error {
	return os.MkdirAll(config.OutputDir, 0755)
}
