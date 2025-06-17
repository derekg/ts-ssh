package main

import (
	"sync"

	"golang.org/x/term"
)

// TerminalStateManager provides thread-safe terminal state management
type TerminalStateManager struct {
	mu       sync.RWMutex
	oldState *term.State
	fd       int
	isRaw    bool
}

// NewTerminalStateManager creates a new terminal state manager
func NewTerminalStateManager() *TerminalStateManager {
	return &TerminalStateManager{
		fd: -1,
	}
}

// MakeRaw sets the terminal to raw mode safely
func (tsm *TerminalStateManager) MakeRaw(fd int) error {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	
	if tsm.isRaw {
		return nil // Already in raw mode
	}
	
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	
	tsm.oldState = oldState
	tsm.fd = fd
	tsm.isRaw = true
	return nil
}

// Restore restores the terminal to its original state safely
func (tsm *TerminalStateManager) Restore() error {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	
	if !tsm.isRaw || tsm.oldState == nil {
		return nil // Nothing to restore
	}
	
	err := term.Restore(tsm.fd, tsm.oldState)
	if err == nil {
		tsm.isRaw = false
		tsm.oldState = nil
		tsm.fd = -1
	}
	return err
}

// IsRaw returns whether the terminal is currently in raw mode
func (tsm *TerminalStateManager) IsRaw() bool {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()
	return tsm.isRaw
}

// GetFD returns the current file descriptor
func (tsm *TerminalStateManager) GetFD() int {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()
	return tsm.fd
}

// Global terminal state manager instance
var globalTerminalState = NewTerminalStateManager()

// GetGlobalTerminalState returns the global terminal state manager
func GetGlobalTerminalState() *TerminalStateManager {
	return globalTerminalState
}