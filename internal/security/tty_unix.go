//go:build !windows
// +build !windows

package security

import (
	"fmt"
	"os"
	"syscall"
)

// validateTTYOwnership performs platform-specific TTY ownership validation
// This is the Unix/Linux/macOS implementation
func validateTTYOwnership(info os.FileInfo, ttyPath string) error {
	// Get file system stat for ownership checks
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot get TTY ownership information on this platform")
	}

	currentUID := uint32(os.Getuid())
	currentGID := uint32(os.Getgid())

	// Check ownership - TTY should be owned by current user OR root (for system terminals)
	// Also allow if owned by current user's group (common in some environments)
	if stat.Uid != currentUID && stat.Uid != 0 && stat.Gid != currentGID {
		return fmt.Errorf("TTY not owned by current user, root, or current group (owned by UID %d, GID %d, current UID %d, GID %d)",
			stat.Uid, stat.Gid, currentUID, currentGID)
	}

	return nil
}

// validateTTYPermissions performs platform-specific TTY permission validation
func validateTTYPermissions(info os.FileInfo, ttyPath string) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot get TTY ownership information for permission check")
	}

	mode := info.Mode()

	// For /dev/tty specifically, permissions are often 666 and that's normal
	// since it's a special device that redirects to the controlling terminal
	if ttyPath == "/dev/tty" {
		// /dev/tty is special - it's safe even with wide permissions because
		// it only gives access to the process's own controlling terminal
		return nil
	}

	// For actual TTY devices (like /dev/pts/0), be more careful about permissions
	// but still allow common patterns for root-owned TTYs
	if stat.Uid == 0 {
		// Root-owned TTYs can have group/other read access but not write
		if mode&0022 != 0 { // Check group-write and other-write
			return fmt.Errorf("TTY has unsafe permissions: %v (group/world-writable on root-owned TTY)", mode)
		}
		return nil
	}

	// For user-owned TTYs, be strict about permissions
	if mode&0077 != 0 {
		return fmt.Errorf("TTY has unsafe permissions: %v (allows group/other access on user-owned TTY)", mode)
	}

	return nil
}

// validateOpenTTYOwnership performs platform-specific ownership validation on opened TTY
func validateOpenTTYOwnership(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot get opened TTY ownership information")
	}

	currentUID := uint32(os.Getuid())
	currentGID := uint32(os.Getgid())

	// Use same relaxed ownership logic as validateTTYOwnership
	if stat.Uid != currentUID && stat.Uid != 0 && stat.Gid != currentGID {
		return fmt.Errorf("opened TTY ownership changed (owned by UID %d, GID %d, current UID %d, GID %d)",
			stat.Uid, stat.Gid, currentUID, currentGID)
	}

	return nil
}
