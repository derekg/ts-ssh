//go:build !windows && !linux
// +build !windows,!linux

package main

import (
	"runtime"
)

// maskProcessTitlePlatform sets a process title on Unix-like systems (non-Linux)
func maskProcessTitlePlatform(title string) {
	switch runtime.GOOS {
	case "darwin":
		maskProcessTitleDarwin(title)
	default:
		// For other Unix-like systems, we don't have specific implementations
		// but the SSH config file approach still protects credentials
	}
}

// maskProcessTitleDarwin sets process title on macOS/Darwin
func maskProcessTitleDarwin(title string) {
	if runtime.GOOS != "darwin" {
		return
	}
	
	// On macOS, we can set the process title by modifying argv[0]
	// This is more complex and platform-specific, so for now we skip it
	// The SSH config file approach provides the main security benefit
}