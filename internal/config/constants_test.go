package config

import (
	"testing"
)

// TestConstants verifies that all constants are properly defined
func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		nonZero  bool
		expected interface{}
	}{
		{
			name:     "DefaultSSHPort",
			value:    DefaultSSHPort,
			nonZero:  true,
			expected: "22",
		},
		{
			name:     "DefaultTerminalWidth",
			value:    DefaultTerminalWidth,
			nonZero:  true,
			expected: 80,
		},
		{
			name:     "DefaultTerminalHeight",
			value:    DefaultTerminalHeight,
			nonZero:  true,
			expected: 24,
		},
		{
			name:     "DefaultTerminalType",
			value:    DefaultTerminalType,
			nonZero:  true,
			expected: "xterm-256color",
		},
		{
			name:     "ClientName",
			value:    ClientName,
			nonZero:  true,
			expected: "ts-ssh",
		},
		{
			name:    "DefaultConnectionTimeout",
			value:   DefaultConnectionTimeout,
			nonZero: true,
		},
		{
			name:    "DefaultCommandTimeout",
			value:   DefaultCommandTimeout,
			nonZero: true,
		},
		{
			name:    "SecureFilePermissions",
			value:   SecureFilePermissions,
			nonZero: true,
		},
		{
			name:    "SecureDirectoryPermissions",
			value:   SecureDirectoryPermissions,
			nonZero: true,
		},
		{
			name:     "ModernKeyTypes length",
			value:    len(ModernKeyTypes),
			nonZero:  true,
			expected: nil, // We just check it's non-zero
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expected != nil {
				if tt.value != tt.expected {
					t.Errorf("%s = %v, want %v", tt.name, tt.value, tt.expected)
				}
			} else if tt.nonZero {
				switch v := tt.value.(type) {
				case string:
					if v == "" {
						t.Errorf("%s should not be empty", tt.name)
					}
				case int:
					if v == 0 {
						t.Errorf("%s should not be zero", tt.name)
					}
				}
			}
		})
	}
}

// TestDefaultSSHPortValidity tests that the default SSH port is valid
func TestDefaultSSHPortValidity(t *testing.T) {
	if DefaultSSHPort != "22" {
		t.Errorf("DefaultSSHPort = %s, want 22", DefaultSSHPort)
	}
}

// TestTimeoutValues tests that timeout values are reasonable
func TestTimeoutValues(t *testing.T) {
	timeouts := map[string]int{
		"DefaultConnectionTimeout": DefaultConnectionTimeout,
		"DefaultCommandTimeout":    DefaultCommandTimeout,
		"SSHAuthTimeout":           SSHAuthTimeout,
		"SSHConnectTimeout":        SSHConnectTimeout,
		"SSHHandshakeTimeout":      SSHHandshakeTimeout,
	}

	for name, timeout := range timeouts {
		t.Run(name, func(t *testing.T) {
			// Timeouts should be positive
			if timeout <= 0 {
				t.Errorf("%s should be positive, got %v", name, timeout)
			}

			// Timeouts should be reasonable (not too large)
			maxReasonable := 600 // 10 minutes in seconds
			if timeout > maxReasonable {
				t.Errorf("%s seems too large: %v (max reasonable: %v)", name, timeout, maxReasonable)
			}

			// Timeouts should not be too small
			minReasonable := 1 // 1 second
			if timeout < minReasonable {
				t.Errorf("%s seems too small: %v (min reasonable: %v)", name, timeout, minReasonable)
			}
		})
	}
}

// TestApplicationValues tests application configuration values
func TestApplicationValues(t *testing.T) {
	if MaxConcurrentConnections <= 0 {
		t.Errorf("MaxConcurrentConnections should be positive, got %d", MaxConcurrentConnections)
	}

	if MaxConcurrentConnections > 100 {
		t.Errorf("MaxConcurrentConnections seems too large: %d", MaxConcurrentConnections)
	}

	if DefaultBatchSize <= 0 {
		t.Errorf("DefaultBatchSize should be positive, got %d", DefaultBatchSize)
	}

	if MaxLogFileSize <= 0 {
		t.Errorf("MaxLogFileSize should be positive, got %d", MaxLogFileSize)
	}

	if MaxLogFiles <= 0 {
		t.Errorf("MaxLogFiles should be positive, got %d", MaxLogFiles)
	}

	// Check that hostname length is reasonable
	if MaxHostnameLength <= 0 || MaxHostnameLength > 255 {
		t.Errorf("MaxHostnameLength should be between 1 and 255, got %d", MaxHostnameLength)
	}
}

// TestModernKeyTypes tests SSH key type configuration
func TestModernKeyTypes(t *testing.T) {
	// Array should not be empty
	if len(ModernKeyTypes) == 0 {
		t.Error("ModernKeyTypes should not be empty")
	}

	// Array should not contain empty strings
	for i, keyType := range ModernKeyTypes {
		if keyType == "" {
			t.Errorf("ModernKeyTypes[%d] should not be empty", i)
		}
	}

	// Array should not contain duplicates
	seen := make(map[string]bool)
	for i, keyType := range ModernKeyTypes {
		if seen[keyType] {
			t.Errorf("ModernKeyTypes[%d] contains duplicate: %s", i, keyType)
		}
		seen[keyType] = true
	}
}

// TestModernKeyTypesContent tests specific expected key types
func TestModernKeyTypesContent(t *testing.T) {
	// Check that expected key types are present
	expectedKeyTypes := []string{
		"id_ed25519",
		"id_ecdsa",
		"id_rsa",
	}

	keyTypeSet := make(map[string]bool)
	for _, keyType := range ModernKeyTypes {
		keyTypeSet[keyType] = true
	}

	for _, expected := range expectedKeyTypes {
		if !keyTypeSet[expected] {
			t.Errorf("ModernKeyTypes should contain %s", expected)
		}
	}
}

// TestFilePermissions tests file permission constants
func TestFilePermissions(t *testing.T) {
	// Test secure file permissions (0600 = owner read/write only)
	if SecureFilePermissions != 0600 {
		t.Errorf("SecureFilePermissions = %o, want 0600", SecureFilePermissions)
	}

	// Test secure directory permissions (0700 = owner read/write/execute only)
	if SecureDirectoryPermissions != 0700 {
		t.Errorf("SecureDirectoryPermissions = %o, want 0700", SecureDirectoryPermissions)
	}
}

// TestSSHConfigConstants tests SSH configuration constants
func TestSSHConfigConstants(t *testing.T) {
	if KnownHostsFileName == "" {
		t.Error("KnownHostsFileName should not be empty")
	}

	if SSHConfigDirName == "" {
		t.Error("SSHConfigDirName should not be empty")
	}

	if SSHConfigDirName != ".ssh" {
		t.Errorf("SSHConfigDirName = %s, want .ssh", SSHConfigDirName)
	}

	if KnownHostsFileName != "known_hosts" {
		t.Errorf("KnownHostsFileName = %s, want known_hosts", KnownHostsFileName)
	}
}

// TestTmuxConfiguration tests Tmux-related constants
func TestTmuxConfiguration(t *testing.T) {
	if TmuxSessionPrefix == "" {
		t.Error("TmuxSessionPrefix should not be empty")
	}

	if TmuxSessionPrefix != "ts-ssh" {
		t.Errorf("TmuxSessionPrefix = %s, want ts-ssh", TmuxSessionPrefix)
	}
}

// TestVersionVariables tests version-related variables
func TestVersionVariables(t *testing.T) {
	// These should have default values even if not set by build process
	if Version == "" {
		t.Error("Version should not be empty")
	}

	if GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}

	if BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}

	// Test default values
	if Version != "dev" {
		t.Logf("Version is set to: %s (expected 'dev' for development builds)", Version)
	}
}

// TestPreferredKeyTypes tests the key type preference string
func TestPreferredKeyTypes(t *testing.T) {
	if PreferredKeyTypes == "" {
		t.Error("PreferredKeyTypes should not be empty")
	}

	// Should contain expected key types
	expectedTypes := []string{"ed25519", "ecdsa", "rsa"}
	for _, keyType := range expectedTypes {
		if !contains(PreferredKeyTypes, keyType) {
			t.Errorf("PreferredKeyTypes should contain %s", keyType)
		}
	}
}

// TestConfigurationConsistency tests that related configurations are consistent
func TestConfigurationConsistency(t *testing.T) {
	// Connection timeout should be longer than SSH handshake timeout
	if DefaultConnectionTimeout <= SSHHandshakeTimeout {
		t.Errorf("DefaultConnectionTimeout (%d) should be longer than SSHHandshakeTimeout (%d)",
			DefaultConnectionTimeout, SSHHandshakeTimeout)
	}

	// SSH auth timeout should be reasonable compared to connect timeout
	if SSHAuthTimeout > DefaultConnectionTimeout {
		t.Errorf("SSHAuthTimeout (%d) should not be longer than DefaultConnectionTimeout (%d)",
			SSHAuthTimeout, DefaultConnectionTimeout)
	}

	// Command timeout should be longer than connection timeout
	if DefaultCommandTimeout <= DefaultConnectionTimeout {
		t.Errorf("DefaultCommandTimeout (%d) should be longer than DefaultConnectionTimeout (%d)",
			DefaultCommandTimeout, DefaultConnectionTimeout)
	}

	// Batch size should not exceed max concurrent connections
	if DefaultBatchSize > MaxConcurrentConnections {
		t.Errorf("DefaultBatchSize (%d) should not exceed MaxConcurrentConnections (%d)",
			DefaultBatchSize, MaxConcurrentConnections)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findInString(s, substr))))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
