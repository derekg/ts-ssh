package platform

import (
	"os"
)

// maskProcessTitle sets a generic process title to hide sensitive information
// from process lists like 'ps aux'
func maskProcessTitle(title string) {
	if title == "" {
		title = "ts-ssh [secure connection]"
	}
	
	// Use platform-specific implementation
	maskProcessTitlePlatform(title)
}

// SetSecureEnvironment clears potentially sensitive environment variables
// and sets up a clean environment for SSH operations
func SetSecureEnvironment() {
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

// HideCredentialsInProcessList applies various techniques to prevent
// credential exposure in process lists
func HideCredentialsInProcessList() {
	// Set generic process title
	maskProcessTitle("ts-ssh [secure]")
	
	// Clean environment of sensitive variables
	SetSecureEnvironment()
}