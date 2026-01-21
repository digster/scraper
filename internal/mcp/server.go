package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"scraper/internal/api"
)

// Server wraps the MCP server with scraper functionality
type Server struct {
	mcpServer  *server.MCPServer
	jobManager *api.JobManager
}

// NewServer creates a new MCP server for the scraper
func NewServer(maxJobs int) *Server {
	// Create job manager
	jobManager := api.NewJobManager(maxJobs)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"Web Scraper",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	s := &Server{
		mcpServer:  mcpServer,
		jobManager: jobManager,
	}

	// Register all tools
	s.registerTools()

	return s
}

// registerTools adds all scraper tools to the MCP server
func (s *Server) registerTools() {
	// scraper_start - Start a new crawl job
	s.mcpServer.AddTool(
		mcp.NewTool("scraper_start",
			mcp.WithDescription("Start a new web crawl job. Returns immediately with a job ID that can be used to track progress."),
			mcp.WithString("url",
				mcp.Required(),
				mcp.Description("Target URL to start crawling from"),
			),
			mcp.WithNumber("maxDepth",
				mcp.Description("Maximum link depth to crawl (default: 10)"),
			),
			mcp.WithBoolean("concurrent",
				mcp.Description("Enable concurrent crawling for faster processing"),
			),
			mcp.WithString("delay",
				mcp.Description("Delay between requests (e.g. '500ms' or '1s')"),
			),
			mcp.WithString("outputDir",
				mcp.Description("Directory to save crawled content"),
			),
			mcp.WithString("prefixFilter",
				mcp.Description("Only crawl URLs starting with this prefix"),
			),
			mcp.WithString("fetchMode",
				mcp.Description("Fetch mode: 'http' for fast requests or 'browser' for JavaScript-rendered pages"),
				mcp.Enum("http", "browser"),
			),
			mcp.WithBoolean("headless",
				mcp.Description("Run browser in headless mode (default: true)"),
			),
			mcp.WithBoolean("waitForLogin",
				mcp.Description("Wait for manual login before starting crawl (browser mode only)"),
			),
			mcp.WithString("userAgent",
				mcp.Description("Custom User-Agent string"),
			),
			mcp.WithBoolean("ignoreRobots",
				mcp.Description("Ignore robots.txt restrictions"),
			),
			mcp.WithNumber("minContent",
				mcp.Description("Minimum content length to save a page (default: 100)"),
			),
			mcp.WithObject("antiBot",
				mcp.Description("Anti-bot detection evasion settings (browser mode only)"),
			),
			mcp.WithBoolean("normalizeUrls",
				mcp.Description("Enable URL normalization for better duplicate detection (default: true)"),
			),
			mcp.WithBoolean("lowercasePaths",
				mcp.Description("Lowercase URL paths during normalization (default: false, use with caution)"),
			),
			mcp.WithString("pageLoadWait",
				mcp.Description("Time to wait after page load for dynamic content (browser mode, e.g. '500ms', '2s')"),
			),
			mcp.WithBoolean("disableReadability",
				mcp.Description("Disable readability content extraction and save raw HTML"),
			),
			mcp.WithObject("pagination",
				mcp.Description("Click-based pagination settings (browser mode only). Properties: enable (bool), selector (CSS selector), maxClicks (int), waitAfterClick (duration), waitSelector (CSS), stopOnDuplicate (bool)"),
			),
			mcp.WithArray("excludeExtensions",
				mcp.Description("File extensions to exclude from crawling (e.g. ['.pdf', '.zip', '.png'])"),
			),
			mcp.WithArray("linkSelectors",
				mcp.Description("CSS selectors to find links (defaults to standard link tags, e.g. ['a.nav-link', '.content a'])"),
			),
		),
		s.handleStart,
	)

	// scraper_list - List all jobs
	s.mcpServer.AddTool(
		mcp.NewTool("scraper_list",
			mcp.WithDescription("List all crawl jobs with their current status"),
		),
		s.handleList,
	)

	// scraper_get - Get job details
	s.mcpServer.AddTool(
		mcp.NewTool("scraper_get",
			mcp.WithDescription("Get detailed information about a specific crawl job including metrics"),
			mcp.WithString("jobId",
				mcp.Required(),
				mcp.Description("Job ID returned from scraper_start"),
			),
		),
		s.handleGet,
	)

	// scraper_stop - Stop a job
	s.mcpServer.AddTool(
		mcp.NewTool("scraper_stop",
			mcp.WithDescription("Stop a running or paused crawl job"),
			mcp.WithString("jobId",
				mcp.Required(),
				mcp.Description("Job ID to stop"),
			),
		),
		s.handleStop,
	)

	// scraper_pause - Pause a job
	s.mcpServer.AddTool(
		mcp.NewTool("scraper_pause",
			mcp.WithDescription("Pause a running crawl job (can be resumed later)"),
			mcp.WithString("jobId",
				mcp.Required(),
				mcp.Description("Job ID to pause"),
			),
		),
		s.handlePause,
	)

	// scraper_resume - Resume a job
	s.mcpServer.AddTool(
		mcp.NewTool("scraper_resume",
			mcp.WithDescription("Resume a paused crawl job"),
			mcp.WithString("jobId",
				mcp.Required(),
				mcp.Description("Job ID to resume"),
			),
		),
		s.handleResume,
	)

	// scraper_metrics - Get real-time metrics
	s.mcpServer.AddTool(
		mcp.NewTool("scraper_metrics",
			mcp.WithDescription("Get real-time metrics for a crawl job (URLs processed, saved, errors, etc.)"),
			mcp.WithString("jobId",
				mcp.Required(),
				mcp.Description("Job ID to get metrics for"),
			),
		),
		s.handleMetrics,
	)

	// scraper_confirm_login - Confirm browser login
	s.mcpServer.AddTool(
		mcp.NewTool("scraper_confirm_login",
			mcp.WithDescription("Confirm that manual browser login is complete (for jobs waiting for login)"),
			mcp.WithString("jobId",
				mcp.Required(),
				mcp.Description("Job ID waiting for login confirmation"),
			),
		),
		s.handleConfirmLogin,
	)

	// scraper_wait - Wait for job completion
	s.mcpServer.AddTool(
		mcp.NewTool("scraper_wait",
			mcp.WithDescription("Wait for a crawl job to complete, returning final metrics. Polls every 2 seconds by default."),
			mcp.WithString("jobId",
				mcp.Required(),
				mcp.Description("Job ID to wait for"),
			),
			mcp.WithNumber("timeoutSeconds",
				mcp.Description("Maximum seconds to wait (default: 300)"),
			),
			mcp.WithNumber("pollIntervalMs",
				mcp.Description("Polling interval in milliseconds (default: 2000)"),
			),
		),
		s.handleWait,
	)
}

// Serve starts the MCP server with stdio transport
func (s *Server) Serve() error {
	return server.ServeStdio(s.mcpServer)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	s.jobManager.Shutdown()
}

// GetJobManager returns the job manager for testing
func (s *Server) GetJobManager() *api.JobManager {
	return s.jobManager
}
