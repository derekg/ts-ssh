# ts-ssh Developer Guide

This guide is for developers who want to contribute to, extend, or understand the ts-ssh codebase.

## Table of Contents
- [Architecture Overview](#architecture-overview)
- [Development Setup](#development-setup)
- [Code Structure](#code-structure)
- [CLI Framework](#cli-framework)
- [Testing](#testing)
- [Contributing](#contributing)

## Architecture Overview

### Core Components

```
ts-ssh/
├── CLI Layer (cmd.go, main.go)      # User interface
├── Power Operations (power_cli.go)  # Multi-host features  
├── Client Layer (internal/client/)  # SSH/SCP implementation
├── Security (internal/security/)    # Security validations
├── Platform (internal/platform/)    # OS-specific code
├── Config (internal/config/)        # Configuration constants
└── Utils (utils.go, i18n.go)       # Shared utilities
```

### Key Design Principles

1. **Dual CLI Support**: Modern (Fang) + Legacy compatibility
2. **Security First**: All operations undergo security validation
3. **Cross-Platform**: Consistent behavior across OS platforms
4. **Modular**: Clean separation of concerns
5. **Testable**: Comprehensive test coverage

## Development Setup

### Prerequisites
```bash
# Go 1.24+ required
go version

# Development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Clone and Build
```bash
git clone https://github.com/derekg/ts-ssh.git
cd ts-ssh

# Build
go build -o ts-ssh .

# Run tests
go test ./...

# Run linter
golangci-lint run
```

### Development Dependencies

Key external libraries:
```go
// CLI Framework
github.com/charmbracelet/fang    // Modern CLI wrapper
github.com/charmbracelet/lipgloss // Terminal styling
github.com/charmbracelet/huh     // Interactive prompts
github.com/spf13/cobra           // Command structure

// Core Functionality  
tailscale.com                    // Tailscale integration
golang.org/x/crypto/ssh          // SSH client
golang.org/x/text               // Internationalization

// File Transfer
github.com/bramvdbogaerde/go-scp // SCP implementation
```

## Code Structure

### Main Entry Points

**main.go**: Application entry point with CLI mode detection
```go
func main() {
    if shouldUseLegacyCLI() {
        runLegacyCLI()    // Original interface
    } else {
        runModernCLI()    // Fang-powered interface
    }
}
```

**cmd.go**: Modern CLI implementation using Fang framework
```go
func createFangApp() *fang.Application {
    app := fang.New()
    app.AddCommand(connectCmd)
    app.AddCommand(listCmd)
    // ... other commands
}
```

### Core Modules

#### internal/client/ssh/
- `client.go`: Main SSH client implementation
- `config.go`: SSH configuration management
- `helpers.go`: Connection utilities
- `key_discovery.go`: SSH key detection and prioritization

#### internal/client/scp/
- `client.go`: SCP file transfer implementation
- Supports both upload and download operations
- Multi-host file distribution

#### internal/security/
- `validation.go`: Security validation functions
- `tty.go`: Terminal security checks
- `fileops.go`: Secure file operations

#### internal/platform/
- `process.go`: Process management utilities
- Platform-specific implementations for Windows/Unix

### Configuration Management

**internal/config/constants.go**: All configuration constants
```go
const (
    DefaultSSHPort = "22"
    DefaultTerminalWidth = 80
    SecureFilePermissions = 0600
    // ... other constants
)

var ModernKeyTypes = []string{
    "id_ed25519",  // Preferred
    "id_ecdsa",    // Good
    "id_rsa",      // Legacy
}
```

## CLI Framework

### Modern CLI (Fang-powered)

The modern CLI uses Charmbracelet's Fang framework for enhanced UX:

```go
// cmd.go
var connectCmd = &cobra.Command{
    Use:   "connect [user@]hostname[:port]",
    Short: T("connect.short"),
    Long:  T("connect.long"),
    RunE: func(cmd *cobra.Command, args []string) error {
        // Enhanced connection logic with styling
    },
}
```

### Legacy CLI Compatibility

Legacy mode maintains 100% backward compatibility:

```go
// main_legacy.go  
func runLegacyCLI() {
    // Original flag parsing and logic
    flag.Parse()
    // ... handle original commands
}
```

### CLI Mode Detection

Automatic detection logic:
```go
func shouldUseLegacyCLI() bool {
    // Environment variable override
    if os.Getenv("TS_SSH_LEGACY_CLI") == "1" {
        return true
    }
    
    // Auto-detect script-friendly patterns
    return detectLegacyUsage()
}
```

## Testing

### Test Categories

1. **Unit Tests**: Test individual functions
2. **Integration Tests**: Test component interactions  
3. **Security Tests**: Validate security features
4. **Cross-Platform Tests**: Ensure platform compatibility

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test ./... -cover

# Specific categories
go test ./... -run "Test.*[Ss]ecure"     # Security tests
go test ./... -run "Test.*[Ii]ntegration" # Integration tests

# Race condition detection
go test ./... -race

# Cross-platform testing
GOOS=windows go test ./...
GOOS=darwin go test ./...
```

### Test Structure

Example test pattern:
```go
func TestSSHConnection(t *testing.T) {
    tests := []struct {
        name    string
        host    string
        want    error
        setup   func()
        cleanup func()
    }{
        // ... test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Mock Infrastructure

For testing SSH operations:
```go
// internal/client/ssh/mock_server_test.go
func startMockSSHServer(t *testing.T) (string, func()) {
    // Returns address and cleanup function
}
```

## Internationalization (i18n)

### Adding New Strings

1. **Add to translations map** (i18n.go):
```go
var translations = map[string]map[string]string{
    "en": {
        "connect.short": "Connect to a single host via SSH",
        "new.key": "Your new translatable string",
    },
    "es": {
        "connect.short": "Conectar a un solo servidor vía SSH", 
        "new.key": "Tu nueva cadena traducible",
    },
}
```

2. **Use in code**:
```go
fmt.Println(T("new.key"))
```

### Language Detection

Priority order:
1. `--lang` CLI flag
2. `TS_SSH_LANG` environment variable
3. `LC_ALL` environment variable
4. `LANG` environment variable
5. Default (English)

## Contributing

### Code Style

1. **Format code**:
```bash
go fmt ./...
```

2. **Lint code**:
```bash
golangci-lint run
```

3. **Follow conventions**:
   - Use descriptive variable names
   - Add comments for exported functions
   - Handle errors explicitly
   - Write tests for new functionality

### Security Considerations

1. **Never log sensitive data** (passwords, keys)
2. **Validate all user inputs**
3. **Use secure file permissions** (0600 for keys, 0700 for directories)
4. **Test across platforms** for security consistency

### Pull Request Process

1. **Fork the repository**
2. **Create feature branch**: `git checkout -b feature/your-feature`
3. **Write tests** for new functionality
4. **Ensure all tests pass**: `go test ./...`
5. **Lint code**: `golangci-lint run`
6. **Test both CLI modes**:
   ```bash
   # Modern CLI
   ./ts-ssh --help
   
   # Legacy CLI
   TS_SSH_LEGACY_CLI=1 ./ts-ssh -h
   ```
7. **Submit pull request** with:
   - Clear description of changes
   - Test coverage information
   - Screenshots if UI changes

### Adding New Commands

1. **Create command in cmd.go**:
```go
var newCmd = &cobra.Command{
    Use:   "new [args]",
    Short: T("new.short"),
    Long:  T("new.long"),
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}
```

2. **Add to Fang app**:
```go
func createFangApp() *fang.Application {
    app := fang.New()
    // ... existing commands
    app.AddCommand(newCmd)
    return app
}
```

3. **Add legacy equivalent** (if needed):
```go
// In main_legacy.go
if *newFlag {
    // Legacy implementation
}
```

4. **Add translations**:
```go
// In i18n.go
"new.short": "Short description",
"new.long": "Long description with examples",
```

### Debugging

Enable verbose logging:
```go
if debug {
    log.Printf("Debug: %s", message)
}
```

Use conditional compilation for debug builds:
```go
//go:build debug
// +build debug

func debugLog(msg string) {
    log.Printf("DEBUG: %s", msg)
}
```

### Performance Considerations

1. **Connection pooling**: Reuse SSH connections when possible
2. **Parallel operations**: Use goroutines for multi-host operations
3. **Memory management**: Close resources properly
4. **Minimize allocations**: Reuse buffers and objects

### Release Process

1. **Update version** in constants.go
2. **Update RELEASE_NOTES.md**
3. **Tag release**: `git tag v0.X.Y`
4. **Build cross-platform binaries**:
```bash
./scripts/build-releases.sh  # If script exists
# Or manually:
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ts-ssh-windows.exe
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ts-ssh-darwin-arm64
# ... other platforms
```

## Getting Help

- **GitHub Discussions**: For design questions
- **GitHub Issues**: For bugs and feature requests  
- **Code Review**: Request reviews on pull requests
- **Documentation**: Contribute to guides like this one

## Useful Resources

- [Cobra Documentation](https://cobra.dev/)
- [Charmbracelet Fang](https://github.com/charmbracelet/fang)
- [Tailscale tsnet](https://pkg.go.dev/tailscale.com/tsnet)
- [Go SSH Package](https://pkg.go.dev/golang.org/x/crypto/ssh)