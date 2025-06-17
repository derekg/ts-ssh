package main

import "time"

// Network and connection constants
const (
	// SSH connection timeouts
	DefaultSSHTimeout      = 15 * time.Second
	DefaultSCPTimeout      = 30 * time.Second
	ConnectionWaitTime     = 3 * time.Second
	StatusUpdateTimeout    = 5 * time.Second
	
	// SSH configuration
	DefaultSshPort = "22"
	
	// Terminal defaults
	DefaultTerminalWidth   = 80
	DefaultTerminalHeight  = 24
	DefaultTerminalType    = "xterm-256color"
	
	// Application identifiers
	ClientName        = "ts-ssh-client"
	TmuxSessionPrefix = "ts-ssh-"
	
	// Buffer sizes
	DefaultBufferSize = 4096
	
	// File permissions
	DefaultKeyPermissions = 0600
	DefaultDirPermissions = 0700
)

// Error messages constants
const (
	ErrEmptyTarget     = "target cannot be empty"
	ErrEmptyPath       = "file path cannot be empty"
	ErrInvalidPath     = "file path contains invalid characters"
	ErrConnectionFailed = "failed to establish connection"
)