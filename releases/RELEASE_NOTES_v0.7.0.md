# ts-ssh v0.7.0 - Major CLI Simplification

**Release Date**: November 4, 2025

This release represents a **major simplification** of ts-ssh, transforming it from a complex multi-modal CLI to a simple, SSH-like command.

## 🎯 Design Philosophy

**"Simplicity over features"** - This tool does one thing well: SSH and SCP over Tailscale networks, with a familiar, SSH-like interface.

## 📊 Major Changes

### Code Reduction
- **From**: ~15,000 lines of code
- **To**: ~4,656 lines of code
- **Reduction**: 69% smaller codebase

### Simplified CLI
Now mimics standard SSH command syntax - no subcommands, no complexity.

**Before** (v0.5.0):
```bash
ts-ssh connect hostname
ts-ssh --multi host1,host2,host3
ts-ssh --exec "uptime" --list
ts-ssh --copy file.txt --list
```

**After** (v0.7.0):
```bash
ts-ssh hostname
ts-ssh hostname uptime
ts-ssh -scp file.txt hostname:/tmp/
```

### Removed Features
- ❌ Dual CLI modes (modern/legacy)
- ❌ 11-language internationalization system
- ❌ Multi-host operations (`--list`, `--multi`, `--exec`, `--copy`, `--pick`)
- ❌ Tmux integration
- ❌ Charmbracelet UI frameworks (Fang, Lipgloss, Huh, Cobra)
- ❌ Complex subcommand structure

## ✅ What Remains

All core functionality is preserved:

- ✅ SSH connections with standard syntax
- ✅ SCP file transfers
- ✅ Tailscale tsnet integration (userspace networking)
- ✅ All security features and validation
- ✅ Post-quantum cryptography support
- ✅ Host key verification
- ✅ Multiple authentication methods
- ✅ Cross-platform support (Linux, macOS, Windows, BSD)

## 🚀 CLI Syntax

### SSH Operations
```bash
# Connect to a host
ts-ssh hostname
ts-ssh user@hostname
ts-ssh user@hostname:2222

# Execute remote command
ts-ssh hostname uptime
ts-ssh user@hostname "ls -la /tmp"

# Options
ts-ssh -v hostname                  # Verbose mode
ts-ssh -p 2222 hostname            # Custom port
ts-ssh -l alice hostname           # Specify username
ts-ssh -i ~/.ssh/custom_key hostname  # Custom key
```

### SCP Operations
```bash
# Upload file
ts-ssh -scp file.txt hostname:/tmp/

# Download file
ts-ssh -scp hostname:/tmp/file.txt ./

# With specific port/user
ts-ssh -p 2222 -scp file.txt user@hostname:/tmp/
```

### Help & Version
```bash
ts-ssh --help
ts-ssh --version
```

## ⚠️ Breaking Changes

This is a **BREAKING** release. If you rely on removed features:

### Migration Guidance

**Multi-host operations** → Use shell loops:
```bash
# Old: ts-ssh --exec "uptime" --multi host1,host2,host3
# New:
for host in host1 host2 host3; do
  ts-ssh $host uptime
done
```

**Parallel execution** → Use GNU parallel or xargs:
```bash
# Parallel command execution
echo "host1 host2 host3" | xargs -P 3 -n 1 ts-ssh -c uptime

# Or with GNU parallel
parallel ts-ssh {} uptime ::: host1 host2 host3
```

**Internationalization** → English only
- Non-English users should stay on v0.5.0 or use terminal translation tools

**Tmux integration** → Use tmux directly
- Launch tmux manually and run ts-ssh within it

## 📦 Downloads

### Platform-specific Binaries

- **Linux AMD64**: `ts-ssh-v0.7.0-linux-amd64`
- **Linux ARM64**: `ts-ssh-v0.7.0-linux-arm64`
- **macOS Intel**: `ts-ssh-v0.7.0-darwin-amd64`
- **macOS Apple Silicon**: `ts-ssh-v0.7.0-darwin-arm64`
- **Windows AMD64**: `ts-ssh-v0.7.0-windows-amd64.exe`
- **Windows ARM64**: `ts-ssh-v0.7.0-windows-arm64.exe`
- **FreeBSD AMD64**: `ts-ssh-v0.7.0-freebsd-amd64`
- **OpenBSD AMD64**: `ts-ssh-v0.7.0-openbsd-amd64`

### All Platforms Archive

- **All platforms**: `ts-ssh-v0.7.0-all-platforms.tar.gz` (127 MB)

### Verification

Each binary includes a `.sha256` checksum file. Verify downloads:

```bash
sha256sum -c ts-ssh-v0.7.0-linux-amd64.sha256
```

## 🧪 Testing

- ✅ All tests passing (440+ tests)
- ✅ Cross-platform builds verified
- ✅ Security features validated
- ✅ Test coverage: 35.5%

## 🏗️ Technical Details

### Removed Dependencies
- `github.com/charmbracelet/fang`
- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/huh`
- `github.com/spf13/cobra`
- 11-language translation files
- TUI frameworks and dependencies

### Code Structure (Simplified)
```
ts-ssh/
├── main.go              # ~457 lines - main CLI logic
├── constants.go         # ~52 lines
├── main_test.go         # ~256 lines
├── main_e2e_test.go     # ~410 lines
└── internal/
    ├── client/          # SSH and SCP clients
    ├── config/          # Configuration
    ├── crypto/pqc/      # Post-quantum cryptography
    ├── errors/          # Error handling
    ├── platform/        # Platform-specific code
    └── security/        # Security validation
```

**Total**: ~4,656 lines (down from ~15,000)

## 🔐 Security

All security features maintained:
- Host key verification
- Secure TTY handling for passwords
- Input validation (SQL injection, command injection protection)
- Post-quantum cryptography support
- Secure file transfers with atomic operations

## 📝 Commits

- `56da8af` - chore: Remove dead code and cleanup codebase (#30)
- `a600998` - Simplify CLI to mimic standard SSH command (#29)
- `8f108b4` - Design Simplified SSH Command Implementation (#28)

## 🙏 Acknowledgments

This simplification was driven by the principle that **tools should do one thing well**. By focusing on core SSH/SCP functionality over Tailscale, ts-ssh is now more maintainable, easier to understand, and more aligned with Unix philosophy.

---

**Full Changelog**: https://github.com/derekg/ts-ssh/compare/v0.5.0...v0.7.0
