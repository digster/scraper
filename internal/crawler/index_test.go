package crawler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExtractTextExcerpt(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		maxLen   int
		expected string
	}{
		{
			name:     "simple text",
			html:     "<p>Hello world</p>",
			maxLen:   100,
			expected: "Hello world",
		},
		{
			name:     "strips multiple tags",
			html:     "<div><h1>Title</h1><p>Content here</p></div>",
			maxLen:   100,
			expected: "Title Content here",
		},
		{
			name:     "handles HTML entities",
			html:     "<p>Hello &amp; goodbye &lt;test&gt;</p>",
			maxLen:   100,
			expected: "Hello & goodbye <test>",
		},
		{
			name:     "truncates long text",
			html:     "<p>This is a very long text that should be truncated because it exceeds the maximum length limit.</p>",
			maxLen:   30,
			expected: "This is a very long text that...",
		},
		{
			name:     "normalizes whitespace",
			html:     "<p>  Multiple   spaces   here  </p>",
			maxLen:   100,
			expected: "Multiple spaces here",
		},
		{
			name:     "empty html",
			html:     "",
			maxLen:   100,
			expected: "",
		},
		{
			name:     "only tags",
			html:     "<div><span></span></div>",
			maxLen:   100,
			expected: "",
		},
		{
			name:     "nested tags with text",
			html:     "<article><header><h1>Main Title</h1></header><section><p>First paragraph.</p><p>Second paragraph.</p></section></article>",
			maxLen:   100,
			expected: "Main Title First paragraph. Second paragraph.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextExcerpt(tt.html, tt.maxLen)
			if result != tt.expected {
				t.Errorf("extractTextExcerpt() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestScanMetaFiles(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "index_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files
	testFiles := []string{
		"page1.meta.json",
		"page2.meta.json",
		"subdir/page3.meta.json",
		"subdir/nested/page4.meta.json",
		"not_meta.json",    // Should not be included
		"page.html",        // Should not be included
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte("{}"), 0644)
	}

	// Scan for meta files
	metaFiles, err := scanMetaFiles(tmpDir)
	if err != nil {
		t.Fatalf("scanMetaFiles() error: %v", err)
	}

	// Should find exactly 4 meta files
	if len(metaFiles) != 4 {
		t.Errorf("scanMetaFiles() found %d files, want 4", len(metaFiles))
	}

	// All found files should end with .meta.json
	for _, f := range metaFiles {
		if !strings.HasSuffix(f, ".meta.json") {
			t.Errorf("found non-meta file: %s", f)
		}
	}
}

func TestLoadPageEntry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "index_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a meta file
	meta := metaFileData{
		URL:                  "https://example.com/page",
		Timestamp:            time.Now().Unix(),
		Size:                 1024,
		ContentFile:          "page.content.html",
		ContentSize:          512,
		ContentExtracted: true,
	}

	metaJSON, _ := json.Marshal(meta)
	metaPath := filepath.Join(tmpDir, "page.meta.json")
	os.WriteFile(metaPath, metaJSON, 0644)

	// Create corresponding HTML and content files
	os.WriteFile(filepath.Join(tmpDir, "page.html"), []byte("<html>raw</html>"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "page.content.html"), []byte("<p>Extracted content here</p>"), 0644)

	// Load the page entry
	entry, err := loadPageEntry(tmpDir, metaPath)
	if err != nil {
		t.Fatalf("loadPageEntry() error: %v", err)
	}

	// Verify fields
	if entry.URL != "https://example.com/page" {
		t.Errorf("URL = %q, want %q", entry.URL, "https://example.com/page")
	}
	if entry.Filename != "page.html" {
		t.Errorf("Filename = %q, want %q", entry.Filename, "page.html")
	}
	if entry.ContentFile != "page.content.html" {
		t.Errorf("ContentFile = %q, want %q", entry.ContentFile, "page.content.html")
	}
	if entry.Size != 1024 {
		t.Errorf("Size = %d, want %d", entry.Size, 1024)
	}
	if !entry.HasContent {
		t.Error("HasContent should be true")
	}
	if entry.Excerpt == "" {
		t.Error("Excerpt should not be empty")
	}
}

func TestGenerateIndex(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "index_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test pages
	pages := []struct {
		filename string
		url      string
		content  string
	}{
		{"page1", "https://example.com/page1", "First page content"},
		{"page2", "https://example.com/page2", "Second page content"},
		{"subdir/page3", "https://example.com/subdir/page3", "Third page content"},
	}

	for _, p := range pages {
		basePath := filepath.Join(tmpDir, p.filename)
		os.MkdirAll(filepath.Dir(basePath), 0755)

		// Create HTML file
		os.WriteFile(basePath+".html", []byte("<html>"+p.content+"</html>"), 0644)

		// Create content file
		os.WriteFile(basePath+".content.html", []byte("<p>"+p.content+"</p>"), 0644)

		// Create meta file
		meta := metaFileData{
			URL:                  p.url,
			Timestamp:            time.Now().Unix(),
			Size:                 len(p.content) + 13, // <html></html>
			ContentFile:          p.filename + ".content.html",
			ContentSize:          len(p.content) + 7, // <p></p>
			ContentExtracted: true,
		}
		metaJSON, _ := json.Marshal(meta)
		os.WriteFile(basePath+".meta.json", metaJSON, 0644)
	}

	// Generate index
	err = GenerateIndex(tmpDir)
	if err != nil {
		t.Fatalf("GenerateIndex() error: %v", err)
	}

	// Verify index file was created
	indexPath := filepath.Join(tmpDir, "_index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatal("_index.html was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}

	contentStr := string(content)

	// Check for expected elements
	checks := []string{
		"<!DOCTYPE html>",
		"Scraper Index",
		"<strong>3</strong> pages",
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/subdir/page3",
		"page1.html",
		"page2.html",
		"subdir/page3.html",
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("index should contain %q", check)
		}
	}
}

func TestGenerateIndexEmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "index_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate index for empty directory
	err = GenerateIndex(tmpDir)
	if err != nil {
		t.Fatalf("GenerateIndex() error for empty dir: %v", err)
	}

	// Index file should NOT be created for empty directory
	indexPath := filepath.Join(tmpDir, "_index.html")
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Error("_index.html should not be created for empty directory")
	}
}

func TestGenerateIndexWithMissingContentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "index_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a meta file pointing to non-existent content
	meta := metaFileData{
		URL:                  "https://example.com/page",
		Timestamp:            time.Now().Unix(),
		Size:                 1024,
		ContentFile:          "nonexistent.content.html",
		ContentSize:          0,
		ContentExtracted: false,
	}
	metaJSON, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(tmpDir, "page.meta.json"), metaJSON, 0644)
	os.WriteFile(filepath.Join(tmpDir, "page.html"), []byte("<html>test</html>"), 0644)

	// Should not error, just skip the excerpt
	err = GenerateIndex(tmpDir)
	if err != nil {
		t.Fatalf("GenerateIndex() error: %v", err)
	}

	// Index should still be created
	indexPath := filepath.Join(tmpDir, "_index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("_index.html should be created even with missing content files")
	}
}
