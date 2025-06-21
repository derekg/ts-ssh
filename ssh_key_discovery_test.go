package main

import (
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

// TestSSHKeyDiscovery tests the new SSH key discovery functionality
func TestSSHKeyDiscovery(t *testing.T) {
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

	t.Run("no_keys_returns_ed25519_default", func(t *testing.T) {
		// Test with no keys present
		result := discoverSSHKey(tempHome, logger)
		if result != "" {
			t.Errorf("Expected empty result when no keys present, got: %s", result)
		}

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

// TestKeyDiscoveryDocumentation verifies our recommendations are helpful
func TestKeyDiscoveryDocumentation(t *testing.T) {
	t.Run("recommends_ed25519_for_new_keys", func(t *testing.T) {
		tempHome, err := os.MkdirTemp("", "ssh-key-doc-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp home: %v", err)
		}
		defer os.RemoveAll(tempHome)

		// When no keys exist, should recommend Ed25519
		defaultPath := getDefaultSSHKeyPath(&user.User{HomeDir: tempHome}, nil)
		expectedPath := filepath.Join(tempHome, ".ssh", "id_ed25519")
		
		if defaultPath != expectedPath {
			t.Errorf("Expected recommendation for Ed25519 path %s, got %s", expectedPath, defaultPath)
		}
	})
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
}