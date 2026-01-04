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
