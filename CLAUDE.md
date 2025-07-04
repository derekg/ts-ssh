# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ts-ssh is a Go-based SSH and SCP client that uses Tailscale's `tsnet` library to provide userspace connectivity to Tailscale networks without requiring a full Tailscale daemon. The project enables secure SSH connections and file transfers over a Tailnet with enterprise-grade security, comprehensive cross-platform support, and a modern CLI experience powered by Charmbracelet's Fang framework.

## Guidance Notes

- **Quality Score Tracking**: Do not store quality scores in any artifacts, including markdown files, code comments, commit messages, or pull request descriptions. Quality metrics, including security assessments, should be reported back to the project lead but not memorialized in project artifacts.

## CLI Architecture

ts-ssh supports dual CLI modes for optimal user experience:

### Modern CLI (Default)
- Powered by Charmbracelet Fang framework
- Enhanced styling with Lipgloss
- Interactive prompts with Huh
- Structured subcommand architecture
- Better help organization and styling

### Legacy CLI
- Original interface for backward compatibility
- Script-friendly for automation
- Controlled via `TS_SSH_LEGACY_CLI=1` environment variable
- Auto-detection for legacy usage patterns

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
# Run all tests (unit + integration + security)
go test ./...

# Run specific test categories
go test ./... -run "Test.*[Ss]ecure"        # Security tests only
go test ./... -run "Test.*[Ii]ntegration"   # Integration tests only
go test ./... -run "Test.*[Aa]uth"          # Authentication tests only

# Run tests with verbose output
go test ./... -v

# Run tests with coverage
go test ./... -cover

# Test specific modules that previously had 0% coverage
go test ./internal/errors/... -v -cover     # Error handling (84.6% coverage)
go test ./internal/config/... -v -cover     # Configuration constants
go test ./internal/client/scp/... -v -cover # SCP client (11.2% coverage)

# Run security benchmarks
go test ./... -bench="Benchmark.*[Ss]ecure"
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

### Security Assessment
```bash
# Run comprehensive security test suite
go test ./... -run "Test.*[Ss]ecure" -v

# Validate cross-platform security features
GOOS=windows go test ./... -run "Test.*[Ss]ecure"
GOOS=darwin go test ./... -run "Test.*[Ss]ecure"

# Check for race conditions
go test ./... -race
```

### Run Application

#### Modern CLI (Default)
```bash
# Subcommand structure with enhanced styling
./ts-ssh connect [user@]hostname[:port] [-- command...]
./ts-ssh list                           # Beautiful host listing
./ts-ssh multi host1,host2,host3        # Enhanced tmux experience
./ts-ssh exec --command "uptime" host1,host2  # Styled command execution
./ts-ssh copy file.txt host1:/tmp/      # Enhanced file operations
./ts-ssh pick                           # Interactive host picker
./ts-ssh --help                         # Styled help output
```

#### Legacy CLI (Backward Compatible)
```bash
# Original interface for scripts and automation
export TS_SSH_LEGACY_CLI=1
./ts-ssh [user@]hostname[:port] [command...]
./ts-ssh --list                         # Original host listing
./ts-ssh --multi host1,host2,host3      # Original tmux
./ts-ssh --exec "uptime" host1,host2    # Original command execution
./ts-ssh --copy file.txt host1:/tmp/    # Original file operations
./ts-ssh --pick                         # Original host picker
./ts-ssh -h                             # Original help
```

#### CLI Mode Control
```bash
# Force legacy mode permanently
export TS_SSH_LEGACY_CLI=1

# One-time legacy mode usage
TS_SSH_LEGACY_CLI=1 ./ts-ssh --list

# Check current mode (modern CLI will show subcommands)
./ts-ssh --help
```

## Key Dependencies

### Core Libraries
- **Charmbracelet Fang**: CLI framework for enhanced user experience
- **Charmbracelet Lipgloss**: Terminal styling and color management
- **Charmbracelet Huh**: Interactive prompts and user input
- **Spf13 Cobra**: Underlying command structure (via Fang)
- **Tailscale**: Core networking and `tsnet` integration
- **golang.org/x/crypto/ssh**: SSH client implementation
- **golang.org/x/text**: Internationalization support

### Development Patterns
```bash
# When adding new features, ensure both CLI modes work
go run . connect hostname          # Test modern CLI
TS_SSH_LEGACY_CLI=1 go run . hostname  # Test legacy CLI

# Test internationalization (11 supported languages)
LANG=es go run . --help            # Spanish interface
LANG=zh go run . --help            # Chinese interface
LANG=de go run . --help            # German interface
LANG=fr go run . --help            # French interface
LANG=pt go run . --help            # Portuguese interface
LANG=ru go run . --help            # Russian interface
LANG=ja go run . --help            # Japanese interface
LANG=hi go run . --help            # Hindi interface
LANG=ar go run . --help            # Arabic interface
LANG=bn go run . --help            # Bengali interface
LANG=en go run . --help            # English interface (default)

# Validate security across platforms
GOOS=windows go test ./internal/security/...
GOOS=darwin go test ./internal/security/...
GOOS=linux go test ./internal/security/...
```

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
- **Error handling**: Target 80%+ coverage (currently 84.6%)
- **Security modules**: 100% coverage required
- **Core functionality**: 70%+ coverage minimum
- **Configuration**: Comprehensive constant validation required

## Architecture Insights

### Tailscale Integration (`tsnet` Library)
- **Authentication URL Display**: `tsnet.Server.UserLogf` controls where authentication URLs are shown
- **Key Discovery**: `UserLogf` outputs to stderr by default, but can be redirected to io.Discard in non-verbose mode
- **Critical Pattern**: Always use a dedicated stderr logger for UserLogf to ensure auth URLs are visible:
  ```go
  stderrLogger := log.New(os.Stderr, "", 0)
  srv.UserLogf = stderrLogger.Printf
  ```
- **Logger Configuration**: `srv.Logf` vs `srv.UserLogf` serve different purposes - Logf for debug info, UserLogf for user-facing messages

### Internationalization System
- **Translation Coverage**: Currently supports 11 languages (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)
- **Missing Translation Detection**: Use `rg "T\(\"[^\"]*\"\)" -o -h --no-filename | sort | uniq` to find all translation keys
- **Translation Validation**: Check for missing translations by running app with different LANG settings
- **Key Patterns**: Connection status messages like "Starting Tailscale connection..." need translation coverage
- **Format String Safety**: Use `fmt.Errorf("%s", T("key"))` instead of `fmt.Errorf(T("key"))` to avoid go vet warnings

### CLI Framework (Cobra/Fang)
- **Text Rendering Issue**: Cobra/Fang may strip spaces from Example fields in help text
- **Workaround Patterns**: Use non-breaking spaces or alternative formatting for help examples
- **Translation Integration**: Example text should be translated and may need language-specific formatting

### Code Organization Best Practices
- **Function Extraction**: Break large functions into smaller, focused helpers (e.g., `initTsNet()` refactored into multiple helper functions)
- **Error Handling**: Use consistent error wrapping patterns with translated messages
- **Logger Management**: Distinguish between verbose debug logging and user-facing messages
- **Terminal State**: Centralize terminal state management for consistent behavior across interactive sessions

## Debugging Workflows

### Authentication Issues
1. Check if auth URL appears in verbose mode: `./ts-ssh connect -v target`
2. Verify UserLogf configuration in tsnet_handler.go
3. Test logger output destination (should be stderr, not io.Discard)

### Missing Translations
1. Extract all translation keys: `rg "T\(\"[^\"]*\"\)" -o -h --no-filename | sort | uniq`
2. Test different languages: `LANG=es ./ts-ssh --help`
3. Look for untranslated strings in output (they appear as key names)

### CLI Rendering Issues
1. Check help output formatting: `./ts-ssh --help`
2. Verify example text spacing in different languages
3. Test both modern and legacy CLI modes

## Development Workflow

### Before Committing
```bash
# Format and validate code
go fmt ./...
go vet ./...

# Run comprehensive tests
go test ./...

# Check for translation issues
for lang in es zh hi ar bn pt ru ja de fr; do
    echo "Testing $lang..."
    LANG=$lang ./ts-ssh --help | head -10
done
```

### Code Quality Checks
- Run `go vet ./...` to catch format string issues
- Use `golangci-lint run` for comprehensive linting
- Test auth URL display in both verbose and non-verbose modes
- Validate translation coverage for new user-facing messages