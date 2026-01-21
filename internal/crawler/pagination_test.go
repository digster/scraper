package crawler

import (
	"testing"
	"time"
)

func TestNewPaginationState(t *testing.T) {
	config := PaginationConfig{
		Enable:          true,
		Selector:        "a.next",
		MaxClicks:       50,
		WaitAfterClick:  2 * time.Second,
		StopOnDuplicate: true,
	}
	antiBot := AntiBotConfig{}

	state := NewPaginationState(config, antiBot)

	if state.ClickCount != 0 {
		t.Errorf("Expected ClickCount to be 0, got %d", state.ClickCount)
	}
	if state.SeenHashes == nil {
		t.Error("SeenHashes should be initialized")
	}
	if len(state.SeenHashes) != 0 {
		t.Error("SeenHashes should be empty initially")
	}
	if state.Behavior == nil {
		t.Error("Behavior should be initialized")
	}
	if state.Config.MaxClicks != 50 {
		t.Errorf("Expected MaxClicks to be 50, got %d", state.Config.MaxClicks)
	}
}

func TestPaginationStateCanContinue(t *testing.T) {
	config := PaginationConfig{
		MaxClicks: 3,
	}
	state := NewPaginationState(config, AntiBotConfig{})

	// Should be able to continue initially
	if !state.CanContinue() {
		t.Error("Should be able to continue with 0 clicks")
	}

	// Simulate clicks
	state.ClickCount = 2
	if !state.CanContinue() {
		t.Error("Should be able to continue with 2 clicks when max is 3")
	}

	state.ClickCount = 3
	if state.CanContinue() {
		t.Error("Should not be able to continue at max clicks")
	}

	state.ClickCount = 4
	if state.CanContinue() {
		t.Error("Should not be able to continue past max clicks")
	}
}

func TestPaginationStateRecordClick(t *testing.T) {
	config := PaginationConfig{
		MaxClicks:       10,
		StopOnDuplicate: true,
	}
	state := NewPaginationState(config, AntiBotConfig{})

	// First click with unique hash
	hash1 := "abc123"
	isNew := state.RecordClick(hash1)
	if !isNew {
		t.Error("First click should be new content")
	}
	if state.ClickCount != 1 {
		t.Errorf("ClickCount should be 1, got %d", state.ClickCount)
	}
	if state.ContentHash != hash1 {
		t.Errorf("ContentHash should be %s, got %s", hash1, state.ContentHash)
	}

	// Second click with different hash
	hash2 := "def456"
	isNew = state.RecordClick(hash2)
	if !isNew {
		t.Error("Second click with new hash should be new content")
	}
	if state.ClickCount != 2 {
		t.Errorf("ClickCount should be 2, got %d", state.ClickCount)
	}

	// Third click with duplicate hash
	isNew = state.RecordClick(hash1)
	if isNew {
		t.Error("Click with duplicate hash should not be new content")
	}
	if state.ClickCount != 3 {
		t.Errorf("ClickCount should be 3, got %d", state.ClickCount)
	}
}

func TestEscapeSelector(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a.next", "a.next"},
		{"a[href='next']", "a[href=\\'next\\']"},
		{".btn\\-next", ".btn\\\\-next"},
		{"a[data-page='1']", "a[data-page=\\'1\\']"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		result := escapeSelector(tt.input)
		if result != tt.expected {
			t.Errorf("escapeSelector(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestPaginationConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "pagination with http mode should fail",
			config: Config{
				URL:       "https://example.com",
				MaxDepth:  1,
				FetchMode: FetchModeHTTP,
				Pagination: PaginationConfig{
					Enable:   true,
					Selector: "a.next",
				},
			},
			expectError: true,
			errorMsg:    "pagination requires browser fetch mode",
		},
		{
			name: "pagination without selector should fail",
			config: Config{
				URL:       "https://example.com",
				MaxDepth:  1,
				FetchMode: FetchModeBrowser,
				Pagination: PaginationConfig{
					Enable:   true,
					Selector: "",
				},
			},
			expectError: true,
			errorMsg:    "pagination selector is required",
		},
		{
			name: "valid pagination config should pass",
			config: Config{
				URL:       "https://example.com",
				MaxDepth:  1,
				FetchMode: FetchModeBrowser,
				Pagination: PaginationConfig{
					Enable:   true,
					Selector: "a.next",
				},
			},
			expectError: false,
		},
		{
			name: "pagination disabled should pass validation",
			config: Config{
				URL:       "https://example.com",
				MaxDepth:  1,
				FetchMode: FetchModeHTTP,
				Pagination: PaginationConfig{
					Enable: false,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(&tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorMsg)
				} else if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestPaginationConfigDefaults(t *testing.T) {
	config := Config{
		URL:       "https://example.com",
		MaxDepth:  1,
		FetchMode: FetchModeBrowser,
		Pagination: PaginationConfig{
			Enable:   true,
			Selector: "a.next",
			// MaxClicks and WaitAfterClick are zero
		},
	}

	err := ValidateConfig(&config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Validation should set defaults
	if config.Pagination.MaxClicks != 100 {
		t.Errorf("Expected MaxClicks default to be 100, got %d", config.Pagination.MaxClicks)
	}
	if config.Pagination.WaitAfterClick != 2*time.Second {
		t.Errorf("Expected WaitAfterClick default to be 2s, got %v", config.Pagination.WaitAfterClick)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
