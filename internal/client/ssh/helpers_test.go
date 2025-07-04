package ssh

import (
	"log"
	"testing"
	"time"
)

func TestCreateSSHConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      SSHConnectionConfig
		wantUser    string
		wantTimeout time.Duration
	}{
		{
			name: "basic config",
			config: SSHConnectionConfig{
				User:            "testuser",
				TargetHost:      "testhost",
				TargetPort:      "22",
				InsecureHostKey: true,
				Verbose:         false,
			},
			wantUser:    "testuser",
			wantTimeout: DefaultSSHTimeout,
		},
		{
			name: "insecure host key",
			config: SSHConnectionConfig{
				User:            "alice",
				TargetHost:      "example.com",
				TargetPort:      "2222",
				InsecureHostKey: true,
				Verbose:         true,
			},
			wantUser:    "alice",
			wantTimeout: DefaultSSHTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sshConfig, err := createSSHConfig(tt.config)
			if err != nil {
				t.Errorf("createSSHConfig() error = %v", err)
				return
			}

			if sshConfig.User != tt.wantUser {
				t.Errorf("createSSHConfig() User = %v, want %v", sshConfig.User, tt.wantUser)
			}

			if sshConfig.Timeout != tt.wantTimeout {
				t.Errorf("createSSHConfig() Timeout = %v, want %v", sshConfig.Timeout, tt.wantTimeout)
			}

			if sshConfig.HostKeyCallback == nil {
				t.Error("createSSHConfig() HostKeyCallback should not be nil")
			}

			if len(sshConfig.Auth) == 0 {
				t.Error("createSSHConfig() Auth methods should not be empty")
			}
		})
	}
}

func TestCreateSSHAuthMethods(t *testing.T) {
	tests := []struct {
		name       string
		keyPath    string
		user       string
		targetHost string
		expectKey  bool
		expectPass bool
	}{
		{
			name:       "empty key path",
			keyPath:    "",
			user:       "testuser",
			targetHost: "testhost",
			expectKey:  false,
			expectPass: true,
		},
		{
			name:       "invalid key path",
			keyPath:    "/nonexistent/key",
			user:       "testuser",
			targetHost: "testhost",
			expectKey:  false,
			expectPass: true,
		},
	}

	logger := log.Default()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMethods, err := createSSHAuthMethods(tt.keyPath, tt.user, tt.targetHost, logger)
			if err != nil {
				t.Errorf("createSSHAuthMethods() error = %v", err)
				return
			}

			if len(authMethods) == 0 {
				t.Error("createSSHAuthMethods() should return at least password auth")
			}

			// Should always have password callback as fallback
			if len(authMethods) < 1 {
				t.Error("createSSHAuthMethods() should include password authentication")
			}
		})
	}
}

func TestCreateSSHSession(t *testing.T) {
	// This test would require a real SSH client connection
	// For now, test the function signature and nil handling
	t.Run("nil client", func(t *testing.T) {
		_, err := CreateSSHSession(nil)
		if err == nil {
			t.Error("CreateSSHSession() should return error for nil client")
		}
	})
}

func TestSSHConnectionConfig(t *testing.T) {
	tests := []struct {
		name   string
		config SSHConnectionConfig
	}{
		{
			name: "valid config",
			config: SSHConnectionConfig{
				User:       "testuser",
				TargetHost: "testhost",
				TargetPort: "22",
			},
		},
		{
			name: "config with key path",
			config: SSHConnectionConfig{
				User:       "testuser",
				TargetHost: "testhost",
				TargetPort: "22",
				KeyPath:    "/path/to/key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the config struct can be created and accessed
			if tt.config.User == "" {
				t.Error("User should not be empty")
			}
			if tt.config.TargetHost == "" {
				t.Error("TargetHost should not be empty")
			}
			if tt.config.TargetPort == "" {
				t.Error("TargetPort should not be empty")
			}
		})
	}
}
