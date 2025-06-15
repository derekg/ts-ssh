# ts-ssh 1.0.0 Release Notes

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