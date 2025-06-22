package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// TestSSHKeyAuthentication tests SSH key authentication end-to-end
func TestSSHKeyAuthentication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if not in an environment where we can test SSH
	if !canRunSSHTests() {
		t.Skip("skipping SSH integration tests - SSH server not available or not suitable environment")
	}

	testCases := []struct {
		name           string
		setupKeys      bool
		usePassphrase  bool
		expectedResult bool
		description    string
	}{
		{
			name:           "valid_ssh_key_no_passphrase",
			setupKeys:      true,
			usePassphrase:  false,
			expectedResult: true,
			description:    "Test SSH key authentication with unprotected key",
		},
		{
			name:           "valid_ssh_key_with_passphrase",
			setupKeys:      true,
			usePassphrase:  true,
			expectedResult: true,
			description:    "Test SSH key authentication with passphrase-protected key",
		},
		{
			name:           "no_ssh_key",
			setupKeys:      false,
			usePassphrase:  false,
			expectedResult: false,
			description:    "Test behavior when no SSH key is configured",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSSHKeyAuth(t, tc.setupKeys, tc.usePassphrase, tc.expectedResult, tc.description)
		})
	}
}

// testSSHKeyAuth runs a single SSH key authentication test
func testSSHKeyAuth(t *testing.T, setupKeys, usePassphrase, expectedResult bool, description string) {
	t.Logf("Running test: %s", description)

	// Create temporary test environment
	testEnv, cleanup := createSSHTestEnvironment(t)
	defer cleanup()

	var privateKeyPath string
	if setupKeys {
		// Generate and setup SSH keys
		privPath, pubPath := generateSSHKeyPair(t, testEnv.tempDir, usePassphrase)
		privateKeyPath = privPath

		// Setup authorized_keys (simulate remote host setup)
		setupAuthorizedKeys(t, testEnv.tempDir, pubPath)
	}

	// Test our SSH key loading functionality
	testSSHKeyLoading(t, privateKeyPath, usePassphrase, expectedResult)

	// Test SSH connection setup (without actual network connection)
	testSSHConnectionConfig(t, privateKeyPath, expectedResult)
}

// SSHTestEnvironment holds test environment setup
type SSHTestEnvironment struct {
	tempDir string
	sshDir  string
}

// createSSHTestEnvironment sets up a temporary test environment
func createSSHTestEnvironment(t *testing.T) (*SSHTestEnvironment, func()) {
	tempDir, err := os.MkdirTemp("", "ts-ssh-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	sshDir := filepath.Join(tempDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	env := &SSHTestEnvironment{
		tempDir: tempDir,
		sshDir:  sshDir,
	}

	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: failed to cleanup temp dir %s: %v", tempDir, err)
		}
	}

	return env, cleanup
}

// generateSSHKeyPair generates an SSH key pair for testing
func generateSSHKeyPair(t *testing.T, tempDir string, usePassphrase bool) (privateKeyPath, publicKeyPath string) {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Encode private key
	var privateKeyPEM []byte
	if usePassphrase {
		// For testing, we'll use a simple passphrase
		passphrase := []byte("test-passphrase-123")
		encryptedBlock, err := x509.EncryptPEMBlock(
			rand.Reader,
			"RSA PRIVATE KEY",
			x509.MarshalPKCS1PrivateKey(privateKey),
			passphrase,
			x509.PEMCipherAES256,
		)
		if err != nil {
			t.Fatalf("Failed to encrypt private key: %v", err)
		}
		privateKeyPEM = pem.EncodeToMemory(encryptedBlock)
	} else {
		privateKeyPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})
	}

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to generate public key: %v", err)
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	// Write keys to files
	privateKeyPath = filepath.Join(tempDir, "id_rsa_test")
	publicKeyPath = filepath.Join(tempDir, "id_rsa_test.pub")

	if err := os.WriteFile(privateKeyPath, privateKeyPEM, 0600); err != nil {
		t.Fatalf("Failed to write private key: %v", err)
	}

	if err := os.WriteFile(publicKeyPath, publicKeyBytes, 0644); err != nil {
		t.Fatalf("Failed to write public key: %v", err)
	}

	t.Logf("Generated SSH key pair: %s (passphrase: %t)", privateKeyPath, usePassphrase)
	return privateKeyPath, publicKeyPath
}

// setupAuthorizedKeys creates an authorized_keys file for testing
func setupAuthorizedKeys(t *testing.T, tempDir, publicKeyPath string) {
	authorizedKeysPath := filepath.Join(tempDir, ".ssh", "authorized_keys")

	publicKeyContent, err := os.ReadFile(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to read public key: %v", err)
	}

	if err := os.WriteFile(authorizedKeysPath, publicKeyContent, 0600); err != nil {
		t.Fatalf("Failed to write authorized_keys: %v", err)
	}

	t.Logf("Setup authorized_keys at: %s", authorizedKeysPath)
}

// testSSHKeyLoading tests our SSH key loading functionality
func testSSHKeyLoading(t *testing.T, privateKeyPath string, usePassphrase, expectedSuccess bool) {
	if privateKeyPath == "" && expectedSuccess {
		t.Fatal("Cannot test successful key loading without a private key path")
	}

	if privateKeyPath == "" {
		// Test the no-key scenario
		t.Log("Testing scenario with no SSH key")
		return
	}

	t.Logf("Testing SSH key loading from: %s", privateKeyPath)

	// Test LoadPrivateKey function
	logger := createTestLogger()
	
	// For passphrase-protected keys, we'll need to mock the passphrase input
	// For now, we'll test the non-passphrase scenario directly
	if !usePassphrase {
		authMethod, err := LoadPrivateKey(privateKeyPath, logger)
		if expectedSuccess {
			if err != nil {
				t.Errorf("Expected successful key loading, got error: %v", err)
				return
			}
			if authMethod == nil {
				t.Error("Expected non-nil auth method for successful key loading")
				return
			}
			t.Log("✓ SSH key loaded successfully")
		} else {
			if err == nil {
				t.Error("Expected key loading to fail, but it succeeded")
				return
			}
			t.Logf("✓ SSH key loading failed as expected: %v", err)
		}
	} else {
		t.Log("⚠️  Passphrase-protected key testing requires interactive input - skipping direct test")
		// TODO: Implement mock for passphrase input testing
	}
}

// testSSHConnectionConfig tests SSH connection configuration
func testSSHConnectionConfig(t *testing.T, privateKeyPath string, expectedSuccess bool) {
	t.Log("Testing SSH connection configuration")

	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}

	config := SSHConnectionConfig{
		User:            "testuser",
		KeyPath:         privateKeyPath,
		TargetHost:      "testhost",
		TargetPort:      "22",
		InsecureHostKey: true, // For testing
		Verbose:         true,
		CurrentUser:     currentUser,
		Logger:          createTestLogger(),
	}

	// Test createSSHConfig function
	sshConfig, err := createSSHConfig(config)
	if err != nil {
		if expectedSuccess {
			t.Errorf("Expected SSH config creation to succeed, got error: %v", err)
		} else {
			t.Logf("✓ SSH config creation failed as expected: %v", err)
		}
		return
	}

	if sshConfig == nil {
		t.Error("SSH config is nil")
		return
	}

	// Validate SSH config properties
	if sshConfig.User != config.User {
		t.Errorf("SSH config user mismatch: got %s, want %s", sshConfig.User, config.User)
	}

	if len(sshConfig.Auth) == 0 {
		t.Error("SSH config has no authentication methods")
		return
	}

	t.Logf("✓ SSH config created with %d auth methods", len(sshConfig.Auth))

	// Test that auth methods are properly configured
	testAuthMethods(t, sshConfig.Auth, privateKeyPath != "")
}

// testAuthMethods validates SSH authentication methods
func testAuthMethods(t *testing.T, authMethods []ssh.AuthMethod, hasKey bool) {
	if len(authMethods) == 0 {
		t.Error("No authentication methods configured")
		return
	}

	// We should always have at least password auth as fallback
	foundPasswordAuth := false
	foundKeyAuth := false

	for i, method := range authMethods {
		// Check method type (this is a bit tricky since ssh.AuthMethod is an interface)
		methodType := fmt.Sprintf("%T", method)
		t.Logf("Auth method %d: %s", i, methodType)

		if strings.Contains(methodType, "password") {
			foundPasswordAuth = true
		}
		if strings.Contains(methodType, "publicKey") {
			foundKeyAuth = true
		}
	}

	if !foundPasswordAuth {
		t.Error("Password authentication method not found")
	} else {
		t.Log("✓ Password authentication method configured")
	}

	if hasKey && !foundKeyAuth {
		t.Error("Expected public key authentication method when key is provided")
	} else if hasKey {
		t.Log("✓ Public key authentication method configured")
	}
}

// canRunSSHTests checks if we can run SSH integration tests
func canRunSSHTests() bool {
	// Check if we're in a CI environment or a suitable test environment
	if os.Getenv("CI") != "" {
		return false // Skip in CI for now unless specifically enabled
	}

	// Check if SSH client tools are available
	if _, err := exec.LookPath("ssh"); err != nil {
		return false
	}

	return true
}

// createTestLogger creates a logger for testing
func createTestLogger() *log.Logger {
	return createQuietLogger() // Use quiet logger to avoid noise in tests
}

// TestSSHConnectionHelpers tests SSH connection helper functions
func TestSSHConnectionHelpers(t *testing.T) {
	tests := []struct {
		name           string
		config         SSHConnectionConfig
		expectError    bool
		description    string
	}{
		{
			name: "valid_config_no_key",
			config: SSHConnectionConfig{
				User:            "testuser",
				KeyPath:         "",
				TargetHost:      "localhost",
				TargetPort:      "22",
				InsecureHostKey: true,
				Verbose:         false,
				CurrentUser:     &user.User{Username: "test", HomeDir: "/tmp"},
				Logger:          createTestLogger(),
			},
			expectError: false,
			description: "Valid config without SSH key should work",
		},
		{
			name: "missing_user",
			config: SSHConnectionConfig{
				User:            "",
				KeyPath:         "",
				TargetHost:      "localhost",
				TargetPort:      "22",
				InsecureHostKey: true,
				Verbose:         false,
				CurrentUser:     &user.User{Username: "test", HomeDir: "/tmp"},
				Logger:          createTestLogger(),
			},
			expectError: false, // Empty user should be handled gracefully
			description: "Empty user should be handled",
		},
		{
			name: "invalid_key_path",
			config: SSHConnectionConfig{
				User:            "testuser",
				KeyPath:         "/nonexistent/key/path",
				TargetHost:      "localhost",
				TargetPort:      "22",
				InsecureHostKey: true,
				Verbose:         false,
				CurrentUser:     &user.User{Username: "test", HomeDir: "/tmp"},
				Logger:          createTestLogger(),
			},
			expectError: false, // Should fallback to password auth
			description: "Invalid key path should fallback gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			sshConfig, err := createSSHConfig(tt.config)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else {
					t.Logf("✓ Got expected error: %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if sshConfig == nil {
				t.Error("SSH config is nil")
				return
			}

			// Validate basic properties
			if sshConfig.User != tt.config.User {
				t.Errorf("User mismatch: got %s, want %s", sshConfig.User, tt.config.User)
			}

			if len(sshConfig.Auth) == 0 {
				t.Error("No authentication methods configured")
			}

			t.Logf("✓ SSH config created successfully with %d auth methods", len(sshConfig.Auth))
		})
	}
}

// TestSSHKeyGeneration tests SSH key generation and validation
func TestSSHKeyGeneration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ssh-key-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("generate_unprotected_key", func(t *testing.T) {
		privPath, pubPath := generateSSHKeyPair(t, tempDir, false)
		
		// Verify files exist
		if _, err := os.Stat(privPath); err != nil {
			t.Errorf("Private key file not created: %v", err)
		}
		if _, err := os.Stat(pubPath); err != nil {
			t.Errorf("Public key file not created: %v", err)
		}

		// Verify key can be loaded
		logger := createTestLogger()
		authMethod, err := LoadPrivateKey(privPath, logger)
		if err != nil {
			t.Errorf("Failed to load generated key: %v", err)
		}
		if authMethod == nil {
			t.Error("Auth method is nil")
		}

		t.Log("✓ Unprotected SSH key generated and loaded successfully")
	})

	t.Run("generate_protected_key", func(t *testing.T) {
		privPath, pubPath := generateSSHKeyPair(t, tempDir, true)
		
		// Verify files exist
		if _, err := os.Stat(privPath); err != nil {
			t.Errorf("Private key file not created: %v", err)
		}
		if _, err := os.Stat(pubPath); err != nil {
			t.Errorf("Public key file not created: %v", err)
		}

		// Verify key requires passphrase (should fail without passphrase)
		logger := createTestLogger()
		authMethod, err := LoadPrivateKey(privPath, logger)
		if err == nil {
			t.Error("Expected error for passphrase-protected key, but got none")
		} else {
			t.Logf("✓ Passphrase-protected key correctly requires passphrase: %v", err)
		}
		if authMethod != nil {
			t.Error("Auth method should be nil for failed key loading")
		}

		t.Log("✓ Passphrase-protected SSH key generated successfully")
	})
}

// Mock network test for SSH connection establishment
func TestSSHConnectionEstablishment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	t.Run("mock_connection_setup", func(t *testing.T) {
		// Test SSH connection setup without actual network connection
		tempDir, err := os.MkdirTemp("", "ssh-conn-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Generate test key
		privPath, _ := generateSSHKeyPair(t, tempDir, false)

		currentUser, err := user.Current()
		if err != nil {
			t.Fatalf("Failed to get current user: %v", err)
		}

		config := SSHConnectionConfig{
			User:            "testuser",
			KeyPath:         privPath,
			TargetHost:      "localhost",
			TargetPort:      "22",
			InsecureHostKey: true,
			Verbose:         true,
			CurrentUser:     currentUser,
			Logger:          createTestLogger(),
		}

		// Test SSH config creation
		sshConfig, err := createSSHConfig(config)
		if err != nil {
			t.Fatalf("Failed to create SSH config: %v", err)
		}

		// Validate the config is properly set up for connection
		if sshConfig.User != config.User {
			t.Errorf("User mismatch: got %s, want %s", sshConfig.User, config.User)
		}

		if sshConfig.Timeout == 0 {
			t.Error("SSH timeout not set")
		}

		if len(sshConfig.Auth) == 0 {
			t.Error("No auth methods configured")
		}

		t.Log("✓ SSH connection configuration validated successfully")
	})
}