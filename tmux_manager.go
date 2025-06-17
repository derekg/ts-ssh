package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// TmuxManager handles creating and managing tmux sessions for SSH connections
type TmuxManager struct {
	logger       *log.Logger
	sessionName  string
	sshUser      string
	sshKeyPath   string
	insecureHostKey bool
}

// NewTmuxManager creates a new tmux session manager
func NewTmuxManager(logger *log.Logger, sshUser, sshKeyPath string, insecureHostKey bool) *TmuxManager {
	// Create a unique session name based on timestamp
	sessionName := fmt.Sprintf("ts-ssh-%d", time.Now().Unix())
	
	return &TmuxManager{
		logger:          logger,
		sessionName:     sessionName,
		sshUser:         sshUser,
		sshKeyPath:      sshKeyPath,
		insecureHostKey: insecureHostKey,
	}
}

// StartMultiSession creates a tmux session with multiple SSH connections
func (tm *TmuxManager) StartMultiSession(hosts []string) error {
	if len(hosts) == 0 {
		return fmt.Errorf("no hosts provided")
	}
	
	tm.logger.Printf("Creating tmux session '%s' with %d hosts", tm.sessionName, len(hosts))
	
	// Check if tmux is available
	if !tm.isTmuxAvailable() {
		return fmt.Errorf("tmux is not installed or not available in PATH")
	}
	
	// Kill any existing session with the same name
	tm.killExistingSession()
	
	// Create new tmux session with the first host
	firstHost := hosts[0]
	err := tm.createInitialSession(firstHost)
	if err != nil {
		return fmt.Errorf("failed to create initial tmux session: %w", err)
	}
	
	// Add additional hosts as new windows
	for i, host := range hosts[1:] {
		windowName := fmt.Sprintf("ssh-%d", i+2)
		err := tm.addWindow(windowName, host)
		if err != nil {
			tm.logger.Printf("Warning: failed to add window for %s: %v", host, err)
			// Continue with other hosts even if one fails
		}
	}
	
	// Set up tmux configuration for better experience
	tm.configureTmux()
	
	// Attach to the session
	return tm.attachToSession()
}

// isTmuxAvailable checks if tmux is installed and available
func (tm *TmuxManager) isTmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// killExistingSession kills any existing tmux session with our name
func (tm *TmuxManager) killExistingSession() {
	cmd := exec.Command("tmux", "kill-session", "-t", tm.sessionName)
	// Ignore errors - session might not exist
	cmd.Run()
}

// createInitialSession creates the first tmux session with SSH to the first host
func (tm *TmuxManager) createInitialSession(host string) error {
	sshCmd := tm.buildSSHCommand(host)
	
	// Create new tmux session with SSH command
	cmd := exec.Command("tmux", "new-session", "-d", "-s", tm.sessionName, "-n", "ssh-1")
	cmd.Env = os.Environ()
	
	tm.logger.Printf("Creating tmux session with command: %s", strings.Join(cmd.Args, " "))
	
	err := cmd.Run()
	if err != nil {
		return err
	}
	
	// Send SSH command to the session
	return tm.sendKeysToWindow("ssh-1", sshCmd)
}

// createInitialSessionDryRun creates the first tmux session without sending SSH commands (for testing)
func (tm *TmuxManager) createInitialSessionDryRun(host string) error {
	// Create new tmux session (detached, no commands sent)
	cmd := exec.Command("tmux", "new-session", "-d", "-s", tm.sessionName, "-n", "ssh-1")
	cmd.Env = os.Environ()
	
	tm.logger.Printf("Creating tmux session (dry run) with command: %s", strings.Join(cmd.Args, " "))
	
	return cmd.Run()
}

// addWindow adds a new window to the tmux session with SSH to the specified host
func (tm *TmuxManager) addWindow(windowName, host string) error {
	sshCmd := tm.buildSSHCommand(host)
	
	// Create new window
	cmd := exec.Command("tmux", "new-window", "-t", tm.sessionName, "-n", windowName)
	err := cmd.Run()
	if err != nil {
		return err
	}
	
	// Send SSH command to the new window
	return tm.sendKeysToWindow(windowName, sshCmd)
}

// sendKeysToWindow sends a command to a specific tmux window
func (tm *TmuxManager) sendKeysToWindow(windowName, command string) error {
	target := fmt.Sprintf("%s:%s", tm.sessionName, windowName)
	cmd := exec.Command("tmux", "send-keys", "-t", target, command, "Enter")
	
	tm.logger.Printf("Sending to window %s: %s", windowName, command)
	
	return cmd.Run()
}

// buildSSHCommand constructs the SSH command for connecting to a host
func (tm *TmuxManager) buildSSHCommand(host string) string {
	var sshArgs []string
	
	// Add SSH key if specified
	if tm.sshKeyPath != "" {
		sshArgs = append(sshArgs, "-i", tm.sshKeyPath)
	}
	
	// Add insecure host key option if specified
	if tm.insecureHostKey {
		sshArgs = append(sshArgs, "-o", "StrictHostKeyChecking=no")
		sshArgs = append(sshArgs, "-o", "UserKnownHostsFile=/dev/null")
	}
	
	// Add connection target
	sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", tm.sshUser, host))
	
	// Use our ts-ssh binary instead of regular ssh to get Tailscale connectivity
	// We'll construct a command that uses our binary in non-TUI mode
	cmdParts := []string{os.Args[0]} // Our binary path
	
	// Add our flags
	if tm.sshKeyPath != "" {
		cmdParts = append(cmdParts, "-i", tm.sshKeyPath)
	}
	if tm.insecureHostKey {
		cmdParts = append(cmdParts, "-insecure")
	}
	
	// Add the target
	cmdParts = append(cmdParts, fmt.Sprintf("%s@%s", tm.sshUser, host))
	
	return strings.Join(cmdParts, " ")
}

// configureTmux sets up tmux configuration for a better multi-session experience
func (tm *TmuxManager) configureTmux() {
	configs := [][]string{
		// Set status bar to show window list
		{"set-option", "-t", tm.sessionName, "status", "on"},
		// Enable mouse support
		{"set-option", "-t", tm.sessionName, "mouse", "on"},
		// Set window titles to show hostname
		{"set-option", "-t", tm.sessionName, "automatic-rename", "on"},
		// Set base index to 1 for easier switching
		{"set-option", "-t", tm.sessionName, "base-index", "1"},
		// Enable activity monitoring
		{"set-option", "-t", tm.sessionName, "monitor-activity", "on"},
		// Set escape time for better responsiveness
		{"set-option", "-t", tm.sessionName, "escape-time", "10"},
	}
	
	for _, config := range configs {
		cmd := exec.Command("tmux", config...)
		err := cmd.Run()
		if err != nil {
			tm.logger.Printf("Warning: failed to set tmux option %v: %v", config, err)
		}
	}
	
	// Display helpful message in each window
	tm.displayWelcomeMessage()
}

// displayWelcomeMessage shows a helpful message about tmux controls
func (tm *TmuxManager) displayWelcomeMessage() {
	message := "# ts-ssh Multi-Session Mode\\n" +
		"# Tmux Controls:\\n" +
		"#   Ctrl+B n     - Next window\\n" +
		"#   Ctrl+B p     - Previous window\\n" +
		"#   Ctrl+B 1-9   - Switch to window number\\n" +
		"#   Ctrl+B c     - Create new window\\n" +
		"#   Ctrl+B x     - Close current window\\n" +
		"#   Ctrl+B d     - Detach from session\\n" +
		"#   Ctrl+B ?     - Show all key bindings\\n" +
		"# Connecting..."
	
	// Display message in the first window
	target := fmt.Sprintf("%s:ssh-1", tm.sessionName)
	cmd := exec.Command("tmux", "display-message", "-t", target, "-d", "3000", message)
	cmd.Run() // Ignore errors
}

// attachToSession attaches to the tmux session (this will block until detached)
func (tm *TmuxManager) attachToSession() error {
	tm.logger.Printf("Attaching to tmux session '%s'", tm.sessionName)
	
	// Attach to session - this will transfer control to tmux
	cmd := exec.Command("tmux", "attach-session", "-t", tm.sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	
	// When we get here, user has detached from tmux or session ended
	tm.logger.Printf("Detached from tmux session '%s'", tm.sessionName)
	
	return err
}

// CleanupSession kills the tmux session
func (tm *TmuxManager) CleanupSession() error {
	tm.logger.Printf("Cleaning up tmux session '%s'", tm.sessionName)
	cmd := exec.Command("tmux", "kill-session", "-t", tm.sessionName)
	return cmd.Run()
}

// AddHost adds a new host to the existing tmux session
func (tm *TmuxManager) AddHost(host string) error {
	if !tm.isSessionActive() {
		return fmt.Errorf("tmux session '%s' is not active", tm.sessionName)
	}
	
	// Find next available window number
	windowNum := tm.getNextWindowNumber()
	windowName := fmt.Sprintf("ssh-%d", windowNum)
	
	return tm.addWindow(windowName, host)
}

// isSessionActive checks if the tmux session is still active
func (tm *TmuxManager) isSessionActive() bool {
	cmd := exec.Command("tmux", "has-session", "-t", tm.sessionName)
	return cmd.Run() == nil
}

// getNextWindowNumber finds the next available window number
func (tm *TmuxManager) getNextWindowNumber() int {
	cmd := exec.Command("tmux", "list-windows", "-t", tm.sessionName, "-F", "#{window_index}")
	output, err := cmd.Output()
	if err != nil {
		return 1
	}
	
	// Parse existing window numbers and find the next available
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	maxNum := 0
	for _, line := range lines {
		var num int
		n, _ := fmt.Sscanf(line, "%d", &num)
		if n == 1 && num > maxNum {
			maxNum = num
		}
	}
	
	return maxNum + 1
}