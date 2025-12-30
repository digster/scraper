package crawler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// PageEntry holds metadata for a single scraped page
type PageEntry struct {
	URL         string
	Filename    string // relative path to raw HTML
	ContentFile string // relative path to .content.html (empty if none)
	Excerpt     string // plain text excerpt from content
	Timestamp   time.Time
	Size        int64
	ContentSize int64
	HasContent  bool
}

// IndexData holds all data needed to render the index template
type IndexData struct {
	Title       string
	Pages       []PageEntry
	TotalPages  int
	TotalSize   int64
	GeneratedAt time.Time
	EarliestURL time.Time
	LatestURL   time.Time
}

// metaFileData represents the structure of .meta.json files
type metaFileData struct {
	URL                   string `json:"url"`
	Timestamp             int64  `json:"timestamp"`
	Size                  int    `json:"size"`
	ContentFile           string `json:"content_file"`
	ContentSize           int    `json:"content_size"`
	ReadabilityExtracted  bool   `json:"readability_extracted"`
}

// GenerateIndex creates an _index.html file in the output directory
func GenerateIndex(outputDir string) error {
	// Scan for all meta files
	metaFiles, err := scanMetaFiles(outputDir)
	if err != nil {
		return fmt.Errorf("failed to scan meta files: %v", err)
	}

	if len(metaFiles) == 0 {
		return nil // Nothing to index
	}

	// Load page entries from meta files
	pages := make([]PageEntry, 0, len(metaFiles))
	var totalSize int64
	var earliest, latest time.Time

	for _, metaPath := range metaFiles {
		entry, err := loadPageEntry(outputDir, metaPath)
		if err != nil {
			continue // Skip files that can't be loaded
		}
		pages = append(pages, entry)
		totalSize += entry.Size

		if earliest.IsZero() || entry.Timestamp.Before(earliest) {
			earliest = entry.Timestamp
		}
		if latest.IsZero() || entry.Timestamp.After(latest) {
			latest = entry.Timestamp
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Timestamp.After(pages[j].Timestamp)
	})

	// Prepare template data
	data := IndexData{
		Title:       filepath.Base(outputDir),
		Pages:       pages,
		TotalPages:  len(pages),
		TotalSize:   totalSize,
		GeneratedAt: time.Now(),
		EarliestURL: earliest,
		LatestURL:   latest,
	}

	// Generate the index HTML
	indexPath := filepath.Join(outputDir, "_index.html")
	return writeIndexHTML(indexPath, data)
}

// scanMetaFiles recursively finds all .meta.json files in the directory
func scanMetaFiles(dir string) ([]string, error) {
	var metaFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}
		if !info.IsDir() && strings.HasSuffix(path, ".meta.json") {
			metaFiles = append(metaFiles, path)
		}
		return nil
	})

	return metaFiles, err
}

// loadPageEntry reads a meta file and creates a PageEntry
func loadPageEntry(outputDir, metaPath string) (PageEntry, error) {
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return PageEntry{}, err
	}

	var meta metaFileData
	if err := json.Unmarshal(data, &meta); err != nil {
		return PageEntry{}, err
	}

	// Calculate relative path for the HTML file
	htmlPath := strings.TrimSuffix(metaPath, ".meta.json") + ".html"
	relPath, _ := filepath.Rel(outputDir, htmlPath)

	// Calculate relative path for content file
	var contentRelPath string
	var excerpt string
	if meta.ContentFile != "" {
		contentPath := filepath.Join(outputDir, meta.ContentFile)
		contentRelPath = meta.ContentFile

		// Extract excerpt from content file
		excerpt = extractExcerptFromFile(contentPath, 300)
	}

	return PageEntry{
		URL:         meta.URL,
		Filename:    relPath,
		ContentFile: contentRelPath,
		Excerpt:     excerpt,
		Timestamp:   time.Unix(meta.Timestamp, 0),
		Size:        int64(meta.Size),
		ContentSize: int64(meta.ContentSize),
		HasContent:  meta.ReadabilityExtracted && meta.ContentFile != "",
	}, nil
}

// extractExcerptFromFile reads a file and extracts a text excerpt
func extractExcerptFromFile(path string, maxLen int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return extractTextExcerpt(string(data), maxLen)
}

// extractTextExcerpt strips HTML tags and returns the first maxLen characters
func extractTextExcerpt(html string, maxLen int) string {
	// Remove HTML tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	text := tagRegex.ReplaceAllString(html, " ")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Normalize whitespace
	spaceRegex := regexp.MustCompile(`\s+`)
	text = spaceRegex.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Truncate to maxLen
	if len(text) > maxLen {
		// Try to break at a word boundary
		text = text[:maxLen]
		lastSpace := strings.LastIndex(text, " ")
		if lastSpace > maxLen-50 {
			text = text[:lastSpace]
		}
		text += "..."
	}

	return text
}

// writeIndexHTML generates and writes the index HTML file
func writeIndexHTML(path string, data IndexData) error {
	tmpl, err := template.New("index").Funcs(template.FuncMap{
		"formatBytes": FormatBytes,
		"formatTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006 15:04")
		},
		"formatDate": func(t time.Time) string {
			return t.Format("Jan 2, 2006")
		},
	}).Parse(indexTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create index file: %v", err)
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Scraper Index - {{.Title}}</title>
    <style>
        :root {
            --bg-primary: #ffffff;
            --bg-secondary: #f8f9fa;
            --bg-card: #ffffff;
            --text-primary: #212529;
            --text-secondary: #6c757d;
            --border-color: #dee2e6;
            --accent-color: #0d6efd;
            --accent-hover: #0b5ed7;
            --success-color: #198754;
            --card-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }

        @media (prefers-color-scheme: dark) {
            :root {
                --bg-primary: #1a1a2e;
                --bg-secondary: #16213e;
                --bg-card: #1f2937;
                --text-primary: #f8f9fa;
                --text-secondary: #9ca3af;
                --border-color: #374151;
                --accent-color: #60a5fa;
                --accent-hover: #3b82f6;
                --success-color: #34d399;
                --card-shadow: 0 1px 3px rgba(0,0,0,0.3);
            }
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            line-height: 1.6;
            min-height: 100vh;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }

        header {
            background: var(--bg-secondary);
            border-bottom: 1px solid var(--border-color);
            padding: 24px 0;
            margin-bottom: 24px;
        }

        header .container {
            display: flex;
            flex-wrap: wrap;
            justify-content: space-between;
            align-items: center;
            gap: 16px;
        }

        h1 {
            font-size: 1.5rem;
            font-weight: 600;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .stats {
            display: flex;
            gap: 24px;
            color: var(--text-secondary);
            font-size: 0.9rem;
        }

        .stats span {
            display: flex;
            align-items: center;
            gap: 4px;
        }

        .search-box {
            width: 100%;
            max-width: 400px;
        }

        .search-box input {
            width: 100%;
            padding: 10px 16px;
            border: 1px solid var(--border-color);
            border-radius: 8px;
            background: var(--bg-card);
            color: var(--text-primary);
            font-size: 0.95rem;
        }

        .search-box input:focus {
            outline: none;
            border-color: var(--accent-color);
            box-shadow: 0 0 0 3px rgba(13, 110, 253, 0.15);
        }

        .page-list {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }

        .page-card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 16px;
            box-shadow: var(--card-shadow);
            transition: box-shadow 0.2s, border-color 0.2s;
        }

        .page-card:hover {
            border-color: var(--accent-color);
        }

        .page-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            gap: 12px;
            margin-bottom: 8px;
        }

        .page-title {
            flex: 1;
            min-width: 0;
        }

        .page-title a {
            color: var(--accent-color);
            text-decoration: none;
            font-weight: 500;
            word-break: break-all;
        }

        .page-title a:hover {
            text-decoration: underline;
        }

        .page-url {
            font-size: 0.85rem;
            color: var(--text-secondary);
            word-break: break-all;
            margin-top: 2px;
        }

        .page-url a {
            color: var(--text-secondary);
            text-decoration: none;
        }

        .page-url a:hover {
            color: var(--accent-color);
            text-decoration: underline;
        }

        .page-actions {
            display: flex;
            gap: 8px;
            flex-shrink: 0;
        }

        .btn {
            padding: 6px 12px;
            border-radius: 6px;
            font-size: 0.8rem;
            text-decoration: none;
            border: 1px solid var(--border-color);
            background: var(--bg-secondary);
            color: var(--text-primary);
            cursor: pointer;
            transition: all 0.2s;
        }

        .btn:hover {
            background: var(--accent-color);
            border-color: var(--accent-color);
            color: white;
        }

        .btn-primary {
            background: var(--accent-color);
            border-color: var(--accent-color);
            color: white;
        }

        .btn-primary:hover {
            background: var(--accent-hover);
            border-color: var(--accent-hover);
        }

        .page-excerpt {
            color: var(--text-secondary);
            font-size: 0.9rem;
            margin: 12px 0;
            padding: 12px;
            background: var(--bg-secondary);
            border-radius: 6px;
            display: none;
        }

        .page-excerpt.expanded {
            display: block;
        }

        .page-meta {
            display: flex;
            gap: 16px;
            font-size: 0.8rem;
            color: var(--text-secondary);
            margin-top: 8px;
        }

        .expand-btn {
            background: none;
            border: none;
            color: var(--accent-color);
            cursor: pointer;
            font-size: 0.85rem;
            padding: 4px 0;
        }

        .expand-btn:hover {
            text-decoration: underline;
        }

        .no-results {
            text-align: center;
            padding: 48px;
            color: var(--text-secondary);
        }

        footer {
            text-align: center;
            padding: 24px;
            color: var(--text-secondary);
            font-size: 0.85rem;
            margin-top: 24px;
            border-top: 1px solid var(--border-color);
        }

        @media (max-width: 600px) {
            header .container {
                flex-direction: column;
                align-items: stretch;
            }

            .stats {
                flex-wrap: wrap;
                gap: 12px;
            }

            .page-header {
                flex-direction: column;
            }

            .page-actions {
                width: 100%;
            }

            .btn {
                flex: 1;
                text-align: center;
            }
        }
    </style>
</head>
<body>
    <header>
        <div class="container">
            <div>
                <h1>Scraper Index</h1>
                <div class="stats">
                    <span><strong>{{.TotalPages}}</strong> pages</span>
                    <span><strong>{{formatBytes .TotalSize}}</strong> total</span>
                    {{if not .EarliestURL.IsZero}}
                    <span>{{formatDate .EarliestURL}} - {{formatDate .LatestURL}}</span>
                    {{end}}
                </div>
            </div>
            <div class="search-box">
                <input type="text" id="search" placeholder="Filter pages by URL or content..." autocomplete="off">
            </div>
        </div>
    </header>

    <main class="container">
        <div class="page-list" id="pageList">
            {{range .Pages}}
            <div class="page-card" data-url="{{.URL}}" data-excerpt="{{.Excerpt}}">
                <div class="page-header">
                    <div class="page-title">
                        <a href="{{.Filename}}">{{.Filename}}</a>
                        <div class="page-url">
                            <a href="{{.URL}}" target="_blank" rel="noopener">{{.URL}}</a>
                        </div>
                    </div>
                    <div class="page-actions">
                        <a href="{{.Filename}}" class="btn">Raw HTML</a>
                        {{if .HasContent}}
                        <a href="{{.ContentFile}}" class="btn btn-primary">View Content</a>
                        {{end}}
                    </div>
                </div>
                {{if .Excerpt}}
                <button class="expand-btn" onclick="toggleExcerpt(this)">Show preview</button>
                <div class="page-excerpt">{{.Excerpt}}</div>
                {{end}}
                <div class="page-meta">
                    <span>{{formatBytes .Size}}</span>
                    {{if .HasContent}}<span>Content: {{formatBytes .ContentSize}}</span>{{end}}
                    <span>{{formatTime .Timestamp}}</span>
                </div>
            </div>
            {{end}}
        </div>
        <div class="no-results" id="noResults" style="display: none;">
            No pages match your search.
        </div>
    </main>

    <footer>
        Generated on {{formatTime .GeneratedAt}} by <a href="https://github.com/user/scraper">Web Scraper</a>
    </footer>

    <script>
        const searchInput = document.getElementById('search');
        const pageList = document.getElementById('pageList');
        const noResults = document.getElementById('noResults');
        const cards = document.querySelectorAll('.page-card');

        searchInput.addEventListener('input', function() {
            const query = this.value.toLowerCase().trim();
            let visibleCount = 0;

            cards.forEach(card => {
                const url = card.dataset.url.toLowerCase();
                const excerpt = (card.dataset.excerpt || '').toLowerCase();
                const matches = query === '' || url.includes(query) || excerpt.includes(query);

                card.style.display = matches ? 'block' : 'none';
                if (matches) visibleCount++;
            });

            noResults.style.display = visibleCount === 0 ? 'block' : 'none';
        });

        function toggleExcerpt(btn) {
            const excerpt = btn.nextElementSibling;
            const isExpanded = excerpt.classList.toggle('expanded');
            btn.textContent = isExpanded ? 'Hide preview' : 'Show preview';
        }
    </script>
</body>
</html>
`
