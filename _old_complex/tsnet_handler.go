package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

// init suppresses tsnet logging early unless verbose mode is detected.
// This runs before main() to catch any import-time logging from tsnet.
func init() {
	if !isVerboseMode() {
		log.SetOutput(io.Discard)
	}
}

// isVerboseMode checks command line arguments and environment variables
// to determine if verbose logging should be enabled.
func isVerboseMode() bool {
	// Check command line arguments
	for _, arg := range os.Args {
		if arg == "-v" || arg == "--verbose" || arg == "-verbose" {
			return true
		}
		// Handle combined flags like -va or --verbose-auth
		if strings.HasPrefix(arg, "-v") && len(arg) > 2 {
			return true
		}
		if strings.HasPrefix(arg, "--verbose") {
			return true
		}
	}

	// Check environment variables
	return os.Getenv("TS_DEBUG") != "" || os.Getenv("TS_VERBOSE") != ""
}

// initTsNet initializes the tsnet server and returns the server instance, context,
// current Tailscale status, and any error that occurred.
func initTsNet(tsnetDir string, clientHostname string, logger *log.Logger, tsControlURL string, verbose bool) (*tsnet.Server, context.Context, *ipnstate.Status, error) {
	// Re-apply logging configuration to ensure persistence
	if !verbose {
		log.SetOutput(io.Discard)
	}

	// Setup tsnet directory
	if err := setupTsNetDir(&tsnetDir, clientHostname, logger); err != nil {
		return nil, nil, nil, err
	}

	// Create and configure tsnet server
	srv := createTsNetServer(tsnetDir, clientHostname, tsControlURL, logger, verbose)

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
	if !verbose {
		fmt.Fprintf(os.Stderr, "%s\n", T("starting_tailscale_connection"))
	}

	status, err := srv.Up(ctx)
	if err != nil {
		// If context was cancelled, it might be because of a signal during srv.Up.
		// AvoidFatalf here if ctx.Err is not nil, as it's an expected shutdown.
		if ctx.Err() != nil {
			logger.Printf("initTsNet: Context cancelled during srv.Up: %v", ctx.Err())
			return nil, nil, nil, fmt.Errorf("tsnet setup cancelled: %w", ctx.Err())
		}
		logger.Fatalf("Failed to bring up tsnet: %v. If authentication is required, run with -v to see the auth URL.", err)
		return nil, nil, nil, err // Fatalf will exit, but return for completeness.
	}

	// Display auth URL if available from status
	if status != nil && status.AuthURL != "" {
		displayAuthURL(status.AuthURL)
	}

	// It can take a moment for the connection to be fully established and peers to be visible.
	// A small delay can improve reliability of fetching peers immediately after Up.
	logger.Println("Waiting briefly for Tailscale connection to establish...")
	select {
	case <-time.After(3 * time.Second): // ConnectionWaitTime
		// Continue after delay
	case <-ctx.Done():
		logger.Println("initTsNet: Context cancelled while waiting for connection to establish.")
		return nil, nil, nil, fmt.Errorf("tsnet setup cancelled during peer wait: %w", ctx.Err())
	}

	// Refresh status to get the most current information
	currentStatus := refreshTailscaleStatus(ctx, srv, status, logger)

	return srv, ctx, currentStatus, nil
}

// setupTsNetDir ensures the tsnet state directory exists and is properly configured.
func setupTsNetDir(tsnetDir *string, clientHostname string, logger *log.Logger) error {
	if *tsnetDir == "" {
		// Fallback directory name if user.Current() failed in main or not provided.
		// This is less ideal as it's not user-specific.
		*tsnetDir = clientHostname + "-state-dir"
		logger.Printf("Warning: Using default tsnet state directory: %s (consider setting -tsnet-dir)", *tsnetDir)
	}
	if err := os.MkdirAll(*tsnetDir, 0700); err != nil && !os.IsExist(err) {
		logger.Fatalf("Failed to create tsnet state directory %q: %v", *tsnetDir, err)
		return err
	}
	return nil
}

// createTsNetServer creates and configures a tsnet server with appropriate logging.
func createTsNetServer(tsnetDir, clientHostname, tsControlURL string, logger *log.Logger, verbose bool) *tsnet.Server {
	srv := &tsnet.Server{
		Dir:        tsnetDir,
		Hostname:   clientHostname,
		ControlURL: tsControlURL,
	}

	// Configure logging based on verbose mode
	if verbose {
		srv.Logf = logger.Printf
		srv.UserLogf = logger.Printf
	} else {
		// Use a filtered logger that only shows auth URLs once
		srv.UserLogf = createAuthURLLogger()
		srv.Logf = func(string, ...interface{}) {} // Suppress backend logs
	}

	return srv
}

// createAuthURLLogger returns a logging function that filters out noise
// but displays authentication URLs in a clean format.
func createAuthURLLogger() func(string, ...interface{}) {
	var authURLShown bool

	return func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		// Only show messages that contain authentication URLs
		if strings.Contains(msg, "https://login.tailscale.com/") && !authURLShown {
			// Extract just the URL from the message
			if idx := strings.Index(msg, "https://"); idx != -1 {
				url := msg[idx:]
				// Find the end of the URL (space or newline)
				if endIdx := strings.IndexAny(url, " \n\r\t"); endIdx != -1 {
					url = url[:endIdx]
				}
				displayAuthURL(url)
				authURLShown = true
			}
		}
	}
}

// displayAuthURL shows the authentication URL in a clean, consistent format.
func displayAuthURL(url string) {
	fmt.Fprintf(os.Stderr, "\n%s\n%s\n\n", T("to_authenticate_visit"), url)
}

// refreshTailscaleStatus attempts to get the most current Tailscale status
// with retry logic for improved reliability.
func refreshTailscaleStatus(ctx context.Context, srv *tsnet.Server, initialStatus *ipnstate.Status, logger *log.Logger) *ipnstate.Status {
	currentStatus := initialStatus
	var stateErr error

	for i := 0; i < 3; i++ { // MaxStateRetries
		if i > 0 {
			logger.Printf("Attempting to refresh Tailscale status (attempt %d/%d)...", i+1, 3) // MaxStateRetries
			select {
			case <-time.After(1 * time.Second): // StateRetryDelay
				// Continue after delay
			case <-ctx.Done():
				logger.Println("initTsNet: Context cancelled during status refresh retry.")
				return currentStatus // Return what we have
			}
		}

		// Attempt to get current status after connection is established
		client, err := srv.LocalClient()
		if err != nil {
			stateErr = fmt.Errorf("failed to get local client: %w", err)
			logger.Printf("Warning: %v", stateErr)
			continue
		}

		updatedStatus, err := client.Status(ctx)
		if err != nil {
			stateErr = fmt.Errorf("failed to get updated status: %w", err)
			logger.Printf("Warning: %v", stateErr)
			continue
		}

		currentStatus = updatedStatus
		stateErr = nil
		break
	}

	if stateErr != nil {
		// We have a connection but couldn't refresh status - proceed with initial status
		logger.Printf("Warning: Using initial status due to refresh failures: %v", stateErr)
	}

	return currentStatus
}
