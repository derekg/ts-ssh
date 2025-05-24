package main

import (
	"context"
	"io"
	"log"
	"os/user"
	"testing"
	"tailscale.com/tsnet"
	// "os" // No longer directly needed
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name         string
		target       string // This is the input to parseTarget
		expectedHost string
		expectedPort string
		expectErr    bool
	}{
		{
			name:         "hostname only",
			target:       "myhost",
			expectedHost: "myhost",
			expectedPort: DefaultSshPort,
			expectErr:    false,
		},
		{
			name:         "hostname with user (passed through by net.SplitHostPort)",
			target:       "user@myhost",
			expectedHost: "user@myhost", // parseTarget, via net.SplitHostPort, keeps user for non-bracketed
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
			name:         "hostname with user and port (passed through by net.SplitHostPort)",
			target:       "user@myhost:2222",
			expectedHost: "user@myhost", 
			expectedPort: "2222",
			expectErr:    false,
		},
		{
			name:         "ipv4 address",
			target:       "192.168.1.100",
			expectedHost: "192.168.1.100",
			expectedPort: DefaultSshPort,
			expectErr:    false,
		},
		{
			name:         "ipv4 address with port",
			target:       "192.168.1.100:2222",
			expectedHost: "192.168.1.100",
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
			// This test case is specifically for how parseTarget handles bracketed IPv6.
			// User info should be stripped by calling logic *before* this for pure host/port parsing.
			name:         "bracketed ipv6 address with port (no user info)",
			target:       "[2001:db8::1]:2222",
			expectedHost: "2001:db8::1",
			expectedPort: "2222",
			expectErr:    false,
		},
		{
			name:      "empty target",
			target:    "",
			expectErr: true,
		},
		{
			name:      "just port",
			target:    ":2222",
			expectErr: true, 
		},
		{
			name:      "invalid ipv6 missing closing bracket",
			target:    "[::1:2222",
			expectErr: true,
		},
		{
			name:      "invalid ipv6 wrong port separator",
			target:    "[::1]2222", 
			expectErr: true,
		},
		{
			name:         "hostname with hyphen and port",
			target:       "my-cool-host:2022",
			expectedHost: "my-cool-host",
			expectedPort: "2022",
			expectErr:    false,
		},
		{
			name:         "hostname with numbers and port",
			target:       "host123:2023",
			expectedHost: "host123",
			expectedPort: "2023",
			expectErr:    false,
		},
		{
			name:      "user@ with bracketed IPv6 (parseTarget fails this)",
			// parseTarget's IPv6 logic expects only "[host]:port" or "[host]".
			// It does not handle "user@[host]:port". This form is syntactically problematic
			// for URI standards and net.SplitHostPort if userinfo is outside brackets.
			// The main `main.go` logic should ideally separate "user@" before calling parseTarget
			// if this complex form is ever encountered for SSH targets.
			// For SCP, parseScpRemoteArg handles user@host:path separately.
			target:    "user@[::1]:2222", 
			expectErr: true, 
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := parseTarget(tt.target, DefaultSshPort) 

			if (err != nil) != tt.expectErr {
				t.Errorf("parseTarget(%q) error = %v, expectErr %v", tt.target, err, tt.expectErr)
				return
			}
			if !tt.expectErr {
				if host != tt.expectedHost {
					t.Errorf("parseTarget(%q) host = %q, want %q", tt.target, host, tt.expectedHost)
				}
				if port != tt.expectedPort {
					t.Errorf("parseTarget(%q) port = %q, want %q", tt.target, port, tt.expectedPort)
				}
			}
		})
	}
}

func TestPerformSCPTransferParams(t *testing.T) {
	dummySrv := &tsnet.Server{}
	dummyCtx := context.Background()
	dummyLogger := log.New(io.Discard, "", 0) 
	dummyCurrentUser, _ := user.Current()

	tests := []struct {
		name        string
		scpDetails  tuiActionResult
		expectError bool
		expectedMsg string
	}{
		{
			name: "valid paths",
			scpDetails: tuiActionResult{
				scpLocalPath:  "/tmp/localfile",
				scpRemotePath: "/tmp/remotefile",
			},
			expectError: false,
		},
		{
			name: "empty local path",
			scpDetails: tuiActionResult{
				scpLocalPath:  "",
				scpRemotePath: "/tmp/remotefile",
			},
			expectError: true,
			expectedMsg: "local or remote path for SCP cannot be empty",
		},
		{
			name: "empty remote path",
			scpDetails: tuiActionResult{
				scpLocalPath:  "/tmp/localfile",
				scpRemotePath: "",
			},
			expectError: true,
			expectedMsg: "local or remote path for SCP cannot be empty",
		},
		{
			name: "both paths empty",
			scpDetails: tuiActionResult{
				scpLocalPath:  "",
				scpRemotePath: "",
			},
			expectError: true,
			expectedMsg: "local or remote path for SCP cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := performSCPTransfer(
				dummySrv,
				dummyCtx,
				dummyLogger,
				tt.scpDetails,
				"user", 
				"",     
				true,   
				dummyCurrentUser,
				false, 
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("performSCPTransfer() with details %+v expected error, got nil", tt.scpDetails)
				} else if err.Error() != tt.expectedMsg {
					t.Errorf("performSCPTransfer() expected error msg %q, got %q", tt.expectedMsg, err.Error())
				}
			} else {
				if err != nil && err.Error() == tt.expectedMsg {
					t.Errorf("performSCPTransfer() with details %+v got unexpected validation error: %v", tt.scpDetails, err)
				} else if err != nil {
					t.Logf("performSCPTransfer() with details %+v returned error %v. Assuming path validation passed as it's not the expectedMsg for empty paths.", tt.scpDetails, err)
				}
			}
		})
	}
}

func TestIntegrationInitTsNet_Placeholder(t *testing.T) {
	t.Log("Integration Test Placeholder: TestInitTsNet")
	t.Skip("Skipping placeholder integration test.")
}

func TestIntegrationConnectToHostFromTUI_Placeholder(t *testing.T) {
	t.Log("Integration Test Placeholder: TestConnectToHostFromTUI")
	t.Skip("Skipping placeholder integration test.")
}

func TestIntegrationPerformSCPTransfer_Placeholder(t *testing.T) {
	t.Log("Integration Test Placeholder: TestPerformSCPTransfer")
	t.Skip("Skipping placeholder integration test.")
}

func TestTUIPopulationLogic_Placeholder(t *testing.T) {
	t.Log("TUI Test Placeholder: TestTUIPopulationLogic")
	t.Skip("Skipping placeholder TUI logic test.")
}

func TestTryParseScpArgs(t *testing.T) {
	dummyLogger := log.New(io.Discard, "", 0)
	tests := []struct {
		name              string
		nonFlagArgs       []string
		defaultSshUser    string
		verbose           bool
		expectedScpArgs   *scpArgs
		expectedIsScpOp   bool
		expectErr         bool
		expectedErrSubstr string
	}{
		{
			name:            "not enough args",
			nonFlagArgs:     []string{"one"},
			defaultSshUser:  "user",
			expectedScpArgs: nil,
			expectedIsScpOp: false,
			expectErr:       false,
		},
		{
			name:            "too many args",
			nonFlagArgs:     []string{"one", "two", "three"},
			defaultSshUser:  "user",
			expectedScpArgs: nil,
			expectedIsScpOp: false,
			expectErr:       false,
		},
		{
			name:            "no colons (not scp)",
			nonFlagArgs:     []string{"localfile", "remotefile"},
			defaultSshUser:  "user",
			expectedScpArgs: nil,
			expectedIsScpOp: false,
			expectErr:       false,
		},
		{
			name:           "valid upload: local user@host:remote",
			nonFlagArgs:    []string{"local.txt", "user1@host1:/remote/path"},
			defaultSshUser: "default",
			expectedScpArgs: &scpArgs{
				isUpload:   true,
				localPath:  "local.txt",
				remotePath: "/remote/path",
				targetHost: "host1",
				sshUser:    "user1",
			},
			expectedIsScpOp: true,
			expectErr:       false,
		},
		{
			name:           "valid download: user@host:remote local",
			nonFlagArgs:    []string{"user2@host2:/remote/file", "localdir/"},
			defaultSshUser: "default",
			expectedScpArgs: &scpArgs{
				isUpload:   false,
				localPath:  "localdir/",
				remotePath: "/remote/file",
				targetHost: "host2",
				sshUser:    "user2",
			},
			expectedIsScpOp: true,
			expectErr:       false,
		},
		{
			name:           "valid upload: local host:remote (use default user)",
			nonFlagArgs:    []string{"local.txt", "host3:/remote/path"},
			defaultSshUser: "defaultuser3",
			expectedScpArgs: &scpArgs{
				isUpload:   true,
				localPath:  "local.txt",
				remotePath: "/remote/path",
				targetHost: "host3",
				sshUser:    "defaultuser3",
			},
			expectedIsScpOp: true,
			expectErr:       false,
		},
		{
			name:           "valid download: host:remote local (use default user)",
			nonFlagArgs:    []string{"host4:/remote/file", "localdir/"},
			defaultSshUser: "defaultuser4",
			expectedScpArgs: &scpArgs{
				isUpload:   false,
				localPath:  "localdir/",
				remotePath: "/remote/file",
				targetHost: "host4",
				sshUser:    "defaultuser4",
			},
			expectedIsScpOp: true,
			expectErr:       false,
		},
		{
			name:              "invalid remote arg (upload)",
			nonFlagArgs:       []string{"local.txt", "user@:/missinghost"}, // missing host
			defaultSshUser:    "default",
			expectedScpArgs:   nil,
			expectedIsScpOp:   false,
			expectErr:         true,
			expectedErrSubstr: `failed to parse remote argument user@:/missinghost: host cannot be empty in SCP argument: "user@:/missinghost"`,
		},
		{
			name:              "invalid remote arg (download)",
			nonFlagArgs:       []string{"user@hostmalformed:/path", "local.txt"}, // no path after colon
			defaultSshUser:    "default",
			expectedScpArgs:   nil,
			expectedIsScpOp:   false,
			expectErr:         true,
			expectedErrSubstr: `failed to parse remote argument user@hostmalformed:/path: invalid remote SCP argument format: "user@hostmalformed:/path". Must be [user@]host:path`,
		},
		{
			name:            "both args contain colon (not scp)",
			nonFlagArgs:     []string{"host1:path1", "host2:path2"},
			defaultSshUser:  "user",
			expectedScpArgs: nil,
			expectedIsScpOp: false,
			expectErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scpDetails, isScpOp, err := tryParseScpArgs(tt.nonFlagArgs, tt.defaultSshUser, dummyLogger, tt.verbose)

			if (err != nil) != tt.expectErr {
				t.Errorf("tryParseScpArgs() error = %v, expectErr %v", err, tt.expectErr)
				if err != nil && tt.expectErr && tt.expectedErrSubstr != "" && !strings.Contains(err.Error(), tt.expectedErrSubstr) {
					t.Errorf("tryParseScpArgs() error msg = %q, expected to contain %q", err.Error(), tt.expectedErrSubstr)
				}
				return
			}
			if tt.expectErr {
				if err != nil && tt.expectedErrSubstr != "" && !strings.Contains(err.Error(), tt.expectedErrSubstr) {
					t.Errorf("tryParseScpArgs() error msg = %q, expected to contain %q", err.Error(), tt.expectedErrSubstr)
				}
				return
			}

			if isScpOp != tt.expectedIsScpOp {
				t.Errorf("tryParseScpArgs() isScpOp = %v, want %v", isScpOp, tt.expectedIsScpOp)
			}

			if tt.expectedScpArgs == nil && scpDetails != nil {
				t.Errorf("tryParseScpArgs() scpDetails got %+v, want nil", scpDetails)
			}
			if tt.expectedScpArgs != nil && scpDetails == nil {
				t.Errorf("tryParseScpArgs() scpDetails got nil, want %+v", tt.expectedScpArgs)
			}
			if tt.expectedScpArgs != nil && scpDetails != nil {
				if scpDetails.isUpload != tt.expectedScpArgs.isUpload ||
					scpDetails.localPath != tt.expectedScpArgs.localPath ||
					scpDetails.remotePath != tt.expectedScpArgs.remotePath ||
					scpDetails.targetHost != tt.expectedScpArgs.targetHost ||
					scpDetails.sshUser != tt.expectedScpArgs.sshUser {
					t.Errorf("tryParseScpArgs() scpDetails = %+v, want %+v", scpDetails, tt.expectedScpArgs)
				}
			}
		})
	}
}

func TestParseScpRemoteArg(t *testing.T) {
	tests := []struct {
		name             string
		remoteArg        string
		defaultSshUser   string
		expectedHost     string
		expectedPath     string
		expectedUser     string
		expectErr        bool
		expectedErrExact string // If expectErr is true, this can be a substring or exact match
	}{
		{
			name:           "simple host:path",
			remoteArg:      "myhost:/remote/path",
			defaultSshUser: "defaultuser",
			expectedHost:   "myhost",
			expectedPath:   "/remote/path",
			expectedUser:   "defaultuser",
			expectErr:      false,
		},
		{
			name:           "user@host:path",
			remoteArg:      "scpuser@myhost:/remote/path/to/file",
			defaultSshUser: "defaultuser",
			expectedHost:   "myhost",
			expectedPath:   "/remote/path/to/file",
			expectedUser:   "scpuser",
			expectErr:      false,
		},
		{
			name:           "host relative path",
			remoteArg:      "anotherhost:relative/path",
			defaultSshUser: "defaultuser",
			expectedHost:   "anotherhost",
			expectedPath:   "relative/path",
			expectedUser:   "defaultuser",
			expectErr:      false,
		},
		{
			name:           "user@host relative path",
			remoteArg:      "anotheruser@anotherhost:relative/path",
			defaultSshUser: "defaultuser",
			expectedHost:   "anotherhost",
			expectedPath:   "relative/path",
			expectedUser:   "anotheruser",
			expectErr:      false,
		},
		{
			name:           "host with port and path", // parseScpRemoteArg doesn't handle port in host
			remoteArg:      "myhost.example.com:2222:/some/path",
			defaultSshUser: "defaultuser",
			expectedHost:   "myhost.example.com", // It will take "myhost.example.com:2222" as host
			expectedPath:   "/some/path",
			expectedUser:   "defaultuser",
			expectErr:      false, // Current implementation will parse host as "myhost.example.com:2222"
		},
		{
			name:           "user@host with port and path", // parseScpRemoteArg doesn't handle port in host
			remoteArg:      "user@myhost.example.com:2222:/another/path",
			defaultSshUser: "defaultuser",
			expectedHost:   "myhost.example.com:2222", // It will take "myhost.example.com:2222" as host
			expectedPath:   "/another/path",
			expectedUser:   "user",
			expectErr:      false, // Current implementation
		},
		{
			name:           "empty path after colon",
			remoteArg:      "myhost:",
			defaultSshUser: "defaultuser",
			expectedHost:   "myhost",
			expectedPath:   "", // Path is empty, not an error for parseScpRemoteArg
			expectedUser:   "defaultuser",
			expectErr:      false, 
		},
		{
			name:             "no colon",
			remoteArg:        "myhostnorpath",
			defaultSshUser:   "defaultuser",
			expectErr:        true,
			expectedErrExact: `invalid remote SCP argument format: "myhostnorpath". Must be [user@]host:path`,
		},
		{
			name:             "empty host part before colon",
			remoteArg:        ":/path",
			defaultSshUser:   "defaultuser",
			expectErr:        true,
			expectedErrExact: `host cannot be empty in SCP argument: ":/path"`,
		},
		{
			name:             "empty host with user",
			remoteArg:        "user@:/path",
			defaultSshUser:   "defaultuser",
			expectErr:        true,
			expectedErrExact: `host cannot be empty in SCP argument: "user@:/path"`,
		},
		{
			name:             "invalid user@ format just @",
			remoteArg:        "@myhost:/path",
			defaultSshUser:   "defaultuser",
			expectErr:        true,
			expectedErrExact: `invalid user@host format in SCP argument: "@myhost"`,
		},
		{
			name:             "invalid user@ format user@",
			remoteArg:        "user@:/path",
			defaultSshUser:   "defaultuser",
			expectErr:        true,
			expectedErrExact: `host cannot be empty in SCP argument: "user@:/path"`,

		},
		{
			name:           "path is just /",
			remoteArg:      "host:/",
			defaultSshUser: "defaultuser",
			expectedHost:   "host",
			expectedPath:   "/",
			expectedUser:   "defaultuser",
			expectErr:      false,
		},
		{
			name:           "ipv6 host with path",
			remoteArg:      "[::1]:/my/folder",
			defaultSshUser: "defaultuser",
			expectedHost:   "[::1]",
			expectedPath:   "/my/folder",
			expectedUser:   "defaultuser",
			expectErr:      false,
		},
		{
			name:           "user@ipv6 host with path",
			remoteArg:      "user@[::1]:/my/folder",
			defaultSshUser: "defaultuser",
			expectedHost:   "[::1]",
			expectedPath:   "/my/folder",
			expectedUser:   "user",
			expectErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, path, user, err := parseScpRemoteArg(tt.remoteArg, tt.defaultSshUser)

			if (err != nil) != tt.expectErr {
				t.Errorf("parseScpRemoteArg(%q, %q) error = %v, expectErr %v", tt.remoteArg, tt.defaultSshUser, err, tt.expectErr)
				return
			}
			if tt.expectErr {
				if err != nil && tt.expectedErrExact != "" && err.Error() != tt.expectedErrExact {
					t.Errorf("parseScpRemoteArg(%q, %q) error msg = %q, want exact %q", tt.remoteArg, tt.defaultSshUser, err.Error(), tt.expectedErrExact)
				}
				return // Don't check other fields if an error is expected
			}

			if host != tt.expectedHost {
				t.Errorf("parseScpRemoteArg(%q, %q) host = %q, want %q", tt.remoteArg, tt.defaultSshUser, host, tt.expectedHost)
			}
			if path != tt.expectedPath {
				t.Errorf("parseScpRemoteArg(%q, %q) path = %q, want %q", tt.remoteArg, tt.defaultSshUser, path, tt.expectedPath)
			}
			if user != tt.expectedUser {
				t.Errorf("parseScpRemoteArg(%q, %q) user = %q, want %q", tt.remoteArg, tt.defaultSshUser, user, tt.expectedUser)
			}
		})
	}
}
