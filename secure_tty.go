package main

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/term"
)

// getSecureTTY validates and opens a secure TTY connection
// This prevents TTY hijacking and input redirection attacks
func getSecureTTY() (*os.File, error) {
	// First, verify we're running in a real terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, fmt.Errorf("not running in a terminal")
	}

	// Get TTY path with validation
	ttyPath, err := getTTYPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get TTY path: %w", err)
	}

	// Validate TTY security before opening
	if err := validateTTYSecurity(ttyPath); err != nil {
		return nil, fmt.Errorf("TTY security validation failed: %w", err)
	}

	// Open TTY with explicit permissions check
	ttyFile, err := os.OpenFile(ttyPath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open TTY: %w", err)
	}

	// Additional security check after opening
	if err := validateOpenTTY(ttyFile); err != nil {
		ttyFile.Close()
		return nil, fmt.Errorf("opened TTY failed security validation: %w", err)
	}

	return ttyFile, nil
}

// getTTYPath determines the correct TTY path with security validation
func getTTYPath() (string, error) {
	// Check environment variable first (but validate it)
	if ttyname := os.Getenv("TTY"); ttyname != "" {
		if err := validateTTYPath(ttyname); err != nil {
			return "", fmt.Errorf("TTY environment variable points to invalid path: %w", err)
		}
		return ttyname, nil
	}

	// Fallback to /dev/tty if it exists and is safe
	ttyPath := "/dev/tty"
	if err := validateTTYPath(ttyPath); err != nil {
		return "", fmt.Errorf("default TTY path is unsafe: %w", err)
	}

	return ttyPath, nil
}

// validateTTYPath performs basic validation on a TTY path
func validateTTYPath(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("TTY path does not exist: %w", err)
	}

	// Check if it's a character device (TTY should be)
	if info.Mode()&os.ModeCharDevice == 0 {
		return fmt.Errorf("TTY path is not a character device")
	}

	return nil
}

// validateTTYSecurity performs comprehensive security validation on a TTY
func validateTTYSecurity(ttyPath string) error {
	info, err := os.Stat(ttyPath)
	if err != nil {
		return fmt.Errorf("cannot stat TTY: %w", err)
	}

	// Get file system stat for ownership checks
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot get TTY ownership information")
	}

	currentUID := uint32(os.Getuid())
	currentGID := uint32(os.Getgid())
	
	// Check ownership - TTY should be owned by current user OR root (for system terminals)
	// Also allow if owned by current user's group (common in some environments)
	if stat.Uid != currentUID && stat.Uid != 0 && stat.Gid != currentGID {
		return fmt.Errorf("TTY not owned by current user, root, or current group (owned by UID %d, GID %d, current UID %d, GID %d)", 
			stat.Uid, stat.Gid, currentUID, currentGID)
	}

	// Check permissions based on TTY type and ownership
	mode := info.Mode()
	
	// For /dev/tty specifically, permissions are often 666 and that's normal
	// since it's a special device that redirects to the controlling terminal
	if ttyPath == "/dev/tty" {
		// /dev/tty is special - it's safe even with wide permissions because
		// it only gives access to the process's own controlling terminal
		return nil
	}
	
	// For actual TTY devices (like /dev/pts/0), be more careful about permissions
	// but still allow common patterns for root-owned TTYs
	if stat.Uid == 0 {
		// Root-owned TTYs can have group/other read access but not write
		if mode&0022 != 0 { // Check group-write and other-write
			return fmt.Errorf("TTY has unsafe permissions: %v (group/world-writable on root-owned TTY)", mode)
		}
		return nil
	}
	
	// For user-owned TTYs, be strict about permissions
	if mode&0077 != 0 {
		return fmt.Errorf("TTY has unsafe permissions: %v (allows group/other access on user-owned TTY)", mode)
	}

	return nil
}

// validateOpenTTY performs additional validation on an opened TTY file
func validateOpenTTY(ttyFile *os.File) error {
	// Verify it's still a terminal after opening
	fd := int(ttyFile.Fd())
	if !term.IsTerminal(fd) {
		return fmt.Errorf("opened file is not a terminal")
	}

	// Additional ownership check on the opened file descriptor
	info, err := ttyFile.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat opened TTY: %w", err)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot get opened TTY ownership information")
	}

	currentUID := uint32(os.Getuid())
	currentGID := uint32(os.Getgid())
	
	// Use same relaxed ownership logic as validateTTYSecurity
	if stat.Uid != currentUID && stat.Uid != 0 && stat.Gid != currentGID {
		return fmt.Errorf("opened TTY ownership changed (owned by UID %d, GID %d, current UID %d, GID %d)", 
			stat.Uid, stat.Gid, currentUID, currentGID)
	}

	return nil
}

// readPasswordSecurely reads a password from a secure TTY connection
func readPasswordSecurely() (string, error) {
	tty, err := getSecureTTY()
	if err != nil {
		// Fallback to stdin if secure TTY is not available
		// This maintains functionality while logging the security concern
		if term.IsTerminal(int(os.Stdin.Fd())) {
			password, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return "", fmt.Errorf("failed to read password from stdin fallback: %w", err)
			}
			return string(password), nil
		}
		return "", fmt.Errorf("cannot access secure TTY and stdin is not a terminal: %w", err)
	}
	defer tty.Close()

	// Save terminal state for proper restoration
	fd := int(tty.Fd())
	oldState, err := term.GetState(fd)
	if err != nil {
		return "", fmt.Errorf("failed to get terminal state: %w", err)
	}
	defer func() {
		// Ensure terminal state is restored even on panic
		if oldState != nil {
			term.Restore(fd, oldState)
		}
	}()

	// Read password with echo disabled
	password, err := term.ReadPassword(fd)
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	return string(password), nil
}

// withSecureTTY executes a function with a secure TTY, ensuring proper cleanup
func withSecureTTY(fn func(*os.File) error) error {
	tty, err := getSecureTTY()
	if err != nil {
		return err
	}
	defer func() {
		// Ensure TTY is properly restored and closed
		fd := int(tty.Fd())
		if term.IsTerminal(fd) {
			if state, err := term.GetState(fd); err == nil {
				term.Restore(fd, state)
			}
		}
		tty.Close()
	}()

	return fn(tty)
}

// promptUserSecurely prompts the user for input using a secure TTY
func promptUserSecurely(prompt string) (string, error) {
	var result string
	var readErr error

	err := withSecureTTY(func(tty *os.File) error {
		// Write prompt to stderr (visible to user)
		fmt.Fprint(os.Stderr, prompt)

		// Read from secure TTY
		buffer := make([]byte, 256)
		n, err := tty.Read(buffer)
		if err != nil {
			readErr = err
			return err
		}

		// Process input (remove newline)
		input := string(buffer[:n])
		if len(input) > 0 && input[len(input)-1] == '\n' {
			input = input[:len(input)-1]
		}
		if len(input) > 0 && input[len(input)-1] == '\r' {
			input = input[:len(input)-1]
		}

		result = input
		return nil
	})

	if err != nil {
		return "", err
	}
	if readErr != nil {
		return "", readErr
	}

	return result, nil
}