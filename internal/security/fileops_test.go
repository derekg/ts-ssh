package security

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateSecureFile(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := ioutil.TempDir("", "secure-file-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		filename string
		mode     os.FileMode
		wantErr  bool
	}{
		{
			name:     "create new file with 0600",
			filename: "test1.txt",
			mode:     0600,
			wantErr:  false,
		},
		{
			name:     "create new file with 0644",
			filename: "test2.txt",
			mode:     0644,
			wantErr:  false,
		},
		{
			name:     "fail when file exists",
			filename: "test1.txt", // Same as first test
			mode:     0600,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath := filepath.Join(tempDir, tt.filename)

			file, err := CreateSecureFile(fullPath, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSecureFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if file == nil {
					t.Error("CreateSecureFile() returned nil file but no error")
					return
				}
				defer file.Close()

				// Verify file permissions
				info, err := file.Stat()
				if err != nil {
					t.Errorf("Failed to stat created file: %v", err)
					return
				}

				if info.Mode() != tt.mode {
					t.Errorf("File permissions = %v, want %v", info.Mode(), tt.mode)
				}

				// Verify file exists and is accessible
				if _, err := os.Stat(fullPath); err != nil {
					t.Errorf("Created file not accessible: %v", err)
				}
			}
		})
	}
}

func TestCreateSecureFileForAppend(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "secure-append-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "append-test.txt")

	// First call should create the file
	file1, err := CreateSecureFileForAppend(testFile, 0600)
	if err != nil {
		t.Fatalf("First CreateSecureFileForAppend() failed: %v", err)
	}
	file1.Close()

	// Second call should open existing file
	file2, err := CreateSecureFileForAppend(testFile, 0600)
	if err != nil {
		t.Fatalf("Second CreateSecureFileForAppend() failed: %v", err)
	}
	file2.Close()

	// Verify permissions
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Mode() != 0600 {
		t.Errorf("File permissions = %v, want %v", info.Mode(), 0600)
	}
}

func TestCreateSecureKnownHostsFile(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "known-hosts-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sshDir := filepath.Join(tempDir, ".ssh")
	knownHostsPath := filepath.Join(sshDir, "known_hosts")

	// Test creating known_hosts file
	err = CreateSecureKnownHostsFile(knownHostsPath)
	if err != nil {
		t.Fatalf("CreateSecureKnownHostsFile() failed: %v", err)
	}

	// Verify SSH directory was created with correct permissions
	dirInfo, err := os.Stat(sshDir)
	if err != nil {
		t.Fatalf("SSH directory not created: %v", err)
	}

	if dirInfo.Mode() != os.ModeDir|0700 {
		t.Errorf("SSH directory permissions = %v, want %v", dirInfo.Mode(), os.ModeDir|0700)
	}

	// Verify known_hosts file was created with correct permissions
	fileInfo, err := os.Stat(knownHostsPath)
	if err != nil {
		t.Fatalf("Known hosts file not created: %v", err)
	}

	if fileInfo.Mode() != 0600 {
		t.Errorf("Known hosts file permissions = %v, want %v", fileInfo.Mode(), 0600)
	}

	// Verify file contains initial content
	content, err := ioutil.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatalf("Failed to read known hosts file: %v", err)
	}

	expectedContent := "# SSH Known Hosts managed by ts-ssh\n"
	if string(content) != expectedContent {
		t.Errorf("Known hosts content = %q, want %q", string(content), expectedContent)
	}

	// Test calling again on existing file
	err = CreateSecureKnownHostsFile(knownHostsPath)
	if err != nil {
		t.Errorf("CreateSecureKnownHostsFile() should succeed on existing file: %v", err)
	}
}

func TestVerifyFilePermissions(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "verify-perms-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test-perms.txt")

	// Create file with wrong permissions
	file, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()

	// Verify it fixes permissions
	err = verifyFilePermissions(testFile, 0600)
	if err != nil {
		t.Errorf("verifyFilePermissions() failed: %v", err)
	}

	// Check permissions were actually changed
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Mode() != 0600 {
		t.Errorf("File permissions after verify = %v, want %v", info.Mode(), 0600)
	}
}

func TestSecureFileCopy(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "secure-copy-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create source file
	srcFile := filepath.Join(tempDir, "source.txt")
	srcContent := "This is test content for secure file copy"
	err = ioutil.WriteFile(srcFile, []byte(srcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Test secure copy
	dstFile := filepath.Join(tempDir, "destination.txt")
	err = secureFileCopy(srcFile, dstFile, 0600)
	if err != nil {
		t.Fatalf("secureFileCopy() failed: %v", err)
	}

	// Verify destination file was created with correct permissions
	info, err := os.Stat(dstFile)
	if err != nil {
		t.Fatalf("Destination file not created: %v", err)
	}

	if info.Mode() != 0600 {
		t.Errorf("Destination file permissions = %v, want %v", info.Mode(), 0600)
	}

	// Verify content was copied correctly
	dstContent, err := ioutil.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != srcContent {
		t.Errorf("Destination content = %q, want %q", string(dstContent), srcContent)
	}
}

func TestCreateSecureDownloadFile(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "secure-download-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	downloadFile := filepath.Join(tempDir, "download.txt")

	file, err := createSecureDownloadFile(downloadFile)
	if err != nil {
		t.Fatalf("createSecureDownloadFile() failed: %v", err)
	}
	defer file.Close()

	// Verify permissions
	info, err := file.Stat()
	if err != nil {
		t.Fatalf("Failed to stat download file: %v", err)
	}

	if info.Mode() != 0600 {
		t.Errorf("Download file permissions = %v, want %v", info.Mode(), 0600)
	}
}

func TestGenerateRandomSuffix(t *testing.T) {
	// Test that function returns a non-empty string
	suffix := GenerateRandomSuffix()
	if suffix == "" {
		t.Error("GenerateRandomSuffix() returned empty string")
	}

	// Test that multiple calls return different values
	suffix2 := GenerateRandomSuffix()
	if suffix == suffix2 {
		t.Error("GenerateRandomSuffix() returned same value twice (very unlikely)")
	}

	// Test suffix length is reasonable
	if len(suffix) < 4 {
		t.Errorf("GenerateRandomSuffix() returned too short suffix: %q", suffix)
	}
}

func TestCreateSecureDownloadFileWithReplace(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "atomic-download-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	downloadFile := filepath.Join(tempDir, "download-test.txt")

	// Create existing file first
	existingContent := "existing content"
	err = ioutil.WriteFile(downloadFile, []byte(existingContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Use atomic download file replacement
	atomicFile, err := CreateSecureDownloadFileWithReplace(downloadFile)
	if err != nil {
		t.Fatalf("CreateSecureDownloadFileWithReplace() failed: %v", err)
	}

	// Write new content
	newContent := "new download content"
	_, err = atomicFile.WriteString(newContent)
	if err != nil {
		t.Fatalf("Failed to write to atomic file: %v", err)
	}

	// Complete atomic replacement
	err = CompleteAtomicReplacement(atomicFile)
	if err != nil {
		t.Fatalf("Failed to complete atomic replacement: %v", err)
	}

	// Verify content was replaced
	finalContent, err := ioutil.ReadFile(downloadFile)
	if err != nil {
		t.Fatalf("Failed to read final file: %v", err)
	}

	if string(finalContent) != newContent {
		t.Errorf("File content = %q, want %q", string(finalContent), newContent)
	}

	// Verify permissions are secure
	info, err := os.Stat(downloadFile)
	if err != nil {
		t.Fatalf("Failed to stat final file: %v", err)
	}

	if info.Mode() != 0600 {
		t.Errorf("Final file permissions = %v, want %v", info.Mode(), 0600)
	}
}

func TestAtomicDownloadFileCleanupOnError(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "atomic-cleanup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	downloadFile := filepath.Join(tempDir, "cleanup-test.txt")

	// Create atomic file but don't close it properly
	atomicFile, err := CreateSecureDownloadFileWithReplace(downloadFile)
	if err != nil {
		t.Fatalf("CreateSecureDownloadFileWithReplace() failed: %v", err)
	}

	// Write some content
	_, err = atomicFile.WriteString("test content")
	if err != nil {
		t.Fatalf("Failed to write to atomic file: %v", err)
	}

	// Complete atomic replacement
	err = CompleteAtomicReplacement(atomicFile)
	if err != nil {
		t.Fatalf("Failed to complete atomic replacement: %v", err)
	}

	// Verify the file was created at the final location
	if _, err := os.Stat(downloadFile); err != nil {
		t.Errorf("Expected final file to exist after close: %v", err)
	}
}

func TestSecureFileOperationsConcurrency(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "concurrent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test concurrent file creation (should fail for same filename)
	testFile := filepath.Join(tempDir, "concurrent.txt")

	// Create channels for coordination
	results := make(chan error, 2)

	// Start two goroutines trying to create the same file
	for i := 0; i < 2; i++ {
		go func() {
			file, err := CreateSecureFile(testFile, 0600)
			if err == nil {
				file.Close()
			}
			results <- err
		}()
	}

	// Collect results
	var errors []error
	for i := 0; i < 2; i++ {
		err := <-results
		errors = append(errors, err)
	}

	// One should succeed, one should fail
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}

	if successCount != 1 {
		t.Errorf("Expected exactly 1 success in concurrent file creation, got %d", successCount)
	}
}
