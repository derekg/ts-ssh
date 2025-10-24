package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/derekg/ts-ssh/internal/security"
)

// version is set at build time via -ldflags "-X main.version=..."; default is "dev".
var version = "dev"

func main() {
	// Initialize security audit logging early (if enabled via environment variables)
	if err := security.InitSecurityLogger(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize security audit logging: %v\n", err)
	}
	// Ensure security logger is properly closed on exit
	defer security.CloseSecurityLogger()

	// Check if we should use the legacy CLI (for backwards compatibility)
	if shouldUseLegacyCLI() {
		runLegacyCLI()
		return
	}

	// Use the new Fang-enhanced CLI
	ctx := context.Background()
	if err := ExecuteWithFang(ctx); err != nil {
		// Don't print the error here as Fang will handle it with proper styling
		os.Exit(1)
	}
}

// shouldUseLegacyCLI determines if we should use the legacy CLI
// This can be controlled by an environment variable for compatibility
func shouldUseLegacyCLI() bool {
	// Check for explicit legacy mode
	if os.Getenv("TS_SSH_LEGACY_CLI") == "1" {
		return true
	}

	// Check if the command line looks like it's using the old style
	// (this helps with backwards compatibility during transition)
	if len(os.Args) > 1 {
		firstArg := os.Args[1]
		// If first arg starts with user@ or contains :, it's likely a connection target
		if strings.Contains(firstArg, "@") ||
			(strings.Contains(firstArg, ":") && !strings.HasPrefix(firstArg, "-")) {
			// Insert "connect" subcommand for backwards compatibility
			newArgs := []string{os.Args[0], "connect"}
			newArgs = append(newArgs, os.Args[1:]...)
			os.Args = newArgs
		}
	}

	return false
}

// runLegacyCLI runs the original simple CLI implementation
func runLegacyCLI() {
	// Create the CLI application
	cli := NewCLI()

	// Handle special case for backwards compatibility: if first arg looks like a target,
	// and no subcommand is specified, default to connect command
	if len(os.Args) > 1 && !isSubcommand(os.Args[1]) {
		// Insert "connect" as the first argument to maintain compatibility
		args := []string{os.Args[0], "connect"}
		args = append(args, os.Args[1:]...)
		os.Args = args
	}

	// Run the CLI
	ctx := context.Background()
	if err := cli.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// isSubcommand checks if the given argument is a known subcommand
func isSubcommand(arg string) bool {
	subcommands := []string{
		"connect", "scp", "list", "exec", "multi", "config", "pqc", "version",
		"help", "-h", "--help", "-v", "--version",
	}

	for _, cmd := range subcommands {
		if arg == cmd {
			return true
		}
	}

	return false
}
