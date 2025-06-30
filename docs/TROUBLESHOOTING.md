# ts-ssh Troubleshooting Guide

This guide helps you diagnose and resolve common issues when using ts-ssh.

## Table of Contents
- [Connection Issues](#connection-issues)
- [Authentication Problems](#authentication-problems)
- [Tailscale Issues](#tailscale-issues)
- [CLI Mode Issues](#cli-mode-issues)
- [SSH Key Issues](#ssh-key-issues)
- [Multi-Host Operations](#multi-host-operations)
- [File Transfer Problems](#file-transfer-problems)
- [Performance Issues](#performance-issues)
- [Platform-Specific Issues](#platform-specific-issues)

## Connection Issues

### Problem: "Connection refused" or "Host unreachable"

**Symptoms:**
```
Error: ssh_connect: host: example-host: dial tcp: connection refused
```

**Solutions:**
1. **Verify host is online:**
   ```bash
   ts-ssh list -v  # Check if host shows as online
   ```

2. **Check SSH service:**
   ```bash
   # On the target host, verify SSH is running
   sudo systemctl status ssh    # Ubuntu/Debian
   sudo systemctl status sshd   # CentOS/RHEL
   ```

3. **Verify Tailscale connectivity:**
   ```bash
   tailscale ping example-host  # Test basic Tailscale connectivity
   ```

4. **Check port:**
   ```bash
   ts-ssh connect example-host:2222  # If SSH runs on non-standard port
   ```

### Problem: "Host key verification failed"

**Symptoms:**
```
Error: host key verification failed
```

**Solutions:**
1. **First connection to new host:**
   ```bash
   # Remove old key if host was rebuilt
   ssh-keygen -R example-host
   ssh-keygen -R 100.64.0.1  # Also remove by IP
   ```

2. **Temporarily bypass (DANGEROUS - use only for testing):**
   ```bash
   ts-ssh connect --insecure example-host
   ```

3. **Proper solution - add host key:**
   ```bash
   # Connect once to add key to known_hosts
   ssh example-host  # Using regular SSH first
   ```

## Authentication Problems

### Problem: SSH key authentication fails

**Symptoms:**
```
Error: ssh_auth: user: myuser, host: example-host: ssh: handshake failed
```

**Solutions:**
1. **Check SSH key exists:**
   ```bash
   ls -la ~/.ssh/id_*
   ```

2. **Specify key explicitly:**
   ```bash
   ts-ssh connect --identity ~/.ssh/id_ed25519 user@example-host
   ```

3. **Check key permissions:**
   ```bash
   chmod 600 ~/.ssh/id_rsa
   chmod 700 ~/.ssh
   ```

4. **Test key manually:**
   ```bash
   ssh -i ~/.ssh/id_rsa user@example-host
   ```

5. **Add key to ssh-agent:**
   ```bash
   ssh-add ~/.ssh/id_rsa
   ```

### Problem: Permission denied with correct credentials

**Solutions:**
1. **Check authorized_keys on target host:**
   ```bash
   # On target host
   ls -la ~/.ssh/authorized_keys
   chmod 600 ~/.ssh/authorized_keys
   ```

2. **Verify SSH configuration:**
   ```bash
   # On target host, check /etc/ssh/sshd_config
   sudo grep -E "(PubkeyAuthentication|AuthorizedKeysFile)" /etc/ssh/sshd_config
   ```

3. **Check SSH logs:**
   ```bash
   # On target host
   sudo tail -f /var/log/auth.log    # Ubuntu/Debian
   sudo tail -f /var/log/secure      # CentOS/RHEL
   ```

## Tailscale Issues

### Problem: "tsnet initialization failed"

**Symptoms:**
```
Error: tsnet_init: failed to initialize tsnet
```

**Solutions:**
1. **Check Tailscale authentication:**
   ```bash
   # Clear tsnet state and re-authenticate
   rm -rf ~/.config/ts-ssh-client
   ts-ssh list  # Will prompt for re-authentication
   ```

2. **Use custom tsnet directory:**
   ```bash
   ts-ssh list --tsnet-dir /tmp/ts-ssh-test
   ```

3. **Check network connectivity:**
   ```bash
   ping controlplane.tailscale.com
   ```

### Problem: Authentication URL not accessible

**Solutions:**
1. **Copy URL manually:**
   ```
   Copy the authentication URL and open in browser manually
   ```

2. **Use headless authentication:**
   ```bash
   # Get auth key from Tailscale admin console
   export TS_AUTHKEY="your-auth-key"
   ts-ssh list
   ```

## CLI Mode Issues

### Problem: Modern CLI not working, shows legacy interface

**Solutions:**
1. **Check environment variables:**
   ```bash
   echo $TS_SSH_LEGACY_CLI  # Should be empty for modern CLI
   unset TS_SSH_LEGACY_CLI
   ```

2. **Force modern CLI:**
   ```bash
   TS_SSH_LEGACY_CLI="" ts-ssh --help  # Should show subcommands
   ```

### Problem: Scripts breaking with modern CLI

**Solutions:**
1. **Use legacy mode for scripts:**
   ```bash
   export TS_SSH_LEGACY_CLI=1
   # Your existing scripts will work unchanged
   ```

2. **Update scripts to use modern CLI:**
   ```bash
   # Old: ts-ssh --list
   # New: ts-ssh list
   
   # Old: ts-ssh --exec "uptime" host1,host2
   # New: ts-ssh exec --command "uptime" host1,host2
   ```

## SSH Key Issues

### Problem: Ed25519 keys not being used

**Solutions:**
1. **Check key discovery order:**
   ```bash
   ts-ssh connect --verbose user@host  # Shows which keys are tried
   ```

2. **Generate Ed25519 key:**
   ```bash
   ssh-keygen -t ed25519 -C "your_email@example.com"
   ```

3. **Force specific key type:**
   ```bash
   ts-ssh connect --identity ~/.ssh/id_ed25519 user@host
   ```

## Multi-Host Operations

### Problem: tmux sessions not starting

**Symptoms:**
```
Error: tmux_operation: session_create: tmux not found
```

**Solutions:**
1. **Install tmux:**
   ```bash
   # Ubuntu/Debian
   sudo apt update && sudo apt install tmux
   
   # macOS
   brew install tmux
   
   # CentOS/RHEL
   sudo yum install tmux
   ```

2. **Check tmux is in PATH:**
   ```bash
   which tmux
   tmux -V
   ```

### Problem: Parallel execution not working

**Solutions:**
1. **Verify hosts are reachable:**
   ```bash
   ts-ssh list  # Check all target hosts are online
   ```

2. **Test sequential first:**
   ```bash
   # Test without --parallel first
   ts-ssh exec --command "uptime" host1,host2
   
   # Then add --parallel
   ts-ssh exec --parallel --command "uptime" host1,host2
   ```

## File Transfer Problems

### Problem: SCP transfers failing

**Solutions:**
1. **Check local file exists:**
   ```bash
   ls -la /path/to/local/file
   ```

2. **Verify remote directory exists:**
   ```bash
   ts-ssh connect host "ls -la /path/to/remote/directory"
   ```

3. **Check permissions:**
   ```bash
   # On target host
   ls -la /path/to/remote/directory
   chmod 755 /path/to/remote/directory
   ```

4. **Test with single host first:**
   ```bash
   # Test single host transfer first
   ts-ssh copy file.txt host1:/tmp/
   
   # Then multi-host
   ts-ssh copy file.txt host1,host2:/tmp/
   ```

## Performance Issues

### Problem: Slow connections

**Solutions:**
1. **Enable verbose logging to identify bottlenecks:**
   ```bash
   ts-ssh connect --verbose user@host
   ```

2. **Check Tailscale route:**
   ```bash
   tailscale status
   tailscale netcheck
   ```

3. **Test direct connection:**
   ```bash
   # Compare with direct SSH
   time ssh user@host "echo test"
   time ts-ssh connect user@host -- echo test
   ```

### Problem: High memory usage

**Solutions:**
1. **Check for memory leaks:**
   ```bash
   # Monitor memory usage
   top -p $(pgrep ts-ssh)
   ```

2. **Reduce concurrent connections:**
   ```bash
   # Reduce batch size for multi-host operations
   ts-ssh exec --command "uptime" host1,host2,host3  # Instead of many hosts
   ```

## Platform-Specific Issues

### Windows Issues

**Problem: Terminal not resizing properly**
```bash
# Use Windows Terminal or PowerShell 7+
# Ensure TERM environment variable is set
set TERM=xterm-256color
```

**Problem: SSH keys not found**
```bash
# Specify full Windows path
ts-ssh connect --identity C:\Users\username\.ssh\id_rsa user@host
```

### macOS Issues

**Problem: Permission denied on macOS Catalina+**
```bash
# Grant Full Disk Access to Terminal in System Preferences
# Or specify explicit paths
ts-ssh connect --identity /Users/username/.ssh/id_rsa user@host
```

### Linux Issues

**Problem: SELinux blocking connections**
```bash
# Check SELinux status
sestatus

# Allow SSH in SELinux
sudo setsebool -P ssh_sysadm_login on
```

## Debugging Commands

### Enable Debug Mode
```bash
# Modern CLI
ts-ssh connect --verbose user@host

# Legacy CLI  
ts-ssh --verbose user@host
```

### Check Configuration
```bash
# List hosts with detailed info
ts-ssh list --verbose

# Test specific host connectivity
ts-ssh connect --dry-run user@host
```

### Collect Debug Information
```bash
# Create debug log
ts-ssh list --verbose > debug.log 2>&1

# System information
uname -a >> debug.log
go version >> debug.log
echo "Tailscale status:" >> debug.log
tailscale status >> debug.log
```

## Getting Help

If you continue to experience issues:

1. **Check GitHub Issues:** https://github.com/derekg/ts-ssh/issues
2. **Create detailed issue with:**
   - Operating system and version
   - ts-ssh version (`ts-ssh version`)
   - Complete error message
   - Steps to reproduce
   - Debug logs (`ts-ssh --verbose`)

3. **Include environment info:**
   ```bash
   ts-ssh version
   go version
   tailscale version
   uname -a
   ```