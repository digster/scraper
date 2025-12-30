package crawler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
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
}

// NewBrowserFetcher creates a new browser-based fetcher
func NewBrowserFetcher(headless bool) (*BrowserFetcher, error) {
	return NewBrowserFetcherWithUserAgent(headless, DefaultUserAgent)
}

// NewBrowserFetcherWithUserAgent creates a new browser-based fetcher with custom user agent
func NewBrowserFetcherWithUserAgent(headless bool, userAgent string) (*BrowserFetcher, error) {
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

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	browserCtx, cancelFunc := chromedp.NewContext(allocCtx)

	// Start browser
	if err := chromedp.Run(browserCtx); err != nil {
		allocCancel()
		return nil, fmt.Errorf("failed to start browser: %w", err)
	}

	return &BrowserFetcher{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		browserCtx:  browserCtx,
		cancelFunc:  cancelFunc,
		headless:    headless,
		userAgent:   userAgent,
	}, nil
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

	actions := []chromedp.Action{
		network.Enable(),
		chromedp.Navigate(rawURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond), // Small delay for dynamic content
		chromedp.Location(&finalURL),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	}

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

// Close releases browser resources
func (f *BrowserFetcher) Close() error {
	f.cancelFunc()
	f.allocCancel()
	return nil
}
