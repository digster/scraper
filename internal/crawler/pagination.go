package crawler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

// PaginationResult represents the result of a pagination click attempt
type PaginationResult struct {
	Success     bool   // Whether the click was successful
	Exhausted   bool   // Whether pagination is exhausted (no more pages)
	Reason      string // Reason for exhaustion or failure
	ContentHash string // Hash of the page content after click
	PageNumber  int    // Current page number (1-indexed)
}

// PaginationState tracks the state of pagination across multiple clicks
type PaginationState struct {
	ClickCount  int               // Number of clicks performed
	ContentHash string            // Hash of current page content
	SeenHashes  map[string]bool   // Set of previously seen content hashes
	Config      PaginationConfig  // Pagination configuration
	Behavior    *HumanBehavior    // Human behavior helper for natural clicks
}

// NewPaginationState creates a new pagination state tracker
func NewPaginationState(config PaginationConfig, antiBot AntiBotConfig) *PaginationState {
	return &PaginationState{
		ClickCount:  0,
		SeenHashes:  make(map[string]bool),
		Config:      config,
		Behavior:    NewHumanBehavior(antiBot),
	}
}

// CanContinue checks if pagination can continue
func (ps *PaginationState) CanContinue() bool {
	return ps.ClickCount < ps.Config.MaxClicks
}

// RecordClick records a pagination click and its content hash
// Returns true if this is new content, false if duplicate
func (ps *PaginationState) RecordClick(contentHash string) bool {
	ps.ClickCount++
	ps.ContentHash = contentHash

	if ps.SeenHashes[contentHash] {
		return false // Duplicate content
	}

	ps.SeenHashes[contentHash] = true
	return true
}

// ClickPagination attempts to click the pagination element and waits for page update
// Returns a PaginationResult indicating success/failure and reason
func ClickPagination(ctx context.Context, selector string, config PaginationConfig, behavior *HumanBehavior) (*PaginationResult, error) {
	result := &PaginationResult{
		Success:   false,
		Exhausted: false,
	}

	// Check if element exists
	exists, err := elementExists(ctx, selector)
	if err != nil {
		return result, fmt.Errorf("failed to check element existence: %w", err)
	}
	if !exists {
		result.Exhausted = true
		result.Reason = "pagination element not found"
		return result, nil
	}

	// Check if element is disabled
	disabled, err := isPaginationDisabled(ctx, selector)
	if err != nil {
		return result, fmt.Errorf("failed to check disabled state: %w", err)
	}
	if disabled {
		result.Exhausted = true
		result.Reason = "pagination element is disabled"
		return result, nil
	}

	// Check if element is visible
	visible, err := isPaginationVisible(ctx, selector)
	if err != nil {
		return result, fmt.Errorf("failed to check visibility: %w", err)
	}
	if !visible {
		result.Exhausted = true
		result.Reason = "pagination element is not visible"
		return result, nil
	}

	// Apply pre-click delay if human behavior is enabled
	behavior.ApplyActionDelay()

	// Scroll element into view naturally
	if err := scrollToElementNaturally(ctx, selector, behavior); err != nil {
		return result, fmt.Errorf("failed to scroll to element: %w", err)
	}

	// Small delay after scrolling
	behavior.ApplyActionDelay()

	// Click the pagination element
	if err := behavior.ApplyClick(ctx, selector); err != nil {
		return result, fmt.Errorf("failed to click pagination element: %w", err)
	}

	// Wait for page to update
	if err := waitForPageUpdate(ctx, config); err != nil {
		return result, fmt.Errorf("failed waiting for page update: %w", err)
	}

	// Get content hash
	contentHash, err := getContentHash(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to get content hash: %w", err)
	}

	result.Success = true
	result.ContentHash = contentHash
	return result, nil
}

// elementExists checks if an element matching the selector exists
func elementExists(ctx context.Context, selector string) (bool, error) {
	var exists bool
	script := fmt.Sprintf(`document.querySelector('%s') !== null`, escapeSelector(selector))
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &exists)); err != nil {
		return false, err
	}
	return exists, nil
}

// isPaginationDisabled checks if the pagination element is disabled
func isPaginationDisabled(ctx context.Context, selector string) (bool, error) {
	var disabled bool
	script := fmt.Sprintf(`
		(function() {
			const el = document.querySelector('%s');
			if (!el) return true;

			// Check disabled attribute
			if (el.disabled === true) return true;
			if (el.getAttribute('disabled') !== null) return true;

			// Check aria-disabled
			if (el.getAttribute('aria-disabled') === 'true') return true;

			// Check common disabled classes
			const classes = el.className || '';
			if (classes.includes('disabled') || classes.includes('is-disabled')) return true;

			return false;
		})()
	`, escapeSelector(selector))
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &disabled)); err != nil {
		return false, err
	}
	return disabled, nil
}

// isPaginationVisible checks if the pagination element is visible
func isPaginationVisible(ctx context.Context, selector string) (bool, error) {
	var visible bool
	script := fmt.Sprintf(`
		(function() {
			const el = document.querySelector('%s');
			if (!el) return false;

			const style = window.getComputedStyle(el);

			// Check display and visibility
			if (style.display === 'none') return false;
			if (style.visibility === 'hidden') return false;
			if (style.opacity === '0') return false;

			// Check dimensions
			const rect = el.getBoundingClientRect();
			if (rect.width === 0 || rect.height === 0) return false;

			return true;
		})()
	`, escapeSelector(selector))
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &visible)); err != nil {
		return false, err
	}
	return visible, nil
}

// scrollToElementNaturally scrolls the pagination element into view
func scrollToElementNaturally(ctx context.Context, selector string, behavior *HumanBehavior) error {
	// First, check if element is already in view
	var inView bool
	checkScript := fmt.Sprintf(`
		(function() {
			const el = document.querySelector('%s');
			if (!el) return false;
			const rect = el.getBoundingClientRect();
			return rect.top >= 0 && rect.bottom <= window.innerHeight;
		})()
	`, escapeSelector(selector))

	if err := chromedp.Run(ctx, chromedp.Evaluate(checkScript, &inView)); err != nil {
		return err
	}

	if inView {
		return nil
	}

	// Get the element's position
	var elementTop float64
	posScript := fmt.Sprintf(`
		(function() {
			const el = document.querySelector('%s');
			if (!el) return 0;
			const rect = el.getBoundingClientRect();
			return window.scrollY + rect.top - (window.innerHeight / 2);
		})()
	`, escapeSelector(selector))

	if err := chromedp.Run(ctx, chromedp.Evaluate(posScript, &elementTop)); err != nil {
		return err
	}

	// Get current scroll position
	var currentScroll float64
	if err := chromedp.Run(ctx, chromedp.Evaluate(`window.scrollY`, &currentScroll)); err != nil {
		return err
	}

	// Calculate scroll delta
	scrollDelta := int(elementTop - currentScroll)

	// Apply natural scrolling
	return behavior.ApplyScroll(ctx, scrollDelta)
}

// waitForPageUpdate waits for the page to update after a pagination click
func waitForPageUpdate(ctx context.Context, config PaginationConfig) error {
	if config.WaitSelector != "" {
		// Wait for a specific element to appear
		ctxTimeout, cancel := context.WithTimeout(ctx, config.WaitAfterClick)
		defer cancel()

		err := chromedp.Run(ctxTimeout,
			chromedp.WaitVisible(config.WaitSelector, chromedp.ByQuery),
		)
		if err != nil {
			// If timeout, continue anyway - element might already be there
			if ctxTimeout.Err() == context.DeadlineExceeded {
				return nil
			}
			return err
		}
	} else {
		// Wait for a fixed duration
		time.Sleep(config.WaitAfterClick)
	}
	return nil
}

// getContentHash returns a hash of the main page content for duplicate detection
func getContentHash(ctx context.Context) (string, error) {
	var content string

	// Try to get main content area, fall back to body
	script := `
		(function() {
			// Try common main content selectors
			const mainSelectors = ['main', 'article', '#content', '#main', '.content', '.main'];
			for (const sel of mainSelectors) {
				const el = document.querySelector(sel);
				if (el && el.innerText.length > 100) {
					return el.innerText;
				}
			}
			// Fall back to body
			return document.body.innerText;
		})()
	`

	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &content)); err != nil {
		return "", err
	}

	// Hash the content
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:]), nil
}

// escapeSelector escapes a CSS selector for use in JavaScript strings
func escapeSelector(selector string) string {
	// Escape single quotes and backslashes
	result := ""
	for _, c := range selector {
		switch c {
		case '\'':
			result += "\\'"
		case '\\':
			result += "\\\\"
		default:
			result += string(c)
		}
	}
	return result
}
