package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// MaxDirNameLength is the maximum length for directory names to avoid filesystem issues
const MaxDirNameLength = 100

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

// validateConfig checks that configuration values are valid
func validateConfig(config *Config) error {
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

func sanitizeDirName(name string) string {
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

func main() {
	var config Config
	var excludeExtensions string
	var linkSelectors string

	flag.StringVar(&config.URL, "url", "", "Starting URL to scrape")
	flag.BoolVar(&config.Concurrent, "concurrent", false, "Run in concurrent mode")
	flag.DurationVar(&config.Delay, "delay", time.Second, "Delay between fetches")
	flag.IntVar(&config.MaxDepth, "depth", 10, "Maximum crawl depth")
	flag.StringVar(&config.OutputDir, "output", "", "Output directory (defaults to URL-based name)")
	flag.StringVar(&config.StateFile, "state", "", "State file for resume functionality (defaults to folder name)")
	flag.StringVar(&config.PrefixFilterURL, "prefix-filter", "", "URL prefix to filter by (if not specified, no prefix filtering is applied)")
	flag.StringVar(&excludeExtensions, "exclude-extensions", "", "Comma-separated list of asset extensions to exclude (e.g., js,css,png)")
	flag.StringVar(&linkSelectors, "link-selectors", "", "Comma-separated list of CSS selectors to filter links (e.g., 'a.internal,.nav-link')")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose debug output")
	flag.StringVar(&config.UserAgent, "user-agent", "", "Custom User-Agent header (defaults to WebScraper/1.0)")
	flag.BoolVar(&config.IgnoreRobots, "ignore-robots", false, "Ignore robots.txt rules")
	flag.IntVar(&config.MinContentLength, "min-content", 100, "Minimum text content length (characters) for a page to be saved")
	flag.BoolVar(&config.ShowProgress, "progress", true, "Show progress bar and statistics")
	flag.StringVar(&config.MetricsFile, "metrics-json", "", "Output final metrics to JSON file")
	flag.Parse()

	// Parse exclude extensions
	if excludeExtensions != "" {
		config.ExcludeExtensions = strings.Split(excludeExtensions, ",")
		for i, ext := range config.ExcludeExtensions {
			config.ExcludeExtensions[i] = strings.TrimSpace(strings.ToLower(ext))
		}
	}

	// Parse link selectors
	if linkSelectors != "" {
		config.LinkSelectors = strings.Split(linkSelectors, ",")
		for i, selector := range config.LinkSelectors {
			config.LinkSelectors[i] = strings.TrimSpace(selector)
		}
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		fmt.Printf("Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// Generate output directory name from URL if not provided
	if config.OutputDir == "" {
		parsedURL, err := url.Parse(config.URL)
		if err != nil {
			log.Fatal("Invalid URL:", err)
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
		dirName = sanitizeDirName(dirName)
		config.OutputDir = filepath.Join("backup", dirName)
	}

	// Generate state file name if not provided
	if config.StateFile == "" {
		// Use just the folder name (without backup path) for state file
		folderName := filepath.Base(config.OutputDir)
		config.StateFile = folderName + "_state.json"
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := SetupSignalHandler()
	defer cancel()

	crawler := NewCrawler(config, ctx)
	if err := crawler.Start(); err != nil {
		log.Fatal(err)
	}
}
