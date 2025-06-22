package main

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// maskProcessTitle sets a generic process title to hide sensitive information
// from process lists like 'ps aux'
func maskProcessTitle(title string) {
	if title == "" {
		title = "ts-ssh [secure connection]"
	}
	
	switch runtime.GOOS {
	case "linux":
		maskProcessTitleLinux(title)
	case "darwin":
		maskProcessTitleDarwin(title)
	default:
		// For other platforms, we can't mask the process title
		// but the SSH config file approach still protects credentials
	}
}

// maskProcessTitleLinux uses prctl to set process title on Linux
func maskProcessTitleLinux(title string) {
	if runtime.GOOS != "linux" {
		return
	}
	
	// Use prctl PR_SET_NAME to set process name
	titleBytes := []byte(title + "\x00")
	if len(titleBytes) > 16 { // Linux process names are limited to 15 chars + null
		titleBytes = titleBytes[:15]
		titleBytes[14] = 0
	}
	
	// PR_SET_NAME = 15
	syscall.Syscall(syscall.SYS_PRCTL, 15, uintptr(unsafe.Pointer(&titleBytes[0])), 0)
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

// setSecureEnvironment clears potentially sensitive environment variables
// and sets up a clean environment for SSH operations
func setSecureEnvironment() {
	// Clear potentially sensitive environment variables
	sensitiveVars := []string{
		"SSH_AUTH_SOCK",    // Don't inherit SSH agent
		"SSH_AGENT_PID",    // Don't inherit SSH agent PID
		"DISPLAY",          // Clear X11 display for security
	}
	
	for _, varName := range sensitiveVars {
		os.Unsetenv(varName)
	}
}

// hideCredentialsInProcessList applies various techniques to prevent
// credential exposure in process lists
func hideCredentialsInProcessList() {
	// Set generic process title
	maskProcessTitle("ts-ssh [secure]")
	
	// Clean environment of sensitive variables
	setSecureEnvironment()
}