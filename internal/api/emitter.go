package api

import (
	"sync"

	"scraper/internal/crawler"
)

// SSEEmitter implements crawler.EventEmitter and broadcasts events to SSE clients
type SSEEmitter struct {
	mu      sync.RWMutex
	clients map[chan SSEEvent]struct{}
	closed  bool
}

// NewSSEEmitter creates a new SSE event emitter
func NewSSEEmitter() *SSEEmitter {
	return &SSEEmitter{
		clients: make(map[chan SSEEvent]struct{}),
	}
}

// Emit implements crawler.EventEmitter interface
// It broadcasts the crawler event to all connected SSE clients
func (e *SSEEmitter) Emit(event crawler.CrawlerEvent) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.closed {
		return
	}

	sseEvent := FromCrawlerEvent(event)

	// Non-blocking broadcast to all clients
	for clientChan := range e.clients {
		select {
		case clientChan <- sseEvent:
			// Event sent successfully
		default:
			// Client channel is full, skip this event
			// This prevents slow clients from blocking the crawler
		}
	}
}

// Subscribe creates a new client channel for receiving events
// Returns the channel and a cleanup function
func (e *SSEEmitter) Subscribe() (<-chan SSEEvent, func()) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		// Return a closed channel if emitter is closed
		ch := make(chan SSEEvent)
		close(ch)
		return ch, func() {}
	}

	// Buffer size of 100 to prevent blocking during bursts
	clientChan := make(chan SSEEvent, 100)
	e.clients[clientChan] = struct{}{}

	// Return unsubscribe function
	unsubscribe := func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		if _, exists := e.clients[clientChan]; exists {
			delete(e.clients, clientChan)
			close(clientChan)
		}
	}

	return clientChan, unsubscribe
}

// ClientCount returns the number of connected clients
func (e *SSEEmitter) ClientCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.clients)
}

// Close closes all client channels and prevents new subscriptions
func (e *SSEEmitter) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return
	}

	e.closed = true

	// Close all client channels
	for clientChan := range e.clients {
		close(clientChan)
	}
	e.clients = make(map[chan SSEEvent]struct{})
}

// IsClosed returns whether the emitter has been closed
func (e *SSEEmitter) IsClosed() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.closed
}
