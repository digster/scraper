package crawler

import (
	"math"
	"testing"
)

func TestAntiBotConfigDefaults(t *testing.T) {
	config := AntiBotConfig{}

	// All boolean options should be false by default
	if config.HideWebdriver {
		t.Error("HideWebdriver should be false by default")
	}
	if config.SpoofPlugins {
		t.Error("SpoofPlugins should be false by default")
	}
	if config.SpoofLanguages {
		t.Error("SpoofLanguages should be false by default")
	}
	if config.SpoofWebGL {
		t.Error("SpoofWebGL should be false by default")
	}
	if config.AddCanvasNoise {
		t.Error("AddCanvasNoise should be false by default")
	}
	if config.NaturalMouseMovement {
		t.Error("NaturalMouseMovement should be false by default")
	}
	if config.RandomTypingDelays {
		t.Error("RandomTypingDelays should be false by default")
	}
	if config.NaturalScrolling {
		t.Error("NaturalScrolling should be false by default")
	}
	if config.RandomActionDelays {
		t.Error("RandomActionDelays should be false by default")
	}
	if config.RandomClickOffset {
		t.Error("RandomClickOffset should be false by default")
	}
	if config.RotateUserAgent {
		t.Error("RotateUserAgent should be false by default")
	}
	if config.RandomViewport {
		t.Error("RandomViewport should be false by default")
	}
	if config.MatchTimezone {
		t.Error("MatchTimezone should be false by default")
	}
	if config.Timezone != "" {
		t.Error("Timezone should be empty by default")
	}
}

func TestBuildInjectionScripts(t *testing.T) {
	// Test with empty config - no scripts should be returned
	config := AntiBotConfig{}
	scripts := BuildInjectionScripts(config)
	if len(scripts) != 0 {
		t.Errorf("Expected 0 scripts for empty config, got %d", len(scripts))
	}

	// Test with single option enabled
	config.HideWebdriver = true
	scripts = BuildInjectionScripts(config)
	if len(scripts) != 1 {
		t.Errorf("Expected 1 script for HideWebdriver only, got %d", len(scripts))
	}

	// Test with all fingerprint options enabled
	config = AntiBotConfig{
		HideWebdriver:  true,
		SpoofPlugins:   true,
		SpoofLanguages: true,
		SpoofWebGL:     true,
		AddCanvasNoise: true,
	}
	scripts = BuildInjectionScripts(config)
	if len(scripts) != 5 {
		t.Errorf("Expected 5 scripts for all fingerprint options, got %d", len(scripts))
	}

	// Verify scripts are non-empty strings
	for i, script := range scripts {
		if script == "" {
			t.Errorf("Script %d is empty", i)
		}
		if len(script) < 50 {
			t.Errorf("Script %d is suspiciously short: %d chars", i, len(script))
		}
	}
}

func TestBezierCurveGeneration(t *testing.T) {
	start := BezierPoint{0, 0}
	end := BezierPoint{100, 100}
	steps := 20

	points := GenerateBezierCurve(start, end, steps)

	// Check correct number of points
	if len(points) != steps {
		t.Errorf("Expected %d points, got %d", steps, len(points))
	}

	// First point should be at start
	if points[0].X != start.X || points[0].Y != start.Y {
		t.Errorf("First point should be at start position, got (%f, %f)", points[0].X, points[0].Y)
	}

	// Last point should be at end
	lastIdx := len(points) - 1
	if points[lastIdx].X != end.X || points[lastIdx].Y != end.Y {
		t.Errorf("Last point should be at end position, got (%f, %f)", points[lastIdx].X, points[lastIdx].Y)
	}

	// Points should progress generally from start to end
	for i := 1; i < len(points); i++ {
		// Allow for some curvature, but generally should move toward end
		if i > len(points)/2 {
			// In second half, X and Y should be closer to end than to start
			if math.Abs(points[i].X-end.X) > math.Abs(start.X-end.X) {
				// This would mean we went backwards, which is unlikely but possible with curves
				// So we just verify the point is within a reasonable range
				if points[i].X < -50 || points[i].X > 150 {
					t.Errorf("Point %d X coordinate (%f) is out of reasonable range", i, points[i].X)
				}
			}
		}
	}
}

func TestBezierCurveWithDifferentSteps(t *testing.T) {
	start := BezierPoint{10, 20}
	end := BezierPoint{200, 150}

	// Test with various step counts
	stepCounts := []int{5, 10, 30, 50}
	for _, steps := range stepCounts {
		points := GenerateBezierCurve(start, end, steps)
		if len(points) != steps {
			t.Errorf("With %d steps, expected %d points, got %d", steps, steps, len(points))
		}
	}
}

func TestUserAgentRotation(t *testing.T) {
	userAgents := GetChromeUserAgents()

	// Should have at least some user agents
	if len(userAgents) == 0 {
		t.Error("User agent list should not be empty")
	}

	// Each user agent should be non-empty and look like a real UA
	for i, ua := range userAgents {
		if ua == "" {
			t.Errorf("User agent %d is empty", i)
		}
		if len(ua) < 50 {
			t.Errorf("User agent %d is suspiciously short: %s", i, ua)
		}
		// Should contain Mozilla (all Chrome UAs have this)
		if !containsString(ua, "Mozilla") {
			t.Errorf("User agent %d doesn't contain Mozilla: %s", i, ua)
		}
		// Should contain Chrome
		if !containsString(ua, "Chrome") {
			t.Errorf("User agent %d doesn't contain Chrome: %s", i, ua)
		}
	}
}

func TestRandomViewport(t *testing.T) {
	viewports := GetCommonViewports()

	// Should have at least some viewports
	if len(viewports) == 0 {
		t.Error("Viewport list should not be empty")
	}

	// Each viewport should have valid dimensions
	for i, vp := range viewports {
		if vp.Width < 800 || vp.Width > 4000 {
			t.Errorf("Viewport %d has unusual width: %d", i, vp.Width)
		}
		if vp.Height < 600 || vp.Height > 3000 {
			t.Errorf("Viewport %d has unusual height: %d", i, vp.Height)
		}
	}

	// GetRandomViewport should return one of the common viewports
	for i := 0; i < 20; i++ {
		viewport := GetRandomViewport()
		found := false
		for _, v := range viewports {
			if v.Width == viewport.Width && v.Height == viewport.Height {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetRandomViewport returned viewport not in list: %dx%d", viewport.Width, viewport.Height)
		}
	}
}

func TestRandomActionDelay(t *testing.T) {
	// Run multiple times to check range
	for i := 0; i < 100; i++ {
		delay := RandomActionDelay()
		ms := delay.Milliseconds()
		if ms < 100 || ms > 500 {
			t.Errorf("RandomActionDelay returned %dms, expected 100-500ms", ms)
		}
	}
}

func TestRandomTypingDelay(t *testing.T) {
	// Run multiple times to check range
	hasLongDelay := false
	for i := 0; i < 100; i++ {
		delay := RandomTypingDelay()
		ms := delay.Milliseconds()
		// Base is 50-149ms, with occasional longer pauses adding 200-499ms
		// Max possible: 149 + 499 = 648ms
		if ms < 50 || ms > 650 {
			t.Errorf("RandomTypingDelay returned %dms, expected 50-650ms", ms)
		}
		if ms > 200 {
			hasLongDelay = true
		}
	}
	// With 100 iterations and 10% chance of long delay, we should see at least one
	if !hasLongDelay {
		t.Log("Warning: No long typing delays observed in 100 iterations (expected ~10%)")
	}
}

func TestRandomScrollDelay(t *testing.T) {
	for i := 0; i < 100; i++ {
		delay := RandomScrollDelay()
		ms := delay.Milliseconds()
		if ms < 20 || ms > 50 {
			t.Errorf("RandomScrollDelay returned %dms, expected 20-50ms", ms)
		}
	}
}

func TestHumanBehaviorHelper(t *testing.T) {
	// Test with all options disabled
	config := AntiBotConfig{}
	hb := NewHumanBehavior(config)

	if hb == nil {
		t.Error("NewHumanBehavior should not return nil")
	}

	// Test ApplyActionDelay with disabled config (should not block)
	hb.ApplyActionDelay() // Should return immediately

	// Test with options enabled
	config = AntiBotConfig{
		NaturalMouseMovement: true,
		RandomTypingDelays:   true,
		NaturalScrolling:     true,
		RandomClickOffset:    true,
		RandomActionDelays:   true,
	}
	hb = NewHumanBehavior(config)

	if hb == nil {
		t.Error("NewHumanBehavior with options should not return nil")
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
