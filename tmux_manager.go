package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/derekg/ts-ssh/internal/security"
)

// TmuxManager handles creating and managing tmux sessions for SSH connections
type TmuxManager struct {
	logger          *log.Logger
	sessionName     string
	sshUser         string
	sshKeyPath      string
	insecureHostKey bool
	tempConfigFiles []string // Track temporary config files for cleanup
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
	sshCmd, configFile, err := tm.buildSecureSSHCommand(host)
	if err != nil {
		return err
	}
	
	// Store config file for cleanup
	tm.tempConfigFiles = append(tm.tempConfigFiles, configFile)
	
	// Create new tmux session with SSH command
	cmd := exec.Command("tmux", "new-session", "-d", "-s", tm.sessionName, "-n", "ssh-1")
	cmd.Env = os.Environ()
	
	tm.logger.Printf("Creating tmux session with secure command (credentials protected)")
	
	err = cmd.Run()
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
	sshCmd, configFile, err := tm.buildSecureSSHCommand(host)
	if err != nil {
		return err
	}
	
	// Store config file for cleanup
	tm.tempConfigFiles = append(tm.tempConfigFiles, configFile)
	
	// Create new window
	cmd := exec.Command("tmux", "new-window", "-t", tm.sessionName, "-n", windowName)
	err = cmd.Run()
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

// buildSecureSSHCommand constructs a secure SSH command using temporary config files
// to avoid exposing credentials in process lists
func (tm *TmuxManager) buildSecureSSHCommand(host string) (string, string, error) {
	// SECURITY: Validate hostname to prevent command injection
	if err := security.ValidateHostname(host); err != nil {
		return "", "", fmt.Errorf("hostname validation failed: %w", err)
	}
	
	// Create temporary SSH config file to avoid credential exposure
	tempConfigFile, err := tm.createTemporarySSHConfig(host)
	if err != nil {
		return "", "", fmt.Errorf("failed to create temporary SSH config: %w", err)
	}
	
	// Build command using config file instead of command line args
	cmdParts := []string{os.Args[0]} // Our binary path
	cmdParts = append(cmdParts, "-F", tempConfigFile) // Use SSH config file
	
	// SECURITY: Sanitize hostname for shell execution
	sanitizedHost := security.SanitizeShellArg(host)
	cmdParts = append(cmdParts, sanitizedHost) // Safely escaped hostname
	
	return strings.Join(cmdParts, " "), tempConfigFile, nil
}

// createTemporarySSHConfig creates a temporary SSH config file with secure permissions
func (tm *TmuxManager) createTemporarySSHConfig(host string) (string, error) {
	// SECURITY: Validate hostname again for config file creation
	if err := security.ValidateHostname(host); err != nil {
		return "", fmt.Errorf("hostname validation failed: %w", err)
	}
	
	// Generate unique filename for temporary config using secure random suffix
	// Use a sanitized version of hostname for the filename
	safeHostname := strings.ReplaceAll(host, ":", "_")
	safeHostname = strings.ReplaceAll(safeHostname, "/", "_")
	tempFileName := fmt.Sprintf("/tmp/ts-ssh-config-%s-%s.conf", safeHostname, security.GenerateRandomSuffix())
	
	// Create temporary file with secure permissions atomically
	tempFile, err := security.CreateSecureFile(tempFileName, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create secure temporary SSH config: %w", err)
	}
	
	// Generate SSH config content
	config := fmt.Sprintf(`# Temporary SSH config for ts-ssh tmux session
Host %s
    User %s
`, host, tm.sshUser)
	
	if tm.sshKeyPath != "" {
		config += fmt.Sprintf("    IdentityFile %s\n", tm.sshKeyPath)
	}
	
	if tm.insecureHostKey {
		config += "    StrictHostKeyChecking no\n"
		config += "    UserKnownHostsFile /dev/null\n"
	} else {
		config += "    StrictHostKeyChecking yes\n"
	}
	
	config += "    LogLevel QUIET\n"
	config += "    BatchMode no\n" // Allow password prompts
	
	// Write config to file
	if _, err := tempFile.WriteString(config); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", err
	}
	
	tempFile.Close()
	return tempFile.Name(), nil
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
	
	// Check if we're in a terminal environment
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		tm.logger.Printf("Not running in a terminal, cannot attach to tmux session")
		fmt.Printf("Tmux session '%s' created successfully!\n", tm.sessionName)
		fmt.Printf("To connect manually, run: tmux attach-session -t %s\n", tm.sessionName)
		fmt.Printf("Or list sessions with: tmux list-sessions\n")
		return nil
	}
	
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

// CleanupSession kills the tmux session and cleans up temporary files
func (tm *TmuxManager) CleanupSession() error {
	tm.logger.Printf("Cleaning up tmux session '%s'", tm.sessionName)
	
	// Kill tmux session
	cmd := exec.Command("tmux", "kill-session", "-t", tm.sessionName)
	err := cmd.Run()
	
	// Clean up temporary SSH config files
	tm.cleanupTempConfigFiles()
	
	return err
}

// cleanupTempConfigFiles removes all temporary SSH config files
func (tm *TmuxManager) cleanupTempConfigFiles() {
	for _, configFile := range tm.tempConfigFiles {
		if err := os.Remove(configFile); err != nil {
			tm.logger.Printf("Warning: failed to remove temporary config file %s: %v", configFile, err)
		} else {
			tm.logger.Printf("Cleaned up temporary config file: %s", configFile)
		}
	}
	tm.tempConfigFiles = nil
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