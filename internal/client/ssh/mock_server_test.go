package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// TestSSHAuthenticationWithMockServer tests SSH authentication against a mock SSH server
func TestSSHAuthenticationWithMockServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping mock server test in short mode")
	}

	t.Run("successful_key_authentication", func(t *testing.T) {
		testSuccessfulKeyAuth(t)
	})

	t.Run("failed_key_authentication", func(t *testing.T) {
		testFailedKeyAuth(t)
	})
}

// testSuccessfulKeyAuth tests successful SSH key authentication
func testSuccessfulKeyAuth(t *testing.T) {
	// Create temporary directory for test keys
	tempDir, err := os.MkdirTemp("", "ssh-auth-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate client key pair
	clientPrivKey, clientPubKey := generateTestKeyPair(t)
	
	// Write client private key to file
	clientKeyPath := filepath.Join(tempDir, "client_key")
	if err := writePrivateKeyToFile(clientPrivKey, clientKeyPath); err != nil {
		t.Fatalf("Failed to write client key: %v", err)
	}

	// Start mock SSH server that accepts the client public key
	serverAddr, cleanup := startMockSSHServer(t, clientPubKey)
	defer cleanup()

	// Test our SSH authentication
	testSSHConnection(t, serverAddr, clientKeyPath, true)
}

// testFailedKeyAuth tests failed SSH key authentication
func testFailedKeyAuth(t *testing.T) {
	// Create temporary directory for test keys
	tempDir, err := os.MkdirTemp("", "ssh-auth-fail-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate client key pair
	clientPrivKey, _ := generateTestKeyPair(t)
	
	// Generate different server-accepted key
	_, serverPubKey := generateTestKeyPair(t)
	
	// Write client private key to file
	clientKeyPath := filepath.Join(tempDir, "client_key")
	if err := writePrivateKeyToFile(clientPrivKey, clientKeyPath); err != nil {
		t.Fatalf("Failed to write client key: %v", err)
	}

	// Start mock SSH server that accepts different public key
	serverAddr, cleanup := startMockSSHServer(t, serverPubKey)
	defer cleanup()

	// Test our SSH authentication (should fail)
	testSSHConnection(t, serverAddr, clientKeyPath, false)
}

// generateTestKeyPair generates a test RSA key pair
func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, ssh.PublicKey) {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create SSH public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to create public key: %v", err)
	}

	return privateKey, publicKey
}

// writePrivateKeyToFile writes an RSA private key to a file in PEM format
func writePrivateKeyToFile(privateKey *rsa.PrivateKey, filename string) error {
	// Convert to PEM format (standard RSA private key format)
	privKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	
	privateKeyBytes := pem.EncodeToMemory(privKeyPEM)
	return os.WriteFile(filename, privateKeyBytes, 0600)
}

// MockSSHServer represents a mock SSH server for testing
type MockSSHServer struct {
	listener   net.Listener
	authorizedKey ssh.PublicKey
	serverKey   ssh.Signer
}

// startMockSSHServer starts a mock SSH server that accepts the given public key
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

// testSSHConnection tests SSH connection using our authentication code
func testSSHConnection(t *testing.T, serverAddr, keyPath string, expectSuccess bool) {
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}

	// Parse server address
	host, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		t.Fatalf("Failed to parse server address: %v", err)
	}

	config := SSHConnectionConfig{
		User:            "testuser",
		KeyPath:         keyPath,
		TargetHost:      host,
		TargetPort:      port,
		InsecureHostKey: true, // Accept any host key for testing
		Verbose:         false,
		CurrentUser:     currentUser,
		Logger:          createTestLogger(),
	}

	// Create SSH config
	sshConfig, err := createSSHConfig(config)
	if err != nil {
		t.Fatalf("Failed to create SSH config: %v", err)
	}

	// Attempt connection
	conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
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

// TestCreateSSHAuthMethodsMock tests the createSSHAuthMethods function specifically in mock context
func TestCreateSSHAuthMethodsMock(t *testing.T) {
	tests := []struct {
		name        string
		keyPath     string
		user        string
		targetHost  string
		expectError bool
		description string
	}{
		{
			name:        "no_key_path_mock",
			keyPath:     "",
			user:        "testuser",
			targetHost:  "testhost",
			expectError: false,
			description: "Should work with password auth only in mock context",
		},
		{
			name:        "invalid_key_path_mock",
			keyPath:     "/nonexistent/key",
			user:        "testuser", 
			targetHost:  "testhost",
			expectError: false,
			description: "Should fallback to password auth in mock context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := createTestLogger()
			
			authMethods, err := createSSHAuthMethods(tt.keyPath, tt.user, tt.targetHost, logger)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if len(authMethods) == 0 {
				t.Error("No authentication methods returned")
				return
			}
			
			// Should always have at least password authentication
			if len(authMethods) < 1 {
				t.Error("Expected at least password authentication method")
				return
			}
			
			t.Logf("✓ %s: Got %d authentication methods", tt.description, len(authMethods))
		})
	}
}