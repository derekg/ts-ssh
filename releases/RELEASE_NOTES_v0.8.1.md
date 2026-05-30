# ts-ssh v0.8.1 - Bug Fix Release

**Release Date**: May 2026

Patch release fixing auth URL visibility, SOCKS5 protocol correctness, and an IPv6 bind address parsing bug.

## Bug Fixes

### Auth URL Always Displayed on Re-authentication
The `UserLogf` filter that surfaces Tailscale authentication URLs was matching only `login.tailscale.com`. Re-auth URLs from custom control servers were silently dropped, leaving the user with a hung connection and no URL to visit. The filter now matches any `https://` URL in the user-facing log channel.

### SOCKS5 Protocol Correctness
- **TCP fragmentation**: Raw `Read()` calls assumed the entire SOCKS5 greeting and CONNECT request arrived in a single TCP segment. Replaced with `io.ReadFull` for each protocol segment so fragmented messages are handled correctly under real network conditions.
- **IPv6 bind address**: `ts-ssh -D [::1]:1080` now parses correctly. The old `strings.Split` approach failed on IPv6 addresses; replaced with `net.SplitHostPort` which handles bracket notation properly.
- **Dead context removed**: `context.WithCancel` in `setupDynamicForward` was only ever cancelled by the goroutine's own defer — unreachable from the caller. Removed and replaced with `errors.Is(err, net.ErrClosed)` for the listener shutdown check.
- **IPv6 target address**: `net.JoinHostPort` is now used when constructing the SOCKS5 dial target so IPv6 destination addresses get correct bracket notation.

### Audit Log Version Fixed
Security audit logs were recording version `0.4.0` regardless of the actual binary version. The `internal/security` package now receives the build-time version via `security.SetVersion()` called from `main`.

## CLI Syntax (unchanged)

```bash
# SOCKS5 with IPv6 bind (now works correctly)
ts-ssh -D [::1]:1080 hostname

# Standard usage unchanged
ts-ssh hostname
ts-ssh user@hostname
ts-ssh -D 1080 hostname
ts-ssh -scp file.txt hostname:/tmp/
```

## Downloads

### Platform-specific Binaries

- **Linux AMD64**: `ts-ssh-v0.8.1-linux-amd64`
- **Linux ARM64**: `ts-ssh-v0.8.1-linux-arm64`
- **macOS Intel**: `ts-ssh-v0.8.1-darwin-amd64`
- **macOS Apple Silicon**: `ts-ssh-v0.8.1-darwin-arm64`
- **Windows AMD64**: `ts-ssh-v0.8.1-windows-amd64.exe`
- **Windows ARM64**: `ts-ssh-v0.8.1-windows-arm64.exe`
- **FreeBSD AMD64**: `ts-ssh-v0.8.1-freebsd-amd64`
- **OpenBSD AMD64**: `ts-ssh-v0.8.1-openbsd-amd64`

### Verification

Each binary includes a `.sha256` checksum file:

```bash
sha256sum -c ts-ssh-v0.8.1-linux-amd64.sha256
```

## Commits

- `2af2c1b` - fix: auth URL display and SOCKS5 protocol correctness
- `0cd4051` - fix: use net.SplitHostPort for SOCKS5 bind address parsing

## Upgrade Notes

Drop-in replacement for v0.8.0. No CLI changes.

---

**Full Changelog**: https://github.com/derekg/ts-ssh/compare/v0.8.0...v0.8.1
