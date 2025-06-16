package main

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SessionStatus represents the current state of a session
type SessionStatus int

const (
	SessionStatusCreated SessionStatus = iota
	SessionStatusConnecting
	SessionStatusActive
	SessionStatusIdle
	SessionStatusError
	SessionStatusClosed
)

func (s SessionStatus) String() string {
	switch s {
	case SessionStatusCreated:
		return "created"
	case SessionStatusConnecting:
		return "connecting"
	case SessionStatusActive:
		return "active"
	case SessionStatusIdle:
		return "idle"
	case SessionStatusError:
		return "error"
	case SessionStatusClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// Session represents a single SSH connection to a host
type Session struct {
	ID          string
	HostTarget  string
	SSHUser     string
	Status      SessionStatus
	CreatedAt   time.Time
	LastActive  time.Time
	ErrorMsg    string
	
	// SSH connection details
	SSHClient   *ssh.Client
	SSHSession  *ssh.Session
	
	// Terminal handling
	StdinPipe   io.WriteCloser
	StdoutBuf   io.Reader
	StderrBuf   io.Reader
	
	// Context for cancellation
	ctx         context.Context
	cancel      context.CancelFunc
	
	// Synchronization
	mu          sync.RWMutex
}

// NewSession creates a new session instance
func NewSession(id, hostTarget, sshUser string) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Session{
		ID:         id,
		HostTarget: hostTarget,
		SSHUser:    sshUser,
		Status:     SessionStatusCreated,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// GetStatus safely returns the current session status
func (s *Session) GetStatus() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// SetStatus safely updates the session status
func (s *Session) SetStatus(status SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
	s.LastActive = time.Now()
}

// SetError sets the session to error state with a message
func (s *Session) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = SessionStatusError
	s.ErrorMsg = err.Error()
	s.LastActive = time.Now()
}

// GetError safely returns the current error message
func (s *Session) GetError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ErrorMsg
}

// UpdateActivity updates the last active timestamp
func (s *Session) UpdateActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActive = time.Now()
}

// Close terminates the session and cleans up resources
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.Status == SessionStatusClosed {
		return nil // Already closed
	}
	
	// Cancel context to signal shutdown
	if s.cancel != nil {
		s.cancel()
	}
	
	// Close SSH session if active
	if s.SSHSession != nil {
		s.SSHSession.Close()
	}
	
	// Close SSH client connection
	if s.SSHClient != nil {
		s.SSHClient.Close()
	}
	
	// Close stdin pipe if open
	if s.StdinPipe != nil {
		s.StdinPipe.Close()
	}
	
	s.Status = SessionStatusClosed
	return nil
}

// IsActive returns true if the session is currently active
func (s *Session) IsActive() bool {
	status := s.GetStatus()
	return status == SessionStatusActive || status == SessionStatusConnecting
}

// GetDisplayName returns a formatted name for display in UI
func (s *Session) GetDisplayName() string {
	return fmt.Sprintf("%s@%s", s.SSHUser, s.HostTarget)
}

// GetStatusDisplay returns a formatted status string for UI display
func (s *Session) GetStatusDisplay() string {
	status := s.GetStatus()
	switch status {
	case SessionStatusActive:
		return "[green]●[white] Active"
	case SessionStatusIdle:
		return "[yellow]●[white] Idle"
	case SessionStatusError:
		return "[red]●[white] Error"
	case SessionStatusConnecting:
		return "[blue]●[white] Connecting"
	case SessionStatusClosed:
		return "[gray]●[white] Closed"
	default:
		return "[gray]●[white] Unknown"
	}
}

// Write sends data to the session's stdin (for sending commands)
func (s *Session) Write(data []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.StdinPipe == nil {
		return 0, fmt.Errorf("session %s: stdin pipe not available", s.ID)
	}
	
	s.UpdateActivity()
	return s.StdinPipe.Write(data)
}

// SetTerminalSize updates the terminal size for the session
func (s *Session) SetTerminalSize(width, height int) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.SSHSession == nil {
		return fmt.Errorf("session %s: SSH session not available", s.ID)
	}
	
	return s.SSHSession.WindowChange(height, width)
}

// GetAge returns how long ago the session was created
func (s *Session) GetAge() time.Duration {
	return time.Since(s.CreatedAt)
}

// GetIdleTime returns how long the session has been idle
func (s *Session) GetIdleTime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.LastActive)
}