# ts-ssh v0.8.0 - SOCKS5 Proxy & Enhanced Compatibility

**Release Date**: January 2026

This release adds SOCKS5 dynamic port forwarding, PTY control, and improved username validation.

## New Features

### SOCKS5 Dynamic Port Forwarding (`-D` flag)
Full SOCKS5 proxy support for tunneling traffic through your Tailscale connection:

```bash
# Start SOCKS5 proxy on localhost:1080
ts-ssh -D 1080 hostname

# Bind to specific address (with security warning)
ts-ssh -D 0.0.0.0:1080 hostname

# Use with curl
curl --socks5 localhost:1080 http://internal-service.example.com
```

**VSCode Remote SSH Compatible** - Use ts-ssh as a SOCKS proxy for VSCode Remote SSH connections to your Tailnet.

### PTY Allocation Control (`-T` flag)
Disable pseudo-terminal allocation for non-interactive commands:

```bash
# Disable PTY for scripted commands
ts-ssh -T hostname "cat /etc/hostname"

# Useful for piping data
ts-ssh -T hostname "cat /var/log/app.log" | grep error
```

### Flexible Username Validation
Usernames with dots are now fully supported:

```bash
ts-ssh first.last@hostname
ts-ssh -l john.doe hostname
```

## CLI Syntax

```bash
# SSH with SOCKS5 proxy
ts-ssh -D 1080 hostname              # SOCKS proxy on localhost:1080
ts-ssh -D 0.0.0.0:1080 hostname      # Bind to all interfaces

# Disable PTY
ts-ssh -T hostname command           # No pseudo-terminal

# All options combined
ts-ssh -v -D 1080 -T user@hostname command
```

## Downloads

### Platform-specific Binaries

- **Linux AMD64**: `ts-ssh-v0.8.0-linux-amd64`
- **Linux ARM64**: `ts-ssh-v0.8.0-linux-arm64`
- **macOS Intel**: `ts-ssh-v0.8.0-darwin-amd64`
- **macOS Apple Silicon**: `ts-ssh-v0.8.0-darwin-arm64`
- **Windows AMD64**: `ts-ssh-v0.8.0-windows-amd64.exe`
- **Windows ARM64**: `ts-ssh-v0.8.0-windows-arm64.exe`
- **FreeBSD AMD64**: `ts-ssh-v0.8.0-freebsd-amd64`
- **OpenBSD AMD64**: `ts-ssh-v0.8.0-openbsd-amd64`

### Verification

Each binary includes a `.sha256` checksum file:

```bash
sha256sum -c ts-ssh-v0.8.0-linux-amd64.sha256
```

## Technical Details

### SOCKS5 Implementation
- Full SOCKS5 protocol (RFC 1928)
- IPv4, IPv6, and domain name resolution
- No authentication required (local use)
- Context-based lifecycle management
- Security validation for bind addresses

### Code Stats
- **Total**: ~5,250 lines (including comprehensive tests)
- **Test coverage**: SOCKS5 functionality fully tested

## Commits

- `276ceef` - Enhanced SOCKS5 implementation and username validation improvements (#35)

## Upgrade Notes

This is a backwards-compatible release. All v0.7.0 commands continue to work.

---

**Full Changelog**: https://github.com/derekg/ts-ssh/compare/v0.7.0...v0.8.0
