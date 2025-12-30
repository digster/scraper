package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// CrawlerMetrics tracks statistics during crawling
type CrawlerMetrics struct {
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time,omitempty"`
	Duration         float64   `json:"duration_seconds,omitempty"`
	URLsProcessed    int64     `json:"urls_processed"`
	URLsSaved        int64     `json:"urls_saved"`
	URLsSkipped      int64     `json:"urls_skipped"`
	URLsErrored      int64     `json:"urls_errored"`
	BytesDownloaded  int64     `json:"bytes_downloaded"`
	RobotsBlocked    int64     `json:"robots_blocked"`
	DepthLimitHits   int64     `json:"depth_limit_hits"`
	ContentFiltered  int64     `json:"content_filtered"`
	PagesPerSecond   float64   `json:"pages_per_second,omitempty"`
	QueueSize        int       `json:"queue_size"`
	mu               sync.Mutex
	lastDisplayTime  time.Time
	lastDisplayCount int64
}

// MetricsDisplayInterval controls how often progress is displayed
const MetricsDisplayInterval = 2 * time.Second

// NewCrawlerMetrics creates a new metrics tracker
func NewCrawlerMetrics() *CrawlerMetrics {
	return &CrawlerMetrics{
		StartTime:       time.Now(),
		lastDisplayTime: time.Now(),
	}
}

// IncrementProcessed increments the processed URL count
func (m *CrawlerMetrics) IncrementProcessed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.URLsProcessed++
}

// IncrementSaved increments the saved URL count and adds bytes
func (m *CrawlerMetrics) IncrementSaved(bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.URLsSaved++
	m.BytesDownloaded += bytes
}

// IncrementSkipped increments the skipped URL count
func (m *CrawlerMetrics) IncrementSkipped() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.URLsSkipped++
}

// IncrementErrored increments the error count
func (m *CrawlerMetrics) IncrementErrored() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.URLsErrored++
}

// IncrementRobotsBlocked increments the robots.txt blocked count
func (m *CrawlerMetrics) IncrementRobotsBlocked() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RobotsBlocked++
}

// IncrementDepthLimitHits increments the depth limit hit count
func (m *CrawlerMetrics) IncrementDepthLimitHits() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DepthLimitHits++
}

// IncrementContentFiltered increments the content filtered count
func (m *CrawlerMetrics) IncrementContentFiltered() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ContentFiltered++
}

// SetQueueSize updates the current queue size
func (m *CrawlerMetrics) SetQueueSize(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QueueSize = size
}

// Finalize calculates final metrics
func (m *CrawlerMetrics) Finalize() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EndTime = time.Now()
	m.Duration = m.EndTime.Sub(m.StartTime).Seconds()
	if m.Duration > 0 {
		m.PagesPerSecond = float64(m.URLsProcessed) / m.Duration
	}
}

// GetSnapshot returns a copy of current metrics
func (m *CrawlerMetrics) GetSnapshot() CrawlerMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := *m
	elapsed := time.Since(m.StartTime).Seconds()
	if elapsed > 0 {
		snapshot.PagesPerSecond = float64(m.URLsProcessed) / elapsed
	}
	return snapshot
}

// ShouldDisplay checks if enough time has passed to display progress
func (m *CrawlerMetrics) ShouldDisplay() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return time.Since(m.lastDisplayTime) >= MetricsDisplayInterval
}

// MarkDisplayed marks that progress was just displayed
func (m *CrawlerMetrics) MarkDisplayed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastDisplayTime = time.Now()
	m.lastDisplayCount = m.URLsProcessed
}

// DisplayProgress prints current progress to stdout
func (m *CrawlerMetrics) DisplayProgress(verbose bool) {
	snapshot := m.GetSnapshot()
	elapsed := time.Since(m.StartTime)

	// Format bytes downloaded
	bytesStr := FormatBytes(snapshot.BytesDownloaded)

	// Calculate percentage if we have a meaningful denominator
	// (processed + queue gives us an estimate of total)
	total := snapshot.URLsProcessed + int64(snapshot.QueueSize)
	var progressStr string
	if total > 0 {
		pct := float64(snapshot.URLsProcessed) / float64(total) * 100
		progressStr = fmt.Sprintf("%.1f%%", pct)
	} else {
		progressStr = "..."
	}

	if verbose {
		fmt.Printf("\r[%s] Progress: %s | Processed: %d | Saved: %d | Errors: %d | Queue: %d | %.2f p/s | %s   ",
			FormatDuration(elapsed),
			progressStr,
			snapshot.URLsProcessed,
			snapshot.URLsSaved,
			snapshot.URLsErrored,
			snapshot.QueueSize,
			snapshot.PagesPerSecond,
			bytesStr,
		)
	} else {
		fmt.Printf("\r[%s] %s | %d processed | %d saved | %d in queue | %.2f p/s   ",
			FormatDuration(elapsed),
			progressStr,
			snapshot.URLsProcessed,
			snapshot.URLsSaved,
			snapshot.QueueSize,
			snapshot.PagesPerSecond,
		)
	}

	m.MarkDisplayed()
}

// DisplayFinalSummary prints the final crawl summary
func (m *CrawlerMetrics) DisplayFinalSummary() {
	m.Finalize()
	snapshot := m.GetSnapshot()

	fmt.Println() // New line after progress bar
	fmt.Println()
	fmt.Println("=== Crawl Complete ===")
	fmt.Printf("Duration:         %s\n", FormatDuration(time.Duration(snapshot.Duration*float64(time.Second))))
	fmt.Printf("URLs Processed:   %d\n", snapshot.URLsProcessed)
	fmt.Printf("URLs Saved:       %d\n", snapshot.URLsSaved)
	fmt.Printf("URLs Skipped:     %d\n", snapshot.URLsSkipped)
	fmt.Printf("Errors:           %d\n", snapshot.URLsErrored)
	fmt.Printf("Robots Blocked:   %d\n", snapshot.RobotsBlocked)
	fmt.Printf("Depth Limit Hits: %d\n", snapshot.DepthLimitHits)
	fmt.Printf("Content Filtered: %d\n", snapshot.ContentFiltered)
	fmt.Printf("Data Downloaded:  %s\n", FormatBytes(snapshot.BytesDownloaded))
	fmt.Printf("Average Speed:    %.2f pages/second\n", snapshot.PagesPerSecond)
}

// WriteJSON writes metrics to a JSON file
func (m *CrawlerMetrics) WriteJSON(filepath string) error {
	m.Finalize()
	snapshot := m.GetSnapshot()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %v", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %v", err)
	}

	return nil
}

// FormatBytes converts bytes to human readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats a duration as HH:MM:SS
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
