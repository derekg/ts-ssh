package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// Note: SSH key types are now defined in constants.go as ModernKeyTypes

// logSafe safely logs a message only if logger is not nil
func logSafe(logger *log.Logger, message string, args ...interface{}) {
	if logger != nil {
		if len(args) > 0 {
			logger.Printf(message, args...)
		} else {
			logger.Print(message)
		}
	}
}

// discoverSSHKey finds the best available SSH private key in the user's .ssh directory
// It prioritizes modern key types (Ed25519, ECDSA) over legacy RSA keys
// Returns the path to the best available key, or empty string if none found
func discoverSSHKey(homeDir string, logger *log.Logger) string {
	if homeDir == "" {
		logSafe(logger, "Cannot discover SSH keys: home directory unknown")
		return ""
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	
	// Check if .ssh directory exists
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		logSafe(logger, "SSH directory %s does not exist", sshDir)
		return ""
	}

	// Try each key type in order of preference
	for _, keyType := range ModernKeyTypes {
		keyPath := filepath.Join(sshDir, keyType)
		
		// Check if the private key file exists and is readable
		if info, err := os.Stat(keyPath); err == nil && !info.IsDir() {
			// Verify file has secure permissions (not readable by group or others)
			// SSH private keys should be readable only by owner (mode 0600 or stricter)
			const groupReadPerm = os.FileMode(0040)  // Group read permission
			const otherReadPerm = os.FileMode(0004)  // Other (world) read permission
			const insecurePerms = groupReadPerm | otherReadPerm
			
			if info.Mode().Perm() & insecurePerms == 0 { // Ensure neither group nor others can read
				logSafe(logger, "Found SSH key: %s (type: %s)", keyPath, keyType)
				return keyPath
			} else {
				logSafe(logger, "Warning: SSH key %s has overly permissive permissions (%o), skipping for security", keyPath, info.Mode().Perm())
			}
		}
	}

	logSafe(logger, "No suitable SSH private keys found in %s", sshDir)
	logSafe(logger, "Searched for: %v", ModernKeyTypes)
	logSafe(logger, "Tip: Generate a modern Ed25519 key with: ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519")
	
	return ""
}

// getDefaultSSHKeyPath returns the path to the best available SSH key
// or falls back to the Ed25519 default path if no keys are found
func getDefaultSSHKeyPath(currentUser *user.User, logger *log.Logger) string {
	if currentUser == nil || currentUser.HomeDir == "" {
		logSafe(logger, "Cannot determine SSH key path: user or home directory unknown")
		return ""
	}

	// Try to discover an existing key
	foundKey := discoverSSHKey(currentUser.HomeDir, logger)
	if foundKey != "" {
		return foundKey
	}

	// If no keys found, default to Ed25519 path (most modern)
	// This encourages users to generate modern keys
	defaultPath := filepath.Join(currentUser.HomeDir, ".ssh", "id_ed25519")
	logSafe(logger, "No SSH keys found, defaulting to %s", defaultPath)
	logSafe(logger, "Consider generating an Ed25519 key: ssh-keygen -t ed25519 -f %s", defaultPath)
	
	return defaultPath
}

// LoadBestPrivateKey attempts to load SSH keys in order of preference
// This function tries multiple key types automatically rather than relying on a single path
func LoadBestPrivateKey(homeDir string, logger *log.Logger) (keyPath string, authMethod ssh.AuthMethod, err error) {
	if homeDir == "" {
		return "", nil, fmt.Errorf("home directory is required for key discovery")
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	
	// Try each key type in order of preference
	for _, keyType := range ModernKeyTypes {
		keyPath = filepath.Join(sshDir, keyType)
		
		// Check if key exists
		if _, err := os.Stat(keyPath); err != nil {
			continue // Try next key type
		}

		// Try to load the key
		authMethod, loadErr := LoadPrivateKey(keyPath, logger)
		if loadErr == nil {
			logSafe(logger, "Successfully loaded SSH key: %s (type: %s)", keyPath, keyType)
			return keyPath, authMethod, nil
		} else {
			logSafe(logger, "Failed to load %s key at %s: %v", keyType, keyPath, loadErr)
		}
	}

	return "", nil, fmt.Errorf("no usable SSH private keys found in %s (searched: %v)", sshDir, ModernKeyTypes)
}

// createModernSSHAuthMethods creates authentication methods with automatic key discovery
// This is an enhanced version of createSSHAuthMethods that prioritizes modern key types
func createModernSSHAuthMethods(keyPath, sshUser, targetHost string, currentUser *user.User, logger *log.Logger) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	// If a specific key path is provided, try it first
	if keyPath != "" {
		keyAuth, err := LoadPrivateKey(keyPath, logger)
		if err == nil {
			authMethods = append(authMethods, keyAuth)
			logSafe(logger, "Using specified key: %s", keyPath)
		} else {
			logSafe(logger, "Specified key failed to load: %v", err)
			logSafe(logger, "Falling back to automatic key discovery...")
		}
	}

	// If no specific key provided or if it failed, try automatic discovery
	if len(authMethods) == 0 && currentUser != nil {
		discoveredKeyPath, keyAuth, err := LoadBestPrivateKey(currentUser.HomeDir, logger)
		if err == nil && keyAuth != nil {
			authMethods = append(authMethods, keyAuth)
			logSafe(logger, "Using discovered key: %s", discoveredKeyPath)
		} else {
			logSafe(logger, "Key discovery failed: %v", err)
		}
	}

	// Add password authentication as fallback using secure TTY
	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		fmt.Print(T("enter_password", sshUser, targetHost))
		password, err := readPasswordSecurely()
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("failed to read password securely: %w", err)
		}
		return password, nil
	}))

	logSafe(logger, "Created %d authentication methods (key-based: %d, password: 1)", 
		len(authMethods), len(authMethods)-1)

	return authMethods, nil
}