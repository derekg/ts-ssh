package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// Modern SSH key types in order of preference (most secure first)
var modernKeyTypes = []string{
	"id_ed25519",     // Ed25519 - fastest, most secure, smallest
	"id_ecdsa",       // ECDSA - good performance, secure
	"id_rsa",         // RSA - legacy, still supported but deprecated
}

// discoverSSHKey finds the best available SSH private key in the user's .ssh directory
// It prioritizes modern key types (Ed25519, ECDSA) over legacy RSA keys
// Returns the path to the best available key, or empty string if none found
func discoverSSHKey(homeDir string, logger *log.Logger) string {
	if homeDir == "" {
		if logger != nil {
			logger.Printf("Cannot discover SSH keys: home directory unknown")
		}
		return ""
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	
	// Check if .ssh directory exists
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		if logger != nil {
			logger.Printf("SSH directory %s does not exist", sshDir)
		}
		return ""
	}

	// Try each key type in order of preference
	for _, keyType := range modernKeyTypes {
		keyPath := filepath.Join(sshDir, keyType)
		
		// Check if the private key file exists and is readable
		if info, err := os.Stat(keyPath); err == nil && !info.IsDir() {
			// Verify file has reasonable permissions (not world-readable)
			if info.Mode().Perm() & 0044 == 0 { // Check that group and others don't have read permission
				if logger != nil {
					logger.Printf("Found SSH key: %s (type: %s)", keyPath, keyType)
				}
				return keyPath
			} else {
				if logger != nil {
					logger.Printf("Warning: SSH key %s has overly permissive permissions (%o), skipping for security", keyPath, info.Mode().Perm())
				}
			}
		}
	}

	if logger != nil {
		logger.Printf("No suitable SSH private keys found in %s", sshDir)
		logger.Printf("Searched for: %v", modernKeyTypes)
		logger.Printf("Tip: Generate a modern Ed25519 key with: ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519")
	}
	
	return ""
}

// getDefaultSSHKeyPath returns the path to the best available SSH key
// or falls back to the Ed25519 default path if no keys are found
func getDefaultSSHKeyPath(currentUser *user.User, logger *log.Logger) string {
	if currentUser == nil || currentUser.HomeDir == "" {
		if logger != nil {
			logger.Printf("Cannot determine SSH key path: user or home directory unknown")
		}
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
	if logger != nil {
		logger.Printf("No SSH keys found, defaulting to %s", defaultPath)
		logger.Printf("Consider generating an Ed25519 key: ssh-keygen -t ed25519 -f %s", defaultPath)
	}
	
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
	for _, keyType := range modernKeyTypes {
		keyPath = filepath.Join(sshDir, keyType)
		
		// Check if key exists
		if _, err := os.Stat(keyPath); err != nil {
			continue // Try next key type
		}

		// Try to load the key
		authMethod, loadErr := LoadPrivateKey(keyPath, logger)
		if loadErr == nil {
			if logger != nil {
				logger.Printf("Successfully loaded SSH key: %s (type: %s)", keyPath, keyType)
			}
			return keyPath, authMethod, nil
		} else {
			if logger != nil {
				logger.Printf("Failed to load %s key at %s: %v", keyType, keyPath, loadErr)
			}
		}
	}

	return "", nil, fmt.Errorf("no usable SSH private keys found in %s (searched: %v)", sshDir, modernKeyTypes)
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
			if logger != nil {
				logger.Printf("Using specified key: %s", keyPath)
			}
		} else {
			if logger != nil {
				logger.Printf("Specified key failed to load: %v", err)
				logger.Printf("Falling back to automatic key discovery...")
			}
		}
	}

	// If no specific key provided or if it failed, try automatic discovery
	if len(authMethods) == 0 && currentUser != nil {
		discoveredKeyPath, keyAuth, err := LoadBestPrivateKey(currentUser.HomeDir, logger)
		if err == nil && keyAuth != nil {
			authMethods = append(authMethods, keyAuth)
			if logger != nil {
				logger.Printf("Using discovered key: %s", discoveredKeyPath)
			}
		} else {
			if logger != nil {
				logger.Printf("Key discovery failed: %v", err)
			}
		}
	}

	// Add password authentication as fallback
	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		fmt.Print(T("enter_password", sshUser, targetHost))
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		return string(bytePassword), nil
	}))

	if logger != nil {
		logger.Printf("Created %d authentication methods (key-based: %d, password: 1)", 
			len(authMethods), len(authMethods)-1)
	}

	return authMethods, nil
}