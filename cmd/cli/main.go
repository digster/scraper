package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"scraper/internal/crawler"
)

func main() {
	var config crawler.Config
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
	flag.BoolVar(&config.DisableReadability, "no-readability", false, "Disable readability content extraction (extracts main article content by default)")
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
	if err := crawler.ValidateConfig(&config); err != nil {
		fmt.Printf("Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// Set default output directory
	if err := crawler.SetDefaultOutputDir(&config); err != nil {
		log.Fatal("Invalid URL:", err)
	}

	// Set default state file
	crawler.SetDefaultStateFile(&config)

	// Set up signal handling for graceful shutdown
	ctx, cancel := crawler.SetupSignalHandler()
	defer cancel()

	c := crawler.NewCrawler(config, ctx)
	if err := c.Start(); err != nil {
		log.Fatal(err)
	}
}
