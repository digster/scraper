// Command scraper-mcp runs the web scraper as an MCP (Model Context Protocol) server.
// This allows LLM agents like Claude to use the scraper as a tool.
//
// Usage:
//
//	scraper-mcp [flags]
//
// Flags:
//
//	-max-jobs int
//	      Maximum concurrent crawl jobs (default 5)
//
// Configuration in Claude Code (~/.claude/mcp.json):
//
//	{
//	  "mcpServers": {
//	    "scraper": {
//	      "command": "/path/to/scraper-mcp",
//	      "args": ["--max-jobs", "5"]
//	    }
//	  }
//	}
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"scraper/internal/mcp"
)

func main() {
	// Parse command-line flags
	maxJobs := flag.Int("max-jobs", 5, "Maximum concurrent crawl jobs")
	flag.Parse()

	// Create and start the MCP server
	server := mcp.NewServer(*maxJobs)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		server.Shutdown()
		os.Exit(0)
	}()

	// Start serving (blocks until error or shutdown)
	if err := server.Serve(); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
