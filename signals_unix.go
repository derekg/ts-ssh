//go:build !windows

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// handleSignalsAndResizeWithTerminalState handles signals and window resizing with terminal state
func handleSignalsAndResizeWithTerminalState(session *ssh.Session, termState *TerminalStateManager, logger *log.Logger) {
	// Set up signal channel for SIGWINCH (window resize)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	
	for {
		select {
		case <-sigCh:
			if termState.IsRaw() {
				fd := termState.GetFD()
				if width, height, err := term.GetSize(fd); err == nil {
					// Send window size change to remote
					session.WindowChange(height, width)
				}
			}
		}
	}
}