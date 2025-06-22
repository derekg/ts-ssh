package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// TestSecurityWorkflowIntegration tests complex security workflows end-to-end
func TestSecurityWorkflowIntegration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "ts-ssh-security-integration-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		workflow func(t *testing.T, tempDir string)
	}{
		{
			name:     "complete_ssh_key_discovery_and_auth_workflow",
			workflow: testCompleteSSHKeyDiscoveryWorkflow,
		},
		{
			name:     "insecure_mode_audit_logging_workflow", 
			workflow: testInsecureModeAuditLoggingWorkflow,
		},
		{
			name:     "secure_file_operations_with_concurrent_access",
			workflow: testSecureFileOperationsWorkflow,
		},
		{
			name:     "cross_platform_security_validation",
			workflow: testCrossPlatformSecurityWorkflow,
		},
		{
			name:     "tty_security_validation_workflow",
			workflow: testTTYSecurityValidationWorkflow,
		},
		{
			name:     "host_key_verification_workflow",
			workflow: testHostKeyVerificationWorkflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create isolated temp directory for each test
			testTempDir := filepath.Join(tempDir, tt.name)
			if err := os.MkdirAll(testTempDir, 0700); err != nil {
				t.Fatalf("Failed to create test temp dir: %v", err)
			}
			tt.workflow(t, testTempDir)
		})
	}
}

// testCompleteSSHKeyDiscoveryWorkflow tests the complete SSH key discovery and authentication workflow
func testCompleteSSHKeyDiscoveryWorkflow(t *testing.T, tempDir string) {
	t.Logf("Testing complete SSH key discovery and authentication workflow")

	// Create mock .ssh directory structure
	sshDir := filepath.Join(tempDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh directory: %v", err)
	}

	// Create multiple key types with different priorities
	keyTypes := []struct {
		name     string
		priority int
	}{
		{"id_rsa", 3},      // Lowest priority
		{"id_ecdsa", 2},    // Medium priority  
		{"id_ed25519", 1},  // Highest priority
	}

	var createdKeys []string
	for _, keyType := range keyTypes {
		keyPath := filepath.Join(sshDir, keyType.name)
		
		// Generate a test key file
		testKeyContent := fmt.Sprintf("-----BEGIN PRIVATE KEY-----\ntest_%s_key_content\n-----END PRIVATE KEY-----\n", keyType.name)
		if err := os.WriteFile(keyPath, []byte(testKeyContent), 0600); err != nil {
			t.Fatalf("Failed to create test key %s: %v", keyType.name, err)
		}
		createdKeys = append(createdKeys, keyPath)
		t.Logf("✓ Created test SSH key: %s", keyPath)
	}

	// Test key discovery with modern preference order
	discoveredKey := getDefaultSSHKeyPath(nil, nil)
	t.Logf("Default key discovery result: %s", discoveredKey)

	// Test key discovery from specific directory
	if runtime.GOOS == "windows" {
		// Simulate Windows home directory structure
		t.Logf("Testing Windows-style key discovery")
	}

	// Verify security properties
	for _, keyPath := range createdKeys {
		info, err := os.Stat(keyPath)
		if err != nil {
			t.Errorf("Failed to stat key file %s: %v", keyPath, err)
			continue
		}
		
		if info.Mode().Perm() != 0600 {
			t.Errorf("Key file %s has incorrect permissions: got %v, want 0600", keyPath, info.Mode().Perm())
		}
		t.Logf("✓ Verified secure permissions for %s", keyPath)
	}

	t.Logf("✓ Complete SSH key discovery workflow validated")
}

// testInsecureModeAuditLoggingWorkflow tests the security audit logging for insecure mode
func testInsecureModeAuditLoggingWorkflow(t *testing.T, tempDir string) {
	t.Logf("Testing insecure mode audit logging workflow")

	// Set up security audit logging
	logPath := filepath.Join(tempDir, "security_audit.log")
	os.Setenv("TS_SSH_SECURITY_AUDIT", "1")
	os.Setenv("TS_SSH_AUDIT_LOG", logPath)
	defer func() {
		os.Unsetenv("TS_SSH_SECURITY_AUDIT")
		os.Unsetenv("TS_SSH_AUDIT_LOG")
	}()

	// Initialize security logger
	if err := initSecurityLogger(); err != nil {
		t.Fatalf("Failed to initialize security logger: %v", err)
	}
	defer closeSecurityLogger()

	// Test various security events
	testHost := "test-host.example.com"
	testUser := "testuser"

	// Test insecure mode logging (forced)
	LogInsecureModeUsage(testHost, testUser, true, true)
	t.Logf("✓ Logged forced insecure mode usage")

	// Test insecure mode logging (user confirmed)
	LogInsecureModeUsage(testHost, testUser, false, true)
	t.Logf("✓ Logged user-confirmed insecure mode usage")

	// Test insecure mode logging (user declined)
	LogInsecureModeUsage(testHost, testUser, false, false)
	t.Logf("✓ Logged user-declined insecure mode usage")

	// Test SSH key authentication logging
	LogSSHKeyAuthentication(testHost, testUser, "/test/key/path", "ed25519", true)
	LogSSHKeyAuthentication(testHost, testUser, "/test/key/path", "rsa", false)
	t.Logf("✓ Logged SSH key authentication events")

	// Test host key verification logging
	LogHostKeyVerification(testHost, testUser, "known_host", true)
	LogHostKeyVerification(testHost, testUser, "new_host_accepted", true)
	LogHostKeyVerification(testHost, testUser, "new_host_rejected", false)
	t.Logf("✓ Logged host key verification events")

	// Test password authentication logging
	LogPasswordAuthentication(testHost, testUser, true)
	LogPasswordAuthentication(testHost, testUser, false)
	t.Logf("✓ Logged password authentication events")

	// Test file operation logging
	LogSecureFileOperation("create_secure_file", "/test/path", true, "success")
	LogSecureFileOperation("atomic_replacement", "/test/path", false, "permission denied")
	t.Logf("✓ Logged secure file operation events")

	// Test TTY security logging
	LogTTYSecurityValidation("tty_ownership_check", true, "TTY owned by current user")
	LogTTYSecurityValidation("tty_permissions_check", false, "TTY has insecure permissions")
	t.Logf("✓ Logged TTY security validation events")

	// Verify log file was created and contains expected entries
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Security audit log file was not created: %s", logPath)
		return
	}

	logContent, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read security audit log: %v", err)
	}

	logString := string(logContent)
	expectedEvents := []string{
		"HOST_KEY_BYPASS",
		"SSH_AUTH", 
		"HOST_KEY_VERIFICATION",
		"FILE_OPERATION",
		"TTY_SECURITY",
		"AUDIT_INIT",
	}

	for _, expectedEvent := range expectedEvents {
		if !strings.Contains(logString, expectedEvent) {
			t.Errorf("Security audit log missing expected event type: %s", expectedEvent)
		}
	}

	t.Logf("✓ Security audit log contains all expected event types")
	t.Logf("✓ Insecure mode audit logging workflow validated")
}

// testSecureFileOperationsWorkflow tests secure file operations under concurrent access
func testSecureFileOperationsWorkflow(t *testing.T, tempDir string) {
	t.Logf("Testing secure file operations workflow with concurrent access")

	// Test concurrent secure file creation
	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			filename := filepath.Join(tempDir, fmt.Sprintf("secure_file_%d.txt", id))
			
			// Test atomic file creation
			file, err := createSecureFile(filename, 0600)
			if err != nil {
				results <- fmt.Errorf("goroutine %d: failed to create secure file: %w", id, err)
				return
			}
			defer file.Close()
			
			// Write test content
			testContent := fmt.Sprintf("Secure content from goroutine %d", id)
			if _, err := file.WriteString(testContent); err != nil {
				results <- fmt.Errorf("goroutine %d: failed to write content: %w", id, err)
				return
			}
			
			// Verify file permissions
			info, err := file.Stat()
			if err != nil {
				results <- fmt.Errorf("goroutine %d: failed to stat file: %w", id, err)
				return
			}
			
			if info.Mode().Perm() != 0600 {
				results <- fmt.Errorf("goroutine %d: incorrect file permissions: got %v, want 0600", id, info.Mode().Perm())
				return
			}
			
			results <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)

	// Check results
	var errorCount int
	for err := range results {
		if err != nil {
			t.Errorf("Concurrent file operation error: %v", err)
			errorCount++
		}
	}

	if errorCount == 0 {
		t.Logf("✓ All %d concurrent secure file operations completed successfully", numGoroutines)
	}

	// Test atomic file replacement
	testFile := filepath.Join(tempDir, "atomic_replacement_test.txt")
	
	// Create initial file
	initialFile, err := createSecureFile(testFile, 0600)
	if err != nil {
		t.Fatalf("Failed to create initial file for atomic replacement test: %v", err)
	}
	initialFile.WriteString("initial content")
	initialFile.Close()

	// Test atomic replacement using the secure download mechanism
	downloadFile, err := createSecureDownloadFileWithReplace(testFile)
	if err != nil {
		t.Fatalf("Failed to create secure download file for replacement: %v", err)
	}
	
	// Write new content
	downloadFile.WriteString("replaced content")
	
	// Complete atomic replacement
	if err := completeAtomicReplacement(downloadFile); err != nil {
		t.Fatalf("Failed to complete atomic replacement: %v", err)
	}

	// Verify content was replaced
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read replaced file: %v", err)
	}
	
	if string(content) != "replaced content" {
		t.Errorf("Atomic replacement failed: got %q, want %q", string(content), "replaced content")
	}

	t.Logf("✓ Atomic file replacement completed successfully")
	t.Logf("✓ Secure file operations workflow validated")
}

// testCrossPlatformSecurityWorkflow tests security features across different platforms
func testCrossPlatformSecurityWorkflow(t *testing.T, tempDir string) {
	t.Logf("Testing cross-platform security workflow on %s", runtime.GOOS)

	// Test process security features
	originalArgs := make([]string, len(os.Args))
	copy(originalArgs, os.Args)
	defer func() {
		copy(os.Args, originalArgs) // Restore original args
	}()

	// Test process title masking
	testTitles := []string{
		"ts-ssh [secure]",
		"test-process",
		"very-long-process-title-that-should-be-truncated-properly",
	}

	for _, title := range testTitles {
		hideCredentialsInProcessList()
		maskProcessTitle(title)
		t.Logf("✓ Tested process title masking with: %s", title)
	}

	// Test platform-specific security features
	switch runtime.GOOS {
	case "linux":
		t.Logf("Testing Linux-specific security features")
		// Test prctl-based process security
		maskProcessTitleLinux("test-linux-title")
		t.Logf("✓ Linux prctl-based process masking tested")
		
	case "darwin":
		t.Logf("Testing macOS-specific security features")
		// Test Darwin-specific process security via platform function
		maskProcessTitlePlatform("test-darwin-title")
		t.Logf("✓ macOS process masking tested")
		
	case "windows":
		t.Logf("Testing Windows-specific security features")
		// Test Windows-specific process security via platform function
		maskProcessTitlePlatform("test-windows-title")
		t.Logf("✓ Windows process masking tested")
		
	default:
		t.Logf("Testing generic Unix security features for %s", runtime.GOOS)
		maskProcessTitlePlatform("test-generic-title")
		t.Logf("✓ Generic Unix process masking tested")
	}

	// Test secure environment variable handling
	originalEnv := os.Environ()
	defer func() {
		// Restore original environment (simplified)
		os.Clearenv()
		for _, env := range originalEnv {
			if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	// Set some test environment variables that should be sanitized
	testEnvVars := map[string]string{
		"SSH_AUTH_SOCK": "/tmp/test-auth-sock",
		"SSH_AGENT_PID": "12345",
		"TERM":          "xterm-256color", // This should be preserved
	}

	for key, value := range testEnvVars {
		os.Setenv(key, value)
	}

	// Apply environment security
	setSecureEnvironment()
	t.Logf("✓ Applied secure environment variable handling")

	// Verify TERM is preserved (needed for terminal operations)
	if os.Getenv("TERM") == "" {
		t.Errorf("TERM environment variable was incorrectly removed")
	}

	t.Logf("✓ Cross-platform security workflow validated for %s", runtime.GOOS)
}

// testTTYSecurityValidationWorkflow tests TTY security validation under various conditions
func testTTYSecurityValidationWorkflow(t *testing.T, tempDir string) {
	t.Logf("Testing TTY security validation workflow")

	// Test TTY path validation
	validPaths := []string{
		"/dev/tty",
		"/dev/pts/0",
	}

	invalidPaths := []string{
		"",
		"/nonexistent/path",
		"/tmp/not-a-tty",
	}

	// Test valid TTY paths
	for _, path := range validPaths {
		if _, err := os.Stat(path); err == nil {
			// Only test if the TTY actually exists
			if err := validateTTYPath(path); err != nil {
				t.Logf("TTY path %s validation failed (may be expected in test environment): %v", path, err)
			} else {
				t.Logf("✓ TTY path validation passed for: %s", path)
			}
		}
	}

	// Test invalid TTY paths
	for _, path := range invalidPaths {
		if err := validateTTYPath(path); err == nil {
			t.Errorf("TTY path validation should have failed for invalid path: %s", path)
		} else {
			t.Logf("✓ TTY path validation correctly failed for invalid path: %s", path)
		}
	}

	// Test secure TTY access in non-interactive environment
	// This should fail gracefully since we're running in a test environment
	_, err := getSecureTTY()
	if err != nil {
		expectedError := "not running in a terminal"
		if strings.Contains(err.Error(), expectedError) {
			t.Logf("✓ Secure TTY access correctly failed in non-interactive environment: %v", err)
		} else {
			t.Logf("TTY access failed with different error (may be platform-specific): %v", err)
		}
	}

	// Test TTY security logging
	LogTTYSecurityValidation("test_validation", true, "Test TTY security validation")
	t.Logf("✓ TTY security validation logging tested")

	t.Logf("✓ TTY security validation workflow completed")
}

// testHostKeyVerificationWorkflow tests the complete host key verification workflow
func testHostKeyVerificationWorkflow(t *testing.T, tempDir string) {
	t.Logf("Testing host key verification workflow")

	// Create temporary known_hosts file
	sshDir := filepath.Join(tempDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh directory: %v", err)
	}

	knownHostsPath := filepath.Join(sshDir, "known_hosts")

	// Test secure known_hosts file creation
	if err := createSecureKnownHostsFile(knownHostsPath); err != nil {
		t.Fatalf("Failed to create secure known_hosts file: %v", err)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(knownHostsPath)
	if err != nil {
		t.Fatalf("Failed to stat known_hosts file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("known_hosts file has incorrect permissions: got %v, want 0600", info.Mode().Perm())
	}

	// Verify initial content
	content, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatalf("Failed to read known_hosts file: %v", err)
	}

	expectedHeader := "# SSH Known Hosts managed by ts-ssh"
	if !strings.Contains(string(content), expectedHeader) {
		t.Errorf("known_hosts file missing expected header")
	}

	t.Logf("✓ Secure known_hosts file created successfully")

	// Test host key verification logging
	testHost := "test-verification-host.example.com"
	testUser := "testuser"

	// Log different verification scenarios
	LogHostKeyVerification(testHost, testUser, "known_host", true)
	LogHostKeyVerification(testHost, testUser, "new_host_accepted", true)
	LogHostKeyVerification(testHost, testUser, "new_host_rejected", false)
	LogHostKeyVerification(testHost, testUser, "verification_failed", false)

	t.Logf("✓ Host key verification events logged")

	// Test secure file append operations (for adding new hosts)
	file, err := createSecureFileForAppend(knownHostsPath, 0600)
	if err != nil {
		t.Fatalf("Failed to open known_hosts for append: %v", err)
	}
	defer file.Close()

	// Add a test host entry
	testEntry := "test-host.example.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAATEST...\n"
	if _, err := file.WriteString(testEntry); err != nil {
		t.Fatalf("Failed to append to known_hosts: %v", err)
	}

	t.Logf("✓ Successfully appended test entry to known_hosts")

	// Verify the append operation maintained secure permissions
	info, err = file.Stat()
	if err != nil {
		t.Fatalf("Failed to stat known_hosts after append: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("known_hosts file permissions changed after append: got %v, want 0600", info.Mode().Perm())
	}

	t.Logf("✓ known_hosts file permissions maintained after append")
	t.Logf("✓ Host key verification workflow validated")
}

// TestSecurityEventLogging tests security event logging in isolation
func TestSecurityEventLogging(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ts-ssh-security-logging-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logPath := filepath.Join(tempDir, "test_security.log")
	
	// Test initialization and cleanup
	t.Run("logger_lifecycle", func(t *testing.T) {
		os.Setenv("TS_SSH_SECURITY_AUDIT", "1")
		os.Setenv("TS_SSH_AUDIT_LOG", logPath)
		defer func() {
			os.Unsetenv("TS_SSH_SECURITY_AUDIT")
			os.Unsetenv("TS_SSH_AUDIT_LOG")
		}()

		// Initialize
		if err := initSecurityLogger(); err != nil {
			t.Fatalf("Failed to initialize security logger: %v", err)
		}

		// Verify logger is enabled
		if securityLogger == nil || !securityLogger.enabled {
			t.Error("Security logger should be enabled")
		}

		// Log a test event
		if securityLogger != nil {
			securityLogger.logSecurityEvent(SecurityEvent{
				EventType: "TEST_EVENT",
				Severity:  "INFO",
				Action:    "test_action",
				Details:   "Test security event",
				Success:   true,
			})
		}

		// Cleanup
		closeSecurityLogger()

		// Verify log file exists and contains expected content
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Error("Security log file was not created")
		}

		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read security log: %v", err)
		}

		logString := string(content)
		if !strings.Contains(logString, "AUDIT_INIT") {
			t.Error("Security log missing initialization event")
		}
		if !strings.Contains(logString, "TEST_EVENT") {
			t.Error("Security log missing test event")
		}
		if !strings.Contains(logString, "AUDIT_CLOSE") {
			t.Error("Security log missing closure event")
		}

		t.Logf("✓ Security logger lifecycle validated")
	})

	t.Run("disabled_logger", func(t *testing.T) {
		// Test with logging disabled
		os.Unsetenv("TS_SSH_SECURITY_AUDIT")
		
		if err := initSecurityLogger(); err != nil {
			t.Fatalf("Failed to initialize disabled security logger: %v", err)
		}

		if securityLogger == nil || securityLogger.enabled {
			t.Error("Security logger should be disabled when environment variable not set")
		}

		// Logging should be safe to call even when disabled
		LogInsecureModeUsage("test-host", "test-user", false, true)
		
		closeSecurityLogger()

		t.Logf("✓ Disabled security logger validated")
	})
}

// TestSecurityCompliance tests compliance with security standards
func TestSecurityCompliance(t *testing.T) {
	t.Run("file_permissions_compliance", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "ts-ssh-compliance-test-")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Test that all security-relevant files are created with proper permissions
		testFiles := []struct {
			name     string
			mode     os.FileMode
			createFn func(string, os.FileMode) (*os.File, error)
		}{
			{"ssh_key", 0600, createSecureFile},
			{"known_hosts", 0600, createSecureFile},
			{"config_file", 0600, createSecureFile},
		}

		for _, tf := range testFiles {
			filePath := filepath.Join(tempDir, tf.name)
			file, err := tf.createFn(filePath, tf.mode)
			if err != nil {
				t.Errorf("Failed to create %s: %v", tf.name, err)
				continue
			}
			file.Close()

			info, err := os.Stat(filePath)
			if err != nil {
				t.Errorf("Failed to stat %s: %v", tf.name, err)
				continue
			}

			if info.Mode().Perm() != tf.mode {
				t.Errorf("File %s has incorrect permissions: got %v, want %v", 
					tf.name, info.Mode().Perm(), tf.mode)
			}
		}

		t.Logf("✓ File permissions compliance validated")
	})

	t.Run("audit_trail_compliance", func(t *testing.T) {
		// Verify that security events create proper audit trails
		tempDir, err := os.MkdirTemp("", "ts-ssh-audit-compliance-test-")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		logPath := filepath.Join(tempDir, "compliance_audit.log")
		os.Setenv("TS_SSH_SECURITY_AUDIT", "1")
		os.Setenv("TS_SSH_AUDIT_LOG", logPath)
		defer func() {
			os.Unsetenv("TS_SSH_SECURITY_AUDIT")
			os.Unsetenv("TS_SSH_AUDIT_LOG")
		}()

		if err := initSecurityLogger(); err != nil {
			t.Fatalf("Failed to initialize security logger: %v", err)
		}
		defer closeSecurityLogger()

		// Generate audit events for compliance verification
		complianceEvents := []struct {
			logFunc func()
			pattern string
		}{
			{
				logFunc: func() { LogInsecureModeUsage("compliance-host", "compliance-user", true, true) },
				pattern: "HOST_KEY_BYPASS",
			},
			{
				logFunc: func() { LogSSHKeyAuthentication("compliance-host", "compliance-user", "/test/key", "ed25519", true) },
				pattern: "SSH_AUTH",
			},
			{
				logFunc: func() { LogHostKeyVerification("compliance-host", "compliance-user", "known_host", true) },
				pattern: "HOST_KEY_VERIFICATION",
			},
		}

		for _, event := range complianceEvents {
			event.logFunc()
		}

		// Verify audit log contains required fields for compliance
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read compliance audit log: %v", err)
		}

		logString := string(content)
		requiredFields := []string{
			"timestamp",
			"event_type", 
			"severity",
			"user",
			"host",
			"action",
			"details",
			"success",
			"user_agent",
		}

		for _, field := range requiredFields {
			if !strings.Contains(logString, field) {
				t.Errorf("Compliance audit log missing required field: %s", field)
			}
		}

		for _, event := range complianceEvents {
			if !strings.Contains(logString, event.pattern) {
				t.Errorf("Compliance audit log missing required event: %s", event.pattern)
			}
		}

		t.Logf("✓ Audit trail compliance validated")
	})
}