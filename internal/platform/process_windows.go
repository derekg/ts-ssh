//go:build windows
// +build windows

package platform

// maskProcessTitlePlatform sets a process title on Windows
func maskProcessTitlePlatform(title string) {
	// On Windows, process title masking is more limited
	// Windows doesn't have the same prctl mechanism as Linux
	// The main security benefit comes from using SSH config files
	// instead of command line arguments

	// Potential Windows-specific implementations could use:
	// - SetConsoleTitle() for console applications
	// - Process name changes via Windows APIs
	// For now, this is a no-op on Windows
}
