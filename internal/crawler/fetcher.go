package crawler

// FetchResult contains the result of fetching a URL
type FetchResult struct {
	Body        []byte
	StatusCode  int
	ContentType string
	FinalURL    string // URL after any redirects
}

// Fetcher is the interface for fetching web pages
type Fetcher interface {
	// Fetch retrieves a URL and returns the result
	Fetch(url string, userAgent string) (*FetchResult, error)
	// Close releases any resources held by the fetcher
	Close() error
}
