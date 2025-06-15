# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ts-ssh is a Go-based SSH and SCP client that uses Tailscale's `tsnet` library to provide userspace connectivity to Tailscale networks without requiring a full Tailscale daemon. The project enables secure SSH connections and file transfers over a Tailnet with both command-line and interactive TUI interfaces.

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

- **main.go**: Entry point with CLI argument parsing and mode routing (SSH/SCP/TUI/ProxyCommand)
- **tsnet_handler.go**: Tailscale network integration using `tsnet.Server` for userspace connectivity
- **ssh_client.go**: SSH connection logic with authentication, host key verification, and terminal handling
- **scp_client.go**: SCP file transfer implementation supporting both CLI and TUI-initiated transfers
- **tui.go**: Interactive terminal interface using `tview` for peer selection and action management
- **utils.go**: Target parsing utilities for various hostname/IP formats

### Key Features

- **Authentication**: Public key (including passphrase-protected) and password authentication
- **Security**: Host key verification against `~/.ssh/known_hosts` with MITM protection
- **Connectivity**: Userspace Tailscale connectivity with browser-based auth flow
- **Interfaces**: Both CLI and TUI modes for different use cases
- **File Transfer**: Bidirectional SCP with context-aware cancellation

### Dependencies

- `tailscale.com v1.82.0` - Core Tailscale networking
- `golang.org/x/crypto` - SSH protocol implementation
- `github.com/bramvdbogaerde/go-scp v1.5.0` - SCP protocol
- `github.com/rivo/tview` - TUI framework
- `golang.org/x/term` - Terminal manipulation

### Current Development

The project is on the `feat/scp-tui-refactor` branch, actively developing SCP and TUI functionality. Recent work includes:
- Enhanced SCP capabilities with both CLI and TUI interfaces
- Comprehensive TUI functionality for peer selection
- Improved error handling and user experience
- Integration of multiple operation modes

### Testing

Unit tests are in `main_test.go` with comprehensive coverage of:
- Target parsing for various hostname/IP formats
- SCP parameter validation
- Placeholder integration tests for live network testing

The test suite includes both unit tests (runnable without external dependencies) and placeholder integration tests that would require a live Tailscale network for execution.