//go:build windows

package main

import (
	"log"

	"golang.org/x/crypto/ssh"
)

// handleSignalsAndResizeWithTerminalState handles signals and window resizing with terminal state
// On Windows, SIGWINCH doesn't exist, so this is a no-op
func handleSignalsAndResizeWithTerminalState(session *ssh.Session, termState *TerminalStateManager, logger *log.Logger) {
	// Windows doesn't support SIGWINCH signal for terminal resize
	// This function is a no-op on Windows
	return
}
