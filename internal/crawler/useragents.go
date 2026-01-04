package crawler

import (
	"math/rand"
)

// ChromeUserAgents contains realistic Chrome user agent strings for rotation
var chromeUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
}

// GetChromeUserAgents returns the list of Chrome user agents for rotation
func GetChromeUserAgents() []string {
	return chromeUserAgents
}

// Viewport represents a screen resolution
type Viewport struct {
	Width  int
	Height int
}

// CommonViewports contains common screen resolutions for randomization
var commonViewports = []Viewport{
	{1920, 1080}, // Full HD - most common
	{1366, 768},  // HD - very common on laptops
	{1536, 864},  // Common Windows scaling
	{1440, 900},  // MacBook Pro 15"
	{1280, 720},  // HD
	{1600, 900},  // HD+
	{2560, 1440}, // QHD
	{1680, 1050}, // WSXGA+
}

// GetRandomViewport returns a random common viewport
func GetRandomViewport() *Viewport {
	viewport := commonViewports[rand.Intn(len(commonViewports))]
	return &viewport
}

// GetCommonViewports returns the list of common viewports
func GetCommonViewports() []Viewport {
	return commonViewports
}
