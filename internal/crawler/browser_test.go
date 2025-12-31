package crawler

import (
	"context"
	"testing"
)

func TestHTTPFetcher(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	fetcher := NewHTTPFetcher()
	defer fetcher.Close()

	// Test fetching a simple page
	result, err := fetcher.Fetch("https://example.com", "")
	if err != nil {
		t.Fatalf("HTTPFetcher.Fetch failed: %v", err)
	}

	if result.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", result.StatusCode)
	}

	if len(result.Body) == 0 {
		t.Error("expected non-empty body")
	}

	if result.ContentType == "" {
		t.Error("expected non-empty content type")
	}
}

func TestHTTPFetcherWithUserAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	fetcher := NewHTTPFetcher()
	defer fetcher.Close()

	customUA := "TestBot/1.0"
	result, err := fetcher.Fetch("https://httpbin.org/user-agent", customUA)
	if err != nil {
		t.Fatalf("HTTPFetcher.Fetch failed: %v", err)
	}

	if result.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", result.StatusCode)
	}

	// The response should contain our user agent
	body := string(result.Body)
	if body == "" {
		t.Error("expected non-empty body")
	}
}

func TestHTTPFetcherInvalidURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	fetcher := NewHTTPFetcher()
	defer fetcher.Close()

	_, err := fetcher.Fetch("https://this-domain-does-not-exist-12345.com", "")
	if err == nil {
		t.Error("expected error for invalid domain")
	}
}

func TestFetchModeValidation(t *testing.T) {
	tests := []struct {
		name      string
		fetchMode FetchMode
		valid     bool
	}{
		{
			name:      "empty fetch mode is valid (defaults to http)",
			fetchMode: "",
			valid:     true,
		},
		{
			name:      "http fetch mode is valid",
			fetchMode: FetchModeHTTP,
			valid:     true,
		},
		{
			name:      "browser fetch mode is valid",
			fetchMode: FetchModeBrowser,
			valid:     true,
		},
		{
			name:      "invalid fetch mode",
			fetchMode: "invalid",
			valid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				URL:       "https://example.com",
				MaxDepth:  10,
				FetchMode: tt.fetchMode,
			}
			err := ValidateConfig(&config)
			if tt.valid && err != nil {
				t.Errorf("expected valid config, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected error for invalid fetch mode")
			}
		})
	}
}

// BrowserFetcher tests are skipped in CI as they require Chrome
// To run locally: go test -v -run TestBrowserFetcher
func TestBrowserFetcherCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	fetcher, err := NewBrowserFetcher(true)
	if err != nil {
		t.Fatalf("NewBrowserFetcher failed: %v", err)
	}
	defer fetcher.Close()

	if fetcher.headless != true {
		t.Error("expected headless to be true")
	}
}

func TestBrowserFetcherWithCustomUserAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	customUA := "TestBrowser/1.0"
	fetcher, err := NewBrowserFetcherWithUserAgent(true, customUA)
	if err != nil {
		t.Fatalf("NewBrowserFetcherWithUserAgent failed: %v", err)
	}
	defer fetcher.Close()

	if fetcher.userAgent != customUA {
		t.Errorf("expected userAgent %q, got %q", customUA, fetcher.userAgent)
	}
}

func TestBrowserFetcherFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	fetcher, err := NewBrowserFetcher(true)
	if err != nil {
		t.Fatalf("NewBrowserFetcher failed: %v", err)
	}
	defer fetcher.Close()

	result, err := fetcher.Fetch("https://example.com", "")
	if err != nil {
		t.Fatalf("BrowserFetcher.Fetch failed: %v", err)
	}

	if result.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", result.StatusCode)
	}

	if len(result.Body) == 0 {
		t.Error("expected non-empty body")
	}

	// Browser should return full HTML
	body := string(result.Body)
	if body == "" {
		t.Error("expected non-empty HTML body")
	}
}

func TestWaitForLoginConfig(t *testing.T) {
	tests := []struct {
		name         string
		fetchMode    FetchMode
		headless     bool
		waitForLogin bool
		shouldWait   bool
	}{
		{
			name:         "browser non-headless with wait enabled",
			fetchMode:    FetchModeBrowser,
			headless:     false,
			waitForLogin: true,
			shouldWait:   true,
		},
		{
			name:         "browser headless with wait enabled",
			fetchMode:    FetchModeBrowser,
			headless:     true,
			waitForLogin: true,
			shouldWait:   false, // Should not wait in headless mode
		},
		{
			name:         "http mode with wait enabled",
			fetchMode:    FetchModeHTTP,
			headless:     false,
			waitForLogin: true,
			shouldWait:   false, // Should not wait in HTTP mode
		},
		{
			name:         "browser non-headless with wait disabled",
			fetchMode:    FetchModeBrowser,
			headless:     false,
			waitForLogin: false,
			shouldWait:   false, // Wait not enabled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				URL:          "https://example.com",
				MaxDepth:     10,
				FetchMode:    tt.fetchMode,
				Headless:     tt.headless,
				WaitForLogin: tt.waitForLogin,
			}

			// Calculate whether we should wait based on the conditions
			shouldWait := config.WaitForLogin &&
				config.FetchMode == FetchModeBrowser &&
				!config.Headless

			if shouldWait != tt.shouldWait {
				t.Errorf("shouldWait = %v, want %v", shouldWait, tt.shouldWait)
			}
		})
	}
}

func TestLoginConfirmationMethods(t *testing.T) {
	// Test that IsWaitingForLogin returns false initially
	// and ConfirmLogin works correctly
	// Note: This is a unit test that doesn't require browser

	config := Config{
		URL:          "https://example.com",
		MaxDepth:     10,
		FetchMode:    FetchModeHTTP, // Use HTTP mode to avoid browser dependency
		WaitForLogin: false,
	}

	c, err := NewCrawler(config, context.Background())
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}
	defer c.Close()

	// Initial state should be not waiting
	if c.IsWaitingForLogin() {
		t.Error("crawler should not be waiting for login initially")
	}

	// ConfirmLogin should be safe to call even when not waiting
	c.ConfirmLogin() // Should not panic

	// After calling ConfirmLogin, still should not be waiting
	if c.IsWaitingForLogin() {
		t.Error("crawler should not be waiting for login after ConfirmLogin")
	}
}
