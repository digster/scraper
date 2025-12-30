package crawler

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPFetcher implements Fetcher using standard HTTP client
type HTTPFetcher struct {
	client *http.Client
}

// NewHTTPFetcher creates a new HTTP-based fetcher
func NewHTTPFetcher() *HTTPFetcher {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &HTTPFetcher{
		client: &http.Client{
			Timeout:   HTTPTimeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= MaxRedirects {
					return fmt.Errorf("stopped after %d redirects", MaxRedirects)
				}
				// User-Agent is preserved from original request
				return nil
			},
		},
	}
}

// Fetch retrieves a URL using HTTP client
func (f *HTTPFetcher) Fetch(rawURL string, userAgent string) (*FetchResult, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}

	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &FetchResult{
		Body:        body,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		FinalURL:    resp.Request.URL.String(),
	}, nil
}

// Close releases resources (no-op for HTTP fetcher)
func (f *HTTPFetcher) Close() error {
	return nil
}
