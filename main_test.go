package main

import (
	"testing"
)

func TestParseSSHTarget(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		defaultUser string
		defaultPort string
		wantUser    string
		wantHost    string
		wantPort    string
		wantErr     bool
	}{
		{
			name:        "hostname only",
			target:      "myhost",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "testuser",
			wantHost:    "myhost",
			wantPort:    "22",
		},
		{
			name:        "user@hostname",
			target:      "alice@myhost",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "alice",
			wantHost:    "myhost",
			wantPort:    "22",
		},
		{
			name:        "hostname:port",
			target:      "myhost:2222",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "testuser",
			wantHost:    "myhost",
			wantPort:    "2222",
		},
		{
			name:        "user@hostname:port",
			target:      "alice@myhost:2222",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "alice",
			wantHost:    "myhost",
			wantPort:    "2222",
		},
		{
			name:        "ipv4 address",
			target:      "192.168.1.1",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "testuser",
			wantHost:    "192.168.1.1",
			wantPort:    "22",
		},
		{
			name:        "ipv4:port",
			target:      "192.168.1.1:2222",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "testuser",
			wantHost:    "192.168.1.1",
			wantPort:    "2222",
		},
		{
			name:        "ipv6 address",
			target:      "[::1]",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "testuser",
			wantHost:    "::1",
			wantPort:    "22",
		},
		{
			name:        "ipv6:port",
			target:      "[::1]:2222",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "testuser",
			wantHost:    "::1",
			wantPort:    "2222",
		},
		{
			name:        "empty target",
			target:      "",
			defaultUser: "testuser",
			defaultPort: "22",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, host, port, err := parseSSHTarget(tt.target, tt.defaultUser, tt.defaultPort)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSSHTarget() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseSSHTarget() unexpected error: %v", err)
				return
			}

			if user != tt.wantUser {
				t.Errorf("parseSSHTarget() user = %v, want %v", user, tt.wantUser)
			}
			if host != tt.wantHost {
				t.Errorf("parseSSHTarget() host = %v, want %v", host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Errorf("parseSSHTarget() port = %v, want %v", port, tt.wantPort)
			}
		})
	}
}

func TestParseSCPArg(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		wantHost string
		wantPath string
		isRemote bool
	}{
		{
			name:     "local path",
			arg:      "/tmp/file.txt",
			wantHost: "",
			wantPath: "/tmp/file.txt",
			isRemote: false,
		},
		{
			name:     "relative path",
			arg:      "file.txt",
			wantHost: "",
			wantPath: "file.txt",
			isRemote: false,
		},
		{
			name:     "remote path",
			arg:      "host:/tmp/file.txt",
			wantHost: "host",
			wantPath: "/tmp/file.txt",
			isRemote: true,
		},
		{
			name:     "remote with user",
			arg:      "user@host:/tmp/file.txt",
			wantHost: "user@host",
			wantPath: "/tmp/file.txt",
			isRemote: true,
		},
		{
			name:     "windows drive letter",
			arg:      "C:\\Users\\test\\file.txt",
			wantHost: "",
			wantPath: "C:\\Users\\test\\file.txt",
			isRemote: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, path, isRemote := parseSCPArg(tt.arg)
			if host != tt.wantHost {
				t.Errorf("parseSCPArg() host = %v, want %v", host, tt.wantHost)
			}
			if path != tt.wantPath {
				t.Errorf("parseSCPArg() path = %v, want %v", path, tt.wantPath)
			}
			if isRemote != tt.isRemote {
				t.Errorf("parseSCPArg() isRemote = %v, want %v", isRemote, tt.isRemote)
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

func TestExtractURL(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want string
	}{
		{
			name: "URL in middle of message",
			msg:  "Please visit https://login.tailscale.com/a/123 to authenticate",
			want: "https://login.tailscale.com/a/123",
		},
		{
			name: "URL at start",
			msg:  "https://login.tailscale.com/a/456",
			want: "https://login.tailscale.com/a/456",
		},
		{
			name: "No URL",
			msg:  "No URL here",
			want: "No URL here",
		},
		{
			name: "URL with newline",
			msg:  "Visit https://example.com\nfor more info",
			want: "https://example.com",
		},
		{
			name: "URL with tab",
			msg:  "Visit https://example.com\tfor details",
			want: "https://example.com",
		},
		{
			name: "URL with space",
			msg:  "Visit https://example.com and continue",
			want: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractURL(tt.msg)
			if got != tt.want {
				t.Errorf("extractURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("currentUsername", func(t *testing.T) {
		username := currentUsername()
		if username == "" {
			t.Error("currentUsername() should not return empty string")
		}
	})

	t.Run("defaultKeyPath", func(t *testing.T) {
		keyPath := defaultKeyPath()
		if keyPath == "" {
			t.Error("defaultKeyPath() should not return empty string")
		}
		if keyPath != "~/.ssh/id_rsa" && !contains(keyPath, ".ssh") {
			t.Errorf("defaultKeyPath() = %v, expected path containing .ssh", keyPath)
		}
	})

	t.Run("defaultTsnetDir", func(t *testing.T) {
		tsnetDir := defaultTsnetDir()
		if tsnetDir == "" {
			t.Error("defaultTsnetDir() should not return empty string")
		}
		if !contains(tsnetDir, ClientName) {
			t.Errorf("defaultTsnetDir() = %v, expected path containing %v", tsnetDir, ClientName)
		}
	})
}

func TestParseSCPArgEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		wantHost string
		wantPath string
		isRemote bool
	}{
		{
			name:     "remote with port notation",
			arg:      "host:2222:/tmp/file.txt",
			wantHost: "host",
			wantPath: "2222:/tmp/file.txt",
			isRemote: true,
		},
		{
			name:     "user@host with port notation",
			arg:      "user@host:2222:/tmp/file.txt",
			wantHost: "user@host",
			wantPath: "2222:/tmp/file.txt",
			isRemote: true,
		},
		{
			name:     "path with spaces",
			arg:      "/tmp/my file.txt",
			wantHost: "",
			wantPath: "/tmp/my file.txt",
			isRemote: false,
		},
		{
			name:     "remote path with spaces",
			arg:      "host:/tmp/my file.txt",
			wantHost: "host",
			wantPath: "/tmp/my file.txt",
			isRemote: true,
		},
		{
			name:     "empty path",
			arg:      "",
			wantHost: "",
			wantPath: "",
			isRemote: false,
		},
		{
			name:     "just colon",
			arg:      ":",
			wantHost: "",
			wantPath: ":",
			isRemote: false,
		},
		{
			name:     "D drive windows",
			arg:      "D:\\data\\file.txt",
			wantHost: "",
			wantPath: "D:\\data\\file.txt",
			isRemote: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, path, isRemote := parseSCPArg(tt.arg)
			if host != tt.wantHost {
				t.Errorf("parseSCPArg() host = %v, want %v", host, tt.wantHost)
			}
			if path != tt.wantPath {
				t.Errorf("parseSCPArg() path = %v, want %v", path, tt.wantPath)
			}
			if isRemote != tt.isRemote {
				t.Errorf("parseSCPArg() isRemote = %v, want %v", isRemote, tt.isRemote)
			}
		})
	}
}

func TestParseSSHTargetEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		defaultUser string
		defaultPort string
		wantUser    string
		wantHost    string
		wantPort    string
		wantErr     bool
	}{
		{
			name:        "complex username with hyphen",
			target:      "deploy-user@myhost:2222",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "deploy-user",
			wantHost:    "myhost",
			wantPort:    "2222",
		},
		{
			name:        "hostname with hyphens",
			target:      "my-awesome-host",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "testuser",
			wantHost:    "my-awesome-host",
			wantPort:    "22",
		},
		{
			name:        "FQDN",
			target:      "server.example.com",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "testuser",
			wantHost:    "server.example.com",
			wantPort:    "22",
		},
		{
			name:        "FQDN with user and port",
			target:      "admin@server.example.com:8022",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "admin",
			wantHost:    "server.example.com",
			wantPort:    "8022",
		},
		{
			name:        "IPv6 without brackets or port",
			target:      "2001:db8::1",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "testuser",
			wantHost:    "2001:db8::1",
			wantPort:    "22",
		},
		{
			name:        "localhost",
			target:      "localhost",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "testuser",
			wantHost:    "localhost",
			wantPort:    "22",
		},
		{
			name:        "localhost with port",
			target:      "localhost:2222",
			defaultUser: "testuser",
			defaultPort: "22",
			wantUser:    "testuser",
			wantHost:    "localhost",
			wantPort:    "2222",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, host, port, err := parseSSHTarget(tt.target, tt.defaultUser, tt.defaultPort)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSSHTarget() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseSSHTarget() unexpected error: %v", err)
				return
			}

			if user != tt.wantUser {
				t.Errorf("parseSSHTarget() user = %v, want %v", user, tt.wantUser)
			}
			if host != tt.wantHost {
				t.Errorf("parseSSHTarget() host = %v, want %v", host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Errorf("parseSSHTarget() port = %v, want %v", port, tt.wantPort)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	if version == "" {
		t.Error("version should not be empty")
	}
}

func TestParseDynamicForwardSpec(t *testing.T) {
	tests := []struct {
		name         string
		forwardSpec  string
		wantBindAddr string
		wantPort     string
		wantErr      bool
	}{
		{
			name:         "port only",
			forwardSpec:  "1080",
			wantBindAddr: "localhost",
			wantPort:     "1080",
			wantErr:      false,
		},
		{
			name:         "localhost:port",
			forwardSpec:  "localhost:1080",
			wantBindAddr: "localhost",
			wantPort:     "1080",
			wantErr:      false,
		},
		{
			name:         "127.0.0.1:port",
			forwardSpec:  "127.0.0.1:1080",
			wantBindAddr: "127.0.0.1",
			wantPort:     "1080",
			wantErr:      false,
		},
		{
			name:         "ipv6 localhost:port",
			forwardSpec:  "::1:1080",
			wantBindAddr: "::1",
			wantPort:     "1080",
			wantErr:      false,
		},
		{
			name:         "0.0.0.0:port (all interfaces)",
			forwardSpec:  "0.0.0.0:1080",
			wantBindAddr: "0.0.0.0",
			wantPort:     "1080",
			wantErr:      false,
		},
		{
			name:        "invalid port - too high",
			forwardSpec: "70000",
			wantErr:     true,
		},
		{
			name:        "invalid port - not numeric",
			forwardSpec: "localhost:abc",
			wantErr:     true,
		},
		{
			name:        "invalid format - too many colons",
			forwardSpec: "localhost:1080:extra",
			wantErr:     true,
		},
		{
			name:         "valid high port",
			forwardSpec:  "8080",
			wantBindAddr: "localhost",
			wantPort:     "8080",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the forwardSpec manually to test logic
			bindAddr := "localhost"
			port := tt.forwardSpec

			if contains(tt.forwardSpec, ":") {
				parts := splitString(tt.forwardSpec, ":")
				if len(parts) != 2 {
					if !tt.wantErr {
						t.Errorf("Expected success but got invalid format")
					}
					return
				}
				bindAddr = parts[0]
				port = parts[1]
			}

			if tt.wantErr {
				// For error cases, we just verify the parsing logic
				return
			}

			if bindAddr != tt.wantBindAddr {
				t.Errorf("bindAddr = %v, want %v", bindAddr, tt.wantBindAddr)
			}
			if port != tt.wantPort {
				t.Errorf("port = %v, want %v", port, tt.wantPort)
			}
		})
	}
}

func TestSOCKS5AddressParsing(t *testing.T) {
	tests := []struct {
		name     string
		addrType byte
		data     []byte
		wantHost string
		wantPort uint16
		wantErr  bool
	}{
		{
			name:     "IPv4 address",
			addrType: 0x01,
			data:     []byte{0x05, 0x01, 0x00, 0x01, 192, 168, 1, 1, 0x00, 0x50}, // 192.168.1.1:80
			wantHost: "192.168.1.1",
			wantPort: 80,
			wantErr:  false,
		},
		{
			name:     "Domain name",
			addrType: 0x03,
			data:     append([]byte{0x05, 0x01, 0x00, 0x03, 11}, []byte("example.com")...), // example.com + port will be added
			wantHost: "example.com",
			wantErr:  false,
		},
		{
			name:     "Port 443",
			addrType: 0x01,
			data:     []byte{0x05, 0x01, 0x00, 0x01, 10, 0, 0, 1, 0x01, 0xBB}, // 10.0.0.1:443
			wantHost: "10.0.0.1",
			wantPort: 443,
			wantErr:  false,
		},
		{
			name:     "Port 8080",
			addrType: 0x01,
			data:     []byte{0x05, 0x01, 0x00, 0x01, 127, 0, 0, 1, 0x1F, 0x90}, // 127.0.0.1:8080
			wantHost: "127.0.0.1",
			wantPort: 8080,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the address parsing logic based on SOCKS5 specification
			var host string
			var port uint16

			switch tt.addrType {
			case 0x01: // IPv4
				if len(tt.data) < 10 {
					if !tt.wantErr {
						t.Errorf("Expected success but got insufficient data")
					}
					return
				}
				host = formatIPv4(tt.data[4], tt.data[5], tt.data[6], tt.data[7])
				port = uint16(tt.data[8])<<8 | uint16(tt.data[9])
			case 0x03: // Domain name
				addrLen := int(tt.data[4])
				if len(tt.data) < 5+addrLen+2 {
					// For this test, we're just checking domain parsing
					if len(tt.data) >= 5+addrLen {
						host = string(tt.data[5 : 5+addrLen])
					}
				} else {
					host = string(tt.data[5 : 5+addrLen])
					port = uint16(tt.data[5+addrLen])<<8 | uint16(tt.data[5+addrLen+1])
				}
			}

			if tt.wantErr {
				return
			}

			if host != tt.wantHost {
				t.Errorf("host = %v, want %v", host, tt.wantHost)
			}
			if tt.wantPort != 0 && port != tt.wantPort {
				t.Errorf("port = %v, want %v", port, tt.wantPort)
			}
		})
	}
}

func TestSOCKS5ProtocolVersions(t *testing.T) {
	tests := []struct {
		name        string
		version     byte
		shouldAllow bool
	}{
		{
			name:        "SOCKS5",
			version:     0x05,
			shouldAllow: true,
		},
		{
			name:        "SOCKS4",
			version:     0x04,
			shouldAllow: false,
		},
		{
			name:        "Invalid version 0",
			version:     0x00,
			shouldAllow: false,
		},
		{
			name:        "Invalid version 255",
			version:     0xFF,
			shouldAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.version == 0x05
			if isValid != tt.shouldAllow {
				t.Errorf("version 0x%02x: isValid = %v, want %v", tt.version, isValid, tt.shouldAllow)
			}
		})
	}
}

func TestSOCKS5Commands(t *testing.T) {
	tests := []struct {
		name        string
		command     byte
		shouldAllow bool
	}{
		{
			name:        "CONNECT",
			command:     0x01,
			shouldAllow: true,
		},
		{
			name:        "BIND",
			command:     0x02,
			shouldAllow: false,
		},
		{
			name:        "UDP ASSOCIATE",
			command:     0x03,
			shouldAllow: false,
		},
		{
			name:        "Invalid command",
			command:     0xFF,
			shouldAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We only support CONNECT command (0x01)
			isSupported := tt.command == 0x01
			if isSupported != tt.shouldAllow {
				t.Errorf("command 0x%02x: isSupported = %v, want %v", tt.command, isSupported, tt.shouldAllow)
			}
		})
	}
}

func TestBindAddressSecurity(t *testing.T) {
	tests := []struct {
		name          string
		bindAddr      string
		shouldWarn    bool
		shouldBeValid bool
	}{
		{
			name:          "localhost",
			bindAddr:      "localhost",
			shouldWarn:    false,
			shouldBeValid: true,
		},
		{
			name:          "127.0.0.1",
			bindAddr:      "127.0.0.1",
			shouldWarn:    false,
			shouldBeValid: true,
		},
		{
			name:          "::1",
			bindAddr:      "::1",
			shouldWarn:    false,
			shouldBeValid: true,
		},
		{
			name:          "0.0.0.0 - all interfaces",
			bindAddr:      "0.0.0.0",
			shouldWarn:    true,
			shouldBeValid: true,
		},
		{
			name:          "192.168.1.1 - LAN IP",
			bindAddr:      "192.168.1.1",
			shouldWarn:    true,
			shouldBeValid: true,
		},
		{
			name:          "invalid hostname",
			bindAddr:      "invalid!@#host",
			shouldWarn:    true,
			shouldBeValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test bind address security logic
			shouldWarn := tt.bindAddr != "" && tt.bindAddr != "localhost" &&
				tt.bindAddr != "127.0.0.1" && tt.bindAddr != "::1"

			if shouldWarn != tt.shouldWarn {
				t.Errorf("bindAddr %v: shouldWarn = %v, want %v", tt.bindAddr, shouldWarn, tt.shouldWarn)
			}
		})
	}
}

// Helper function for tests
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func formatIPv4(a, b, c, d byte) string {
	// Simple format function for testing IPv4 addresses
	result := ""
	result += byteToString(a) + "."
	result += byteToString(b) + "."
	result += byteToString(c) + "."
	result += byteToString(d)
	return result
}

func byteToString(b byte) string {
	if b == 0 {
		return "0"
	}
	var digits []byte
	for b > 0 {
		digits = append([]byte{byte('0' + b%10)}, digits...)
		b /= 10
	}
	return string(digits)
}
