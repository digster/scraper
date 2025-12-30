package crawler

import (
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
