# Web Scraper

A Go-based web scraper that creates offline backups of websites by crawling and downloading content.

## Features

- **URL Prefix Filtering**: Only scrapes pages with URLs that have the input URL as a prefix
- **Concurrent/Sequential Mode**: Choose between concurrent or sequential crawling
- **Configurable Delays**: Set delays between fetches to be respectful to servers
- **Content Validation**: Only saves pages with meaningful content (>100 characters of text)
- **Resume Functionality**: Automatically resumes from where it left off if interrupted
- **State Persistence**: Saves crawling state to JSON file for resumption

## Installation

```bash
go mod tidy
go build -o scraper
```

## Usage

### Basic Usage
```bash
./scraper -url https://example.com/docs
```

### Advanced Options
```bash
./scraper -url https://example.com/docs \
          -concurrent \
          -delay 2s \
          -depth 15 \
          -output my_backup \
          -state crawler.json
```

### Command Line Arguments

- `-url`: Starting URL to scrape (required)
- `-concurrent`: Run in concurrent mode (default: false)
- `-delay`: Delay between fetches (default: 1s)
- `-depth`: Maximum crawl depth (default: 10)
- `-output`: Output directory for scraped content (default: "scraped_content")
- `-state`: State file for resume functionality (default: "crawler_state.json")

## How It Works

1. **URL Validation**: Only processes URLs that have the input URL as a prefix
   - Input: `https://example.com/docs`
   - Valid: `https://example.com/docs/page1`, `https://example.com/docs/sub/page2`
   - Invalid: `https://example.com/other`, `https://other.com/docs`

2. **Content Filtering**: Pages are only saved if they contain meaningful content (>100 characters of text after removing scripts and styles)

3. **File Storage**: Each page is saved as:
   - `{hash}.html`: The actual HTML content
   - `{hash}.meta.json`: Metadata including original URL, timestamp, and size

4. **Resume Capability**: State is saved periodically and can be resumed by running the same command again

## Output Structure

```
scraped_content/
├── a1b2c3d4.html          # HTML content
├── a1b2c3d4.meta.json     # Metadata
├── e5f6g7h8.html
├── e5f6g7h8.meta.json
└── ...
```

## Examples

### Sequential crawling with 2-second delays
```bash
./scraper -url https://docs.example.com -delay 2s
```

### Concurrent crawling (faster but more resource intensive)
```bash
./scraper -url https://docs.example.com -concurrent -delay 500ms
```

### Resume interrupted crawling
Simply run the same command again - it will automatically resume from the state file.

## Notes

- The scraper respects the URL prefix constraint strictly
- Only HTML pages with substantial content are saved
- Concurrent mode limits to 10 simultaneous requests to avoid overwhelming servers
- State is saved every 10 processed URLs for resilience