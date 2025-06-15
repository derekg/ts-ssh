# ts-ssh: Tailscale tsnet SSH Client

A command-line SSH client and SCP utility written in Go that utilizes `tsnet` to connect to hosts on your Tailscale network without requiring the full Tailscale client daemon to be running.

This allows you to establish a userspace connection to your Tailscale network and then SSH into your nodes directly from this tool, or transfer files using SCP.

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
*   **Interactive Text-based User Interface (TUI) mode (`-tui` flag):**
    *   Lists peers on your Tailscale network.
    *   Displays host online/offline status and prevents actions on offline peers.
    *   Allows selection of hosts for SSH or SCP operations.
*   **Direct command-line SCP functionality:**
    *   Supports `local_path user@hostname:remote_path` for uploads.
    *   Supports `user@hostname:remote_path local_path` for downloads.
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
       ts-ssh [options] local_path user@hostname:remote_path
       ts-ssh [options] user@hostname:remote_path local_path

Connects to a host on your Tailscale network via SSH or SCP using tsnet.
Can also launch an interactive TUI with the -tui flag.

Options:
  -i string
        Path to SSH private key (default "~/.ssh/id_rsa")
  -insecure
        Disable host key checking (INSECURE!)
  -l string
        SSH Username (default: current OS user)
  -tsnet-dir string
        Directory to store tsnet state (default "~/.config/ts-ssh-client")
  -tui
        Enable interactive TUI mode
  -W string
        Forward stdio to destination host:port (for use as ProxyCommand)
  -version
        Print version and exit
  -v    Verbose logging
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
*   **Launch the Interactive TUI:**
    ```bash
    ts-ssh -tui
    ```
*   **Direct SCP Upload:**
    ```bash
    ts-ssh mylocalfile.txt your-server:/remote/path/
    # OR with a specific user
    ts-ssh mylocalfile.txt an_user@your-server:/remote/path/
    # OR using -l flag for user
    ts-ssh -l an_user mylocalfile.txt your-server:/remote/path/
    ```
*   **Direct SCP Download:**
    ```bash
    ts-ssh your-server:/remote/file.txt .
    # OR with a specific user
    ts-ssh another_user@your-server:/remote/file.txt /local/destination/
    ```
*   Proxy raw TCP via your tailnet (e.g., for standard `scp` or other `ProxyCommand` uses):
    ```bash
    ts-ssh -W your-server:22
    # Or with standard scp, using ts-ssh as a ProxyCommand:
    scp -o ProxyCommand="ts-ssh -W %h:%p" localfile your-server:/remote/path
    # If 'your-server' in the scp command needs a specific user that ts-ssh's -l flag should handle:
    scp -o ProxyCommand="ts-ssh -l proxyuser -W %h:%p" localfile actualuser@your-server:/remote/path
    ```
*   During an interactive SSH session, type `~.` at the start of a line to terminate the session.
  
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
