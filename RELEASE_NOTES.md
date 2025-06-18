# ts-ssh Release Notes

## Version 1.2.0 - Code Quality & Architecture Improvements

Date: 2025-06-18

### Major Improvements

- **🧹 Complete TUI Code Cleanup**  
  Removed all dead terminal UI code and dependencies (180+ lines removed):
  - Eliminated unused `connectToHostFromTUI` function
  - Removed `tuiMode` parameter throughout codebase
  - Cleaned TUI dependencies: `github.com/rivo/tview`, `github.com/gdamore/tcell/v2`
  - Updated documentation to reflect streamlined CLI-only architecture

- **🔧 SSH Code Consolidation**  
  Major refactoring of SSH connection logic:
  - Extracted shared `executeCommandOnHost` helper (~85 lines of duplication removed)
  - Standardized SSH connection patterns across all modules
  - Created comprehensive SSH helper functions in `ssh_helpers.go`
  - Thread-safe authentication with proper mutex protection

- **🐛 Critical i18n Formatting Fixes**  
  Resolved double-formatting issues affecting user experience:
  - **Fixed**: Password prompts now display `derek@bar` instead of `%!!(string=derek)s(MISSING)@%!!(string=bar)s(MISSING)`
  - **Fixed**: Error messages with proper argument substitution
  - **Improved**: Consistent T() function usage patterns across codebase

- **🧪 Enhanced Test Coverage**  
  Comprehensive test suite expansion (14.5% → 22% coverage):
  - **New**: `i18n_test.go` - Race condition testing for concurrent translations
  - **New**: `ssh_helpers_test.go` - SSH connection and authentication testing
  - **New**: `terminal_state_test.go` - Thread-safe terminal state management
  - **Enhanced**: `main_test.go` with additional utility function coverage

### Architecture Improvements

- **📁 Modular Code Organization**  
  Split monolithic functions into focused, maintainable modules:
  - `main_helpers.go` - Command-line argument parsing and operation routing
  - `ssh_helpers.go` - Standardized SSH connection establishment
  - `terminal_state.go` - Thread-safe terminal state management
  - `constants.go` - Centralized application constants and configuration

- **🏗️ Race Condition Fixes**  
  Comprehensive thread safety improvements:
  - Thread-safe i18n system with proper mutex protection
  - Concurrent SSH authentication with shared mutex for password prompts
  - Terminal state management with atomic operations

- **📚 Enhanced Documentation**  
  Comprehensive documentation for all public functions:
  - Detailed parameter descriptions and return value documentation
  - Usage examples and workflow explanations
  - Updated `CLAUDE.md` with current architecture overview

### Performance & Quality

- **⚡ Build Improvements**  
  - Removed unused imports and dependencies
  - Centralized magic numbers into named constants
  - Improved function naming for clarity and consistency

- **🔒 Better Error Handling**  
  - Consistent error patterns across all modules
  - Enhanced error context with better user-facing messages
  - Proper resource cleanup and defer patterns

### Breaking Changes

- **None** - All changes are backward compatible
- Users will notice improved password prompt formatting
- CLI behavior and flags remain unchanged

### Technical Details

**Files Added:**
- `constants.go` - Application-wide constants
- `main_helpers.go` - Refactored CLI argument handling  
- `ssh_helpers.go` - SSH connection utilities
- `terminal_state.go` - Thread-safe terminal management
- `*_test.go` - Comprehensive test suites

**Dependencies Removed:**
- `github.com/rivo/tview` (TUI framework)
- `github.com/gdamore/tcell/v2` (Terminal cell library)
- Related indirect dependencies automatically cleaned

**Code Metrics:**
- **Removed**: 568 lines (dead code elimination)
- **Added**: 796 lines (tests + architectural improvements)  
- **Net Result**: Improved maintainability, better test coverage, cleaner codebase

---

## Version 1.1.0 - Spanish Language Support

Date: 2025-06-17

### New Features

- **🌍 Spanish Language Support**  
  Complete Spanish localization for all CLI help text, usage examples, and error messages. Supports multiple language detection methods:
  - CLI flag: `--lang es` or `--lang=es`
  - Environment variables: `TS_SSH_LANG=es`, `LANG=es`, `LC_ALL=es`
  - Dynamic help display that respects language preferences immediately
  - Language priority: CLI flag > TS_SSH_LANG > LC_ALL > LANG > default (English)

- **Enhanced Internationalization Framework**  
  Robust i18n infrastructure using `golang.org/x/text` with support for:
  - Multiple locale environment variable detection
  - Runtime language switching
  - Proper message formatting with parameter substitution
  - Extensible design for future language additions

### Improvements

- **Improved Language Detection**  
  Now supports standard locale environment variables (`LANG`, `LC_ALL`) in addition to the custom `TS_SSH_LANG` variable
- **Dynamic Help System**  
  Help text (`-h`/`--help`) now immediately reflects the selected language, including when using CLI flags

### Usage Examples

```bash
# Spanish interface via CLI flag
ts-ssh --lang es --help
ts-ssh --lang es --list

# Spanish interface via environment
LANG=es ts-ssh --help
export TS_SSH_LANG=es && ts-ssh --list

# Override environment with CLI flag
LANG=es ts-ssh --lang en --help  # Shows English
```

---

## Version 1.0.0 - Initial Release

Date: 2025-04-18

## Overview
This is the first stable release of **ts-ssh**, a userspace SSH client powered by Tailscale’s `tsnet` library. It lets you reach machines on your Tailnet without the full Tailscale daemon, and brings the majority of standard SSH UX into a single, self-contained binary.

## New Features

- **Interactive Escape Sequence (`~.`)**  
  At any point in an interactive session, type `~.` at the start of a new line to immediately terminate the SSH connection and restore your terminal.
- **Non-Interactive Command Execution**  
  Pass a remote command directly on the command line (e.g. `ts-ssh host uname -a`). The client runs the command, streams its output, and returns its exit code.
- **ProxyCommand-Style TCP Forwarding (`-W`)**  
  Implements `ssh -W host:port` behavior over Tailscale. Use `ts-ssh -W target:22` as a `ProxyCommand` in `ssh` or `scp` configurations to transparently tunnel traffic:

      scp -o ProxyCommand="ts-ssh -W %h:%p user@gateway" localfile remote:/path
- **Version Flag (`-version`)**  
  Print the client’s version string and exit. During your build, embed a version via:

      go build -ldflags "-X main.version=1.0.0" -o ts-ssh .
- **Improved Usage Examples**  
  The built-in `-h`/`--help` output and README now include clear, copy-and-paste examples for interactive sessions, remote commands, ProxyCommand usage, and the escape sequence.
- **Enhanced Documentation**  
  Comprehensive examples and security notes have been added to `README.md`, including warnings about host-key verification, the `-insecure` flag, and interleaved auth-flow logging.

## Security & Stability

- Secure host-key verification against `~/.ssh/known_hosts` by default, with an interactive prompt for unknown hosts and strict detection of changed keys (MITM protection).
- Insecure mode (`-insecure`) remains available for testing but is strongly discouraged.
- Graceful shutdown on `SIGINT`/`SIGTERM`, with terminal state restoration even if you hit the new escape sequence.

## Bug Fixes & Polishing

- Fixed quoting in example `ProxyCommand` snippets.
- Synchronized Tailscale auth-flow and client logs for clarity (use `-v` for verbose, ordered output).
- Cleaned up exit-status propagation for both interactive and non-interactive commands.

## How to Upgrade

If you’re already using Go:

    go install github.com/derekg/ts-ssh@v1.0.0

Or clone and build:

    git clone https://github.com/derekg/ts-ssh.git
    cd ts-ssh
    go build -ldflags "-X main.version=1.0.0" -o ts-ssh .

Then verify:

    ./ts-ssh -version
    # => 1.0.0

Enjoy seamless, userspace SSH over your Tailnet!  
For issues or feature requests, please file a ticket on the project’s issue tracker.