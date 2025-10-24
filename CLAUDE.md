# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ts-ssh is a simplified Go-based SSH and SCP client that uses Tailscale's `tsnet` library to provide userspace connectivity to Tailscale networks without requiring a full Tailscale daemon. The project enables secure SSH connections and file transfers over a Tailnet with enterprise-grade security and a minimal, ssh-like CLI interface.

**Design Philosophy**: Simplicity over features. This tool mimics the standard `ssh` command with minimal flags and maximum clarity.

## Guidance Notes

- **Quality Score Tracking**: Do not store quality scores in any artifacts, including markdown files, code comments, commit messages, or pull request descriptions. Quality metrics, including security assessments, should be reported back to the project lead but not memorialized in project artifacts.
- **Code Simplicity**: Keep the codebase small and maintainable. Avoid adding complexity unless absolutely necessary.

## CLI Architecture

ts-ssh uses a simple, flag-based CLI that mimics the standard `ssh` command:

### Basic Usage
```bash
ts-ssh [options] [user@]host[:port] [command...]
ts-ssh -scp source dest
```

### Core Features
- **SSH Connection**: Just like `ssh`, connect to any host on your Tailnet
- **SCP Transfer**: Simple file transfer with `-scp` flag
- **Port Specification**: Use `-p` flag or `host:port` syntax
- **Verbose Mode**: `-v` for debugging and authentication URLs

### No Subcommands
- No complex subcommand structure
- No dual CLI modes
- No internationalization
- No styling frameworks
- Just simple, direct flags

## Release Considerations

- Ensure the `--version` flag works correctly during release builds
- Verify proper version flag implementation when cross-compiling for different platforms

## Common Commands

### Build
```bash
go build -o ts-ssh .
```

### Run Tests
```bash
# Run all tests
go test ./...

# Run with verbose output
go test ./... -v

# Run with coverage
go test ./... -cover

# Run security tests
go test ./... -run "Test.*[Ss]ecure" -v

# Check for race conditions
go test ./... -race
```

### Cross-compile Examples
```bash
# Windows AMD64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ts-ssh-windows.exe .

# macOS ARM64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ts-ssh-darwin-arm64 .

# Linux AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ts-ssh-linux-amd64 .
```

### Run Application

```bash
# Basic SSH connection
./ts-ssh hostname
./ts-ssh user@hostname
./ts-ssh user@hostname:2222

# Execute remote command
./ts-ssh hostname uptime
./ts-ssh user@hostname "ls -la /tmp"

# SCP file transfer
./ts-ssh -scp file.txt hostname:/tmp/
./ts-ssh -scp hostname:/tmp/file.txt ./downloads/

# With options
./ts-ssh -v hostname                  # Verbose mode
./ts-ssh -p 2222 hostname            # Custom port
./ts-ssh -l alice hostname           # Specify username
./ts-ssh -i ~/.ssh/custom_key hostname  # Custom key

# Get help
./ts-ssh --help
./ts-ssh --version
```

## Key Dependencies

### Core Libraries (Minimal Set)
- **Tailscale**: Core networking and `tsnet` integration
- **golang.org/x/crypto/ssh**: SSH client implementation
- **golang.org/x/term**: Terminal handling for interactive sessions
- **github.com/bramvdbogaerde/go-scp**: SCP file transfer

### Removed Dependencies
The following were removed to simplify the codebase:
- ❌ Charmbracelet Fang, Lipgloss, Huh (UI frameworks)
- ❌ Spf13 Cobra (command framework)
- ❌ Internationalization (i18n) system
- ❌ Complex CLI modes

## Code Structure

```
ts-ssh/
├── main.go              # ~457 lines - main CLI logic
├── constants.go         # ~52 lines - constants
├── main_test.go         # ~256 lines - tests
└── internal/
    ├── client/
    │   ├── scp/         # SCP client implementation
    │   └── ssh/         # SSH client implementation
    ├── config/          # Configuration constants
    ├── crypto/pqc/      # Post-quantum cryptography
    ├── errors/          # Error handling
    ├── platform/        # Platform-specific code
    └── security/        # Security validation
```

**Total**: ~4,656 lines (down from 15,000 - 69% reduction)

## Code Quality Standards

### Linting and Formatting
```bash
# Format code (required before commits)
go fmt ./...

# Run linter (should show no issues)
golangci-lint run

# Vet for potential issues
go vet ./...
```

### Test Coverage Expectations
- **Error handling**: Target 80%+ coverage
- **Security modules**: 100% coverage required
- **Core functionality**: 70%+ coverage minimum
- **Main CLI**: Test all parsing functions

## Architecture Insights

### Tailscale Integration (`tsnet` Library)
- **Authentication URL Display**: `tsnet.Server.UserLogf` controls where authentication URLs are shown
- **Key Discovery**: In non-verbose mode, only authentication URLs are shown
- **Critical Pattern**: Use UserLogf for auth URLs, Logf for debug info
  ```go
  if verbose {
      srv.Logf = logger.Printf
      srv.UserLogf = logger.Printf
  } else {
      srv.Logf = func(string, ...interface{}) {}
      srv.UserLogf = func(format string, args ...interface{}) {
          // Filter and show only auth URLs
      }
  }
  ```

### SSH Connection Flow
1. **Parse target**: `parseSSHTarget()` handles `[user@]host[:port]` syntax
2. **Validate inputs**: Security validation for user, host, port
3. **Initialize tsnet**: Set up Tailscale connection
4. **Establish SSH**: Connect using internal SSH client
5. **Session mode**: Either execute command or start interactive shell

### SCP Transfer Flow
1. **Parse arguments**: Determine local vs remote paths
2. **Detect direction**: Upload or download
3. **Parse remote target**: Extract user, host, port
4. **Initialize tsnet**: Same as SSH
5. **Perform transfer**: Use SCP client library

### Code Organization Best Practices
- **Single responsibility**: Each function does one thing well
- **Minimal abstraction**: Avoid over-engineering
- **Clear naming**: Function names describe what they do
- **Error handling**: Use `fmt.Errorf` with clear messages
- **No magic**: Explicit is better than implicit

## Debugging Workflows

### Authentication Issues
1. Run with verbose mode: `./ts-ssh -v hostname`
2. Check for auth URL in output
3. Visit the URL to authenticate
4. Retry connection

### Connection Issues
1. Verify hostname is on your Tailnet
2. Check if Tailscale is configured: `~/.config/ts-ssh/`
3. Test with verbose mode for detailed logs
4. Ensure port is correct (default: 22)

### SCP Issues
1. Verify syntax: `ts-ssh -scp source dest`
2. Check that exactly one path is remote (`host:path`)
3. Ensure remote path uses colon notation
4. Test with verbose mode

## Development Workflow

### Before Committing
```bash
# Format and validate code
go fmt ./...
go vet ./...

# Run comprehensive tests
go test ./...

# Build to ensure no errors
go build -o ts-ssh .

# Test basic functionality
./ts-ssh --help
./ts-ssh --version
```

### Adding New Features
1. **Ask: Is this necessary?** - Keep it simple
2. **Keep it small** - Avoid adding bloat
3. **Test it** - Add tests for new functionality
4. **Document it** - Update help text and examples
5. **Maintain compatibility** - Don't break existing usage

### Code Quality Checks
- Run `go vet ./...` to catch potential issues
- Use `golangci-lint run` for comprehensive linting
- Test with and without `-v` flag
- Verify security validation is working

## Historical Context

This codebase was simplified from a complex multi-modal CLI with:
- Dual CLI modes (modern/legacy)
- 11-language internationalization
- Charmbracelet UI framework
- Multiple subcommands (connect, list, multi, exec, copy, pick)
- ~15,000 lines of code

The simplification reduced it to ~4,656 lines while maintaining core functionality:
- SSH connections
- SCP file transfers
- Tailscale integration
- Security features

The old complex code is preserved in `_old_complex/` for reference.
