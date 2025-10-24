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
