# ts-ssh: Simple SSH/SCP Client for Tailscale

A streamlined command-line SSH client and SCP utility that connects to your Tailscale network using `tsnet` - **without requiring a full Tailscale daemon**. Designed with simplicity in mind, ts-ssh mimics the standard `ssh` command with minimal flags and maximum clarity.

**Design Philosophy**: Simplicity over features. Just like ssh, but over Tailscale.

## Features

### 🚀 Core Functionality
- **Userspace Tailscale connection** using `tsnet` - no daemon required
- **Standard SSH syntax**: Works just like regular `ssh`
- **SCP file transfers**: Simple `-scp` flag for file operations
- **Interactive SSH sessions** with full PTY support
- **SOCKS5 dynamic port forwarding**: `-D` flag for proxy support (VSCode Remote SSH compatible)
- **Secure host key verification** using `~/.ssh/known_hosts`
- **Multiple authentication methods**: SSH keys, password prompts
- **Flexible username support**: Allows dots in usernames (e.g., `first.last`)

### 🛠️ Technical Features
- **Cross-platform**: Linux, macOS (Intel/ARM), Windows, FreeBSD, OpenBSD
- **Fast startup** - no frameworks or complex initialization
- **Composable** - works perfectly in scripts and automation
- **Clear error handling** and helpful feedback
- **Enterprise-grade security** features built-in
- **Post-quantum cryptography** support

## Prerequisites

- **Go:** Version 1.18 or later (`go version`)
- **Tailscale Account:** An active Tailscale account
- **Target Node:** A machine within your Tailscale network running an SSH server

## Installation

### Using `go install` (Recommended)

```bash
go install github.com/derekg/ts-ssh@latest
```
*Make sure your `$GOPATH/bin` or `$HOME/go/bin` is in your system's `PATH`*

### Manual Build

1. Clone the repository:
   ```bash
   git clone https://github.com/derekg/ts-ssh.git
   cd ts-ssh
   ```

2. Build the executable:
   ```bash
   go build -o ts-ssh .
   ```

### Cross-Compilation

```bash
# macOS (Apple Silicon)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ts-ssh-darwin-arm64 .

# macOS (Intel)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ts-ssh-darwin-amd64 .

# Linux (amd64)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ts-ssh-linux-amd64 .

# Windows (amd64)
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ts-ssh-windows.exe .

# FreeBSD
CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -o ts-ssh-freebsd .

# OpenBSD
CGO_ENABLED=0 GOOS=openbsd GOARCH=amd64 go build -o ts-ssh-openbsd .
```

## Usage

```
Usage: ts-ssh [options] [user@]host[:port] [command...]
       ts-ssh -scp source dest

SSH over Tailscale without requiring a full Tailscale daemon

Options:
  -D string
        SOCKS5 dynamic port forwarding on [bind_address:]port
  -T    Disable pseudo-terminal allocation
  -control-url string
        Tailscale control server URL
  -i string
        SSH private key path (default "~/.ssh/id_rsa")
  -insecure
        Skip host key verification (insecure)
  -l string
        SSH username (default: current user)
  -p string
        SSH port (default "22")
  -scp
        SCP mode: ts-ssh -scp source dest
  -tsnet-dir string
        Tailscale state directory (default "~/.config/ts-ssh")
  -v    Verbose output
  -version
        Show version
```

## Examples

### Basic SSH Operations

```bash
# Connect to a host
ts-ssh hostname

# Connect as specific user
ts-ssh user@hostname
ts-ssh -l user hostname

# Connect to specific port
ts-ssh hostname:2222
ts-ssh -p 2222 hostname

# Full syntax
ts-ssh user@hostname:2222

# Execute remote command
ts-ssh hostname uptime
ts-ssh user@hostname "ls -la /tmp"

# Use specific SSH key
ts-ssh -i ~/.ssh/custom_key hostname

# Verbose mode (shows Tailscale connection details)
ts-ssh -v hostname
```

### SCP File Transfer

```bash
# Upload file
ts-ssh -scp file.txt hostname:/tmp/
ts-ssh -scp file.txt user@hostname:/remote/path/

# Download file
ts-ssh -scp hostname:/tmp/file.txt ./
ts-ssh -scp user@hostname:/remote/file.txt ./downloads/

# With specific port
ts-ssh -p 2222 -scp file.txt hostname:/tmp/

# With custom key
ts-ssh -i ~/.ssh/custom_key -scp file.txt hostname:/tmp/

# Verbose mode
ts-ssh -v -scp file.txt hostname:/tmp/
```

### Advanced Usage

```bash
# Version information
ts-ssh --version

# Help
ts-ssh --help

# Custom Tailscale state directory
ts-ssh -tsnet-dir ~/my-ts-state hostname

# Skip host key verification (INSECURE - use only for testing)
ts-ssh -insecure hostname

# Custom Tailscale control URL
ts-ssh -control-url https://controlplane.tailscale.com hostname
```

### SOCKS5 Dynamic Port Forwarding

Use the `-D` flag to set up a SOCKS5 proxy for forwarding connections through the SSH tunnel. This is particularly useful for tools like VSCode Remote SSH.

```bash
# Start SOCKS5 proxy on localhost:1080
ts-ssh -D 1080 hostname

# Bind to specific address and port
ts-ssh -D localhost:1080 hostname
ts-ssh -D 127.0.0.1:8080 hostname

# Bind to all interfaces (WARNING: exposes proxy to network)
ts-ssh -D 0.0.0.0:1080 hostname

# Use with verbose mode to see proxy connections
ts-ssh -v -D 1080 hostname

# Combine with other options
ts-ssh -D 1080 -p 2222 user@hostname
```

**Security Notes:**
- Binding to `localhost`, `127.0.0.1`, or `::1` is safe (proxy only accessible locally)
- Binding to `0.0.0.0` or specific network IPs exposes the proxy to your network
- The tool will warn you when binding to non-localhost addresses

### Disable PTY Allocation

Use the `-T` flag to disable pseudo-terminal allocation. Useful for non-interactive commands and automation:

```bash
# Run command without PTY (like ssh -T)
ts-ssh -T hostname "cat /etc/hostname"

# Useful for piping output
ts-ssh -T hostname "journalctl -f" | grep error

# Combine with other flags
ts-ssh -T -p 2222 hostname uptime
```

## Tailscale Authentication

The first time you run `ts-ssh` on a machine, or if its Tailscale authentication expires, it will need to authenticate to your Tailscale network.

The program will print a URL to the console:

```
To authenticate, visit:
https://login.tailscale.com/a/abc123def456
```

Copy this URL and open it in a web browser. Log in to your Tailscale account to authorize this client.

Once authorized, `ts-ssh` stores authentication keys in the state directory (`~/.config/ts-ssh` by default, configurable with `-tsnet-dir`) so you don't need to re-authenticate every time.

**Tip**: Use `-v` (verbose mode) to see detailed authentication and connection information.

## Security

### 🔒 Security Features
- **Modern SSH Key Support**: Ed25519 prioritized over legacy RSA keys
- **Host Key Verification**: Comprehensive verification against `~/.ssh/known_hosts`
- **TTY Security**: Multi-layer validation preventing hijacking attacks
- **Process Protection**: Credential masking in process lists and environment
- **Atomic File Operations**: Race condition prevention in file handling
- **Cross-Platform Security**: Platform-specific implementations for Windows/macOS/Linux
- **Post-Quantum Cryptography**: Future-proof encryption support

### ⚠️ Security Warnings
- **`-insecure` Flag**: Disables host key checking - **USE WITH CAUTION**
- Only use on trusted networks where MITM attacks are not a concern
- The program will warn you before proceeding in insecure mode

For detailed security information, see [Security Documentation](docs/security/)

## Architecture

ts-ssh is built with simplicity and security in mind:

```
ts-ssh/
├── main.go              # ~457 lines - main CLI logic
├── constants.go         # ~52 lines - constants
├── main_test.go         # ~256 lines - unit tests
├── main_e2e_test.go     # ~410 lines - E2E tests
└── internal/
    ├── client/
    │   ├── scp/         # SCP client implementation
    │   └── ssh/         # SSH client implementation
    ├── config/          # Configuration constants
    ├── crypto/pqc/      # Post-quantum cryptography
    │   errors/          # Error handling
    ├── platform/        # Platform-specific code
    └── security/        # Security validation
```

**Total**: ~4,656 lines (69% smaller than previous versions)

### Design Principles
- **Single responsibility**: Each function does one thing well
- **Minimal abstraction**: Avoid over-engineering
- **Clear naming**: Function names describe what they do
- **Explicit is better than implicit**: No magic

## Testing

Comprehensive test suite with 440+ tests covering unit tests, integration tests, and E2E scenarios:

```bash
# Run all tests
go test ./...

# Run with verbose output
go test ./... -v

# Run with coverage
go test ./... -cover

# Run security tests
go test ./... -run "Test.*[Ss]ecure" -v

# Run E2E tests
go test ./... -run "TestE2E" -v

# Check for race conditions
go test ./... -race

# Cross-platform testing
GOOS=windows go test ./...
GOOS=darwin go test ./...
```

**Test Coverage:**
- Overall: 35.5%
- Security modules: 69.4%
- Error handling: 84.6%
- Platform utilities: 100%
- Critical parsing functions: 95-100%

## Comparison to Standard SSH

ts-ssh works just like standard SSH, but connects over your Tailnet:

| Standard SSH | ts-ssh |
|--------------|--------|
| `ssh hostname` | `ts-ssh hostname` |
| `ssh user@hostname` | `ts-ssh user@hostname` |
| `ssh -p 2222 hostname` | `ts-ssh -p 2222 hostname` |
| `ssh hostname command` | `ts-ssh hostname command` |
| `scp file host:/path` | `ts-ssh -scp file host:/path` |

The key difference: ts-ssh uses Tailscale's userspace networking (`tsnet`) so you don't need the Tailscale daemon running.

## Historical Context

This version represents a **major simplification** of the ts-ssh codebase:

- **Before**: ~15,000 lines with dual CLI modes, 11-language internationalization, Charmbracelet UI frameworks, multiple subcommands
- **After**: ~4,656 lines with simple flag-based CLI mimicking standard SSH

The old complex code is preserved in `_old_complex/` for reference.

**What was removed:**
- ❌ Dual CLI modes (modern/legacy)
- ❌ Charmbracelet Fang, Lipgloss, Huh (UI frameworks)
- ❌ Spf13 Cobra (command framework)
- ❌ 11-language internationalization system
- ❌ Multi-host operations (list, exec, multi, copy, pick)
- ❌ Tmux integration

**What was kept:**
- ✅ SSH connections with standard syntax
- ✅ SCP file transfers
- ✅ All security features
- ✅ Post-quantum cryptography
- ✅ Cross-platform support
- ✅ Tailscale tsnet integration

## Contributing

Contributions are welcome! Please ensure:
- Code follows Go best practices
- Tests are included for new features
- Documentation is updated
- Keep it simple - avoid adding complexity

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: https://github.com/derekg/ts-ssh/issues
- **Discussions**: https://github.com/derekg/ts-ssh/discussions

---

**Built with ❤️ for simplicity**
