# ts-ssh v0.5.0 Release Notes

## 🎉 Major User Experience Improvements

### ✨ Clean SSH Connection Experience
**Problem Solved**: Eliminated verbose, distracting tsnet logging that cluttered SSH connections
- ❌ **Before**: `2025/06/30 20:19:36 tsnet running state path /home/derek/.config/ts-ssh/tailscaled.state`
- ❌ **Before**: `2025/06/30 20:25:11 AuthLoop: state is Running; done`
- ✅ **After**: Clean, professional connection output

### 🛠️ Technical Fixes
- **Suppress verbose tsnet logging**: All internal library noise eliminated in normal operation
- **Fix undefined logger references**: Corrected multiple CLI commands with proper logger initialization
- **Improve terminal formatting**: Optimized escape sequence message placement
- **Preserve debugging capabilities**: Full verbose logging still available with `-v` flag

## 📚 Comprehensive Feature Set

### 🌍 **Multi-Language Support (11 Languages)**
Complete internationalization covering 4+ billion speakers worldwide:
- English, Spanish, Chinese, Hindi, Arabic, Bengali, Portuguese, Russian, Japanese, German, French
- All CLI help text, commands, and interface elements fully translated
- Smart language detection from environment or `--lang` flag

### 🔒 **Enterprise-Grade Security**
- Post-quantum cryptography (PQC) support with FIPS 140-2 compliance
- Modern SSH key prioritization (Ed25519 over RSA)
- Comprehensive host key verification
- Security audit logging and monitoring
- Cross-platform security implementations

### 💪 **Powerful Multi-Host Operations**
- **Real tmux integration** for multiple SSH sessions
- **Batch command execution** across hosts (sequential or parallel)
- **Multi-host file distribution** with automatic SCP handling
- **Interactive host picker** with enhanced UX
- **Fast host discovery** with online/offline status

### 🎨 **Modern CLI Experience**
- **Dual CLI modes**: Modern (Fang/Cobra) and Legacy for backward compatibility
- **Beautiful styling** with consistent colors and formatting
- **Enhanced help system** with organized subcommands
- **Automatic CLI detection** for optimal user experience

## 🚀 Installation

### Pre-built Binaries
Download the appropriate binary for your platform:

```bash
# Linux AMD64
curl -L -o ts-ssh https://github.com/derekg/ts-ssh/releases/download/v0.5.0/ts-ssh-v0.5.0-linux-amd64
chmod +x ts-ssh

# macOS Apple Silicon
curl -L -o ts-ssh https://github.com/derekg/ts-ssh/releases/download/v0.5.0/ts-ssh-v0.5.0-darwin-arm64
chmod +x ts-ssh

# macOS Intel
curl -L -o ts-ssh https://github.com/derekg/ts-ssh/releases/download/v0.5.0/ts-ssh-v0.5.0-darwin-amd64
chmod +x ts-ssh

# Windows AMD64
curl -L -o ts-ssh.exe https://github.com/derekg/ts-ssh/releases/download/v0.5.0/ts-ssh-v0.5.0-windows-amd64.exe
```

### Go Install
```bash
go install github.com/derekg/ts-ssh@v0.5.0
```

## ✅ Verification

Verify download integrity with checksums:
```bash
curl -L https://github.com/derekg/ts-ssh/releases/download/v0.5.0/checksums.sha256
sha256sum -c checksums.sha256
```

## 🧪 What's Tested
- ✅ All existing functionality preserved
- ✅ Cross-platform builds (Linux, macOS, Windows - AMD64/ARM64)
- ✅ Comprehensive test suite (73+ tests)
- ✅ Security features validated
- ✅ Multi-language support verified
- ✅ Clean SSH connection experience confirmed

## 🔧 For Developers

### Build from Source
```bash
git clone https://github.com/derekg/ts-ssh.git
cd ts-ssh
go build -o ts-ssh .
```

### Cross-Compilation
```bash
# See CLAUDE.md for detailed cross-compilation examples
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ts-ssh-darwin-arm64 .
```

---

**Full Changelog**: https://github.com/derekg/ts-ssh/compare/v0.4.0...v0.5.0
