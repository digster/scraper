# Web Scraper

A Go-based web scraper that creates offline backups of websites by crawling and downloading content.

## Initial Prompt
I want to create a go program to create an offline backup of the url provided.
When a url is provided, go through all the links like a crawler and fetch the content from the pages.
Make sure only pages with content are scraped.
The program should take an argument whether to run concurrent or not.
There should be an argument to specify the delay between the fetches.
For the url provided, make sure only the pages with the url having the input url as prefix are fetched, example -  if www.a.com/a is provided, www.a.com/a/c should be parsed but not www.a.com/c
If the task is interrupted, it should resume with the remaining workload, not from the start.
Ask any clarifying questions if needed.

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
          -state crawler.json \
          -disable-prefix-filter \
          -exclude-extensions js,css,png,jpg
```

### Command Line Arguments

- `-url`: Starting URL to scrape (required)
- `-concurrent`: Run in concurrent mode (default: false)
- `-delay`: Delay between fetches (default: 1s)
- `-depth`: Maximum crawl depth (default: 10)
- `-output`: Output directory for scraped content (default: "scraped_content")
- `-state`: State file for resume functionality (default: "crawler_state.json")
- `-disable-prefix-filter`: Disable URL prefix filtering (allows crawling outside input URL prefix) (default: false)
- `-exclude-extensions`: Comma-separated list of asset extensions to exclude (e.g., js,css,png)

## How It Works

1. **URL Validation**: By default, only processes URLs that have the input URL as a prefix
   - Input: `https://example.com/docs`
   - Valid: `https://example.com/docs/page1`, `https://example.com/docs/sub/page2`
   - Invalid: `https://example.com/other`, `https://other.com/docs`
   - With `-disable-prefix-filter`: All HTTP/HTTPS URLs are valid (allows crawling across domains)

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

### Crawl without URL prefix filtering
```bash
./scraper -url https://example.com -disable-prefix-filter
```

### Exclude specific asset types
```bash
./scraper -url https://example.com -exclude-extensions js,css,png,jpg,gif
```

## Notes

- By default, the scraper respects the URL prefix constraint strictly
- Use `-disable-prefix-filter` to allow crawling across domains (be careful with this option)
- Only HTML pages with substantial content are saved
- Use `-exclude-extensions` to skip downloading specific asset types (js, css, images, etc.)
- Concurrent mode limits to 10 simultaneous requests to avoid overwhelming servers
- State is saved every 10 processed URLs for resilience