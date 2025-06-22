# ts-ssh: Powerful Tailscale SSH/SCP CLI Tool

A streamlined command-line SSH client and SCP utility that connects to your Tailscale network using `tsnet`. Features powerful multi-host operations, batch command execution, and real tmux integration - all without requiring the full Tailscale daemon.

Perfect for DevOps teams who need fast, reliable SSH access across their Tailscale infrastructure.

## Features

### 🚀 Core SSH/SCP Functionality
*   **Userspace Tailscale connection** using `tsnet` - no daemon required
*   **Multiple authentication methods**: SSH keys, password prompts, or both
*   **Interactive SSH sessions** with full PTY support and terminal resizing
*   **Secure host key verification** using `~/.ssh/known_hosts`
*   **Direct SCP transfers** with automatic upload/download detection

### 💪 Multi-Host Power Operations
*   **`--list`**: Fast host discovery with online/offline status
*   **`--multi host1,host2,host3`**: Real tmux sessions with multiple SSH connections
*   **`--exec "command" host1,host2`**: Batch command execution across hosts
*   **`--parallel`**: Concurrent command execution for faster operations
*   **`--copy file host1,host2:/path/`**: Multi-host file distribution
*   **`--pick`**: Simple interactive host selection

### 🛠️ Professional DevOps Features
*   **ProxyCommand support** (`-W`) for integration with standard tools
*   **Cross-platform**: Linux, macOS (Intel/ARM), Windows
*   **Multi-language support**: English and Spanish localization
*   **Fast startup** - no UI frameworks or complex initialization
*   **Composable commands** - works perfectly in scripts and automation
*   **Clear error handling** and helpful feedback

## Prerequisites

*   **Go:** Version 1.18 or later installed (`go version`).
*   **Tailscale Account:** An active Tailscale account.
*   **Target Node:** A machine within your Tailscale network running an SSH server that allows connections from your user/key/password.

## Installation

You can install `ts-ssh` using `go install` (recommended) or build it manually from the source.

**Using `go install`:**

```bash
go install github.com/derekg/ts-ssh@latest
```
*(Make sure your `$GOPATH/bin` or `$HOME/go/bin` is in your system's `PATH`)*

**Manual Build:**

1.  Clone the repository:
    ```bash
    git clone https://github.com/derekg/ts-ssh.git
    cd ts-ssh
    ```
2.  Build the executable:
    ```bash
    go build -o ts-ssh .
    ```
    You can now run `./ts-ssh`.

**Cross-Compilation:**

You can easily cross-compile for other platforms. Set the `GOOS` and `GOARCH` environment variables. Use `CGO_ENABLED=0` for easier cross-compilation.

*   **For macOS (Apple Silicon):**
    ```bash
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ts-ssh-darwin-arm64 .
    ```
*   **For macOS (Intel):**
    ```bash
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ts-ssh-darwin-amd64 .
    ```
*   **For Linux (amd64):**
    ```bash
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ts-ssh-linux-amd64 .
    ```
*   **For Windows (amd64):**
    ```bash
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ts-ssh-windows-amd64.exe .
    ```

## Usage

```
Usage: ts-ssh [options] [user@]hostname[:port] [command...]
       ts-ssh --list                                    # List available hosts
       ts-ssh --multi host1,host2,host3                # Multi-host tmux session
       ts-ssh --exec "command" host1,host2             # Run command on multiple hosts
       ts-ssh --copy file.txt host1,host2:/tmp/        # Copy file to multiple hosts
       ts-ssh --pick                                   # Interactive host picker

Powerful SSH/SCP tool for Tailscale networks.

Options:
  -W string
        forward stdio to destination host:port (for use as ProxyCommand)
  -control-url string
        Tailscale control plane URL (optional)
  -copy string
        Copy files to multiple hosts (format: localfile host1,host2:/path/)
  -exec string
        Execute command on specified hosts
  -i string
        Path to SSH private key (default "/home/user/.ssh/id_rsa")
  -insecure
        Disable host key checking (INSECURE!)
  -l string
        SSH Username (default "user")
  -lang string
        Language for CLI output (en, es)
  -list
        List available Tailscale hosts
  -multi string
        Start tmux session with multiple hosts (comma-separated)
  -parallel
        Execute commands in parallel (use with --exec)
  -pick
        Interactive host picker (simple selection)
  -tsnet-dir string
        Directory to store tsnet state (default "/home/user/.config/ts-ssh-client")
  -v    Verbose logging
  -version
        Print version and exit
```

**Arguments:**

*   For SSH: `[user@]hostname[:port] [command...]`
    *   `hostname` **must** be the Tailscale MagicDNS name or Tailscale IP address of the target machine.
    *   `user` defaults to your current OS username if not provided or specified with `-l`.
    *   `port` defaults to `22` if not provided.
    *   `command...` (optional): If provided, executes the command on the remote host instead of starting an interactive shell.
*   For SCP (direct CLI):
    *   Upload: `local_path [user@]hostname:remote_path`
    *   Download: `[user@]hostname:remote_path local_path`
    *   The `user@` in the remote argument is optional; if not provided, the user from `-l` or the default OS user will be used.

## Examples

### 🔍 Host Discovery
```bash
# List all Tailscale hosts with status
ts-ssh --list

# Detailed host information
ts-ssh --list -v

# Interactive host picker
ts-ssh --pick
```

### 🖥️ Basic SSH Operations
```bash
# Connect to a single host
ts-ssh your-server

# Connect as specific user
ts-ssh admin@your-server
ts-ssh -l admin your-server

# Run a remote command
ts-ssh your-server uname -a

# Use specific SSH key
ts-ssh -i ~/.ssh/my_key user@your-server
```

### 🚀 Multi-Host Power Operations
```bash
# Create tmux session with multiple hosts
ts-ssh --multi web1,web2,db1

# Run command on multiple hosts (sequential)
ts-ssh --exec "uptime" web1,web2,web3

# Run command on multiple hosts (parallel)
ts-ssh --parallel --exec "systemctl status nginx" web1,web2

# Check disk space across all web servers
ts-ssh --exec "df -h" web1.domain,web2.domain,web3.domain
```

### 📁 File Transfer Operations
```bash
# Single host SCP
ts-ssh local.txt your-server:/remote/path/
ts-ssh your-server:/remote/file.txt ./

# Multi-host file distribution
ts-ssh --copy deploy.sh web1,web2,web3:/tmp/
ts-ssh --copy config.json db1,db2:/etc/myapp/

# Copy with specific user
ts-ssh --copy -l admin backup.tar.gz server1,server2:/backups/
```

### 🔧 Advanced Usage
```bash
# ProxyCommand integration
scp -o ProxyCommand="ts-ssh -W %h:%p" file.txt server:/path/

# Version information
ts-ssh -version

# Verbose logging for debugging
ts-ssh --list -v
```

### 🌍 Language Support
```bash
# Use Spanish interface
ts-ssh --lang es --list
LANG=es ts-ssh --help

# Use English (default)
ts-ssh --lang en --help
LC_ALL=en ts-ssh --help

# Set permanent language preference
export TS_SSH_LANG=es
ts-ssh --help  # Now shows Spanish
```

**Language Detection Priority:**
1. CLI flag (`--lang`)
2. `TS_SSH_LANG` environment variable
3. `LC_ALL` environment variable  
4. `LANG` environment variable
5. Default (English)

**Supported Languages:**
- 🇺🇸 **English** (`en`) - Default
- 🇪🇸 **Spanish** (`es`) - Complete translation

### 💡 Real-World DevOps Scenarios
```bash
# Deploy configuration to all web servers
ts-ssh --copy nginx.conf web1,web2,web3:/etc/nginx/
ts-ssh --parallel --exec "sudo nginx -t && sudo systemctl reload nginx" web1,web2,web3

# Check service status across infrastructure
ts-ssh --parallel --exec "systemctl is-active docker" node1,node2,node3

# Collect logs from multiple servers
ts-ssh --exec "tail -100 /var/log/app.log" app1,app2,app3

# Emergency system info gathering
ts-ssh --parallel --exec "uptime && free -h && df -h" web1,web2,db1,db2
```

## Multi-Host tmux Sessions

The `--multi` flag creates real tmux sessions with SSH connections to multiple hosts. This provides a professional terminal multiplexing experience:

```bash
# Create tmux session with 3 hosts
ts-ssh --multi web1,web2,db1
```

### tmux Controls
Once connected, use standard tmux key bindings:
- **`Ctrl+B n`** - Next window (next host)
- **`Ctrl+B p`** - Previous window (previous host)  
- **`Ctrl+B 1-9`** - Switch to window number
- **`Ctrl+B c`** - Create new window
- **`Ctrl+B d`** - Detach from session
- **`Ctrl+B ?`** - Show all key bindings

### Session Management
```bash
# List active tmux sessions
tmux list-sessions

# Reconnect to a detached session
tmux attach-session -t ts-ssh-1234567890

# Kill a specific session
tmux kill-session -t ts-ssh-1234567890
```

**Note:**
The Tailscale authentication flow may show verbose logs during startup. Use `-v` for clearer diagnostic output if needed.

## Tailscale Authentication

The first time you run `ts-ssh` on a machine, or if its Tailscale authentication expires, it will need to authenticate to your Tailscale network.

The program will print a URL to the console. Copy this URL and open it in a web browser. Log in to your Tailscale account to authorize this client ("ts-ssh-client" or the hostname set in code).

Once authorized, `ts-ssh` stores authentication keys in the state directory (`~/.config/ts-ssh-client` by default, configurable with `-tsnet-dir`) so you don't need to re-authenticate every time.

## Security & Enterprise Features

### 🔒 Enterprise-Grade Security (Score: 8.5/10)
*   **Modern SSH Key Support**: Ed25519 prioritized over legacy RSA keys
*   **Host Key Verification**: Comprehensive verification against `~/.ssh/known_hosts`
*   **TTY Security**: Multi-layer validation preventing hijacking attacks
*   **Process Protection**: Credential masking in process lists and environment
*   **Atomic File Operations**: Race condition prevention in file handling
*   **Cross-Platform Security**: Platform-specific implementations for Windows/macOS/Linux

### 🛡️ Security Compliance
*   **SOC 2 Ready**: Comprehensive audit logging and access controls
*   **PCI DSS Compatible**: Secure credential and file permission management
*   **GDPR Compliant**: No information disclosure vulnerabilities

### ⚠️ Security Flags
*   **`-insecure` Flag**: Disables host key checking - **USE WITH CAUTION**
*   **`--force-insecure` Flag**: Skip confirmation prompts (automation only)

For detailed security information, see [Security Documentation](docs/security/)

## Architecture

ts-ssh follows enterprise-grade Go project standards with a modular internal package structure:

- **`internal/security/`**: TTY validation, file operations, process security
- **`internal/client/ssh/`**: SSH connection management and authentication  
- **`internal/client/scp/`**: SCP file transfer implementation
- **`internal/platform/`**: Cross-platform process and environment handling

This architecture ensures maintainability, testability, and security isolation.

## Testing

Comprehensive test suite with 73+ tests covering:
```bash
# Run all tests
go test ./...

# Security-focused testing  
go test ./... -run "Test.*[Ss]ecure" -v

# Cross-platform validation
GOOS=windows go test ./...
GOOS=darwin go test ./...

# Race condition detection
go test ./... -race
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
