package crawler

import "time"

// EventType represents different types of crawler events
type EventType string

const (
	EventProgress        EventType = "progress"
	EventLogMessage      EventType = "log"
	EventURLProcessed    EventType = "url_processed"
	EventStateChanged    EventType = "state_changed"
	EventCrawlStarted    EventType = "crawl_started"
	EventCrawlStopped    EventType = "crawl_stopped"
	EventCrawlPaused     EventType = "crawl_paused"
	EventCrawlResumed    EventType = "crawl_resumed"
	EventCrawlCompleted  EventType = "crawl_completed"
	EventError           EventType = "error"
	EventWaitingForLogin EventType = "waiting_for_login"
)

// CrawlerEvent represents an event emitted by the crawler
type CrawlerEvent struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// ProgressData contains real-time progress information
type ProgressData struct {
	ElapsedTime     string  `json:"elapsedTime"`
	Percentage      float64 `json:"percentage"`
	URLsProcessed   int64   `json:"urlsProcessed"`
	URLsSaved       int64   `json:"urlsSaved"`
	URLsErrored     int64   `json:"urlsErrored"`
	QueueSize       int     `json:"queueSize"`
	PagesPerSecond  float64 `json:"pagesPerSecond"`
	BytesDownloaded int64   `json:"bytesDownloaded"`
	CurrentURL      string  `json:"currentUrl"`
}

// LogData contains log message information
type LogData struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// EventEmitter interface for emitting events to the GUI
type EventEmitter interface {
	Emit(event CrawlerEvent)
}

// EmitProgress sends a progress event to the event emitter
func EmitProgress(emitter EventEmitter, metrics *CrawlerMetrics, currentURL string) {
	if emitter == nil {
		return
	}

	snapshot := metrics.GetSnapshot()
	elapsed := time.Since(metrics.StartTime)

	total := snapshot.URLsProcessed + int64(snapshot.QueueSize)
	var percentage float64
	if total > 0 {
		percentage = float64(snapshot.URLsProcessed) / float64(total) * 100
	}

	emitter.Emit(CrawlerEvent{
		Type:      EventProgress,
		Timestamp: time.Now(),
		Data: ProgressData{
			ElapsedTime:     FormatDuration(elapsed),
			Percentage:      percentage,
			URLsProcessed:   snapshot.URLsProcessed,
			URLsSaved:       snapshot.URLsSaved,
			URLsErrored:     snapshot.URLsErrored,
			QueueSize:       snapshot.QueueSize,
			PagesPerSecond:  snapshot.PagesPerSecond,
			BytesDownloaded: snapshot.BytesDownloaded,
			CurrentURL:      currentURL,
		},
	})
}

// EmitLog sends a log event to the event emitter
func EmitLog(emitter EventEmitter, level, message string) {
	if emitter == nil {
		return
	}

	emitter.Emit(CrawlerEvent{
		Type:      EventLogMessage,
		Timestamp: time.Now(),
		Data: LogData{
			Level:   level,
			Message: message,
		},
	})
}

// EmitStateChange sends a state change event
func EmitStateChange(emitter EventEmitter, eventType EventType) {
	if emitter == nil {
		return
	}

	emitter.Emit(CrawlerEvent{
		Type:      eventType,
		Timestamp: time.Now(),
	})
}

// EmitError sends an error event
func EmitError(emitter EventEmitter, message string) {
	if emitter == nil {
		return
	}

	emitter.Emit(CrawlerEvent{
		Type:      EventError,
		Timestamp: time.Now(),
		Data: LogData{
			Level:   "error",
			Message: message,
		},
	})
}

// EmitWaitingForLogin sends a waiting for login event
func EmitWaitingForLogin(emitter EventEmitter, url string) {
	if emitter == nil {
		return
	}

	emitter.Emit(CrawlerEvent{
		Type:      EventWaitingForLogin,
		Timestamp: time.Now(),
		Data: map[string]string{
			"url": url,
		},
	})
}
