package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	URL              string
	Concurrent       bool
	Delay            time.Duration
	MaxDepth         int
	OutputDir        string
	StateFile        string
	DisablePrefixFilter bool
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
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	
	return sanitized
}

func main() {
	var config Config
	
	flag.StringVar(&config.URL, "url", "", "Starting URL to scrape")
	flag.BoolVar(&config.Concurrent, "concurrent", false, "Run in concurrent mode")
	flag.DurationVar(&config.Delay, "delay", time.Second, "Delay between fetches")
	flag.IntVar(&config.MaxDepth, "depth", 10, "Maximum crawl depth")
	flag.StringVar(&config.OutputDir, "output", "", "Output directory (defaults to URL-based name)")
	flag.StringVar(&config.StateFile, "state", "crawler_state.json", "State file for resume functionality")
	flag.BoolVar(&config.DisablePrefixFilter, "disable-prefix-filter", false, "Disable URL prefix filtering (allows crawling outside input URL prefix)")
	flag.Parse()

	if config.URL == "" {
		fmt.Println("Error: URL is required")
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
		config.OutputDir = dirName
	}

	crawler := NewCrawler(config)
	if err := crawler.Start(); err != nil {
		log.Fatal(err)
	}
}