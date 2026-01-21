package crawler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// BrowserFetcher implements Fetcher using a real browser via chromedp
type BrowserFetcher struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	browserCtx  context.Context
	cancelFunc  context.CancelFunc
	headless    bool
	userAgent   string
	antiBot     AntiBotConfig
	userAgents  []string
	uaIndex     int
	uaMu        sync.Mutex
}

// NewBrowserFetcher creates a new browser-based fetcher
func NewBrowserFetcher(headless bool) (*BrowserFetcher, error) {
	return NewBrowserFetcherWithUserAgent(headless, DefaultUserAgent)
}

// NewBrowserFetcherWithUserAgent creates a new browser-based fetcher with custom user agent
func NewBrowserFetcherWithUserAgent(headless bool, userAgent string) (*BrowserFetcher, error) {
	return NewBrowserFetcherWithAntiBot(headless, userAgent, AntiBotConfig{})
}

// NewBrowserFetcherWithAntiBot creates a new browser-based fetcher with anti-bot configuration
func NewBrowserFetcherWithAntiBot(headless bool, userAgent string, antiBot AntiBotConfig) (*BrowserFetcher, error) {
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(userAgent),
	)

	// Anti-bot: Hide automation indicators
	if antiBot.HideWebdriver {
		opts = append(opts,
			chromedp.Flag("disable-blink-features", "AutomationControlled"),
		)
	}

	// Anti-bot: Random viewport
	var viewport *Viewport
	if antiBot.RandomViewport {
		viewport = GetRandomViewport()
		opts = append(opts,
			chromedp.WindowSize(viewport.Width, viewport.Height),
		)
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	browserCtx, cancelFunc := chromedp.NewContext(allocCtx)

	// Start browser
	if err := chromedp.Run(browserCtx); err != nil {
		allocCancel()
		return nil, fmt.Errorf("failed to start browser: %w", err)
	}

	// Anti-bot: Set timezone if configured
	if antiBot.MatchTimezone && antiBot.Timezone != "" {
		if err := chromedp.Run(browserCtx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return emulation.SetTimezoneOverride(antiBot.Timezone).Do(ctx)
			}),
		); err != nil {
			// Log but don't fail - timezone override is non-critical
			fmt.Printf("Warning: failed to set timezone override: %v\n", err)
		}
	}

	// Set up user agent rotation if enabled
	var userAgentPool []string
	if antiBot.RotateUserAgent {
		userAgentPool = GetChromeUserAgents()
	}

	return &BrowserFetcher{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		browserCtx:  browserCtx,
		cancelFunc:  cancelFunc,
		headless:    headless,
		userAgent:   userAgent,
		antiBot:     antiBot,
		userAgents:  userAgentPool,
		uaIndex:     0,
	}, nil
}

// GetNextUserAgent returns the next user agent in rotation (thread-safe)
func (f *BrowserFetcher) GetNextUserAgent() string {
	if len(f.userAgents) == 0 {
		return f.userAgent
	}

	f.uaMu.Lock()
	defer f.uaMu.Unlock()

	ua := f.userAgents[f.uaIndex]
	f.uaIndex = (f.uaIndex + 1) % len(f.userAgents)
	return ua
}

// Fetch retrieves a URL using the browser
func (f *BrowserFetcher) Fetch(rawURL string, userAgent string) (*FetchResult, error) {
	// Create a new tab context for this request
	tabCtx, cancel := chromedp.NewContext(f.browserCtx)
	defer cancel()

	// Set timeout for the page load
	tabCtx, cancelTimeout := context.WithTimeout(tabCtx, HTTPTimeout)
	defer cancelTimeout()

	var html string
	var finalURL string
	var statusCode int
	var contentType string

	// Set up response listener to capture status code and content type
	chromedp.ListenTarget(tabCtx, func(ev interface{}) {
		if resp, ok := ev.(*network.EventResponseReceived); ok {
			if resp.Type == network.ResourceTypeDocument {
				statusCode = int(resp.Response.Status)
				contentType = resp.Response.MimeType
			}
		}
	})

	// Build actions - user agent is set at browser startup via allocator options
	// The userAgent parameter is ignored here as it's configured when creating the fetcher
	_ = userAgent

	// Inject anti-bot scripts before navigation
	scripts := BuildInjectionScripts(f.antiBot)

	var actions []chromedp.Action

	// First, inject scripts to run on new documents
	if len(scripts) > 0 {
		actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
			for _, script := range scripts {
				_, err := page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
				if err != nil {
					return fmt.Errorf("failed to inject anti-bot script: %w", err)
				}
			}
			return nil
		}))
	}

	// Add random action delay if enabled
	if f.antiBot.RandomActionDelays {
		actions = append(actions, chromedp.Sleep(RandomActionDelay()))
	}

	// Core navigation actions
	actions = append(actions,
		network.Enable(),
		chromedp.Navigate(rawURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond), // Small delay for dynamic content
		chromedp.Location(&finalURL),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)

	err := chromedp.Run(tabCtx, actions...)
	if err != nil {
		// Check if it's a navigation error that might still have some content
		if strings.Contains(err.Error(), "net::ERR_") {
			return nil, fmt.Errorf("navigation failed: %w", err)
		}
		return nil, fmt.Errorf("browser fetch failed: %w", err)
	}

	// Default status code if not captured
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	// Default content type
	if contentType == "" {
		contentType = "text/html"
	}

	return &FetchResult{
		Body:        []byte(html),
		StatusCode:  statusCode,
		ContentType: contentType,
		FinalURL:    finalURL,
	}, nil
}

// NavigateForLogin opens a URL in the browser and returns a cancel function to close the tab.
// This is used for manual login - the tab stays open until the cancel function is called.
// Session data (cookies, etc.) will persist in the browser context for subsequent fetches.
func (f *BrowserFetcher) NavigateForLogin(rawURL string) (context.CancelFunc, error) {
	// Create a new tab context for login
	tabCtx, cancel := chromedp.NewContext(f.browserCtx)

	// Set timeout for the page load (but not for the overall login wait)
	loadCtx, loadCancel := context.WithTimeout(tabCtx, HTTPTimeout)
	defer loadCancel()

	// Navigate to the URL
	err := chromedp.Run(loadCtx,
		network.Enable(),
		chromedp.Navigate(rawURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to navigate for login: %w", err)
	}

	// Return the cancel function - caller should call it after login is complete
	return cancel, nil
}

// Close releases browser resources
func (f *BrowserFetcher) Close() error {
	f.cancelFunc()
	f.allocCancel()
	return nil
}

// PageCallback is called for each page fetched during pagination
// It receives the page content, page number (1-indexed), and a virtual URL with page parameter
// Return an error to stop pagination early
type PageCallback func(result *FetchResult, pageNumber int, virtualURL string) error

// PaginatedFetchResult contains the result of a paginated fetch operation
type PaginatedFetchResult struct {
	TotalPages       int    // Total number of pages fetched
	ExhaustedReason  string // Reason pagination stopped (if applicable)
	LastError        error  // Last error encountered (if any)
}

// FetchWithPagination fetches a URL and handles click-based pagination
// It calls the callback for each page of content (including the initial page)
func (f *BrowserFetcher) FetchWithPagination(rawURL string, userAgent string, config PaginationConfig, callback PageCallback) (*PaginatedFetchResult, error) {
	result := &PaginatedFetchResult{
		TotalPages: 0,
	}

	// Create a persistent tab context for the entire pagination session
	tabCtx, cancel := chromedp.NewContext(f.browserCtx)
	defer cancel()

	// Set timeout for the entire pagination operation
	// Use a longer timeout: base timeout + (waitAfterClick * maxClicks)
	totalTimeout := HTTPTimeout + (config.WaitAfterClick * time.Duration(config.MaxClicks))
	tabCtx, cancelTimeout := context.WithTimeout(tabCtx, totalTimeout)
	defer cancelTimeout()

	var statusCode int
	var contentType string

	// Set up response listener
	chromedp.ListenTarget(tabCtx, func(ev interface{}) {
		if resp, ok := ev.(*network.EventResponseReceived); ok {
			if resp.Type == network.ResourceTypeDocument {
				statusCode = int(resp.Response.Status)
				contentType = resp.Response.MimeType
			}
		}
	})

	// Build initial navigation actions with anti-bot scripts
	scripts := BuildInjectionScripts(f.antiBot)
	var actions []chromedp.Action

	if len(scripts) > 0 {
		actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
			for _, script := range scripts {
				_, err := page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
				if err != nil {
					return fmt.Errorf("failed to inject anti-bot script: %w", err)
				}
			}
			return nil
		}))
	}

	// Add random action delay if enabled
	if f.antiBot.RandomActionDelays {
		actions = append(actions, chromedp.Sleep(RandomActionDelay()))
	}

	// Navigate to the initial URL
	actions = append(actions,
		network.Enable(),
		chromedp.Navigate(rawURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)

	if err := chromedp.Run(tabCtx, actions...); err != nil {
		return result, fmt.Errorf("initial navigation failed: %w", err)
	}

	// Initialize pagination state
	paginationState := NewPaginationState(config, f.antiBot)

	// Process the initial page (page 1)
	initialResult, err := f.fetchCurrentPage(tabCtx, rawURL, statusCode, contentType)
	if err != nil {
		return result, fmt.Errorf("failed to fetch initial page: %w", err)
	}

	// Get initial content hash
	initialHash, err := getContentHash(tabCtx)
	if err != nil {
		return result, fmt.Errorf("failed to get initial content hash: %w", err)
	}
	paginationState.SeenHashes[initialHash] = true
	paginationState.ContentHash = initialHash

	result.TotalPages = 1

	// Call callback for initial page
	if err := callback(initialResult, 1, rawURL); err != nil {
		return result, err
	}

	// Pagination loop
	for paginationState.CanContinue() {
		// Check context cancellation
		select {
		case <-tabCtx.Done():
			result.ExhaustedReason = "context cancelled"
			return result, nil
		default:
		}

		// Attempt to click pagination
		clickResult, err := ClickPagination(tabCtx, config.Selector, config, paginationState.Behavior)
		if err != nil {
			result.LastError = err
			result.ExhaustedReason = fmt.Sprintf("click error: %v", err)
			return result, nil
		}

		if clickResult.Exhausted {
			result.ExhaustedReason = clickResult.Reason
			return result, nil
		}

		if !clickResult.Success {
			result.ExhaustedReason = "click unsuccessful"
			return result, nil
		}

		// Check for duplicate content
		if config.StopOnDuplicate {
			isNew := paginationState.RecordClick(clickResult.ContentHash)
			if !isNew {
				result.ExhaustedReason = "duplicate content detected"
				return result, nil
			}
		} else {
			paginationState.ClickCount++
		}

		// Fetch the new page content
		pageResult, err := f.fetchCurrentPage(tabCtx, rawURL, statusCode, contentType)
		if err != nil {
			result.LastError = err
			result.ExhaustedReason = fmt.Sprintf("fetch error: %v", err)
			return result, nil
		}

		result.TotalPages++
		pageNumber := result.TotalPages

		// Generate virtual URL with page parameter
		virtualURL := fmt.Sprintf("%s?_page=%d", rawURL, pageNumber)

		// Call callback for this page
		if err := callback(pageResult, pageNumber, virtualURL); err != nil {
			result.LastError = err
			return result, nil
		}
	}

	result.ExhaustedReason = fmt.Sprintf("max clicks reached (%d)", config.MaxClicks)
	return result, nil
}

// fetchCurrentPage fetches the content of the currently loaded page in the tab
func (f *BrowserFetcher) fetchCurrentPage(ctx context.Context, originalURL string, statusCode int, contentType string) (*FetchResult, error) {
	var html string
	var finalURL string

	err := chromedp.Run(ctx,
		chromedp.Location(&finalURL),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		return nil, err
	}

	// Default status code if not captured
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	// Default content type
	if contentType == "" {
		contentType = "text/html"
	}

	return &FetchResult{
		Body:        []byte(html),
		StatusCode:  statusCode,
		ContentType: contentType,
		FinalURL:    finalURL,
	}, nil
}
