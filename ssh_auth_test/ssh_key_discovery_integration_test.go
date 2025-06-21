package main

import (
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"testing"
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

// TestSSHKeyDiscoveryIntegration tests the new SSH key discovery functionality
func TestSSHKeyDiscoveryIntegration(t *testing.T) {
	// Create a temporary directory to simulate user's home
	tempHome, err := os.MkdirTemp("", "ssh-key-discovery-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp home dir: %v", err)
	}
	defer os.RemoveAll(tempHome)

	sshDir := filepath.Join(tempHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	// Create a quiet logger for testing
	logger := log.New(io.Discard, "", 0)

	t.Run("no_keys_returns_empty", func(t *testing.T) {
		// Test with no keys present
		result := discoverSSHKey(tempHome, logger)
		if result != "" {
			t.Errorf("Expected empty result when no keys present, got: %s", result)
		}
	})

	t.Run("defaults_to_ed25519_when_no_keys_found", func(t *testing.T) {
		// Test default path suggestion
		defaultPath := getDefaultSSHKeyPath(&user.User{HomeDir: tempHome}, logger)
		expectedDefault := filepath.Join(tempHome, ".ssh", "id_ed25519")
		if defaultPath != expectedDefault {
			t.Errorf("Expected default path %s, got %s", expectedDefault, defaultPath)
		}
	})

	t.Run("prioritizes_ed25519_over_rsa", func(t *testing.T) {
		// Create both RSA and Ed25519 keys
		rsaPath := filepath.Join(sshDir, "id_rsa")
		ed25519Path := filepath.Join(sshDir, "id_ed25519")

		// Create RSA key file
		if err := os.WriteFile(rsaPath, []byte("fake-rsa-key"), 0600); err != nil {
			t.Fatalf("Failed to create RSA key file: %v", err)
		}

		// Create Ed25519 key file  
		if err := os.WriteFile(ed25519Path, []byte("fake-ed25519-key"), 0600); err != nil {
			t.Fatalf("Failed to create Ed25519 key file: %v", err)
		}

		// Discovery should return Ed25519, not RSA
		result := discoverSSHKey(tempHome, logger)
		if result != ed25519Path {
			t.Errorf("Expected Ed25519 key %s, got %s", ed25519Path, result)
		}

		// Default path should also return the discovered Ed25519 key
		defaultPath := getDefaultSSHKeyPath(&user.User{HomeDir: tempHome}, logger)
		if defaultPath != ed25519Path {
			t.Errorf("Expected default to return discovered Ed25519 key %s, got %s", ed25519Path, defaultPath)
		}

		// Clean up for next test
		os.Remove(rsaPath)
		os.Remove(ed25519Path)
	})

	t.Run("skips_keys_with_bad_permissions", func(t *testing.T) {
		// Create a key with overly permissive permissions
		badKeyPath := filepath.Join(sshDir, "id_ed25519")
		if err := os.WriteFile(badKeyPath, []byte("fake-key"), 0644); err != nil { // World-readable
			t.Fatalf("Failed to create key with bad permissions: %v", err)
		}

		// Should skip the key due to bad permissions
		result := discoverSSHKey(tempHome, logger)
		if result != "" {
			t.Errorf("Expected no key found due to bad permissions, got: %s", result)
		}

		// Fix permissions and try again
		if err := os.Chmod(badKeyPath, 0600); err != nil {
			t.Fatalf("Failed to fix key permissions: %v", err)
		}

		result = discoverSSHKey(tempHome, logger)
		if result != badKeyPath {
			t.Errorf("Expected key found after fixing permissions: %s, got: %s", badKeyPath, result)
		}

		// Clean up
		os.Remove(badKeyPath)
	})

	t.Run("key_type_preference_order", func(t *testing.T) {
		// Create all supported key types
		keyFiles := map[string]string{
			"id_rsa":      "fake-rsa-key",
			"id_ecdsa":    "fake-ecdsa-key", 
			"id_ed25519":  "fake-ed25519-key",
		}

		// Create all key files
		for keyName, content := range keyFiles {
			keyPath := filepath.Join(sshDir, keyName)
			if err := os.WriteFile(keyPath, []byte(content), 0600); err != nil {
				t.Fatalf("Failed to create %s: %v", keyName, err)
			}
		}

		// Should return Ed25519 (highest priority)
		result := discoverSSHKey(tempHome, logger)
		expectedPath := filepath.Join(sshDir, "id_ed25519")
		if result != expectedPath {
			t.Errorf("Expected Ed25519 key %s, got %s", expectedPath, result)
		}

		// Remove Ed25519, should fall back to ECDSA
		os.Remove(expectedPath)
		result = discoverSSHKey(tempHome, logger)
		expectedPath = filepath.Join(sshDir, "id_ecdsa")
		if result != expectedPath {
			t.Errorf("Expected ECDSA key %s after Ed25519 removed, got %s", expectedPath, result)
		}

		// Remove ECDSA, should fall back to RSA
		os.Remove(expectedPath)
		result = discoverSSHKey(tempHome, logger)
		expectedPath = filepath.Join(sshDir, "id_rsa")
		if result != expectedPath {
			t.Errorf("Expected RSA key %s after ECDSA removed, got %s", expectedPath, result)
		}

		// Clean up
		for keyName := range keyFiles {
			os.Remove(filepath.Join(sshDir, keyName))
		}
	})
}

// TestModernKeyTypes verifies our key type preferences are correctly ordered
func TestModernKeyTypes(t *testing.T) {
	expected := []string{
		"id_ed25519",  // Most secure and modern
		"id_ecdsa",    // Good performance and security
		"id_rsa",      // Legacy but still supported
	}

	if len(modernKeyTypes) != len(expected) {
		t.Errorf("Expected %d key types, got %d", len(expected), len(modernKeyTypes))
	}

	for i, keyType := range expected {
		if i >= len(modernKeyTypes) || modernKeyTypes[i] != keyType {
			t.Errorf("Expected key type %d to be %s, got %s", i, keyType, modernKeyTypes[i])
		}
	}
}

// TestLegacyCompatibility ensures we still support existing RSA setups
func TestLegacyCompatibility(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "ssh-legacy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	sshDir := filepath.Join(tempHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	// Create only an RSA key (legacy setup)
	rsaPath := filepath.Join(sshDir, "id_rsa")
	if err := os.WriteFile(rsaPath, []byte("fake-rsa-key"), 0600); err != nil {
		t.Fatalf("Failed to create RSA key: %v", err)
	}

	logger := log.New(io.Discard, "", 0)
	
	// Should still find and use the RSA key
	result := discoverSSHKey(tempHome, logger)
	if result != rsaPath {
		t.Errorf("Expected to find RSA key %s in legacy setup, got %s", rsaPath, result)
	}

	// Default path should return the discovered RSA key
	defaultPath := getDefaultSSHKeyPath(&user.User{HomeDir: tempHome}, logger)
	if defaultPath != rsaPath {
		t.Errorf("Expected default to return discovered RSA key %s, got %s", rsaPath, defaultPath)
	}
}

// TestSecurityFeatures verifies security-related functionality
func TestSecurityFeatures(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "ssh-security-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	sshDir := filepath.Join(tempHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	logger := log.New(io.Discard, "", 0)

	t.Run("ignores_world_readable_keys", func(t *testing.T) {
		// Create a key that's readable by everyone (security risk)
		unsafeKeyPath := filepath.Join(sshDir, "id_ed25519")
		if err := os.WriteFile(unsafeKeyPath, []byte("fake-key"), 0644); err != nil {
			t.Fatalf("Failed to create unsafe key: %v", err)
		}

		// Should not find the unsafe key
		result := discoverSSHKey(tempHome, logger)
		if result != "" {
			t.Errorf("Expected to ignore world-readable key, but found: %s", result)
		}

		os.Remove(unsafeKeyPath)
	})

	t.Run("ignores_group_readable_keys", func(t *testing.T) {
		// Create a key that's readable by group (also a security risk)
		unsafeKeyPath := filepath.Join(sshDir, "id_ed25519")
		if err := os.WriteFile(unsafeKeyPath, []byte("fake-key"), 0640); err != nil {
			t.Fatalf("Failed to create group-readable key: %v", err)
		}

		// Should not find the group-readable key
		result := discoverSSHKey(tempHome, logger)
		if result != "" {
			t.Errorf("Expected to ignore group-readable key, but found: %s", result)
		}

		os.Remove(unsafeKeyPath)
	})

	t.Run("accepts_user_only_readable_keys", func(t *testing.T) {
		// Create a key that's only readable by the user (secure)
		safeKeyPath := filepath.Join(sshDir, "id_ed25519")
		if err := os.WriteFile(safeKeyPath, []byte("fake-key"), 0600); err != nil {
			t.Fatalf("Failed to create safe key: %v", err)
		}

		// Should find the safe key
		result := discoverSSHKey(tempHome, logger)
		if result != safeKeyPath {
			t.Errorf("Expected to find safe key %s, got: %s", safeKeyPath, result)
		}

		os.Remove(safeKeyPath)
	})
}