# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ts-ssh is a Go-based SSH and SCP client that uses Tailscale's `tsnet` library to provide userspace connectivity to Tailscale networks without requiring a full Tailscale daemon. The project enables secure SSH connections and file transfers over a Tailnet with a powerful command-line interface.

## Common Commands

### Build
```bash
go build -o ts-ssh .
```

### Run Tests
```bash
go test ./...
```

### Cross-compile Examples
```bash
# macOS ARM64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ts-ssh-darwin-arm64 .

# Linux AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ts-ssh-linux-amd64 .
```

### Run Application
```bash
./ts-ssh [user@]hostname[:port] [command...]
./ts-ssh -h  # for help
```

## Architecture

### Core Components

- **main.go**: Entry point with CLI argument parsing and mode routing (SSH/SCP/ProxyCommand)
- **main_helpers.go**: Refactored helper functions for command-line argument parsing and operation handling
- **tsnet_handler.go**: Tailscale network integration using `tsnet.Server` for userspace connectivity
- **ssh_client.go**: SSH connection logic with authentication, host key verification, and terminal handling
- **ssh_helpers.go**: Standardized SSH connection establishment and authentication helpers
- **scp_client.go**: SCP file transfer implementation
- **power_cli.go**: Advanced CLI features for multi-host operations, command execution, and file transfers
- **terminal_state.go**: Thread-safe terminal state management
- **i18n.go**: Internationalization support with race condition protection
- **constants.go**: Application-wide constants for timeouts, terminal settings, and configuration
- **utils.go**: Target parsing utilities for various hostname/IP formats

### Key Features

- **Authentication**: Public key (including passphrase-protected) and password authentication
- **Security**: Host key verification against `~/.ssh/known_hosts` with MITM protection
- **Connectivity**: Userspace Tailscale connectivity with browser-based auth flow
- **Power CLI**: Advanced command-line features including host discovery, multi-host operations, parallel execution, and tmux integration
- **File Transfer**: Bidirectional SCP with context-aware cancellation

### Dependencies

- `tailscale.com v1.82.0` - Core Tailscale networking
- `golang.org/x/crypto` - SSH protocol implementation
- `github.com/bramvdbogaerde/go-scp v1.5.0` - SCP protocol
- `golang.org/x/term` - Terminal manipulation
- `golang.org/x/text` - Internationalization support

### Current Development

The project is on the `main` branch with recent major improvements including:
- Code quality refactoring with 94% reduction in main() function size
- Race condition fixes for thread-safe operation
- Enhanced power CLI with multi-host operations and parallel execution
- Comprehensive test suite with improved coverage
- Internationalization support with Spanish translations
- Modular architecture with helper functions and constants extraction

### Testing

Comprehensive test suite across multiple files:
- `main_test.go`: Core utility functions and configuration testing
- `ssh_helpers_test.go`: SSH connection and authentication testing
- `terminal_state_test.go`: Thread-safe terminal state management testing
- `i18n_test.go`: Internationalization and race condition testing

Tests cover:
- Target parsing for various hostname/IP formats
- SSH helper functions and configuration
- Race condition protection and thread safety
- Internationalization system
- Power CLI functionality
- Constants and configuration validation

The test suite focuses on unit testing without external dependencies, with substantial improvements in coverage and race condition validation.