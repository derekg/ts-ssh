package platform

import (
	"os"
	"runtime"
	"testing"
)

func TestMaskProcessTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{
			name:  "default title",
			title: "",
		},
		{
			name:  "custom title",
			title: "custom-secure-title",
		},
		{
			name:  "long title that should be truncated",
			title: "this-is-a-very-long-title-that-exceeds-the-linux-process-name-limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that function doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("maskProcessTitle() panicked: %v", r)
				}
			}()

			maskProcessTitle(tt.title)
			// Function should complete without error
			// On platforms that don't support process title masking, it should be a no-op
		})
	}
}

func TestMaskProcessTitleLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}

	tests := []struct {
		name  string
		title string
	}{
		{
			name:  "normal title",
			title: "ts-ssh-secure",
		},
		{
			name:  "empty title",
			title: "",
		},
		{
			name:  "exactly 15 chars",
			title: "exactly15charsX", // 15 characters
		},
		{
			name:  "long title (should be truncated)",
			title: "this-title-is-way-too-long-for-linux-process-names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that function doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("maskProcessTitlePlatform() panicked: %v", r)
				}
			}()

			maskProcessTitlePlatform(tt.title)
			// Function should complete without error
		})
	}
}

func TestMaskProcessTitleDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin-specific test")
	}

	// Test the platform-specific function through the main interface
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("maskProcessTitlePlatform() panicked: %v", r)
		}
	}()

	maskProcessTitlePlatform("test-title")
}

func TestSetSecureEnvironment(t *testing.T) {
	// Save original environment
	originalVars := map[string]string{
		"SSH_AUTH_SOCK": os.Getenv("SSH_AUTH_SOCK"),
		"SSH_AGENT_PID": os.Getenv("SSH_AGENT_PID"),
		"DISPLAY":       os.Getenv("DISPLAY"),
	}

	// Restore environment after test
	defer func() {
		for varName, value := range originalVars {
			if value != "" {
				os.Setenv(varName, value)
			} else {
				os.Unsetenv(varName)
			}
		}
	}()

	// Set some test values
	os.Setenv("SSH_AUTH_SOCK", "/tmp/test-ssh-auth-sock")
	os.Setenv("SSH_AGENT_PID", "12345")
	os.Setenv("DISPLAY", ":0")

	// Call setSecureEnvironment
	setSecureEnvironment()

	// Verify sensitive variables were cleared
	sensitiveVars := []string{"SSH_AUTH_SOCK", "SSH_AGENT_PID", "DISPLAY"}
	for _, varName := range sensitiveVars {
		if value := os.Getenv(varName); value != "" {
			t.Errorf("Environment variable %s was not cleared: %s", varName, value)
		}
	}
}

func TestHideCredentialsInProcessList(t *testing.T) {
	// Save original environment for restoration
	originalVars := map[string]string{
		"SSH_AUTH_SOCK": os.Getenv("SSH_AUTH_SOCK"),
		"SSH_AGENT_PID": os.Getenv("SSH_AGENT_PID"),
		"DISPLAY":       os.Getenv("DISPLAY"),
	}

	defer func() {
		for varName, value := range originalVars {
			if value != "" {
				os.Setenv(varName, value)
			} else {
				os.Unsetenv(varName)
			}
		}
	}()

	// Set some test values
	os.Setenv("SSH_AUTH_SOCK", "/tmp/test-auth-sock")
	os.Setenv("SSH_AGENT_PID", "9999")
	os.Setenv("DISPLAY", ":1")

	// Test that function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("hideCredentialsInProcessList() panicked: %v", r)
		}
	}()

	hideCredentialsInProcessList()

	// Verify sensitive variables were cleared
	sensitiveVars := []string{"SSH_AUTH_SOCK", "SSH_AGENT_PID", "DISPLAY"}
	for _, varName := range sensitiveVars {
		if value := os.Getenv(varName); value != "" {
			t.Errorf("Environment variable %s was not cleared by hideCredentialsInProcessList: %s", varName, value)
		}
	}
}

func TestProcessSecurityIntegration(t *testing.T) {
	// Integration test to verify all process security measures work together
	
	// Save original environment
	originalEnv := map[string]string{
		"SSH_AUTH_SOCK": os.Getenv("SSH_AUTH_SOCK"),
		"SSH_AGENT_PID": os.Getenv("SSH_AGENT_PID"),
		"DISPLAY":       os.Getenv("DISPLAY"),
	}

	defer func() {
		for varName, value := range originalEnv {
			if value != "" {
				os.Setenv(varName, value)
			} else {
				os.Unsetenv(varName)
			}
		}
	}()

	// Set up test environment with sensitive data
	os.Setenv("SSH_AUTH_SOCK", "/tmp/sensitive-auth-sock")
	os.Setenv("SSH_AGENT_PID", "sensitive-pid-12345")
	os.Setenv("DISPLAY", ":0.0")

	// Apply comprehensive process security
	hideCredentialsInProcessList()

	// Verify all sensitive data was cleared
	for varName := range originalEnv {
		if value := os.Getenv(varName); value != "" {
			t.Errorf("Sensitive environment variable %s still contains data after security measures: %s", varName, value)
		}
	}

	// Test process title masking with different titles
	testTitles := []string{
		"",
		"ts-ssh-test",
		"sensitive-connection-data-should-be-hidden",
	}

	for _, title := range testTitles {
		t.Run("mask_title_"+title, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Process title masking panicked with title %q: %v", title, r)
				}
			}()

			maskProcessTitle(title)
		})
	}
}

func TestProcessSecurityCrossPlatform(t *testing.T) {
	// Test that process security works across different platforms
	t.Run("current_platform", func(t *testing.T) {
		switch runtime.GOOS {
		case "linux":
			t.Log("Testing on Linux - full process security available")
		case "darwin":
			t.Log("Testing on macOS - limited process security")
		case "windows":
			t.Log("Testing on Windows - limited process security")
		default:
			t.Log("Testing on", runtime.GOOS, "- limited process security")
		}

		// All platforms should handle the function calls without panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Cross-platform process security failed: %v", r)
			}
		}()

		hideCredentialsInProcessList()
	})
}

func BenchmarkMaskProcessTitle(b *testing.B) {
	for i := 0; i < b.N; i++ {
		maskProcessTitle("benchmark-test-title")
	}
}

func BenchmarkSetSecureEnvironment(b *testing.B) {
	// Set up environment for benchmark
	os.Setenv("SSH_AUTH_SOCK", "/tmp/bench-test")
	os.Setenv("SSH_AGENT_PID", "12345")
	os.Setenv("DISPLAY", ":0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		setSecureEnvironment()
		
		// Reset for next iteration
		os.Setenv("SSH_AUTH_SOCK", "/tmp/bench-test")
		os.Setenv("SSH_AGENT_PID", "12345")
		os.Setenv("DISPLAY", ":0")
	}
}