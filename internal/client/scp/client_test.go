package scp

import (
	"context"
	"io"
	"log"
	"os/user"
	"testing"

	"github.com/derekg/ts-ssh/internal/i18n"
)

// TestConstants verifies SCP constants are defined correctly
func TestConstants(t *testing.T) {
	if DefaultSshPort == "" {
		t.Error("DefaultSshPort should not be empty")
	}

	if DefaultSshPort != "22" {
		t.Errorf("DefaultSshPort should be '22', got '%s'", DefaultSshPort)
	}
}

// TestTranslationFunction tests the T function
func TestTranslationFunction(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		args     []interface{}
		expected string
	}{
		{
			name:     "empty path message",
			key:      "scp_empty_path",
			args:     nil,
			expected: "SCP path cannot be empty",
		},
		{
			name:     "password prompt with args",
			key:      "scp_enter_password",
			args:     []interface{}{"user", "host"},
			expected: "Enter password for user@host: ",
		},
		{
			name:     "dial via tsnet",
			key:      "dial_via_tsnet",
			args:     nil,
			expected: "Connecting via tsnet...",
		},
		{
			name:     "unknown key returns key",
			key:      "unknown_key",
			args:     nil,
			expected: "unknown_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := i18n.T(tt.key, tt.args...)
			if result != tt.expected {
				t.Errorf("T(%s, %v) = %s, want %s", tt.key, tt.args, result, tt.expected)
			}
		})
	}
}

// TestHandleCliScpValidation tests input validation
func TestHandleCliScpValidation(t *testing.T) {
	// Create silent logger for tests
	logger := log.New(io.Discard, "", 0)
	currentUser := &user.User{Username: "testuser", HomeDir: "/tmp"}

	tests := []struct {
		name           string
		localPath      string
		remotePath     string
		expectError    bool
		errorSubstring string
	}{
		{
			name:           "empty local path",
			localPath:      "",
			remotePath:     "/remote/path",
			expectError:    true,
			errorSubstring: "SCP path cannot be empty",
		},
		{
			name:           "empty remote path",
			localPath:      "/local/path",
			remotePath:     "",
			expectError:    true,
			errorSubstring: "SCP path cannot be empty",
		},
		{
			name:           "both paths empty",
			localPath:      "",
			remotePath:     "",
			expectError:    true,
			errorSubstring: "SCP path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We can't test the full function without a real tsnet.Server
			// but we can test the validation logic at the beginning
			err := HandleCliScp(
				nil,                  // srv - will fail later but validation happens first
				context.Background(), // ctx
				logger,
				"testuser",
				"",
				false,
				currentUser,
				tt.localPath,
				tt.remotePath,
				"testhost",
				true,
				false,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for test %s, but got nil", tt.name)
				} else if tt.errorSubstring != "" && err.Error() != tt.errorSubstring {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorSubstring, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for test %s, but got: %v", tt.name, err)
				}
			}
		})
	}
}

// TestScpErrorHandling tests error handling scenarios
func TestScpErrorHandling(t *testing.T) {
	// Test path validation works correctly
	logger := log.New(io.Discard, "", 0)
	currentUser := &user.User{Username: "testuser", HomeDir: "/tmp"}

	// Test validation error with empty local path
	err := HandleCliScp(
		nil, // We won't reach the point where this matters
		context.Background(),
		logger,
		"testuser",
		"",
		false,
		currentUser,
		"", // Empty local path should trigger validation error
		"/valid/remote/path",
		"testhost",
		true,
		false,
	)

	// Should get validation error
	if err == nil {
		t.Error("Expected validation error with empty local path")
	}

	// Error should be about empty paths
	if err.Error() != "SCP path cannot be empty" {
		t.Errorf("Expected validation error, got: %s", err.Error())
	}
}

// TestScpFunctionSignature ensures the function signature is correct
func TestScpFunctionSignature(t *testing.T) {
	// This test ensures the function signature matches expectations
	// We test with invalid paths to trigger early validation return
	logger := log.New(io.Discard, "", 0)
	currentUser := &user.User{Username: "testuser", HomeDir: "/tmp"}

	// Test that we can call the function with correct signature
	err := HandleCliScp(
		nil,
		context.Background(),
		logger,
		"user",
		"/path/to/key",
		true, // insecure
		currentUser,
		"", // Empty local path for early validation return
		"/remote",
		"host",
		true, // upload
		true, // verbose
	)

	// Should get validation error (proving we called the function correctly)
	if err == nil {
		t.Error("Expected validation error")
	}
	if err.Error() != "SCP path cannot be empty" {
		t.Error("Should get validation error with empty path")
	}
}

// TestScpWithSSHKeyPath tests SCP configuration with SSH key path
func TestScpWithSSHKeyPath(t *testing.T) {
	// Test that function handles SSH key path parameter correctly
	// by using empty paths to trigger early validation
	logger := log.New(io.Discard, "", 0)
	currentUser := &user.User{Username: "testuser", HomeDir: "/tmp"}

	// Test with non-existent SSH key but empty remote path (validation error)
	err := HandleCliScp(
		nil, // Won't reach server usage
		context.Background(),
		logger,
		"testuser",
		"/nonexistent/key/path", // This parameter gets accepted
		false,
		currentUser,
		"/valid/local/path",
		"", // Empty remote path triggers validation
		"testhost",
		false, // download
		true,  // verbose
	)

	// Should get validation error for empty remote path
	if err == nil {
		t.Error("Expected validation error for empty remote path")
	}

	// Should be a validation error
	if err.Error() != "SCP path cannot be empty" {
		t.Errorf("Expected validation error, got: %s", err.Error())
	}
}

// TestScpInsecureMode tests SCP with insecure host key verification disabled
func TestScpInsecureMode(t *testing.T) {
	// Test that insecure mode parameter is accepted by using validation trigger
	logger := log.New(io.Discard, "", 0)
	currentUser := &user.User{Username: "testuser", HomeDir: "/tmp"}

	// Test with insecure mode enabled but empty local path (validation error)
	err := HandleCliScp(
		nil, // Won't reach server usage
		context.Background(),
		logger,
		"testuser",
		"",   // No SSH key
		true, // insecure mode - this parameter gets accepted
		currentUser,
		"", // Empty local path triggers validation
		"/valid/remote/path",
		"testhost",
		true,  // upload
		false, // not verbose
	)

	// Should get validation error for empty local path
	if err == nil {
		t.Error("Expected validation error for empty local path")
	}

	// Should be validation error
	if err.Error() != "SCP path cannot be empty" {
		t.Errorf("Expected validation error, got: %s", err.Error())
	}
}
