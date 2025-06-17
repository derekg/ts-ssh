package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

// ScreenStyleSwitcher implements a screen/tmux-like session manager
type ScreenStyleSwitcher struct {
	sessionManager   *SessionManager
	sessionConnector *SessionConnector
	logger           *log.Logger
	appCtx           context.Context
	
	// Terminal state
	terminalFd       int
	originalTerminal *term.State
	
	// Mode state
	inCommandMode    bool
	
	// SSH connection parameters for new sessions
	sshUser          string
	sshKeyPath       string
	insecureHostKey  bool
}

// NewScreenStyleSwitcher creates a new screen-style session switcher
func NewScreenStyleSwitcher(sessionManager *SessionManager, sessionConnector *SessionConnector, 
	logger *log.Logger, appCtx context.Context, sshUser, sshKeyPath string, insecureHostKey bool) *ScreenStyleSwitcher {
	
	return &ScreenStyleSwitcher{
		sessionManager:   sessionManager,
		sessionConnector: sessionConnector,
		logger:           logger,
		appCtx:           appCtx,
		sshUser:          sshUser,
		sshKeyPath:       sshKeyPath,
		insecureHostKey:  insecureHostKey,
		terminalFd:       int(os.Stdin.Fd()),
	}
}

// Start begins the screen-style session management
func (s *ScreenStyleSwitcher) Start() error {
	s.logger.Println("Starting screen-style session switcher...")
	
	// Set terminal to raw mode for proper input handling
	if term.IsTerminal(s.terminalFd) {
		var err error
		s.originalTerminal, err = term.MakeRaw(s.terminalFd)
		if err != nil {
			return fmt.Errorf("failed to set terminal to raw mode: %w", err)
		}
		defer s.restoreTerminal()
	}
	
	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		s.logger.Println("Signal received, shutting down gracefully...")
		s.restoreTerminal()
		s.sessionManager.CloseAllSessions()
		os.Exit(0)
	}()
	
	// Start output forwarding for all sessions
	s.startOutputForwarding()
	
	// Show initial status
	s.showWelcomeMessage()
	
	// Start input handling loop
	return s.handleInput()
}

// restoreTerminal restores the original terminal state
func (s *ScreenStyleSwitcher) restoreTerminal() {
	if s.originalTerminal != nil && term.IsTerminal(s.terminalFd) {
		term.Restore(s.terminalFd, s.originalTerminal)
	}
}

// showWelcomeMessage displays the initial welcome and help
func (s *ScreenStyleSwitcher) showWelcomeMessage() {
	fmt.Print("\r\n")
	fmt.Printf("=== Screen-Style Multi-Session Manager ===\r\n")
	fmt.Printf("Connected to %d session(s)\r\n", s.sessionManager.GetConnectedSessionCount())
	fmt.Printf("\r\n")
	s.showSessionList()
	fmt.Printf("\r\n")
	fmt.Printf("Commands: Ctrl+A then...\r\n")
	fmt.Printf("  n - New session\r\n")
	fmt.Printf("  Space - Next session\r\n")
	fmt.Printf("  p - Previous session\r\n")
	fmt.Printf("  l - List sessions\r\n")
	fmt.Printf("  d - Detach current session\r\n")
	fmt.Printf("  q - Quit all sessions\r\n")
	fmt.Printf("  ? - Show this help\r\n")
	fmt.Printf("\r\n")
	
	// Show current session
	if activeSession, err := s.sessionManager.GetActiveSession(); err == nil {
		fmt.Printf("Active session: %s (Status: %s)\r\n", activeSession.GetDisplayName(), activeSession.GetStatus())
		fmt.Printf("Type normally to send to active session. Use Ctrl+A for commands.\r\n")
		fmt.Printf("If input seems stuck, try Ctrl+A ? for help or Ctrl+A q to quit.\r\n")
	} else {
		fmt.Printf("No active session. Use Ctrl+A n to create a new session.\r\n")
	}
	fmt.Printf("\r\n")
}

// showSessionList displays all current sessions
func (s *ScreenStyleSwitcher) showSessionList() {
	sessions := s.sessionManager.ListSessions()
	if len(sessions) == 0 {
		fmt.Printf("No sessions active.\r\n")
		return
	}
	
	activeSessionID := ""
	if activeSession, err := s.sessionManager.GetActiveSession(); err == nil {
		activeSessionID = activeSession.ID
	}
	
	fmt.Printf("Sessions:\r\n")
	for i, session := range sessions {
		if session.GetStatus() == SessionStatusClosed {
			continue
		}
		
		prefix := "  "
		if session.ID == activeSessionID {
			prefix = "* "
		}
		
		fmt.Printf("%s%d: %s [%s]\r\n", prefix, i, session.GetDisplayName(), session.GetStatus())
	}
}

// handleInput processes keyboard input
func (s *ScreenStyleSwitcher) handleInput() error {
	buffer := make([]byte, 1)
	
	for {
		select {
		case <-s.appCtx.Done():
			return nil
		default:
			// Read single byte directly from stdin
			n, err := os.Stdin.Read(buffer)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				s.logger.Printf("Input read error: %v", err)
				return err
			}
			
			if n == 0 {
				continue
			}
			
			b := buffer[0]
			
			// Debug logging for Ctrl+A detection
			if b == 1 {
				s.logger.Printf("Ctrl+A detected (byte value: %d)", b)
			}
			
			// Check for Ctrl+A (ASCII 1)
			if b == 1 { // Ctrl+A
				s.inCommandMode = true
				s.showCommandPrompt()
				continue
			}
			
			// If in command mode, handle command
			if s.inCommandMode {
				s.logger.Printf("Command mode: received key %c (byte: %d)", b, b)
				s.inCommandMode = false
				s.handleCommand(b)
				continue
			}
			
			// Normal mode - send to active session
			s.sendToActiveSession([]byte{b})
		}
	}
}

// showCommandPrompt shows the command mode prompt
func (s *ScreenStyleSwitcher) showCommandPrompt() {
	fmt.Printf("\r\n[Command mode - press key] ")
}

// handleCommand processes command mode input
func (s *ScreenStyleSwitcher) handleCommand(cmd byte) {
	defer func() {
		// Always ensure we're out of command mode
		s.inCommandMode = false
	}()
	
	switch cmd {
	case 'n', 'N':
		s.createNewSession()
	case ' ': // Space for next session
		s.switchToNextSession()
	case 'p', 'P':
		s.switchToPreviousSession()
	case 'l', 'L':
		s.listSessions()
	case 'd', 'D':
		s.detachCurrentSession()
	case 'q', 'Q':
		s.quitAllSessions()
		return
	case '?', 'h', 'H':
		s.showHelp()
	case 1: // Ctrl+A again - send literal Ctrl+A to session
		s.sendToActiveSession([]byte{1})
		return // Don't show "back to session" message
	default:
		fmt.Printf("\r\nUnknown command: %c (press Ctrl+A ? for help)\r\n", cmd)
	}
	
	// Return to session after command
	fmt.Printf("\r\n")
	if activeSession, err := s.sessionManager.GetActiveSession(); err == nil {
		fmt.Printf("Back to session: %s\r\n", activeSession.GetDisplayName())
	} else {
		fmt.Printf("No active session. Use Ctrl+A n to create one.\r\n")
	}
}

// createNewSession prompts for a new session
func (s *ScreenStyleSwitcher) createNewSession() {
	fmt.Printf("\r\nEnter hostname for new session: ")
	
	// Temporarily restore terminal for input
	if s.originalTerminal != nil {
		term.Restore(s.terminalFd, s.originalTerminal)
	}
	
	reader := bufio.NewReader(os.Stdin)
	hostname, err := reader.ReadString('\n')
	
	// Set back to raw mode
	if term.IsTerminal(s.terminalFd) {
		s.originalTerminal, _ = term.MakeRaw(s.terminalFd)
	}
	
	if err != nil {
		fmt.Printf("Error reading hostname: %v\r\n", err)
		return
	}
	
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		fmt.Printf("No hostname provided.\r\n")
		return
	}
	
	// Create and connect new session
	fmt.Printf("Creating session for %s...\r\n", hostname)
	session, err := s.sessionManager.CreateSession(hostname, s.sshUser)
	if err != nil {
		fmt.Printf("Failed to create session: %v\r\n", err)
		return
	}
	
	err = s.sessionConnector.ConnectSession(session)
	if err != nil {
		fmt.Printf("Failed to connect to %s: %v\r\n", hostname, err)
		s.sessionManager.CloseSession(session.ID)
		return
	}
	
	// Start output forwarding for the new session
	go s.forwardSessionOutput(session)
	
	// Switch to the new session
	s.sessionManager.SetActiveSession(session.ID)
	fmt.Printf("Connected to %s. Now active session.\r\n", hostname)
}

// switchToNextSession switches to the next session
func (s *ScreenStyleSwitcher) switchToNextSession() {
	err := s.sessionManager.SwitchToNextSession()
	if err != nil {
		fmt.Printf("\r\nError switching session: %v\r\n", err)
		return
	}
	
	if activeSession, err := s.sessionManager.GetActiveSession(); err == nil {
		fmt.Printf("\r\nSwitched to: %s\r\n", activeSession.GetDisplayName())
	}
}

// switchToPreviousSession switches to the previous session
func (s *ScreenStyleSwitcher) switchToPreviousSession() {
	err := s.sessionManager.SwitchToPrevSession()
	if err != nil {
		fmt.Printf("\r\nError switching session: %v\r\n", err)
		return
	}
	
	if activeSession, err := s.sessionManager.GetActiveSession(); err == nil {
		fmt.Printf("\r\nSwitched to: %s\r\n", activeSession.GetDisplayName())
	}
}

// listSessions shows all sessions
func (s *ScreenStyleSwitcher) listSessions() {
	fmt.Printf("\r\n")
	s.showSessionList()
}

// detachCurrentSession closes the current session
func (s *ScreenStyleSwitcher) detachCurrentSession() {
	activeSession, err := s.sessionManager.GetActiveSession()
	if err != nil {
		fmt.Printf("\r\nNo active session to detach.\r\n")
		return
	}
	
	sessionName := activeSession.GetDisplayName()
	sessionID := activeSession.ID
	
	err = s.sessionManager.CloseSession(sessionID)
	if err != nil {
		fmt.Printf("\r\nError detaching session: %v\r\n", err)
		return
	}
	
	fmt.Printf("\r\nDetached session: %s\r\n", sessionName)
	
	// If no more sessions, show message
	if s.sessionManager.GetConnectedSessionCount() == 0 {
		fmt.Printf("No more sessions. Use Ctrl+A n to create a new session or Ctrl+A q to quit.\r\n")
	}
}

// quitAllSessions exits the switcher
func (s *ScreenStyleSwitcher) quitAllSessions() {
	fmt.Printf("\r\nClosing all sessions...\r\n")
	s.sessionManager.CloseAllSessions()
	fmt.Printf("All sessions closed. Goodbye!\r\n")
	s.restoreTerminal()
	os.Exit(0)
}

// showHelp displays help information
func (s *ScreenStyleSwitcher) showHelp() {
	fmt.Printf("\r\n=== Screen-Style Commands ===\r\n")
	fmt.Printf("All commands start with Ctrl+A, then:\r\n")
	fmt.Printf("  n       - Create new session\r\n")
	fmt.Printf("  Space   - Switch to next session\r\n")
	fmt.Printf("  p       - Switch to previous session\r\n")
	fmt.Printf("  l       - List all sessions\r\n")
	fmt.Printf("  d       - Detach (close) current session\r\n")
	fmt.Printf("  q       - Quit all sessions\r\n")
	fmt.Printf("  ?       - Show this help\r\n")
	fmt.Printf("  Ctrl+A  - Send literal Ctrl+A to session\r\n")
	fmt.Printf("\r\n")
	fmt.Printf("Type normally to send input to the active session.\r\n")
}

// startOutputForwarding sets up output forwarding from all sessions
func (s *ScreenStyleSwitcher) startOutputForwarding() {
	// Start output forwarding for existing sessions
	for _, session := range s.sessionManager.ListSessions() {
		if session.GetStatus() == SessionStatusActive || session.GetStatus() == SessionStatusIdle {
			go s.forwardSessionOutput(session)
		}
	}
}

// forwardSessionOutput forwards output from a session to stdout
func (s *ScreenStyleSwitcher) forwardSessionOutput(session *Session) {
	if session.StdoutBuf == nil {
		s.logger.Printf("Warning: Session %s has no stdout buffer", session.ID)
		return
	}
	
	s.logger.Printf("Starting output forwarding for session %s", session.ID)
	buffer := make([]byte, 1024)
	
	for {
		select {
		case <-s.appCtx.Done():
			s.logger.Printf("Stopping output forwarding for session %s (context done)", session.ID)
			return
		default:
			// Check if session is still valid
			if session.GetStatus() == SessionStatusClosed || session.GetStatus() == SessionStatusError {
				s.logger.Printf("Stopping output forwarding for session %s (status: %s)", session.ID, session.GetStatus())
				return
			}
			
			// Check if this is still the active session
			activeSession, err := s.sessionManager.GetActiveSession()
			if err != nil || activeSession.ID != session.ID {
				// Not active, sleep and check again
				time.Sleep(100 * time.Millisecond)
				continue
			}
			
			// Read from session output with timeout
			n, err := session.StdoutBuf.Read(buffer)
			if err != nil {
				if err == io.EOF {
					s.logger.Printf("Session %s stdout EOF", session.ID)
					return // Session ended
				}
				// Other error, mark session as error
				s.logger.Printf("Session %s stdout error: %v", session.ID, err)
				session.SetError(err)
				return
			}
			
			if n > 0 {
				// Write to stdout
				os.Stdout.Write(buffer[:n])
				os.Stdout.Sync() // Ensure output is flushed
			}
		}
	}
}

// sendToActiveSession sends input to the currently active session
func (s *ScreenStyleSwitcher) sendToActiveSession(data []byte) {
	activeSession, err := s.sessionManager.GetActiveSession()
	if err != nil {
		// No active session - could show a message or ignore
		return
	}
	
	// Send data to the session
	_, writeErr := activeSession.Write(data)
	if writeErr != nil {
		fmt.Printf("\r\nError sending to session %s: %v\r\n", activeSession.GetDisplayName(), writeErr)
		// Mark session as error state
		activeSession.SetError(writeErr)
	}
}