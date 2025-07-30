package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

type Config struct {
	URL        string
	Concurrent bool
	Delay      time.Duration
	MaxDepth   int
	OutputDir  string
	StateFile  string
}

func main() {
	var config Config
	
	flag.StringVar(&config.URL, "url", "", "Starting URL to scrape")
	flag.BoolVar(&config.Concurrent, "concurrent", false, "Run in concurrent mode")
	flag.DurationVar(&config.Delay, "delay", time.Second, "Delay between fetches")
	flag.IntVar(&config.MaxDepth, "depth", 10, "Maximum crawl depth")
	flag.StringVar(&config.OutputDir, "output", "scraped_content", "Output directory")
	flag.StringVar(&config.StateFile, "state", "crawler_state.json", "State file for resume functionality")
	flag.Parse()

	if config.URL == "" {
		fmt.Println("Error: URL is required")
		flag.Usage()
		os.Exit(1)
	}

	crawler := NewCrawler(config)
	if err := crawler.Start(); err != nil {
		log.Fatal(err)
	}
}