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

- **Hierarchical Crawling**: Only crawls URLs discovered from the input URL and its children (tree-based discovery)
- **Depth Control**: Respects maximum crawl depth based on discovery hierarchy
- **Asset Filtering**: Exclude specific file extensions (js, css, images, etc.) from being downloaded
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
          -prefix-filter https://example.com/api \
          -exclude-extensions js,css,png,jpg \
          -link-selectors "a.internal,.nav-link"
```

### Command Line Arguments

- `-url`: Starting URL to scrape (required)
- `-concurrent`: Run in concurrent mode (default: false)
- `-delay`: Delay between fetches (default: 1s)
- `-depth`: Maximum crawl depth based on discovery hierarchy (default: 10)
- `-output`: Output directory for scraped content (default: "scraped_content")
- `-state`: State file for resume functionality (default: "crawler_state.json")
- `-prefix-filter`: URL prefix to filter by (if not specified, no prefix filtering is applied)
- `-exclude-extensions`: Comma-separated list of asset extensions to exclude (e.g., js,css,png)
- `-link-selectors`: Comma-separated list of CSS selectors to filter links (e.g., 'a.internal,.nav-link')
- `-verbose`: Enable verbose debug output (default: false)

## How It Works

1. **Hierarchical Discovery**: Only processes URLs discovered through the crawling tree starting from the input URL
   - Input: `https://a.com/a`
   - If `b.com` is linked from `a.com/a`, it will be crawled
   - If `c.com` is linked from `b.com`, it will also be crawled  
   - But if `d.com` is linked from `a.com/e` (not discovered through our tree), it won't be crawled
   - Depth is tracked based on discovery steps, not URL structure

2. **URL Filtering Modes**:
   - **Default (No Prefix Filtering)**: Crawls any HTTP/HTTPS URL discovered through the tree, regardless of domain
     - Input: `https://example.com/docs` → Will crawl any domain linked from the discovery tree
   - **With Prefix Filtering**: Use `-prefix-filter <url>` to only crawl URLs matching a specific prefix
     - Example: `-prefix-filter https://example.com/api` → Only crawls URLs starting with `https://example.com/api`
     - Even if other URLs are discovered through the tree, they'll be skipped if they don't match the prefix

3. **Content Filtering**: Pages are only saved if they contain meaningful content (>100 characters of text after removing scripts and styles)

4. **Asset Filtering**: URLs with excluded extensions (specified via `-exclude-extensions`) are skipped

5. **Link Selector Filtering**: Only processes links that match specified CSS selectors
   - **Default**: Processes all links with `href` attributes (`a[href]`)
   - **With `-link-selectors`**: Only processes links matching the specified selectors
   - Examples: `a.internal` (links with class 'internal'), `.nav-link` (any element with class 'nav-link'), `#menu a` (links inside element with id 'menu')

6. **File Storage**: Each page is saved as:
   - `{path}.html`: The actual HTML content
   - `{path}.meta.json`: Metadata including original URL, timestamp, and size
   - Query parameters are included in filenames to avoid collisions (e.g., `/articles?id=1` → `articles_id-1.html`)

7. **Resume Capability**: State is saved periodically and can be resumed by running the same command again

## Output Structure

```
scraped_content/
├── index.html                    # Root page
├── index.meta.json
├── articles.html                 # /articles
├── articles.meta.json
├── articles_id-1.html            # /articles?id=1
├── articles_id-1.meta.json
├── articles_id-2.html            # /articles?id=2
├── articles_id-2.meta.json
├── blog/
│   └── posts_page-1.html         # /blog/posts?page=1
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

### Exclude specific asset types
```bash
./scraper -url https://example.com -exclude-extensions js,css,png,jpg,gif
```

### Limit crawl depth
```bash
./scraper -url https://example.com -depth 3
```

### Use prefix filtering to limit to specific URLs
```bash
./scraper -url https://example.com -prefix-filter https://api.example.com
```

### Only follow specific link types
```bash
./scraper -url https://example.com -link-selectors "a.internal,.nav-link,#menu a"
```

## Notes

- The scraper uses hierarchical discovery - only URLs found through the crawling tree are processed
- By default, no prefix filtering is applied - any domain discovered through the tree will be crawled
- Use `-prefix-filter <url>` to limit crawling to URLs matching a specific prefix
- Use `-link-selectors` to only follow links matching specific CSS selectors (default: all links with href)
- Depth is measured by discovery steps, not URL path depth
- Only HTML pages with substantial content are saved
- Use `-exclude-extensions` to skip downloading specific asset types (js, css, images, etc.)
- Concurrent mode limits to 10 simultaneous requests to avoid overwhelming servers
- State is saved every 10 processed URLs for resilience