package main

import (
	"fmt"
	"log"
	"net"
	"os/user"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
	"tailscale.com/tsnet"
)

// SessionConnector handles establishing SSH connections for sessions
type SessionConnector struct {
	tsnetServer     *tsnet.Server
	logger          *log.Logger
	sshKeyPath      string
	insecureHostKey bool
	currentUser     *user.User
}

// NewSessionConnector creates a new session connector
func NewSessionConnector(tsnetServer *tsnet.Server, logger *log.Logger, sshKeyPath string, insecureHostKey bool, currentUser *user.User) *SessionConnector {
	return &SessionConnector{
		tsnetServer:     tsnetServer,
		logger:          logger,
		sshKeyPath:      sshKeyPath,
		insecureHostKey: insecureHostKey,
		currentUser:     currentUser,
	}
}

// ConnectSession establishes an SSH connection for the given session
func (sc *SessionConnector) ConnectSession(session *Session) error {
	session.SetStatus(SessionStatusConnecting)
	
	// Create SSH target address
	sshTargetAddr := net.JoinHostPort(session.HostTarget, DefaultSshPort)
	
	// Set up SSH authentication
	authMethods := []ssh.AuthMethod{}
	
	// Try public key authentication
	if sc.sshKeyPath != "" {
		keyAuth, err := LoadPrivateKey(sc.sshKeyPath, sc.logger)
		if err == nil {
			authMethods = append(authMethods, keyAuth)
			sc.logger.Printf("Session %s: Using public key authentication: %s", session.ID, sc.sshKeyPath)
		} else {
			sc.logger.Printf("Session %s: Could not load private key %s: %v", session.ID, sc.sshKeyPath, err)
		}
	}
	
	// Add password authentication as fallback
	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		// For multi-session mode, we can't prompt interactively
		// This would need to be handled differently in the TUI
		return "", fmt.Errorf("password authentication not supported in multi-session mode")
	}))
	
	// Set up host key callback
	var hostKeyCallback ssh.HostKeyCallback
	if sc.insecureHostKey {
		sc.logger.Printf("Session %s: WARNING! Host key verification is disabled!", session.ID)
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		var err error
		hostKeyCallback, err = CreateKnownHostsCallback(sc.currentUser, sc.logger)
		if err != nil {
			session.SetError(fmt.Errorf("could not set up host key verification: %w", err))
			return err
		}
	}
	
	// Create SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User:            session.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         0, // No timeout for connection
	}
	
	// Dial the SSH connection via tsnet
	sc.logger.Printf("Session %s: Dialing %s via tsnet...", session.ID, sshTargetAddr)
	conn, err := sc.tsnetServer.Dial(session.ctx, "tcp", sshTargetAddr)
	if err != nil {
		session.SetError(fmt.Errorf("failed to dial %s via tsnet: %w", sshTargetAddr, err))
		return err
	}
	
	// Establish SSH connection
	sc.logger.Printf("Session %s: Establishing SSH connection...", session.ID)
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, sshTargetAddr, sshConfig)
	if err != nil {
		conn.Close()
		session.SetError(fmt.Errorf("failed to establish SSH connection: %w", err))
		return err
	}
	
	// Create SSH client
	sshClient := ssh.NewClient(sshConn, chans, reqs)
	
	// Create SSH session
	sshSession, err := sshClient.NewSession()
	if err != nil {
		sshClient.Close()
		session.SetError(fmt.Errorf("failed to create SSH session: %w", err))
		return err
	}
	
	// Get stdin pipe for sending commands
	stdinPipe, err := sshSession.StdinPipe()
	if err != nil {
		sshSession.Close()
		sshClient.Close()
		session.SetError(fmt.Errorf("failed to create stdin pipe: %w", err))
		return err
	}
	
	// Get stdout and stderr pipes for reading output
	stdoutPipe, err := sshSession.StdoutPipe()
	if err != nil {
		stdinPipe.Close()
		sshSession.Close()
		sshClient.Close()
		session.SetError(fmt.Errorf("failed to create stdout pipe: %w", err))
		return err
	}
	
	stderrPipe, err := sshSession.StderrPipe()
	if err != nil {
		stdinPipe.Close()
		sshSession.Close()
		sshClient.Close()
		session.SetError(fmt.Errorf("failed to create stderr pipe: %w", err))
		return err
	}
	
	// Request PTY for interactive session
	termType := "xterm-256color"
	termWidth := 80
	termHeight := 24
	
	// Try to get actual terminal size if available
	if fd := int(0); term.IsTerminal(fd) { // Check stdin
		if width, height, err := term.GetSize(fd); err == nil {
			termWidth = width
			termHeight = height
		}
	}
	
	err = sshSession.RequestPty(termType, termHeight, termWidth, ssh.TerminalModes{})
	if err != nil {
		stdinPipe.Close()
		sshSession.Close()
		sshClient.Close()
		session.SetError(fmt.Errorf("failed to request PTY: %w", err))
		return err
	}
	
	// Start shell
	err = sshSession.Shell()
	if err != nil {
		stdinPipe.Close()
		sshSession.Close()
		sshClient.Close()
		session.SetError(fmt.Errorf("failed to start shell: %w", err))
		return err
	}
	
	// Update session with connection details
	session.mu.Lock()
	session.SSHClient = sshClient
	session.SSHSession = sshSession
	session.StdinPipe = stdinPipe
	session.StdoutBuf = stdoutPipe
	session.StderrBuf = stderrPipe
	session.Status = SessionStatusActive
	session.mu.Unlock()
	
	sc.logger.Printf("Session %s: Successfully connected to %s@%s", session.ID, session.SSHUser, session.HostTarget)
	
	// Start monitoring session in background
	go sc.monitorSession(session, sshSession)
	
	return nil
}

// monitorSession monitors a session for completion and errors
func (sc *SessionConnector) monitorSession(session *Session, sshSession *ssh.Session) {
	// Wait for session to complete
	err := sshSession.Wait()
	
	// Update session status based on completion
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			sc.logger.Printf("Session %s: Remote command exited with status %d", session.ID, exitErr.ExitStatus())
		} else {
			sc.logger.Printf("Session %s: SSH session ended with error: %v", session.ID, err)
		}
		session.SetError(err)
	} else {
		sc.logger.Printf("Session %s: SSH session ended normally", session.ID)
		session.SetStatus(SessionStatusClosed)
	}
	
	// The session cleanup will be handled by the SessionManager
}

// DisconnectSession cleanly disconnects a session
func (sc *SessionConnector) DisconnectSession(session *Session) error {
	return session.Close()
}

// SendCommand sends a command to a session
func (sc *SessionConnector) SendCommand(session *Session, command string) error {
	if session.GetStatus() != SessionStatusActive && session.GetStatus() != SessionStatusIdle {
		return fmt.Errorf("session %s is not active", session.ID)
	}
	
	_, err := session.Write([]byte(command + "\n"))
	if err != nil {
		session.SetError(err)
		return err
	}
	
	return nil
}

// ResizeSession updates the terminal size for a session
func (sc *SessionConnector) ResizeSession(session *Session, width, height int) error {
	if session.GetStatus() != SessionStatusActive && session.GetStatus() != SessionStatusIdle {
		return fmt.Errorf("session %s is not active", session.ID)
	}
	
	return session.SetTerminalSize(width, height)
}