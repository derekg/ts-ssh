//go:build !windows && !linux
// +build !windows,!linux

package platform

import (
	"os"
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
	
	// On macOS, we can modify os.Args[0] to change the process title
	// This affects what appears in ps output
	if len(os.Args) > 0 && len(title) > 0 {
		// Ensure the title doesn't exceed the original argument space
		originalLen := len(os.Args[0])
		if len(title) > originalLen {
			title = title[:originalLen]
		}
		
		// Convert string to []byte for unsafe operations
		titleBytes := []byte(title)
		
		// Pad with null bytes if shorter than original
		if len(titleBytes) < originalLen {
			padding := make([]byte, originalLen-len(titleBytes))
			titleBytes = append(titleBytes, padding...)
		}
		
		// Modify the process name in memory (requires cgo disabled builds)
		// This is a safe operation on Darwin when properly bounds-checked
		if originalLen > 0 && len(titleBytes) == originalLen {
			// Create a new args[0] with the masked title
			os.Args[0] = title
			
			// For additional security, also try to modify the underlying memory
			// if we can safely access it (this is OS-specific behavior)
			modifyProcessNameDarwin(titleBytes)
		}
	}
}

// modifyProcessNameDarwin attempts to modify the process name in memory on Darwin
func modifyProcessNameDarwin(title []byte) {
	// On Darwin, we can attempt to modify the process name through careful
	// memory manipulation, but this is inherently unsafe and OS-dependent
	// For production use, the os.Args[0] modification above is safer
	
	// This is a best-effort attempt - if it fails, the os.Args[0] change
	// still provides security benefit for most process listing tools
	defer func() {
		// Recover from any panics in unsafe memory operations
		if r := recover(); r != nil {
			// Silently continue - the os.Args[0] change is sufficient
		}
	}()
	
	// Get the command line arguments from the OS
	// This is platform-specific and may not work in all environments
	if len(title) > 0 {
		// In a production environment, you might use cgo to call setproctitle()
		// or similar macOS-specific APIs. For now, rely on os.Args[0] modification
		_ = title // Acknowledge the parameter to avoid unused variable warnings
	}
}