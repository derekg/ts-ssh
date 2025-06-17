package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"tailscale.com/tsnet"
)

// SessionManager manages multiple SSH sessions
type SessionManager struct {
	sessions   map[string]*Session
	activeID   string
	mu         sync.RWMutex
	logger     *log.Logger
	
	// Tailscale connection (shared across sessions)
	tsnetServer *tsnet.Server
}

// NewSessionManager creates a new session manager
func NewSessionManager(tsnetServer *tsnet.Server, logger *log.Logger) *SessionManager {
	return &SessionManager{
		sessions:    make(map[string]*Session),
		tsnetServer: tsnetServer,
		logger:      logger,
	}
}

// CreateSession creates a new session but doesn't connect yet
func (sm *SessionManager) CreateSession(hostTarget, sshUser string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Generate unique session ID
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	
	// Check if we already have a session for this host/user combination
	for _, session := range sm.sessions {
		if session.HostTarget == hostTarget && session.SSHUser == sshUser {
			if session.IsActive() {
				return nil, fmt.Errorf("active session already exists for %s@%s", sshUser, hostTarget)
			}
		}
	}
	
	// Create new session
	session := NewSession(sessionID, hostTarget, sshUser)
	sm.sessions[sessionID] = session
	
	sm.logger.Printf("Created session %s for %s@%s", sessionID, sshUser, hostTarget)
	
	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	
	return session, nil
}

// GetActiveSession returns the currently active session
func (sm *SessionManager) GetActiveSession() (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if sm.activeID == "" {
		return nil, fmt.Errorf("no active session")
	}
	
	session, exists := sm.sessions[sm.activeID]
	if !exists {
		return nil, fmt.Errorf("active session %s not found", sm.activeID)
	}
	
	return session, nil
}

// SetActiveSession sets the currently active session
func (sm *SessionManager) SetActiveSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if sessionID == "" {
		sm.activeID = ""
		return nil
	}
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}
	
	// Update previous active session to idle
	if sm.activeID != "" && sm.activeID != sessionID {
		if prevSession, exists := sm.sessions[sm.activeID]; exists {
			if prevSession.GetStatus() == SessionStatusActive {
				prevSession.SetStatus(SessionStatusIdle)
			}
		}
	}
	
	// Set new active session
	sm.activeID = sessionID
	session.SetStatus(SessionStatusActive)
	session.UpdateActivity()
	
	sm.logger.Printf("Set active session to %s (%s)", sessionID, session.GetDisplayName())
	
	return nil
}

// ListSessions returns all sessions
func (sm *SessionManager) ListSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}
	
	return sessions
}

// GetSessionCount returns the number of sessions
func (sm *SessionManager) GetSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// GetActiveSessionCount returns the number of active sessions
func (sm *SessionManager) GetActiveSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	count := 0
	for _, session := range sm.sessions {
		if session.IsActive() {
			count++
		}
	}
	return count
}

// GetConnectedSessionCount returns the number of truly connected (not just connecting) sessions
func (sm *SessionManager) GetConnectedSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	count := 0
	for _, session := range sm.sessions {
		status := session.GetStatus()
		if status == SessionStatusActive || status == SessionStatusIdle {
			count++
		}
	}
	return count
}

// CloseSession closes and removes a session
func (sm *SessionManager) CloseSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}
	
	// Close the session
	if err := session.Close(); err != nil {
		sm.logger.Printf("Error closing session %s: %v", sessionID, err)
	}
	
	// Remove from sessions map
	delete(sm.sessions, sessionID)
	
	// If this was the active session, clear active ID
	if sm.activeID == sessionID {
		sm.activeID = ""
		
		// Try to set another session as active if available
		for id, sess := range sm.sessions {
			if sess.IsActive() {
				sm.activeID = id
				break
			}
		}
	}
	
	sm.logger.Printf("Closed and removed session %s", sessionID)
	
	return nil
}

// CloseAllSessions closes all sessions
func (sm *SessionManager) CloseAllSessions() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	for sessionID, session := range sm.sessions {
		if err := session.Close(); err != nil {
			sm.logger.Printf("Error closing session %s: %v", sessionID, err)
		}
	}
	
	// Clear all sessions
	sm.sessions = make(map[string]*Session)
	sm.activeID = ""
	
	sm.logger.Printf("Closed all sessions")
	
	return nil
}

// SwitchToNextSession switches to the next session in order
func (sm *SessionManager) SwitchToNextSession() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if len(sm.sessions) == 0 {
		return fmt.Errorf("no sessions available")
	}
	
	// Get ordered list of session IDs
	sessionIDs := make([]string, 0, len(sm.sessions))
	for id := range sm.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	
	if len(sessionIDs) == 1 {
		return nil // Only one session, nothing to switch to
	}
	
	// Find current active session index
	currentIndex := -1
	for i, id := range sessionIDs {
		if id == sm.activeID {
			currentIndex = i
			break
		}
	}
	
	// Calculate next index (wrap around)
	nextIndex := (currentIndex + 1) % len(sessionIDs)
	nextSessionID := sessionIDs[nextIndex]
	
	return sm.setActiveSessionUnsafe(nextSessionID)
}

// SwitchToPrevSession switches to the previous session in order
func (sm *SessionManager) SwitchToPrevSession() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if len(sm.sessions) == 0 {
		return fmt.Errorf("no sessions available")
	}
	
	// Get ordered list of session IDs
	sessionIDs := make([]string, 0, len(sm.sessions))
	for id := range sm.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	
	if len(sessionIDs) == 1 {
		return nil // Only one session, nothing to switch to
	}
	
	// Find current active session index
	currentIndex := -1
	for i, id := range sessionIDs {
		if id == sm.activeID {
			currentIndex = i
			break
		}
	}
	
	// Calculate previous index (wrap around)
	prevIndex := (currentIndex - 1 + len(sessionIDs)) % len(sessionIDs)
	prevSessionID := sessionIDs[prevIndex]
	
	return sm.setActiveSessionUnsafe(prevSessionID)
}

// setActiveSessionUnsafe sets active session without locking (internal use)
func (sm *SessionManager) setActiveSessionUnsafe(sessionID string) error {
	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}
	
	// Update previous active session to idle
	if sm.activeID != "" && sm.activeID != sessionID {
		if prevSession, exists := sm.sessions[sm.activeID]; exists {
			if prevSession.GetStatus() == SessionStatusActive {
				prevSession.SetStatus(SessionStatusIdle)
			}
		}
	}
	
	// Set new active session
	sm.activeID = sessionID
	session.SetStatus(SessionStatusActive)
	session.UpdateActivity()
	
	return nil
}

// GetSessionStats returns statistics about sessions
func (sm *SessionManager) GetSessionStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	totalSessions := len(sm.sessions)
	activeSessions := 0
	idleSessions := 0
	errorSessions := 0
	
	for _, session := range sm.sessions {
		switch session.GetStatus() {
		case SessionStatusActive, SessionStatusConnecting:
			activeSessions++
		case SessionStatusIdle:
			idleSessions++
		case SessionStatusError:
			errorSessions++
		}
	}
	
	stats["total"] = totalSessions
	stats["active"] = activeSessions
	stats["idle"] = idleSessions
	stats["error"] = errorSessions
	stats["activeSessionID"] = sm.activeID
	
	return stats
}