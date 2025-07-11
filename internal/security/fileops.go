package security

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// createSecureFile creates a file with secure permissions atomically
// to prevent race conditions between file creation and permission setting
func CreateSecureFile(filename string, mode os.FileMode) (*os.File, error) {
	// Create file with restrictive permissions atomically
	// Use O_EXCL to ensure we create a new file and fail if it already exists
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to create secure file %s: %w", filename, err)
	}

	// Verify permissions were set correctly (defense in depth)
	info, err := file.Stat()
	if err != nil {
		file.Close()
		os.Remove(filename)
		return nil, fmt.Errorf("failed to verify file permissions: %w", err)
	}

	if info.Mode() != mode {
		file.Close()
		os.Remove(filename)
		return nil, fmt.Errorf("file permissions not set correctly: expected %v, got %v", mode, info.Mode())
	}

	return file, nil
}

// createSecureFileForAppend creates or opens a file for appending with secure permissions
func CreateSecureFileForAppend(filename string, mode os.FileMode) (*os.File, error) {
	// First, try to open existing file
	if _, err := os.Stat(filename); err == nil {
		// File exists, verify its permissions
		if err := verifyFilePermissions(filename, mode); err != nil {
			return nil, fmt.Errorf("existing file has insecure permissions: %w", err)
		}
		// Open for append
		return os.OpenFile(filename, os.O_WRONLY|os.O_APPEND, mode)
	}

	// File doesn't exist, create it securely
	return CreateSecureFile(filename, mode)
}

// CreateSecureKnownHostsFile creates a known_hosts file with secure permissions
func CreateSecureKnownHostsFile(knownHostsPath string) error {
	// Ensure parent directory exists with secure permissions
	dir := filepath.Dir(knownHostsPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create ssh directory: %w", err)
	}

	// Create known_hosts file atomically with secure permissions
	file, err := CreateSecureFileForAppend(knownHostsPath, 0600)
	if err != nil {
		if os.IsExist(err) {
			// File already exists, verify permissions
			return verifyFilePermissions(knownHostsPath, 0600)
		}
		return err
	}
	defer file.Close()

	// Write initial content if it's a new file
	if stat, err := file.Stat(); err == nil && stat.Size() == 0 {
		_, err = file.WriteString("# SSH Known Hosts managed by ts-ssh\n")
		return err
	}

	return nil
}

// verifyFilePermissions checks if a file has the expected permissions
func verifyFilePermissions(filename string, expectedMode os.FileMode) error {
	info, err := os.Stat(filename)
	if err != nil {
		return err
	}

	if info.Mode() != expectedMode {
		return os.Chmod(filename, expectedMode)
	}
	return nil
}

// secureFileCopy performs a secure file copy with atomic operations
func secureFileCopy(src, dst string, mode os.FileMode) error {
	// Create temporary file with secure permissions
	tempFile := dst + ".tmp." + GenerateRandomSuffix()

	file, err := CreateSecureFile(tempFile, mode)
	if err != nil {
		return err
	}
	defer func() {
		file.Close()
		os.Remove(tempFile) // Cleanup on error
	}()

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Copy content
	if _, err := file.ReadFrom(srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close before rename
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, dst); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// createSecureDownloadFile creates a file for SCP download with secure permissions
func createSecureDownloadFile(localPath string) (*os.File, error) {
	// For downloads, we want to create the file with secure permissions initially
	// and allow the user to adjust them later if needed
	return CreateSecureFile(localPath, 0600)
}

// CreateSecureDownloadFileWithReplace creates a temporary file for SCP download
// Returns the file and a function to complete the atomic replacement
func CreateSecureDownloadFileWithReplace(localPath string) (*os.File, error) {
	// Create temporary file in same directory to ensure atomic move is possible
	tempPath := localPath + ".tmp." + GenerateRandomSuffix()

	// Create temporary file with secure permissions
	file, err := CreateSecureFile(tempPath, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary download file: %w", err)
	}

	// Store the paths for later atomic replacement with thread safety
	atomicReplaceFilesMutex.Lock()
	atomicReplaceFiles[file] = atomicReplaceInfo{
		tempPath:  tempPath,
		finalPath: localPath,
	}
	atomicReplaceFilesMutex.Unlock()

	return file, nil
}

// atomicReplaceInfo stores paths for atomic replacement
type atomicReplaceInfo struct {
	tempPath  string
	finalPath string
}

// Global map to track files that need atomic replacement with thread safety
var atomicReplaceFilesMutex sync.RWMutex
var atomicReplaceFiles = make(map[*os.File]atomicReplaceInfo)

// CompleteAtomicReplacement performs atomic replacement for a file
func CompleteAtomicReplacement(file *os.File) error {
	atomicReplaceFilesMutex.Lock()
	info, exists := atomicReplaceFiles[file]
	if exists {
		delete(atomicReplaceFiles, file)
	}
	atomicReplaceFilesMutex.Unlock()

	if !exists {
		// Not an atomic file, just close normally
		return file.Close()
	}

	// Close the file first
	if err := file.Close(); err != nil {
		os.Remove(info.tempPath) // Cleanup temp file
		return fmt.Errorf("failed to close temporary file before rename: %w", err)
	}

	// Perform atomic rename
	if err := os.Rename(info.tempPath, info.finalPath); err != nil {
		os.Remove(info.tempPath) // Clean up temp file
		return fmt.Errorf("failed to atomically replace file: %w", err)
	}

	return nil
}

// generateRandomSuffix generates a random suffix for temporary files
func GenerateRandomSuffix() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to simpler method if crypto/rand fails
		return fmt.Sprintf("%d", os.Getpid())
	}
	return fmt.Sprintf("%x", bytes)
}
