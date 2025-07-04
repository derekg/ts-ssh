package main

import (
	"time"

	"github.com/derekg/ts-ssh/internal/config"
)

// Network and connection constants
const (
	// SSH connection timeouts
	DefaultSSHTimeout   = 15 * time.Second
	DefaultSCPTimeout   = 30 * time.Second
	ConnectionWaitTime  = 3 * time.Second
	StatusUpdateTimeout = 5 * time.Second

	// Buffer sizes
	DefaultBufferSize    = 4096
	InputBufferSize      = 1024
	HostOutputBufferSize = 100

	// Retry and limit constants
	MaxPasswordRetries = 3
	MaxConcurrentHosts = 50
	SessionWaitTimeout = 5 * time.Second
	MaxStateRetries    = 3
	StateRetryDelay    = 1 * time.Second
)

// Import shared constants from config package
const (
	DefaultSshPort        = config.DefaultSSHPort
	DefaultTerminalWidth  = config.DefaultTerminalWidth
	DefaultTerminalHeight = config.DefaultTerminalHeight
	DefaultTerminalType   = config.DefaultTerminalType
	ClientName            = config.ClientName
	DefaultKeyPermissions = config.SecureFilePermissions
	DefaultDirPermissions = config.SecureDirectoryPermissions
)

// Import shared variables from config package
var (
	ModernKeyTypes = config.ModernKeyTypes
)

// Error messages constants
const (
	ErrEmptyTarget      = "target cannot be empty"
	ErrEmptyPath        = "file path cannot be empty"
	ErrInvalidPath      = "file path contains invalid characters"
	ErrConnectionFailed = "failed to establish connection"
)
