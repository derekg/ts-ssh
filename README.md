# ts-ssh: Tailscale tsnet SSH Client

A command-line SSH client written in Go that utilizes `tsnet` to connect to hosts on your Tailscale network without requiring the full Tailscale client daemon to be running.

This allows you to establish a userspace connection to your Tailscale network and then SSH into your nodes directly from this tool.

## Features

*   Connects to your Tailscale network using the userspace `tsnet` library.
*   Handles Tailscale device authentication via a browser-based flow.
*   Supports standard SSH authentication methods:
    *   Public Key (including passphrase-protected keys)
    *   Password (interactive prompt)
*   Provides an interactive SSH session with PTY support.
*   Handles terminal window resizing (`SIGWINCH`).
*   Secure host key verification using `~/.ssh/known_hosts`.
    *   Prompts user to accept and save keys for unknown hosts.
    *   Warns loudly and prevents connection if a known host key changes (potential MITM).
*   Optional insecure mode (`-insecure`) to disable host key checking (Use with extreme caution!).
*   Cross-platform: Can be compiled for Linux, macOS (Intel/ARM), Windows, etc.

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

Connects to a host on your Tailscale network via SSH using tsnet.

Options:
  -i string
        Path to SSH private key (default "~/.ssh/id_rsa")
  -insecure
        Disable host key checking (INSECURE!)
  -l string
        SSH Username (default: current OS user)
  -tsnet-dir string
        Directory to store tsnet state (default "~/.config/ts-ssh-client")
  -W string
        Forward stdio to destination host:port (for use as ProxyCommand)
  -version
        Print version and exit
  -v    Verbose logging
```

**Arguments:**

*   `[user@]hostname[:port]`: The target to connect to.
    *   `hostname` **must** be the Tailscale MagicDNS name or Tailscale IP address of the target machine.
    *   `user` defaults to your current OS username if not provided.
    *   `port` defaults to `22` if not provided.

**Examples:**

*   Connect as current user to `your-server` (MagicDNS name):
    ```bash
    ts-ssh your-server
    ```
*   Connect as user `admin` to `your-server`:
    ```bash
    ts-ssh admin@your-server
    # OR
    ts-ssh -l admin your-server
    ```
*   Connect using a specific private key:
    ```bash
    ts-ssh -i ~/.ssh/my_other_key user@your-server
    ```
*   Connect to a specific Tailscale IP:
    ```bash
    ts-ssh 100.x.y.z
    ```
*   Connect with verbose logging (useful for debugging):
    ```bash
    ts-ssh -v your-server
    ```
*   Connect, disabling host key checks (DANGEROUS - only if you understand the risks!):
    ```bash
    ts-ssh -insecure your-server
    ```
*   Print the client version and exit:
    ```bash
    ts-ssh -version
    ```
  
*   Run a remote command without an interactive shell:
    ```bash
    ts-ssh your-server uname -a
    ```
*   Proxy raw TCP via your tailnet (e.g., for scp or other ProxyCommand uses):
    ```bash
    ts-ssh -W your-server:22
    # Or with scp:
    scp -o ProxyCommand="ts-ssh -W %h:%p" localfile remote:/path
    ```
*   During an interactive session, type `~.` at the start of a line to terminate the session.
  
**Note:**
The Tailscale authentication flow and server logs may interleave in the console during startup, which can be confusing. Use `-v` for more verbose, ordered logging if you need clearer startup output.

## Tailscale Authentication

The first time you run `ts-ssh` on a machine, or if its Tailscale authentication expires, it will need to authenticate to your Tailscale network.

The program will print a URL to the console. Copy this URL and open it in a web browser. Log in to your Tailscale account to authorize this client ("ts-ssh-client" or the hostname set in code).

Once authorized, `ts-ssh` stores authentication keys in the state directory (`~/.config/ts-ssh-client` by default, configurable with `-tsnet-dir`) so you don't need to re-authenticate every time.

## Security Notes

*   **Host Key Verification:** This tool performs host key verification against `~/.ssh/known_hosts` by default. This is a crucial security feature to prevent Man-in-the-Middle (MITM) attacks.
*   **`-insecure` Flag:** The `-insecure` flag disables host key checking entirely. **This is dangerous** and should only be used in trusted environments or for specific testing purposes where you fully understand the security implications. You are vulnerable to MITM attacks if you use this flag carelessly.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

