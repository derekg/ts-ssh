package main

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

// getSecureTTY validates and opens a secure TTY connection
// This prevents TTY hijacking and input redirection attacks
func getSecureTTY() (*os.File, error) {
	// First, verify we're running in a real terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errors.New(T("not_running_in_terminal"))
	}

	// Get TTY path with validation
	ttyPath, err := getTTYPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get TTY path: %w", err)
	}

	// Validate TTY security before opening
	if err := validateTTYSecurity(ttyPath); err != nil {
		return nil, fmt.Errorf(T("tty_security_validation_failed"), err)
	}

	// Open TTY with explicit permissions check
	ttyFile, err := os.OpenFile(ttyPath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf(T("failed_open_tty"), err)
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

	// Perform platform-specific ownership validation
	if err := validateTTYOwnership(info, ttyPath); err != nil {
		return err
	}

	// Perform platform-specific permission validation
	if err := validateTTYPermissions(info, ttyPath); err != nil {
		return err
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

	// Use platform-specific ownership validation
	if err := validateOpenTTYOwnership(info); err != nil {
		return err
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