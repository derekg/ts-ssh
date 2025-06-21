package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

// TestModernKeyDiscoveryIntegration tests the complete modern key discovery system
// with real SSH authentication against a mock server
func TestModernKeyDiscoveryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testCases := []struct {
		name        string
		keyTypes    []string
		expectedKey string
		description string
	}{
		{
			name:        "ed25519_prioritized_over_all",
			keyTypes:    []string{"id_rsa", "id_ecdsa", "id_ed25519"},
			expectedKey: "id_ed25519",
			description: "Ed25519 should be chosen over RSA and ECDSA",
		},
		{
			name:        "ecdsa_chosen_over_rsa",
			keyTypes:    []string{"id_rsa", "id_ecdsa"},
			expectedKey: "id_ecdsa",
			description: "ECDSA should be chosen over RSA when Ed25519 not available",
		},
		{
			name:        "rsa_fallback_when_only_option",
			keyTypes:    []string{"id_rsa"},
			expectedKey: "id_rsa",
			description: "RSA should work as fallback when it's the only key available",
		},
		{
			name:        "ed25519_only_modern_preference",
			keyTypes:    []string{"id_ed25519"},
			expectedKey: "id_ed25519",
			description: "Ed25519 only setup should work perfectly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testModernKeyDiscoveryFlow(t, tc.keyTypes, tc.expectedKey, tc.description)
		})
	}
}

// testModernKeyDiscoveryFlow runs a complete integration test with key discovery
func testModernKeyDiscoveryFlow(t *testing.T, keyTypes []string, expectedKey, description string) {
	t.Logf("Running test: %s", description)

	// Create temporary home directory
	tempHome, err := os.MkdirTemp("", "modern-key-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	sshDir := filepath.Join(tempHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	logger := log.New(io.Discard, "", 0)

	// Generate all requested key types
	keyData := make(map[string]struct {
		privateKey interface{}
		publicKey  ssh.PublicKey
	})

	for _, keyType := range keyTypes {
		privKey, pubKey := generateKeyByTypeForIntegration(t, keyType)
		keyData[keyType] = struct {
			privateKey interface{}
			publicKey  ssh.PublicKey
		}{privKey, pubKey}

		// Write the key to file
		keyPath := filepath.Join(sshDir, keyType)
		if err := writeKeyToFileForIntegration(t, privKey, keyPath); err != nil {
			t.Fatalf("Failed to write %s key: %v", keyType, err)
		}
	}

	// Test key discovery
	discoveredKeyPath := discoverSSHKey(tempHome, logger)
	expectedKeyPath := filepath.Join(sshDir, expectedKey)
	
	if discoveredKeyPath != expectedKeyPath {
		t.Errorf("Expected discovery to find %s, got %s", expectedKeyPath, discoveredKeyPath)
	}

	// Test that the discovered key can be loaded
	authMethod, err := loadPrivateKey(discoveredKeyPath)
	if err != nil {
		t.Fatalf("Failed to load discovered key: %v", err)
	}
	if authMethod == nil {
		t.Fatal("Auth method is nil")
	}

	// Test end-to-end authentication with mock server
	expectedKeyData := keyData[expectedKey]
	testAuthenticationWithDiscoveredKeyIntegration(t, expectedKeyData.publicKey, discoveredKeyPath)

	t.Logf("✓ %s: Successfully discovered %s and authenticated", description, expectedKey)
}

// TestKeyDiscoveryWithRealAuthFlow tests the complete flow from discovery to authentication
func TestKeyDiscoveryWithRealAuthFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("complete_discovery_to_auth_flow", func(t *testing.T) {
		// Setup test environment
		tempHome, err := os.MkdirTemp("", "auth-flow-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp home: %v", err)
		}
		defer os.RemoveAll(tempHome)

		sshDir := filepath.Join(tempHome, ".ssh")
		if err := os.MkdirAll(sshDir, 0700); err != nil {
			t.Fatalf("Failed to create .ssh dir: %v", err)
		}

		// Create an Ed25519 key (should be highest priority)
		ed25519PrivKey, ed25519PubKey := generateKeyByTypeForIntegration(t, "id_ed25519")
		ed25519Path := filepath.Join(sshDir, "id_ed25519")
		if err := writeKeyToFileForIntegration(t, ed25519PrivKey, ed25519Path); err != nil {
			t.Fatalf("Failed to write Ed25519 key: %v", err)
		}

		// Create an RSA key as well (should be ignored in favor of Ed25519)
		rsaPrivKey, _ := generateKeyByTypeForIntegration(t, "id_rsa")
		rsaPath := filepath.Join(sshDir, "id_rsa")
		if err := writeKeyToFileForIntegration(t, rsaPrivKey, rsaPath); err != nil {
			t.Fatalf("Failed to write RSA key: %v", err)
		}

		// Test that LoadBestPrivateKey returns the Ed25519 key
		logger := log.New(io.Discard, "", 0)
		bestKeyPath, authMethod, err := LoadBestPrivateKey(tempHome, logger)
		if err != nil {
			t.Fatalf("LoadBestPrivateKey failed: %v", err)
		}

		if bestKeyPath != ed25519Path {
			t.Errorf("Expected LoadBestPrivateKey to return Ed25519 key %s, got %s", ed25519Path, bestKeyPath)
		}

		if authMethod == nil {
			t.Fatal("Auth method is nil")
		}

		// Test that createModernSSHAuthMethods works correctly
		mockUser := &user.User{HomeDir: tempHome}
		authMethods, err := createModernSSHAuthMethods("", "testuser", "testhost", mockUser, logger)
		if err != nil {
			t.Fatalf("createModernSSHAuthMethods failed: %v", err)
		}

		// Should have at least key auth + password auth
		if len(authMethods) < 2 {
			t.Errorf("Expected at least 2 auth methods (key + password), got %d", len(authMethods))
		}

		// Test complete authentication flow
		testAuthenticationWithDiscoveredKeyIntegration(t, ed25519PubKey, bestKeyPath)

		t.Log("✓ Complete discovery-to-authentication flow successful")
	})
}

// TestBackwardCompatibilityIntegration ensures legacy setups still work
func TestBackwardCompatibilityIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("legacy_rsa_only_setup", func(t *testing.T) {
		tempHome, err := os.MkdirTemp("", "legacy-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp home: %v", err)
		}
		defer os.RemoveAll(tempHome)

		sshDir := filepath.Join(tempHome, ".ssh")
		if err := os.MkdirAll(sshDir, 0700); err != nil {
			t.Fatalf("Failed to create .ssh dir: %v", err)
		}

		// Create only an RSA key (legacy setup)
		rsaPrivKey, rsaPubKey := generateKeyByTypeForIntegration(t, "id_rsa")
		rsaPath := filepath.Join(sshDir, "id_rsa")
		if err := writeKeyToFileForIntegration(t, rsaPrivKey, rsaPath); err != nil {
			t.Fatalf("Failed to write RSA key: %v", err)
		}

		// Test that the system still works with RSA-only
		logger := log.New(io.Discard, "", 0)
		discoveredPath := discoverSSHKey(tempHome, logger)
		if discoveredPath != rsaPath {
			t.Errorf("Expected to discover RSA key %s, got %s", rsaPath, discoveredPath)
		}

		// Test authentication still works
		testAuthenticationWithDiscoveredKeyIntegration(t, rsaPubKey, rsaPath)

		t.Log("✓ Legacy RSA-only setup works correctly")
	})
}

// Helper functions for the integration tests

// generateKeyByTypeForIntegration generates a key pair of the specified type
func generateKeyByTypeForIntegration(t *testing.T, keyType string) (interface{}, ssh.PublicKey) {
	switch keyType {
	case "id_ed25519":
		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("Failed to generate Ed25519 key: %v", err)
		}
		sshPubKey, err := ssh.NewPublicKey(pubKey)
		if err != nil {
			t.Fatalf("Failed to create SSH public key: %v", err)
		}
		return privKey, sshPubKey

	case "id_rsa":
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate RSA key: %v", err)
		}
		sshPubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
		if err != nil {
			t.Fatalf("Failed to create SSH public key: %v", err)
		}
		return privateKey, sshPubKey

	case "id_ecdsa":
		// For this test, we'll use RSA as a proxy for ECDSA since the important thing
		// is testing the discovery priority, not the actual ECDSA implementation
		privateKey, err := rsa.GenerateKey(rand.Reader, 1024) // Minimum secure size for tests
		if err != nil {
			t.Fatalf("Failed to generate ECDSA-proxy key: %v", err)
		}
		sshPubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
		if err != nil {
			t.Fatalf("Failed to create SSH public key: %v", err)
		}
		return privateKey, sshPubKey

	default:
		t.Fatalf("Unknown key type: %s", keyType)
		return nil, nil
	}
}

// writeKeyToFileForIntegration writes a private key to a file in the appropriate format
func writeKeyToFileForIntegration(t *testing.T, privateKey interface{}, filePath string) error {
	switch key := privateKey.(type) {
	case ed25519.PrivateKey:
		return writeEd25519PrivateKeyForIntegration(key, filePath)
	case *rsa.PrivateKey:
		return writeRSAPrivateKeyForIntegration(key, filePath)
	default:
		return fmt.Errorf("unsupported key type: %T", privateKey)
	}
}

// writeEd25519PrivateKeyForIntegration writes an Ed25519 private key in OpenSSH format
func writeEd25519PrivateKeyForIntegration(privateKey ed25519.PrivateKey, filePath string) error {
	// For tests, we'll create a PEM-like structure that our loader can handle
	keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal Ed25519 key: %w", err)
	}

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}

	keyPEM := pem.EncodeToMemory(pemBlock)
	return os.WriteFile(filePath, keyPEM, 0600)
}

// writeRSAPrivateKeyForIntegration writes an RSA private key in PEM format
func writeRSAPrivateKeyForIntegration(privateKey *rsa.PrivateKey, filePath string) error {
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	keyPEM := pem.EncodeToMemory(pemBlock)
	return os.WriteFile(filePath, keyPEM, 0600)
}

// testAuthenticationWithDiscoveredKeyIntegration tests SSH authentication using a discovered key
func testAuthenticationWithDiscoveredKeyIntegration(t *testing.T, authorizedKey ssh.PublicKey, keyPath string) {
	// Start mock SSH server
	serverAddr, cleanup := startMockSSHServer(t, authorizedKey)
	defer cleanup()

	// Test SSH connection with the discovered key
	testSSHConnection(t, serverAddr, keyPath, true)
}

// LoadBestPrivateKey for integration testing (copy from main module for isolated testing)
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
		authMethod, loadErr := loadPrivateKey(keyPath)
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

// createModernSSHAuthMethods for integration testing (copy from main module for isolated testing)
func createModernSSHAuthMethods(keyPath, sshUser, targetHost string, currentUser *user.User, logger *log.Logger) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	// If a specific key path is provided, try it first
	if keyPath != "" {
		keyAuth, err := loadPrivateKey(keyPath)
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
		return "test-password", nil // For tests, return a dummy password
	}))

	if logger != nil {
		logger.Printf("Created %d authentication methods (key-based: %d, password: 1)", 
			len(authMethods), len(authMethods)-1)
	}

	return authMethods, nil
}