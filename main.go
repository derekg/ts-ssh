package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
		log.Fatalf("Error: %v", err)
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