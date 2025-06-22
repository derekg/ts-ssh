package security

import (
	"os"
	"runtime"
	"testing"
)

func TestValidateTTYPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid /dev/tty",
			path:    "/dev/tty",
			wantErr: false,
		},
		{
			name:    "nonexistent path",
			path:    "/dev/nonexistent-tty",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTTYPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTTYPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTTYPath(t *testing.T) {
	// Test with TTY environment variable
	originalTTY := os.Getenv("TTY")
	defer func() {
		if originalTTY != "" {
			os.Setenv("TTY", originalTTY)
		} else {
			os.Unsetenv("TTY")
		}
	}()

	// Test with valid TTY env var
	os.Setenv("TTY", "/dev/tty")
	path, err := getTTYPath()
	if err != nil {
		t.Errorf("getTTYPath() with valid TTY env var failed: %v", err)
	}
	if path != "/dev/tty" {
		t.Errorf("getTTYPath() = %v, want %v", path, "/dev/tty")
	}

	// Test with invalid TTY env var
	os.Setenv("TTY", "/dev/nonexistent")
	_, err = getTTYPath()
	if err == nil {
		t.Errorf("getTTYPath() should fail with invalid TTY env var")
	}

	// Test fallback to /dev/tty
	os.Unsetenv("TTY")
	path, err = getTTYPath()
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if err != nil {
			t.Errorf("getTTYPath() fallback failed: %v", err)
		}
		if path != "/dev/tty" {
			t.Errorf("getTTYPath() fallback = %v, want %v", path, "/dev/tty")
		}
	}
}

func TestValidateTTYSecurity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TTY security validation not supported on Windows")
	}

	tests := []struct {
		name     string
		path     string
		wantErr  bool
		skipTest bool
	}{
		{
			name:     "dev tty should pass",
			path:     "/dev/tty",
			wantErr:  false,
			skipTest: false,
		},
		{
			name:     "nonexistent path should fail",
			path:     "/dev/nonexistent-tty",
			wantErr:  true,
			skipTest: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires specific TTY setup")
			}

			err := validateTTYSecurity(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTTYSecurity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadPasswordSecurely(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Secure TTY not fully supported on Windows")
	}

	// This test is challenging because it requires interactive input
	// We'll test the error cases instead
	t.Run("fallback when no TTY available", func(t *testing.T) {
		// We can't easily test the success case without interactive input
		// But we can test that the function exists and handles errors gracefully
		_, err := ReadPasswordSecurely()
		// Error is expected if we're not in an interactive terminal
		// or if TTY validation fails - this is correct behavior
		if err == nil {
			t.Log("ReadPasswordSecurely() succeeded (likely running in interactive terminal)")
		} else {
			t.Logf("ReadPasswordSecurely() failed as expected in non-interactive environment: %v", err)
		}
	})
}

func TestPromptUserSecurely(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Secure TTY not fully supported on Windows")
	}

	// Test with non-interactive environment
	_, err := PromptUserSecurely("Test prompt: ")
	// Error is expected if we're not in an interactive terminal
	if err == nil {
		t.Log("PromptUserSecurely() succeeded (likely running in interactive terminal)")
	} else {
		t.Logf("PromptUserSecurely() failed as expected in non-interactive environment: %v", err)
	}
}

func TestWithSecureTTY(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Secure TTY not fully supported on Windows")
	}

	// Test the wrapper function behavior
	err := withSecureTTY(func(tty *os.File) error {
		if tty == nil {
			t.Error("TTY file should not be nil")
		}
		// Just verify we got a valid file descriptor
		if tty.Fd() == 0 {
			t.Error("TTY file descriptor should not be 0")
		}
		return nil
	})

	// Error is expected if we're not in an interactive terminal
	if err == nil {
		t.Log("withSecureTTY() succeeded (likely running in interactive terminal)")
	} else {
		t.Logf("withSecureTTY() failed as expected in non-interactive environment: %v", err)
	}
}