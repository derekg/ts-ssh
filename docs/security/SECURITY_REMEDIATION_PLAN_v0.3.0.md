# **SECURITY REMEDIATION PLAN: ts-ssh v0.3.0**
## Priority 1 Critical Vulnerabilities Action Plan

**Plan Date**: 2025-06-21  
**Target Version**: v0.4.0 (Security Hardened)  
**Security Audit Reference**: SECURITY_AUDIT_v0.3.0.md  
**Planned Completion**: 1 week

---

## **🎯 EXECUTIVE SUMMARY**

This document outlines a comprehensive remediation plan for the 4 critical Priority 1 security vulnerabilities identified in our security audit. These vulnerabilities must be addressed before any production deployment. Each fix includes detailed implementation steps, testing requirements, and validation criteria.

**Current Security Score**: 6.2/10  
**Target Security Score**: 8.5/10 (after Priority 1 fixes)

---

## **🔴 PRIORITY 1 VULNERABILITIES - DETAILED REMEDIATION**

### **CVE-TS-SSH-001: Host Key Verification Bypass**
**CVSS Score**: 8.1 (High) | **Target Completion**: Day 1

#### **Current Issue**
- `--insecure` flag completely disables host key verification
- No warnings or confirmation when bypassing security
- Silent MITM vulnerability exposure

#### **Remediation Plan**

**Step 1: Add Security Warnings**
```go
// Location: main.go, ssh_client.go
// Add prominent warnings when insecure mode is used

func validateInsecureMode() error {
    if *insecureHostKey {
        fmt.Fprintf(os.Stderr, "⚠️  WARNING: Host key verification disabled!\n")
        fmt.Fprintf(os.Stderr, "⚠️  This makes you vulnerable to man-in-the-middle attacks.\n")
        fmt.Fprintf(os.Stderr, "⚠️  Only use this in trusted network environments.\n")
        
        if !*forceInsecure {
            fmt.Print("Continue with insecure connection? [y/N]: ")
            reader := bufio.NewReader(os.Stdin)
            response, _ := reader.ReadString('\n')
            if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(response)), "y") {
                return fmt.Errorf("connection cancelled by user")
            }
        }
    }
    return nil
}
```

**Step 2: Add Force Flag**
```go
// Add new CLI flag for automation scenarios
var forceInsecure = flag.Bool("force-insecure", false, "Skip confirmation for insecure connections (automation only)")
```

**Step 3: Enhanced Host Key Verification**
```go
// Implement graduated security levels
type HostKeyVerificationLevel int

const (
    SecureVerification HostKeyVerificationLevel = iota
    WarnAndProceed
    InsecureBypass
)

func createSecureHostKeyCallback(level HostKeyVerificationLevel, currentUser *user.User, logger *log.Logger) ssh.HostKeyCallback {
    switch level {
    case SecureVerification:
        return CreateKnownHostsCallback(currentUser, logger)
    case WarnAndProceed:
        return ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error {
            logger.Printf("⚠️  WARNING: Host key not verified for %s", hostname)
            return nil
        })
    case InsecureBypass:
        return ssh.InsecureIgnoreHostKey()
    }
}
```

**Testing Requirements**:
- Verify warnings appear in all scenarios
- Test confirmation prompt functionality
- Validate force flag behavior for automation

---

### **CVE-TS-SSH-002: Credentials Exposed in Process Lists**
**CVSS Score**: 8.6 (High) | **Target Completion**: Day 2

#### **Current Issue**
- SSH commands with credentials visible in `ps aux`
- Tmux sessions expose connection strings
- Authentication methods visible in process arguments

#### **Remediation Plan**

**Step 1: Secure Command Line Argument Handling**
```go
// Location: tmux_manager.go:165-178, power_cli.go:328
// Remove credentials from command line arguments

func buildSecureSSHCommand(host string, tempConfigFile string) string {
    // Use SSH config file instead of command line arguments
    var cmdParts []string
    cmdParts = append(cmdParts, os.Args[0])
    cmdParts = append(cmdParts, "-F", tempConfigFile) // Use config file
    cmdParts = append(cmdParts, host)
    return strings.Join(cmdParts, " ")
}

func createTemporarySSHConfig(sshUser, sshKeyPath string, insecureHostKey bool) (string, error) {
    tempFile, err := os.CreateTemp("", "ts-ssh-config-*.conf")
    if err != nil {
        return "", err
    }
    
    // Set restrictive permissions immediately
    if err := tempFile.Chmod(0600); err != nil {
        tempFile.Close()
        os.Remove(tempFile.Name())
        return "", err
    }
    
    config := fmt.Sprintf(`
Host *
    User %s
    IdentityFile %s
    StrictHostKeyChecking %s
    UserKnownHostsFile %s
    LogLevel QUIET
`, sshUser, sshKeyPath, 
    map[bool]string{true: "no", false: "yes"}[insecureHostKey],
    map[bool]string{true: "/dev/null", false: "~/.ssh/known_hosts"}[insecureHostKey])
    
    if _, err := tempFile.WriteString(config); err != nil {
        tempFile.Close()
        os.Remove(tempFile.Name())
        return "", err
    }
    
    tempFile.Close()
    return tempFile.Name(), nil
}
```

**Step 2: Process Title Masking**
```go
// Implement process title masking to hide sensitive information
func maskProcessTitle() {
    if runtime.GOOS == "linux" {
        // Use prctl to set process title
        title := "ts-ssh [secure connection]"
        setProcessTitle(title)
    }
}

// Platform-specific implementation
// +build linux
func setProcessTitle(title string) {
    // Implementation for Linux process title masking
    syscall.Syscall(syscall.SYS_PRCTL, 15, uintptr(unsafe.Pointer(&title[0])), 0)
}
```

**Step 3: Secure Environment Variable Handling**
```go
// Use environment variables for sensitive data instead of command line
func setupSecureEnvironment(sshKeyPath string) map[string]string {
    env := make(map[string]string)
    if sshKeyPath != "" {
        env["TS_SSH_KEY_PATH"] = sshKeyPath
    }
    return env
}
```

**Testing Requirements**:
- Verify no credentials appear in `ps aux` output
- Test temporary config file cleanup
- Validate process title masking across platforms

---

### **CVE-TS-SSH-003: File Permission Race Condition**
**CVSS Score**: 7.8 (High) | **Target Completion**: Day 3

#### **Current Issue**
- Known_hosts and SCP files created with default permissions
- Race condition window between file creation and permission setting
- Potential for credential theft during file creation

#### **Remediation Plan**

**Step 1: Atomic File Creation with Secure Permissions**
```go
// Location: ssh_client.go:115-118, scp_client.go:143
// Implement atomic file creation with secure permissions

func createSecureFile(filename string, mode os.FileMode) (*os.File, error) {
    // Create file with restrictive permissions atomically
    file, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
    if err != nil {
        return nil, fmt.Errorf("failed to create secure file %s: %w", filename, err)
    }
    
    // Verify permissions were set correctly
    info, err := file.Stat()
    if err != nil {
        file.Close()
        os.Remove(filename)
        return nil, fmt.Errorf("failed to verify file permissions: %w", err)
    }
    
    if info.Mode() != mode {
        file.Close()
        os.Remove(filename)
        return nil, fmt.Errorf("file permissions not set correctly: expected %v, got %v", mode, info.Mode())
    }
    
    return file, nil
}
```

**Step 2: Secure Known Hosts Management**
```go
// Replace existing known_hosts handling
func createSecureKnownHostsFile(knownHostsPath string) error {
    // Ensure parent directory exists with secure permissions
    dir := filepath.Dir(knownHostsPath)
    if err := os.MkdirAll(dir, 0700); err != nil {
        return fmt.Errorf("failed to create ssh directory: %w", err)
    }
    
    // Create known_hosts file atomically with secure permissions
    file, err := createSecureFile(knownHostsPath, 0600)
    if err != nil {
        if os.IsExist(err) {
            // File already exists, verify permissions
            return verifyFilePermissions(knownHostsPath, 0600)
        }
        return err
    }
    defer file.Close()
    
    // Write initial content if needed
    _, err = file.WriteString("# SSH Known Hosts managed by ts-ssh\n")
    return err
}

func verifyFilePermissions(filename string, expectedMode os.FileMode) error {
    info, err := os.Stat(filename)
    if err != nil {
        return err
    }
    
    if info.Mode() != expectedMode {
        return os.Chmod(filename, expectedMode)
    }
    return nil
}
```

**Step 3: Secure SCP File Operations**
```go
// Implement secure file transfer with atomic operations
func secureFileCopy(src, dst string) error {
    // Create temporary file with secure permissions
    tempFile := dst + ".tmp." + generateRandomSuffix()
    
    file, err := createSecureFile(tempFile, 0600)
    if err != nil {
        return err
    }
    defer func() {
        file.Close()
        os.Remove(tempFile) // Cleanup on error
    }()
    
    // Perform copy operation
    if err := copyFileContent(src, file); err != nil {
        return err
    }
    
    // Close before rename
    if err := file.Close(); err != nil {
        return err
    }
    
    // Atomic rename
    return os.Rename(tempFile, dst)
}
```

**Testing Requirements**:
- Test file creation under concurrent access
- Verify permissions are set atomically
- Test cleanup of temporary files on errors

---

### **CVE-TS-SSH-004: Unsafe TTY Access**
**CVSS Score**: 7.4 (High) | **Target Completion**: Day 4

#### **Current Issue**
- Direct `/dev/tty` access without validation
- No protection against TTY hijacking
- Input redirection vulnerabilities

#### **Remediation Plan**

**Step 1: Secure TTY Validation**
```go
// Location: utils.go:59
// Replace direct /dev/tty access with secure validation

func getSecureTTY() (*os.File, error) {
    // First, verify we're running in a real terminal
    if !term.IsTerminal(int(os.Stdin.Fd())) {
        return nil, fmt.Errorf("not running in a terminal")
    }
    
    // Verify TTY ownership and permissions
    ttyPath, err := getTTYPath()
    if err != nil {
        return nil, fmt.Errorf("failed to get TTY path: %w", err)
    }
    
    if err := validateTTYSecurity(ttyPath); err != nil {
        return nil, fmt.Errorf("TTY security validation failed: %w", err)
    }
    
    // Open TTY with explicit permissions check
    ttyFile, err := os.OpenFile(ttyPath, os.O_RDWR, 0)
    if err != nil {
        return nil, fmt.Errorf("failed to open TTY: %w", err)
    }
    
    return ttyFile, nil
}

func getTTYPath() (string, error) {
    // Get TTY name from controlling terminal
    if ttyname := os.Getenv("TTY"); ttyname != "" {
        return ttyname, nil
    }
    
    // Fallback to /dev/tty if safe
    if _, err := os.Stat("/dev/tty"); err == nil {
        return "/dev/tty", nil
    }
    
    return "", fmt.Errorf("no TTY available")
}

func validateTTYSecurity(ttyPath string) error {
    info, err := os.Stat(ttyPath)
    if err != nil {
        return err
    }
    
    // Check ownership
    stat := info.Sys().(*syscall.Stat_t)
    currentUID := uint32(os.Getuid())
    
    if stat.Uid != currentUID {
        return fmt.Errorf("TTY not owned by current user")
    }
    
    // Check permissions (should not be world-readable/writable)
    mode := info.Mode()
    if mode&0077 != 0 {
        return fmt.Errorf("TTY has unsafe permissions: %v", mode)
    }
    
    return nil
}
```

**Step 2: Secure Password Input**
```go
// Replace direct password reading with secure implementation
func readPasswordSecurely() (string, error) {
    tty, err := getSecureTTY()
    if err != nil {
        return "", fmt.Errorf("cannot access secure TTY: %w", err)
    }
    defer tty.Close()
    
    // Save terminal state
    fd := int(tty.Fd())
    oldState, err := term.GetState(fd)
    if err != nil {
        return "", fmt.Errorf("failed to get terminal state: %w", err)
    }
    defer term.Restore(fd, oldState)
    
    // Read password
    password, err := term.ReadPassword(fd)
    if err != nil {
        return "", fmt.Errorf("failed to read password: %w", err)
    }
    
    return string(password), nil
}
```

**Step 3: TTY State Management**
```go
// Implement proper TTY state cleanup
func withSecureTTY(fn func(*os.File) error) error {
    tty, err := getSecureTTY()
    if err != nil {
        return err
    }
    defer func() {
        // Ensure TTY is properly restored
        fd := int(tty.Fd())
        if state, err := term.GetState(fd); err == nil {
            term.Restore(fd, state)
        }
        tty.Close()
    }()
    
    return fn(tty)
}
```

**Testing Requirements**:
- Test TTY validation across different environments
- Verify protection against TTY hijacking
- Test terminal state restoration

---

## **📋 IMPLEMENTATION TIMELINE**

### **Day 1: CVE-TS-SSH-001 (Host Key Bypass)**
- [ ] Implement security warnings
- [ ] Add confirmation prompts
- [ ] Create force flag for automation
- [ ] Test across all connection methods

### **Day 2: CVE-TS-SSH-002 (Credential Exposure)**
- [ ] Implement SSH config file approach
- [ ] Add process title masking
- [ ] Secure environment variable handling
- [ ] Test process visibility

### **Day 3: CVE-TS-SSH-003 (File Race Conditions)**
- [ ] Implement atomic file creation
- [ ] Secure known_hosts management
- [ ] Fix SCP file operations
- [ ] Test concurrent access scenarios

### **Day 4: CVE-TS-SSH-004 (TTY Security)**
- [ ] Implement TTY validation
- [ ] Secure password input methods
- [ ] TTY state management
- [ ] Test across terminal environments

### **Day 5: Integration & Testing**
- [ ] Integration testing of all fixes
- [ ] Performance impact assessment
- [ ] Cross-platform compatibility testing
- [ ] Security validation testing

### **Day 6-7: Final Validation**
- [ ] Comprehensive security testing
- [ ] Documentation updates
- [ ] Release preparation
- [ ] Security score re-assessment

---

## **🧪 TESTING STRATEGY**

### **Security Test Suite**
1. **Host Key Bypass Tests**
   - Verify warnings appear consistently
   - Test confirmation prompt handling
   - Validate force flag behavior

2. **Credential Exposure Tests**
   - Monitor process lists during connections
   - Test temporary file cleanup
   - Verify environment variable security

3. **File Race Condition Tests**
   - Concurrent file access testing
   - Permission verification
   - Atomic operation validation

4. **TTY Security Tests**
   - TTY ownership validation
   - Terminal state management
   - Input security verification

### **Automated Security Checks**
```bash
# Add to CI/CD pipeline
./security-test-suite.sh
- credential_exposure_test
- file_permission_test  
- tty_security_test
- host_key_bypass_test
```

---

## **📊 SUCCESS CRITERIA**

### **Security Score Improvement**
- **Current**: 6.2/10
- **Target**: 8.5/10
- **Minimum Acceptable**: 8.0/10

### **Vulnerability Status**
- CVE-TS-SSH-001: ✅ Mitigated with warnings and confirmation
- CVE-TS-SSH-002: ✅ Fixed with config file approach
- CVE-TS-SSH-003: ✅ Resolved with atomic file operations
- CVE-TS-SSH-004: ✅ Secured with TTY validation

### **Production Readiness Checklist**
- [ ] All Priority 1 vulnerabilities fixed
- [ ] Security test suite passing
- [ ] Performance impact < 5%
- [ ] Cross-platform compatibility maintained
- [ ] Documentation updated

---

## **🔄 POST-REMEDIATION ACTIVITIES**

1. **Security Re-assessment**
   - Run updated STRIDE analysis
   - Update security score
   - Document remaining risks

2. **Documentation Updates**
   - Update README with security features
   - Create security best practices guide
   - Update CLI help text

3. **Release Management**
   - Tag as v0.4.0 (Security Hardened)
   - Create detailed release notes
   - Publish security advisory

4. **Monitoring & Maintenance**
   - Set up security monitoring
   - Schedule regular security reviews
   - Plan Priority 2 vulnerability fixes

---

**Document Version**: 1.0  
**Last Updated**: 2025-06-21  
**Next Review**: After Priority 1 implementation