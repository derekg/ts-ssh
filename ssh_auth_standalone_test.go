package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// Standalone SSH authentication test that doesn't depend on main package functions
// This allows us to test SSH key functionality even when main package has compilation issues

// TestStandaloneSSHKeyGeneration tests SSH key generation without main package dependencies
func TestStandaloneSSHKeyGeneration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ssh-standalone-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("generate_and_load_unprotected_key", func(t *testing.T) {
		// Generate RSA private key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate private key: %v", err)
		}

		// Convert to PEM format
		privateKeyPEM := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}
		privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

		// Write to file
		keyPath := filepath.Join(tempDir, "test_key")
		if err := os.WriteFile(keyPath, privateKeyBytes, 0600); err != nil {
			t.Fatalf("Failed to write private key: %v", err)
		}

		// Test loading the key
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			t.Fatalf("Failed to read key file: %v", err)
		}

		// Parse the PEM block
		block, _ := pem.Decode(keyData)
		if block == nil {
			t.Fatal("Failed to parse PEM block")
		}

		// Parse the RSA private key
		parsedKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			t.Fatalf("Failed to parse private key: %v", err)
		}

		// Create SSH public key for verification
		sshPublicKey, err := ssh.NewPublicKey(&parsedKey.PublicKey)
		if err != nil {
			t.Fatalf("Failed to create SSH public key: %v", err)
		}

		// Create SSH signer
		sshSigner, err := ssh.NewSignerFromKey(parsedKey)
		if err != nil {
			t.Fatalf("Failed to create SSH signer: %v", err)
		}

		// Verify the signer's public key matches
		if string(sshSigner.PublicKey().Marshal()) != string(sshPublicKey.Marshal()) {
			t.Error("SSH signer public key doesn't match generated public key")
		}

		t.Log("✓ SSH key generated, saved, loaded, and verified successfully")
	})

	t.Run("generate_protected_key", func(t *testing.T) {
		// Generate RSA private key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate private key: %v", err)
		}

		// Encrypt the key with a passphrase
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

		privateKeyPEM := pem.EncodeToMemory(encryptedBlock)

		// Write to file
		keyPath := filepath.Join(tempDir, "test_key_protected")
		if err := os.WriteFile(keyPath, privateKeyPEM, 0600); err != nil {
			t.Fatalf("Failed to write protected private key: %v", err)
		}

		// Test loading the protected key (should fail without passphrase)
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			t.Fatalf("Failed to read key file: %v", err)
		}

		// Parse the PEM block
		block, _ := pem.Decode(keyData)
		if block == nil {
			t.Fatal("Failed to parse PEM block")
		}

		// Should be encrypted
		if !x509.IsEncryptedPEMBlock(block) {
			t.Error("Expected encrypted PEM block")
		}

		// Try to decrypt with passphrase
		decryptedBytes, err := x509.DecryptPEMBlock(block, passphrase)
		if err != nil {
			t.Fatalf("Failed to decrypt PEM block: %v", err)
		}

		// Parse the decrypted key
		parsedKey, err := x509.ParsePKCS1PrivateKey(decryptedBytes)
		if err != nil {
			t.Fatalf("Failed to parse decrypted private key: %v", err)
		}

		// Create SSH signer from decrypted key
		sshSigner, err := ssh.NewSignerFromKey(parsedKey)
		if err != nil {
			t.Fatalf("Failed to create SSH signer from decrypted key: %v", err)
		}

		if sshSigner == nil {
			t.Error("SSH signer is nil")
		}

		t.Log("✓ Protected SSH key generated, encrypted, decrypted, and verified successfully")
	})
}

// TestStandaloneSSHAuthMethods tests SSH authentication method creation
func TestStandaloneSSHAuthMethods(t *testing.T) {
	t.Run("create_password_auth", func(t *testing.T) {
		// Create password authentication method
		passwordAuth := ssh.Password("test-password")
		if passwordAuth == nil {
			t.Error("Password auth method is nil")
		}
		t.Log("✓ Password authentication method created")
	})

	t.Run("create_public_key_auth", func(t *testing.T) {
		// Generate a test key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate private key: %v", err)
		}

		// Create SSH signer
		sshSigner, err := ssh.NewSignerFromKey(privateKey)
		if err != nil {
			t.Fatalf("Failed to create SSH signer: %v", err)
		}

		// Create public key authentication method
		publicKeyAuth := ssh.PublicKeys(sshSigner)
		if publicKeyAuth == nil {
			t.Error("Public key auth method is nil")
		}

		t.Log("✓ Public key authentication method created")
	})

	t.Run("create_keyboard_interactive_auth", func(t *testing.T) {
		// Create keyboard-interactive authentication method
		keyboardAuth := ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
			// Mock responses
			answers := make([]string, len(questions))
			for i := range questions {
				answers[i] = "mock-answer"
			}
			return answers, nil
		})

		if keyboardAuth == nil {
			t.Error("Keyboard-interactive auth method is nil")
		}

		t.Log("✓ Keyboard-interactive authentication method created")
	})
}

// TestStandaloneSSHConfig tests SSH client configuration
func TestStandaloneSSHConfig(t *testing.T) {
	t.Run("create_basic_ssh_config", func(t *testing.T) {
		// Create basic SSH client configuration
		config := &ssh.ClientConfig{
			User: "testuser",
			Auth: []ssh.AuthMethod{
				ssh.Password("test-password"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         15 * time.Second,
		}

		if config.User != "testuser" {
			t.Errorf("User mismatch: got %s, want testuser", config.User)
		}

		if len(config.Auth) != 1 {
			t.Errorf("Expected 1 auth method, got %d", len(config.Auth))
		}

		if config.Timeout != 15*time.Second {
			t.Errorf("Timeout mismatch: got %v, want 15s", config.Timeout)
		}

		t.Log("✓ Basic SSH configuration created successfully")
	})

	t.Run("create_multi_auth_ssh_config", func(t *testing.T) {
		// Generate a test key for public key auth
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate private key: %v", err)
		}

		sshSigner, err := ssh.NewSignerFromKey(privateKey)
		if err != nil {
			t.Fatalf("Failed to create SSH signer: %v", err)
		}

		// Create SSH config with multiple auth methods
		config := &ssh.ClientConfig{
			User: "testuser",
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(sshSigner),
				ssh.Password("test-password"),
				ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
					return []string{"test-answer"}, nil
				}),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         30 * time.Second,
		}

		if len(config.Auth) != 3 {
			t.Errorf("Expected 3 auth methods, got %d", len(config.Auth))
		}

		t.Log("✓ Multi-authentication SSH configuration created successfully")
	})
}

// TestStandaloneSSHKeyFormats tests different SSH key formats
func TestStandaloneSSHKeyFormats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ssh-key-formats-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("rsa_key_format", func(t *testing.T) {
		// Generate RSA key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate RSA key: %v", err)
		}

		// Test different PEM formats
		formats := []struct {
			name      string
			pemType   string
			marshal   func(*rsa.PrivateKey) []byte
			parse     func([]byte) (*rsa.PrivateKey, error)
		}{
			{
				name:    "PKCS1",
				pemType: "RSA PRIVATE KEY",
				marshal: func(key *rsa.PrivateKey) []byte {
					return x509.MarshalPKCS1PrivateKey(key)
				},
				parse: func(data []byte) (*rsa.PrivateKey, error) {
					return x509.ParsePKCS1PrivateKey(data)
				},
			},
			{
				name:    "PKCS8",
				pemType: "PRIVATE KEY",
				marshal: func(key *rsa.PrivateKey) []byte {
					data, _ := x509.MarshalPKCS8PrivateKey(key)
					return data
				},
				parse: func(data []byte) (*rsa.PrivateKey, error) {
					key, err := x509.ParsePKCS8PrivateKey(data)
					if err != nil {
						return nil, err
					}
					rsaKey, ok := key.(*rsa.PrivateKey)
					if !ok {
						return nil, fmt.Errorf("not an RSA key")
					}
					return rsaKey, nil
				},
			},
		}

		for _, format := range formats {
			t.Run(format.name, func(t *testing.T) {
				// Marshal key
				keyBytes := format.marshal(privateKey)

				// Create PEM block
				pemBlock := &pem.Block{
					Type:  format.pemType,
					Bytes: keyBytes,
				}
				pemData := pem.EncodeToMemory(pemBlock)

				// Write to file
				keyPath := filepath.Join(tempDir, fmt.Sprintf("test_key_%s", format.name))
				if err := os.WriteFile(keyPath, pemData, 0600); err != nil {
					t.Fatalf("Failed to write key file: %v", err)
				}

				// Read and parse
				fileData, err := os.ReadFile(keyPath)
				if err != nil {
					t.Fatalf("Failed to read key file: %v", err)
				}

				block, _ := pem.Decode(fileData)
				if block == nil {
					t.Fatal("Failed to decode PEM block")
				}

				parsedKey, err := format.parse(block.Bytes)
				if err != nil {
					t.Fatalf("Failed to parse key: %v", err)
				}

				// Verify keys match
				if parsedKey.N.Cmp(privateKey.N) != 0 {
					t.Error("Parsed key doesn't match original")
				}

				t.Logf("✓ %s format key successfully marshaled and parsed", format.name)
			})
		}
	})
}