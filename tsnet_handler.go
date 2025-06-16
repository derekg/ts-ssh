package main

import (
	"context"
	"fmt"
	"log"
	// "net/http" // Removed as it's not used
	"os"
	"time"

	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

// initTsNet initializes the tsnet server and returns the server instance, context,
// current Tailscale status, and any error that occurred.
func initTsNet(tsnetDir string, clientHostname string, logger *log.Logger, tsControlURL string, verbose bool, tuiMode bool) (*tsnet.Server, context.Context, *ipnstate.Status, error) {
	if tsnetDir == "" {
		// Fallback directory name if user.Current() failed in main or not provided.
		// This is less ideal as it's not user-specific.
		tsnetDir = clientHostname + "-state-dir"
		logger.Printf("Warning: Using default tsnet state directory: %s (consider setting -tsnet-dir)", tsnetDir)
	}
	if err := os.MkdirAll(tsnetDir, 0700); err != nil && !os.IsExist(err) {
		logger.Fatalf("Failed to create tsnet state directory %q: %v", tsnetDir, err)
		return nil, nil, nil, err
	}

	srv := &tsnet.Server{
		Dir:        tsnetDir,
		Hostname:   clientHostname,
		Logf:       logger.Printf,
		ControlURL: tsControlURL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Ensure server is closed when context is done.
	go func() {
		<-ctx.Done()
		logger.Println("initTsNet: Main context cancelled, ensuring tsnet server is closed.")
		if err := srv.Close(); err != nil {
			// Log, but don't make this fatal as we might be shutting down anyway.
			logger.Printf("initTsNet: Error closing tsnet server: %v", err)
		}
		cancel() // Ensure cancel is called to clean up resources associated with this context.
	}()

	logger.Printf("Initializing tsnet in directory: %s for client %s", tsnetDir, clientHostname)

	// Attempt to bring up the tsnet server.
	// srv.Up will block until the server is up or context is cancelled.
	if !verbose && !tuiMode {
		fmt.Fprintf(os.Stderr, "Starting Tailscale connection... You may need to authenticate.\nLook for a URL printed below if needed.\n")
	}
	status, err := srv.Up(ctx)
	if err != nil {
		// If context was cancelled, it might be because of a signal during srv.Up.
		// AvoidFatalf here if ctx.Err is not nil, as it's an expected shutdown.
		if ctx.Err() != nil {
			logger.Printf("initTsNet: Context cancelled during srv.Up: %v", ctx.Err())
			return nil, nil, nil, fmt.Errorf("tsnet setup cancelled: %w", ctx.Err())
		}
		logger.Fatalf("Failed to bring up tsnet: %v. If authentication is required, look for a URL in the logs (run with -v if not already).", err)
		return nil, nil, nil, err // Fatalf will exit, but return for completeness.
	}

	// If not verbose and an AuthURL is present, print it to help the user.
	if !verbose && !tuiMode && status != nil && status.AuthURL != "" {
		fmt.Fprintf(os.Stderr, "\nTo authenticate, visit:\n%s\n", status.AuthURL)
		fmt.Fprintf(os.Stderr, "Please authenticate in the browser. The client will then attempt to connect.\n")
	}

	// It can take a moment for the connection to be fully established and peers to be visible.
	// A small delay can improve reliability of fetching peers immediately after Up.
	logger.Println("Waiting briefly for Tailscale connection to establish...")
	select {
	case <-time.After(3 * time.Second):
		// Continue after delay
	case <-ctx.Done():
		logger.Println("initTsNet: Context cancelled while waiting for connection to establish.")
		return nil, nil, nil, fmt.Errorf("tsnet setup cancelled during peer wait: %w", ctx.Err())
	}
	
	// Attempt to get the most current status after initialization.
	currentStatus := status // Default to initial status from Up()
	lc, errClient := srv.LocalClient()
	if errClient != nil {
		logger.Printf("Warning: Failed to get LocalClient to update Tailscale status: %v. Using potentially stale status from Up().", errClient)
	} else if lc == nil {
		logger.Printf("Warning: LocalClient is nil, cannot update Tailscale status. Using potentially stale status from Up().")
	} else {
		// Use a short timeout for this status check as the connection should be up.
		statusCtx, statusCancel := context.WithTimeout(ctx, 5*time.Second)
		defer statusCancel()
		updatedStatus, errStatus := lc.Status(statusCtx)
		if errStatus != nil {
			logger.Printf("Warning: Failed to get updated Tailscale status after initial Up: %v. Using potentially stale status from Up().", errStatus)
		} else {
			logger.Println("Successfully fetched updated Tailscale status.")
			currentStatus = updatedStatus
		}
	}

	if currentStatus != nil && verbose {
		logger.Printf("Tailscale status: Self: %s, Peers: %d", currentStatus.Self.DNSName, len(currentStatus.Peer))
	}

	return srv, ctx, currentStatus, nil
}
