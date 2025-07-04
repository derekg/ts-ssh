//go:build linux
// +build linux

package platform

import (
	"syscall"
	"unsafe"
)

// maskProcessTitlePlatform sets a process title on Linux
func maskProcessTitlePlatform(title string) {
	maskProcessTitleLinux(title)
}

// maskProcessTitleLinux uses prctl to set process title on Linux
func maskProcessTitleLinux(title string) {
	// Use prctl PR_SET_NAME to set process name
	titleBytes := []byte(title + "\x00")
	if len(titleBytes) > 16 { // Linux process names are limited to 15 chars + null
		titleBytes = titleBytes[:15]
		titleBytes[14] = 0
	}

	// PR_SET_NAME = 15 - Linux-specific syscall
	const PR_SET_NAME = 15
	syscall.Syscall(syscall.SYS_PRCTL, PR_SET_NAME, uintptr(unsafe.Pointer(&titleBytes[0])), 0)
}
