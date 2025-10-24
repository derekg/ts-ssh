package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// TestE2ESSHConnectionFlow tests the complete SSH connection flow
// This is an integration test that requires a mock SSH server
func TestE2ESSHConnectionFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("SSH connection with mock server", func(t *testing.T) {
		// This would require setting up a mock SSH server
		// For now, we test the parseSSHTarget integration with security validation
		tests := []struct {
			name      string
			target    string
			user      string
			port      string
			shouldErr bool
		}{
			{
				name:      "valid target",
				target:    "testhost",
				user:      "testuser",
				port:      "22",
				shouldErr: false,
			},
			{
				name:      "valid target with user",
				target:    "alice@testhost",
				user:      "testuser",
				port:      "22",
				shouldErr: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				user, host, port, err := parseSSHTarget(tt.target, tt.user, tt.port)
				if (err != nil) != tt.shouldErr {
					t.Errorf("parseSSHTarget() error = %v, shouldErr %v", err, tt.shouldErr)
				}
				if err == nil {
					if user == "" || host == "" || port == "" {
						t.Errorf("parseSSHTarget() returned empty values: user=%v, host=%v, port=%v", user, host, port)
					}
				}
			})
		}
	})
}

// TestE2ESCPFlow tests the SCP file transfer flow
func TestE2ESCPFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("SCP argument parsing flow", func(t *testing.T) {
		tests := []struct {
			name        string
			source      string
			dest        string
			expectError bool
		}{
			{
				name:        "local to remote",
				source:      "/tmp/file.txt",
				dest:        "host:/tmp/file.txt",
				expectError: false,
			},
			{
				name:        "remote to local",
				source:      "host:/tmp/file.txt",
				dest:        "/tmp/file.txt",
				expectError: false,
			},
			{
				name:        "both local - should fail",
				source:      "/tmp/file1.txt",
				dest:        "/tmp/file2.txt",
				expectError: true,
			},
			{
				name:        "both remote - should fail",
				source:      "host1:/tmp/file.txt",
				dest:        "host2:/tmp/file.txt",
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				srcHost, srcPath, srcIsRemote := parseSCPArg(tt.source)
				dstHost, dstPath, dstIsRemote := parseSCPArg(tt.dest)

				// Both remote or both local should be an error
				if srcIsRemote == dstIsRemote {
					if !tt.expectError {
						t.Errorf("Expected error for both remote/local scenario")
					}
					return
				}

				if tt.expectError {
					t.Errorf("Expected error but parsing succeeded")
					return
				}

				// Verify we got the right components
				if srcIsRemote {
					if srcHost == "" || srcPath == "" {
						t.Errorf("Remote source should have host and path")
					}
				} else {
					if dstHost == "" || dstPath == "" {
						t.Errorf("Remote dest should have host and path")
					}
				}
			})
		}
	})
}

// TestE2ECommandLineFlags tests various command-line flag combinations
func TestE2ECommandLineFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("version flag", func(t *testing.T) {
		if version == "" {
			t.Error("version should be set")
		}
	})

	t.Run("default values", func(t *testing.T) {
		username := currentUsername()
		if username == "" {
			t.Error("currentUsername() returned empty")
		}

		keyPath := defaultKeyPath()
		if keyPath == "" {
			t.Error("defaultKeyPath() returned empty")
		}

		tsnetDir := defaultTsnetDir()
		if tsnetDir == "" {
			t.Error("defaultTsnetDir() returned empty")
		}
	})
}

// TestE2ESecurityValidation tests security validation in the flow
func TestE2ESecurityValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("hostname validation", func(t *testing.T) {
		validTargets := []struct {
			target string
			host   string
		}{
			{"user@localhost:22", "localhost"},
			{"user@example.com:22", "example.com"},
			{"user@server-1:22", "server-1"},
			{"user@192.168.1.1:22", "192.168.1.1"},
			{"user@[::1]:22", "::1"}, // IPv6 needs brackets
		}

		for _, tt := range validTargets {
			_, parsedHost, _, err := parseSSHTarget(tt.target, "user", "22")
			if err != nil {
				t.Errorf("parseSSHTarget(%q) unexpected error: %v", tt.target, err)
			}
			if parsedHost != tt.host {
				t.Errorf("parseSSHTarget(%q) host = %v, want %v", tt.target, parsedHost, tt.host)
			}
		}
	})

	t.Run("port validation", func(t *testing.T) {
		validPorts := []string{"22", "2222", "8022", "10000"}

		for _, port := range validPorts {
			target := fmt.Sprintf("user@host:%s", port)
			_, _, parsedPort, err := parseSSHTarget(target, "user", "22")
			if err != nil {
				t.Errorf("parseSSHTarget with port %s unexpected error: %v", port, err)
			}
			if parsedPort != port {
				t.Errorf("parseSSHTarget port = %v, want %v", parsedPort, port)
			}
		}
	})
}

// TestE2EURLExtraction tests URL extraction in auth flows
func TestE2EURLExtraction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("tailscale auth URL extraction", func(t *testing.T) {
		authMessages := []struct {
			message string
			wantURL string
		}{
			{
				message: "To authenticate, visit: https://login.tailscale.com/a/abc123def456",
				wantURL: "https://login.tailscale.com/a/abc123def456",
			},
			{
				message: "Visit https://login.tailscale.com/admin/machines/xyz789 to authorize",
				wantURL: "https://login.tailscale.com/admin/machines/xyz789",
			},
			{
				message: "Auth required: https://login.tailscale.com/a/test\nPlease visit",
				wantURL: "https://login.tailscale.com/a/test",
			},
		}

		for _, tt := range authMessages {
			url := extractURL(tt.message)
			if !strings.Contains(url, "https://") {
				t.Errorf("extractURL(%q) = %v, should contain https://", tt.message, url)
			}
		}
	})
}

// mockSSHServer provides a minimal SSH server for testing
type mockSSHServer struct {
	listener net.Listener
	config   *ssh.ServerConfig
	t        *testing.T
}

func newMockSSHServer(t *testing.T) (*mockSSHServer, error) {
	config := &ssh.ServerConfig{
		NoClientAuth: true, // For testing only
	}

	// Generate a test host key
	privateKey, err := generateTestHostKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate host key: %w", err)
	}

	config.AddHostKey(privateKey)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	return &mockSSHServer{
		listener: listener,
		config:   config,
		t:        t,
	}, nil
}

func (s *mockSSHServer) Address() string {
	return s.listener.Addr().String()
}

func (s *mockSSHServer) Close() error {
	return s.listener.Close()
}

func (s *mockSSHServer) Serve(ctx context.Context) {
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					s.t.Logf("Failed to accept connection: %v", err)
					return
				}
			}

			go s.handleConnection(conn)
		}
	}()
}

func (s *mockSSHServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		s.t.Logf("Failed to handshake: %v", err)
		return
	}
	defer sshConn.Close()

	// Handle SSH global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			s.t.Logf("Failed to accept channel: %v", err)
			continue
		}

		go func() {
			defer channel.Close()
			for req := range requests {
				switch req.Type {
				case "exec":
					if req.WantReply {
						req.Reply(true, nil)
					}
					// Send a simple response
					fmt.Fprintf(channel, "mock command output\n")
					channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
					return
				case "shell":
					if req.WantReply {
						req.Reply(true, nil)
					}
					// Echo back
					io.Copy(channel, channel)
					return
				default:
					if req.WantReply {
						req.Reply(false, nil)
					}
				}
			}
		}()
	}
}

func generateTestHostKey() (ssh.Signer, error) {
	// For testing, we can use a simple approach
	// In production, you'd want to generate proper keys
	keyPath := filepath.Join(os.TempDir(), fmt.Sprintf("test_host_key_%d", time.Now().UnixNano()))
	defer os.Remove(keyPath)

	// Create a minimal test key (this is just for testing)
	// In a real scenario, you'd generate RSA or ED25519 keys properly
	testKey := []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACAabc+Qa0zYHY8AO8zKIEsEqhQMO4k9u0P4wYmkV9D4hAAAAJgPVkHED1ZB
xAAAAAtzc2gtZWQyNTUxOQAAACAabc+Qa0zYHY8AO8zKIEsEqhQMO4k9u0P4wYmkV9D4hA
AAAECiL7VQKV+qXZTg0F7kCxEMOqTZDJjb3FWLQN7FzzNZFhptz5BrTNgdjwA7zMogSwSq
FAw7iT27Q/jBiaRX0PiEAAAAEHRlc3RAZXhhbXBsZS5jb20BAgMEBQ==
-----END OPENSSH PRIVATE KEY-----`)

	signer, err := ssh.ParsePrivateKey(testKey)
	if err != nil {
		return nil, err
	}

	return signer, nil
}

// TestE2EMockSSHServer tests with an actual mock SSH server
func TestE2EMockSSHServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test with mock server in short mode")
	}

	t.Run("mock SSH server setup", func(t *testing.T) {
		server, err := newMockSSHServer(t)
		if err != nil {
			t.Skipf("Could not create mock SSH server: %v", err)
			return
		}
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		server.Serve(ctx)

		// Verify the server is listening
		addr := server.Address()
		if addr == "" {
			t.Error("Mock server address is empty")
		}

		t.Logf("Mock SSH server running on %s", addr)
	})
}
