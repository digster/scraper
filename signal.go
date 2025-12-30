package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalHandler creates a context that is cancelled when SIGINT or SIGTERM is received.
// Returns the context and a cancel function.
func SetupSignalHandler() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigChan:
			// Log signal received - use fmt since this is before crawler is created
			println() // New line after any progress output
			println("Received signal:", sig.String())
			println("Initiating graceful shutdown...")
			cancel()
		case <-ctx.Done():
			// Context was cancelled elsewhere
		}
		signal.Stop(sigChan)
	}()

	return ctx, cancel
}
