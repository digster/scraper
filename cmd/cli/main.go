package main

import (
	"bufio"
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
	var fetchMode string
	var paginationWait string

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
	flag.StringVar(&fetchMode, "fetch-mode", "http", "Fetch mode: 'http' for standard HTTP client, 'browser' for real browser via chromedp")
	flag.BoolVar(&config.Headless, "headless", true, "Run browser in headless mode (only applies when fetch-mode=browser)")
	flag.BoolVar(&config.WaitForLogin, "wait-login", false, "Wait for manual login before crawling (only applies when fetch-mode=browser and headless=false)")

	// Pagination flags (only apply when fetch-mode=browser)
	flag.BoolVar(&config.Pagination.Enable, "enable-pagination", false, "Enable click-based pagination (requires fetch-mode=browser)")
	flag.StringVar(&config.Pagination.Selector, "pagination-selector", "", "CSS selector for pagination element (e.g., 'a.next', '.load-more')")
	flag.IntVar(&config.Pagination.MaxClicks, "max-pagination-clicks", 100, "Maximum number of pagination clicks")
	flag.StringVar(&paginationWait, "pagination-wait", "2s", "Time to wait after each pagination click (e.g., 2s, 500ms)")
	flag.StringVar(&config.Pagination.WaitSelector, "pagination-wait-selector", "", "CSS selector to wait for after pagination click")
	flag.BoolVar(&config.Pagination.StopOnDuplicate, "pagination-stop-duplicate", true, "Stop pagination if duplicate content is detected")

	// Anti-bot bypass flags (only apply when fetch-mode=browser and headless=false)
	flag.BoolVar(&config.AntiBot.HideWebdriver, "hide-webdriver", false, "Hide navigator.webdriver flag")
	flag.BoolVar(&config.AntiBot.SpoofPlugins, "spoof-plugins", false, "Inject realistic browser plugins")
	flag.BoolVar(&config.AntiBot.SpoofLanguages, "spoof-languages", false, "Set realistic navigator.languages")
	flag.BoolVar(&config.AntiBot.SpoofWebGL, "spoof-webgl", false, "Override WebGL vendor/renderer")
	flag.BoolVar(&config.AntiBot.AddCanvasNoise, "canvas-noise", false, "Add noise to canvas fingerprint")
	flag.BoolVar(&config.AntiBot.NaturalMouseMovement, "natural-mouse", false, "Use Bezier curve mouse movements")
	flag.BoolVar(&config.AntiBot.RandomTypingDelays, "typing-delays", false, "Add random typing delays")
	flag.BoolVar(&config.AntiBot.NaturalScrolling, "natural-scroll", false, "Use momentum-based scrolling")
	flag.BoolVar(&config.AntiBot.RandomActionDelays, "action-delays", false, "Add jittered action delays")
	flag.BoolVar(&config.AntiBot.RandomClickOffset, "click-offset", false, "Randomize click positions")
	flag.BoolVar(&config.AntiBot.RotateUserAgent, "rotate-ua", false, "Rotate through user agents")
	flag.BoolVar(&config.AntiBot.RandomViewport, "random-viewport", false, "Use random viewport sizes")
	flag.BoolVar(&config.AntiBot.MatchTimezone, "match-timezone", false, "Enable timezone override")
	flag.StringVar(&config.AntiBot.Timezone, "timezone", "", "Timezone to use (e.g., America/New_York)")

	// URL normalization flags
	normalizeURLs := flag.Bool("normalize-urls", true, "Enable URL normalization for better duplicate detection")
	lowercasePaths := flag.Bool("lowercase-paths", false, "Lowercase URL paths during normalization (use with caution)")
	flag.Parse()

	// Set URL normalization options
	config.NormalizeURLs = *normalizeURLs
	config.LowercasePaths = *lowercasePaths

	// Set fetch mode
	config.FetchMode = crawler.FetchMode(fetchMode)

	// Parse pagination wait duration
	if config.Pagination.Enable {
		waitDuration, err := time.ParseDuration(paginationWait)
		if err != nil {
			waitDuration = 2 * time.Second
		}
		config.Pagination.WaitAfterClick = waitDuration
	}

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

	c, err := crawler.NewCrawler(config, ctx)
	if err != nil {
		log.Fatal("Failed to create crawler:", err)
	}
	defer c.Close()

	// If wait-login is enabled, set up a goroutine to wait for Enter key
	if config.WaitForLogin && config.FetchMode == crawler.FetchModeBrowser && !config.Headless {
		go func() {
			// Give the crawler a moment to start and enter login wait state
			time.Sleep(500 * time.Millisecond)

			// Check if the crawler is waiting for login
			for c.IsWaitingForLogin() {
				fmt.Println("\nBrowser opened. Complete login, then press ENTER to start crawling...")
				reader := bufio.NewReader(os.Stdin)
				_, _ = reader.ReadString('\n')
				c.ConfirmLogin()
				break
			}
		}()
	}

	if err := c.Start(); err != nil {
		log.Fatal(err)
	}
}
