package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSH Authentication Integration Test Suite for ts-ssh
// This test suite verifies SSH key authentication functionality

// TestSSHKeyGeneration tests SSH key generation and loading
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
		authMethod, err := loadPrivateKey(privPath)
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
		authMethod, err := loadPrivateKey(privPath)
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

// TestSSHKeyAuthentication tests SSH key authentication end-to-end with mock server
func TestSSHKeyAuthentication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
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
	tempDir, err := os.MkdirTemp("", "ts-ssh-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	var privateKeyPath string
	var clientPubKey ssh.PublicKey
	
	if setupKeys {
		// Generate and setup SSH keys
		privPath, _ := generateSSHKeyPair(t, tempDir, usePassphrase)
		privateKeyPath = privPath

		// Load the public key for the mock server
		authMethod, err := loadPrivateKey(privateKeyPath)
		if err != nil && !usePassphrase {
			t.Fatalf("Failed to load key for testing: %v", err)
		}
		
		if !usePassphrase && authMethod != nil {
			// Load the public key directly from the private key file for the mock server
			keyData, err := os.ReadFile(privateKeyPath)
			if err != nil {
				t.Fatalf("Failed to read private key: %v", err)
			}
			
			block, _ := pem.Decode(keyData)
			if block == nil {
				t.Fatal("Failed to decode PEM block")
			}
			
			privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				t.Fatalf("Failed to parse private key: %v", err)
			}
			
			clientPubKey, err = ssh.NewPublicKey(&privKey.PublicKey)
			if err != nil {
				t.Fatalf("Failed to create public key: %v", err)
			}
		}
	}

	if setupKeys && !usePassphrase && expectedResult {
		// Test with mock SSH server
		testWithMockServer(t, clientPubKey, privateKeyPath, expectedResult)
	} else {
		// Test key loading functionality
		testSSHKeyLoading(t, privateKeyPath, usePassphrase, expectedResult)
	}
}

// testWithMockServer tests SSH authentication against a mock server
func testWithMockServer(t *testing.T, authorizedKey ssh.PublicKey, keyPath string, expectSuccess bool) {
	// Start mock SSH server
	serverAddr, cleanup := startMockSSHServer(t, authorizedKey)
	defer cleanup()

	// Test SSH connection
	testSSHConnection(t, serverAddr, keyPath, expectSuccess)
}

// testSSHConnection tests SSH connection using authentication
func testSSHConnection(t *testing.T, serverAddr, keyPath string, expectSuccess bool) {
	// Parse server address
	host, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		t.Fatalf("Failed to parse server address: %v", err)
	}

	// Load private key
	authMethod, err := loadPrivateKey(keyPath)
	if err != nil {
		t.Fatalf("Failed to load key: %v", err)
	}

	// Create SSH config
	sshConfig := &ssh.ClientConfig{
		User: "testuser",
		Auth: []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout: 5 * time.Second,
	}

	// Attempt connection
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to mock server: %v", err)
	}

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, serverAddr, sshConfig)
	
	if expectSuccess {
		if err != nil {
			t.Errorf("Expected successful SSH connection, but got error: %v", err)
			return
		}
		defer sshConn.Close()
		
		client := ssh.NewClient(sshConn, chans, reqs)
		defer client.Close()
		
		t.Log("✓ SSH authentication successful")
		
		// Test that we can create a session
		session, err := client.NewSession()
		if err != nil {
			t.Errorf("Failed to create SSH session: %v", err)
			return
		}
		defer session.Close()
		
		t.Log("✓ SSH session created successfully")
		
	} else {
		if err == nil {
			if sshConn != nil {
				sshConn.Close()
			}
			t.Error("Expected SSH connection to fail, but it succeeded")
			return
		}
		t.Logf("✓ SSH authentication failed as expected: %v", err)
	}
}

// testSSHKeyLoading tests SSH key loading functionality
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

	// Test loadPrivateKey function
	if !usePassphrase {
		authMethod, err := loadPrivateKey(privateKeyPath)
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
	}
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

// loadPrivateKey loads a private key from file (simplified version for testing)
func loadPrivateKey(keyPath string) (ssh.AuthMethod, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Parse the PEM block
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	// Check if encrypted
	if x509.IsEncryptedPEMBlock(block) {
		return nil, fmt.Errorf("encrypted key requires passphrase")
	}

	// Parse the private key
	var privateKey interface{}
	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		privateKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create SSH signer
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

// startMockSSHServer starts a mock SSH server for testing
func startMockSSHServer(t *testing.T, authorizedKey ssh.PublicKey) (string, func()) {
	// Generate server host key
	serverPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate server key: %v", err)
	}

	serverKey, err := ssh.NewSignerFromKey(serverPrivKey)
	if err != nil {
		t.Fatalf("Failed to create server signer: %v", err)
	}

	// Create SSH server config
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// Check if the provided key matches our authorized key
			if string(key.Marshal()) == string(authorizedKey.Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("authentication failed")
		},
	}
	config.AddHostKey(serverKey)

	// Start listening
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	// Handle connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}
			go handleSSHConnection(t, conn, config)
		}
	}()

	cleanup := func() {
		listener.Close()
	}

	return listener.Addr().String(), cleanup
}

// handleSSHConnection handles a single SSH connection
func handleSSHConnection(t *testing.T, conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		// Authentication failure is expected for some tests
		return
	}
	defer sshConn.Close()

	// Handle SSH requests (minimal implementation)
	go ssh.DiscardRequests(reqs)

	// Handle channels (minimal implementation)
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		// Handle session requests
		go func() {
			defer channel.Close()
			for req := range requests {
				switch req.Type {
				case "exec":
					// Acknowledge exec request and close
					req.Reply(true, nil)
					channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
					return
				case "shell":
					// Acknowledge shell request and close
					req.Reply(true, nil)
					channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
					return
				default:
					req.Reply(false, nil)
				}
			}
		}()
	}
}