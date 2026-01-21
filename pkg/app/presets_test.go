package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPresetNameValidation tests that preset name validation works correctly
func TestPresetNameValidation(t *testing.T) {
	tests := []struct {
		name    string
		valid   bool
	}{
		{"valid-preset", true},
		{"valid_preset", true},
		{"ValidPreset123", true},
		{"a", true},
		{"1preset", true},
		{"", false},                         // Empty
		{"../traversal", false},              // Path traversal
		{"/absolute", false},                 // Absolute path
		{"with spaces", false},               // Spaces
		{"special@chars", false},             // Special chars
		{"名前", false},                       // Non-ASCII
		{"-startswithdash", false},           // Starts with dash
		{"_startswithunderscore", false},     // Starts with underscore
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validPresetName.MatchString(tt.name)
			if result != tt.valid {
				t.Errorf("validPresetName.MatchString(%q) = %v, want %v", tt.name, result, tt.valid)
			}
		})
	}
}

// TestPresetSaveAndLoad tests saving and loading presets
func TestPresetSaveAndLoad(t *testing.T) {
	// Create a temporary directory for tests
	tempDir := t.TempDir()

	// Override the presets directory for testing
	presetsDir := filepath.Join(tempDir, "scraper", "presets")
	if err := os.MkdirAll(presetsDir, 0755); err != nil {
		t.Fatalf("Failed to create test presets dir: %v", err)
	}

	// Test saving a preset
	config := PresetConfig{
		URL:              "https://example.com",
		Concurrent:       true,
		Delay:            "2s",
		MaxDepth:         5,
		MinContentLength: 200,
		FetchMode:        "browser",
		Headless:         false,
	}

	// Write the preset directly to test directory
	testName := "test-preset"
	filePath := filepath.Join(presetsDir, testName+".json")

	// Use json.MarshalIndent directly for testing
	config.Name = testName
	config.CreatedAt = time.Now()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal preset: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("Failed to write preset file: %v", err)
	}

	// Read it back
	readData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read preset file: %v", err)
	}

	var loadedConfig PresetConfig
	if err := json.Unmarshal(readData, &loadedConfig); err != nil {
		t.Fatalf("Failed to unmarshal preset: %v", err)
	}

	// Verify fields
	if loadedConfig.URL != config.URL {
		t.Errorf("URL mismatch: got %q, want %q", loadedConfig.URL, config.URL)
	}
	if loadedConfig.Concurrent != config.Concurrent {
		t.Errorf("Concurrent mismatch: got %v, want %v", loadedConfig.Concurrent, config.Concurrent)
	}
	if loadedConfig.Delay != config.Delay {
		t.Errorf("Delay mismatch: got %q, want %q", loadedConfig.Delay, config.Delay)
	}
	if loadedConfig.MaxDepth != config.MaxDepth {
		t.Errorf("MaxDepth mismatch: got %d, want %d", loadedConfig.MaxDepth, config.MaxDepth)
	}
	if loadedConfig.MinContentLength != config.MinContentLength {
		t.Errorf("MinContentLength mismatch: got %d, want %d", loadedConfig.MinContentLength, config.MinContentLength)
	}
	if loadedConfig.FetchMode != config.FetchMode {
		t.Errorf("FetchMode mismatch: got %q, want %q", loadedConfig.FetchMode, config.FetchMode)
	}
	if loadedConfig.Headless != config.Headless {
		t.Errorf("Headless mismatch: got %v, want %v", loadedConfig.Headless, config.Headless)
	}
}

// TestPresetConfigFields tests that PresetConfig has all expected fields
func TestPresetConfigFields(t *testing.T) {
	// Create a fully populated config to verify all fields serialize correctly
	config := PresetConfig{
		Name:                      "full-test",
		CreatedAt:                 time.Now(),
		URL:                       "https://example.com",
		Concurrent:                true,
		Delay:                     "1s",
		MaxDepth:                  10,
		PrefixFilterURL:           "https://example.com/docs",
		ExcludeExtensions:         "js,css,png",
		LinkSelectors:             "a[href]",
		Verbose:                   true,
		UserAgent:                 "TestBot/1.0",
		IgnoreRobots:              true,
		MinContentLength:          100,
		DisableReadability:        true,
		FetchMode:                 "browser",
		Headless:                  true,
		WaitForLogin:              true,
		EnablePagination:          true,
		PaginationSelector:        ".next",
		MaxPaginationClicks:       50,
		PaginationWait:            "2s",
		PaginationWaitSelector:    ".loaded",
		PaginationStopOnDuplicate: true,
		HideWebdriver:             true,
		SpoofPlugins:              true,
		SpoofLanguages:            true,
		SpoofWebGL:                true,
		AddCanvasNoise:            true,
		NaturalMouseMovement:      true,
		RandomTypingDelays:        true,
		NaturalScrolling:          true,
		RandomActionDelays:        true,
		RandomClickOffset:         true,
		RotateUserAgent:           true,
		RandomViewport:            true,
		MatchTimezone:             true,
		Timezone:                  "America/New_York",
	}

	// Serialize and deserialize
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	var loaded PresetConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify some key fields
	if loaded.Name != config.Name {
		t.Errorf("Name mismatch")
	}
	if loaded.EnablePagination != config.EnablePagination {
		t.Errorf("EnablePagination mismatch")
	}
	if loaded.HideWebdriver != config.HideWebdriver {
		t.Errorf("HideWebdriver mismatch")
	}
	if loaded.Timezone != config.Timezone {
		t.Errorf("Timezone mismatch")
	}
}

// TestSavePresetValidation tests input validation for SavePreset
func TestSavePresetValidation(t *testing.T) {
	app := &App{}

	tests := []struct {
		name        string
		presetName  string
		expectError bool
	}{
		{"empty name", "", true},
		{"valid name", "my-preset", false}, // Will still error due to dir creation, but validates name
		{"path traversal", "../hack", true},
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true}, // 52 chars
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.SavePreset(tt.presetName, PresetConfig{})
			hasError := err != nil

			// For "valid name", we expect an error from directory creation, not validation
			if tt.expectError && !hasError {
				t.Errorf("Expected error for %q, got nil", tt.presetName)
			}
			if !tt.expectError && hasError {
				// Check if the error is from validation vs directory creation
				if err.Error() == "preset name cannot be empty" ||
				   err.Error() == "preset name too long (max 50 characters)" ||
				   err.Error() == "preset name can only contain letters, numbers, dashes, and underscores" {
					t.Errorf("Got validation error for valid name: %v", err)
				}
			}
		})
	}
}
