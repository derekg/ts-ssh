package main

import (
	"testing"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name         string
		target       string
		expectedHost string
		expectedPort string
		expectErr    bool
	}{
		{
			name:         "hostname only",
			target:       "myhost",
			expectedHost: "myhost",
			expectedPort: DefaultSshPort, // "22"
			expectErr:    false,
		},
		{
			name:         "hostname with user",
			target:       "user@myhost",
			expectedHost: "user@myhost", // parseTarget itself doesn't split user, main does
			expectedPort: DefaultSshPort,
			expectErr:    false,
		},
		{
			name:         "hostname with port",
			target:       "myhost:2222",
			expectedHost: "myhost",
			expectedPort: "2222",
			expectErr:    false,
		},
		{
			name:         "hostname with user and port",
			target:       "user@myhost:2222",
			expectedHost: "user@myhost", // parseTarget itself doesn't split user
			expectedPort: "2222",
			expectErr:    false,
		},
		{
			name:         "ipv4 address",
			target:       "192.168.1.1",
			expectedHost: "192.168.1.1",
			expectedPort: DefaultSshPort,
			expectErr:    false,
		},
		{
			name:         "ipv4 address with port",
			target:       "192.168.1.1:2222",
			expectedHost: "192.168.1.1",
			expectedPort: "2222",
			expectErr:    false,
		},
		{
			name:         "ipv6 address",
			target:       "[::1]",
			expectedHost: "::1",
			expectedPort: DefaultSshPort,
			expectErr:    false,
		},
		{
			name:         "ipv6 address with port",
			target:       "[::1]:2222",
			expectedHost: "::1",
			expectedPort: "2222",
			expectErr:    false,
		},
		{
			name:         "invalid port",
			target:       "myhost:abc",
			expectedHost: "",
			expectedPort: "",
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := parseTarget(tt.target, DefaultSshPort)

			if (err != nil) != tt.expectErr {
				t.Errorf("parseTarget() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				if host != tt.expectedHost {
					t.Errorf("parseTarget() host = %v, want %v", host, tt.expectedHost)
				}
				if port != tt.expectedPort {
					t.Errorf("parseTarget() port = %v, want %v", port, tt.expectedPort)
				}
			}
		})
	}
}

func TestParseScpRemoteArg(t *testing.T) {
	tests := []struct {
		name           string
		remoteArg      string
		defaultSSHUser string
		expectedHost   string
		expectedPath   string
		expectedUser   string
		expectErr      bool
	}{
		{
			name:           "host:path",
			remoteArg:      "myhost:/tmp/file",
			defaultSSHUser: "defaultuser",
			expectedHost:   "myhost",
			expectedPath:   "/tmp/file",
			expectedUser:   "defaultuser",
			expectErr:      false,
		},
		{
			name:           "user@host:path",
			remoteArg:      "alice@myhost:/tmp/file",
			defaultSSHUser: "defaultuser",
			expectedHost:   "myhost",
			expectedPath:   "/tmp/file",
			expectedUser:   "alice",
			expectErr:      false,
		},
		{
			name:           "missing path",
			remoteArg:      "myhost:",
			defaultSSHUser: "defaultuser",
			expectedHost:   "",
			expectedPath:   "",
			expectedUser:   "",
			expectErr:      true,
		},
		{
			name:           "missing colon",
			remoteArg:      "myhost",
			defaultSSHUser: "defaultuser",
			expectedHost:   "",
			expectedPath:   "",
			expectedUser:   "",
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, path, user, err := parseScpRemoteArg(tt.remoteArg, tt.defaultSSHUser)

			if (err != nil) != tt.expectErr {
				t.Errorf("parseScpRemoteArg() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				if host != tt.expectedHost {
					t.Errorf("parseScpRemoteArg() host = %v, want %v", host, tt.expectedHost)
				}
				if path != tt.expectedPath {
					t.Errorf("parseScpRemoteArg() path = %v, want %v", path, tt.expectedPath)
				}
				if user != tt.expectedUser {
					t.Errorf("parseScpRemoteArg() user = %v, want %v", user, tt.expectedUser)
				}
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Test that our constants have expected values
	if DefaultSshPort != "22" {
		t.Errorf("DefaultSshPort = %v, want %v", DefaultSshPort, "22")
	}
	
	if ClientName == "" {
		t.Error("ClientName should not be empty")
	}
	
	if DefaultSSHTimeout.Seconds() != 15 {
		t.Errorf("DefaultSSHTimeout = %v seconds, want 15", DefaultSSHTimeout.Seconds())
	}
	
	if DefaultSCPTimeout.Seconds() != 30 {
		t.Errorf("DefaultSCPTimeout = %v seconds, want 30", DefaultSCPTimeout.Seconds())
	}
	
	if DefaultTerminalWidth != 80 {
		t.Errorf("DefaultTerminalWidth = %v, want 80", DefaultTerminalWidth)
	}
	
	if DefaultTerminalHeight != 24 {
		t.Errorf("DefaultTerminalHeight = %v, want 24", DefaultTerminalHeight)
	}
	
	if DefaultTerminalType != "xterm-256color" {
		t.Errorf("DefaultTerminalType = %v, want xterm-256color", DefaultTerminalType)
	}
}

func TestParseHostList(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "empty args",
			args:     []string{},
			expected: nil,
		},
		{
			name:     "single host",
			args:     []string{"host1"},
			expected: []string{"host1"},
		},
		{
			name:     "comma separated hosts",
			args:     []string{"host1,host2,host3"},
			expected: []string{"host1", "host2", "host3"},
		},
		{
			name:     "mixed args",
			args:     []string{"host1", "host2,host3", "host4"},
			expected: []string{"host1", "host2", "host3", "host4"},
		},
		{
			name:     "hosts with spaces",
			args:     []string{" host1 ", "host2, host3 "},
			expected: []string{"host1", "host2", "host3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHostList(tt.args)
			if len(result) != len(tt.expected) {
				t.Errorf("parseHostList() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, host := range result {
				if host != tt.expected[i] {
					t.Errorf("parseHostList()[%d] = %v, want %v", i, host, tt.expected[i])
				}
			}
		})
	}
}

func TestIsPowerCLIMode(t *testing.T) {
	tests := []struct {
		name     string
		config   *AppConfig
		expected bool
	}{
		{
			name:     "no flags set",
			config:   &AppConfig{},
			expected: false,
		},
		{
			name:     "list hosts",
			config:   &AppConfig{ListHosts: true},
			expected: true,
		},
		{
			name:     "multi hosts",
			config:   &AppConfig{MultiHosts: "host1,host2"},
			expected: true,
		},
		{
			name:     "exec command",
			config:   &AppConfig{ExecCmd: "ls -la"},
			expected: true,
		},
		{
			name:     "copy files",
			config:   &AppConfig{CopyFiles: "file host:/path"},
			expected: true,
		},
		{
			name:     "pick host",
			config:   &AppConfig{PickHost: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPowerCLIMode(tt.config)
			if result != tt.expected {
				t.Errorf("isPowerCLIMode() = %v, want %v", result, tt.expected)
			}
		})
	}
}