package main

import (
	"io"
	"log"
	"os"
)

// createQuietLogger creates a logger that doesn't output to stdout/stderr
// This is useful for tests to avoid cluttering test output
func createQuietLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

// createVerboseTestLogger creates a logger for verbose test output
func createVerboseTestLogger() *log.Logger {
	return log.New(os.Stdout, "[TEST] ", log.LstdFlags)
}

// isTestEnvironment checks if we're running in a test environment
func isTestEnvironment() bool {
	return os.Getenv("GO_TEST") != "" ||
		os.Getenv("TESTING") != "" ||
		len(os.Args) > 0 && os.Args[0] == "go"
}
