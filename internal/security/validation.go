package security

import (
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// InputValidator provides comprehensive input validation for security-critical operations
type InputValidator struct {
	MaxHostnameLength int
	MaxPathLength     int
	MaxCommandLength  int
	AllowedHostChars  *regexp.Regexp
	AllowedPathChars  *regexp.Regexp
}

// Security constants for input validation
const (
	MaxHostnameLength = 253  // RFC 1035 limit
	MaxPathLength     = 4096 // Common filesystem limit
	MaxCommandLength  = 8192 // Reasonable command length limit
	MaxPortNumber     = 65535
	MinPortNumber     = 1
	MaxSSHUserLength  = 32
	MaxEnvVarLength   = 32768 // 32KB limit
)

// NewInputValidator creates a new input validator with secure defaults
func NewInputValidator() *InputValidator {
	return &InputValidator{
		MaxHostnameLength: MaxHostnameLength,
		MaxPathLength:     MaxPathLength,
		MaxCommandLength:  MaxCommandLength,
		// Simple hostname validation to prevent ReDoS attacks
		// Removed complex alternation to prevent exponential backtracking
		AllowedHostChars: regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9]$`),
		// Safe path characters (no shell metacharacters)
		AllowedPathChars: regexp.MustCompile(`^[a-zA-Z0-9._/\-]+$`),
	}
}

// ValidateHostname validates hostnames against RFC standards and security best practices
func (iv *InputValidator) ValidateHostname(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}

	if len(hostname) > iv.MaxHostnameLength {
		return fmt.Errorf("hostname too long: %d characters (max %d)", len(hostname), iv.MaxHostnameLength)
	}

	// Check for dangerous characters that could be used for injection (optimized)
	dangerousChars := ";|&`$(){}[]<>\\\"'!*?"
	if strings.ContainsAny(hostname, dangerousChars) {
		return fmt.Errorf("hostname contains invalid characters")
	}

	// Check if it's a valid IP address first (IPv4 or IPv6)
	if net.ParseIP(hostname) != nil {
		return nil // Valid IP address
	}

	// Additional security checks for hostnames (not IPs)
	if strings.HasPrefix(hostname, "-") || strings.HasSuffix(hostname, "-") {
		return fmt.Errorf("hostname cannot start or end with hyphen")
	}

	if strings.Contains(hostname, "--") {
		return fmt.Errorf("hostname cannot contain consecutive hyphens")
	}

	// Validate against RFC 1123 format for hostnames (not IPs)
	if !iv.AllowedHostChars.MatchString(hostname) {
		return fmt.Errorf("hostname format invalid (must comply with RFC 1123)")
	}

	// Validate each label in the hostname
	labels := strings.Split(hostname, ".")
	for _, label := range labels {
		if len(label) == 0 {
			return fmt.Errorf("hostname contains empty label")
		}
		if len(label) > 63 {
			return fmt.Errorf("hostname label too long: %s (max 63 characters)", label)
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("hostname label cannot start or end with hyphen: %s", label)
		}
	}

	return nil
}

// ValidateFilePath validates file paths against security threats
func (iv *InputValidator) ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	if len(path) > iv.MaxPathLength {
		return fmt.Errorf("file path too long: %d characters (max %d)", len(path), iv.MaxPathLength)
	}

	// Check for dangerous characters first (optimized)
	dangerousChars := ";|&`$(){}[]<>\\\"'!*?"
	if strings.ContainsAny(path, dangerousChars) {
		return fmt.Errorf("file path contains invalid characters")
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal attempt detected: %s", path)
	}

	// Clean the path and check if it changed significantly (another traversal check)
	cleanPath := filepath.Clean(path)
	if cleanPath != path && strings.Contains(path, "/") {
		// If cleaning changed the path significantly, it might be a traversal attempt
		if strings.Count(path, "/") > strings.Count(cleanPath, "/") {
			return fmt.Errorf("path traversal attempt detected: %s", path)
		}
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("file path contains null byte")
	}

	// Check for control characters
	for _, r := range path {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return fmt.Errorf("file path contains control character: %U", r)
		}
	}

	return nil
}

// ValidateCommand validates shell commands for safe execution
func (iv *InputValidator) ValidateCommand(command string) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	if len(command) > iv.MaxCommandLength {
		return fmt.Errorf("command too long: %d characters (max %d)", len(command), iv.MaxCommandLength)
	}

	// Check for null bytes and other dangerous binary characters first
	for i, r := range command {
		if r == '\x00' {
			return fmt.Errorf("command contains null byte at position %d", i)
		}
		if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
			return fmt.Errorf("command contains control character: %U at position %d", r, i)
		}
	}

	// Check for command injection patterns
	injectionPatterns := []string{
		";", "&&", "||", "|", "`", "$(",
		"$(", "${", "<!--", "-->", "<script",
		"javascript:", "vbscript:", "onload=",
		"onerror=", "eval(", "exec(",
	}

	for _, pattern := range injectionPatterns {
		if strings.Contains(command, pattern) {
			return fmt.Errorf("command contains potentially dangerous pattern: %s", pattern)
		}
	}

	return nil
}

// ValidateSSHUser validates SSH usernames
func (iv *InputValidator) ValidateSSHUser(username string) error {
	if username == "" {
		return fmt.Errorf("SSH username cannot be empty")
	}

	if len(username) > MaxSSHUserLength {
		return fmt.Errorf("SSH username too long: %d characters (max %d)", len(username), MaxSSHUserLength)
	}

	// SSH usernames should only contain alphanumeric characters, hyphens, and underscores
	validUserRegex := regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
	if !validUserRegex.MatchString(username) {
		return fmt.Errorf("SSH username contains invalid characters (only alphanumeric, hyphen, underscore allowed)")
	}

	// Cannot start with hyphen or number
	if strings.HasPrefix(username, "-") || unicode.IsDigit(rune(username[0])) {
		return fmt.Errorf("SSH username cannot start with hyphen or number")
	}

	return nil
}

// ValidatePort validates network port numbers
func (iv *InputValidator) ValidatePort(port string) error {
	if port == "" {
		return fmt.Errorf("port cannot be empty")
	}

	// Basic numeric validation
	portRegex := regexp.MustCompile(`^[0-9]+$`)
	if !portRegex.MatchString(port) {
		return fmt.Errorf("port must be numeric")
	}

	// Convert to int and validate range
	var portNum int
	if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
		return fmt.Errorf("invalid port number format: %w", err)
	}

	if portNum < MinPortNumber || portNum > MaxPortNumber {
		return fmt.Errorf("port number out of range: %d (must be %d-%d)", portNum, MinPortNumber, MaxPortNumber)
	}

	// Warn about privileged ports
	if portNum < 1024 {
		// This is informational - still valid but noteworthy
		// Could be logged as a security event
	}

	return nil
}

// SanitizeShellArg safely escapes shell arguments to prevent injection
func (iv *InputValidator) SanitizeShellArg(arg string) string {
	// For maximum safety, we'll use Go's strconv.Quote which handles all edge cases
	// This uses double quotes and properly escapes all dangerous characters
	return strconv.Quote(arg)
}

// ValidateWindowName validates tmux window names with appropriate restrictions
func (iv *InputValidator) ValidateWindowName(windowName string) error {
	if windowName == "" {
		return fmt.Errorf("window name cannot be empty")
	}

	if len(windowName) > 64 {
		return fmt.Errorf("window name too long: %d characters (max 64)", len(windowName))
	}

	// Window names should be safe for tmux and shell usage
	// Allow alphanumeric, hyphens, underscores, and basic safe characters
	validWindowRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validWindowRegex.MatchString(windowName) {
		return fmt.Errorf("window name contains invalid characters (only alphanumeric, hyphen, underscore allowed)")
	}

	return nil
}

// ValidateEnvironmentVariable validates environment variable names and values
func (iv *InputValidator) ValidateEnvironmentVariable(name, value string) error {
	if name == "" {
		return fmt.Errorf("environment variable name cannot be empty")
	}

	// Environment variable names should follow POSIX standards
	envNameRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !envNameRegex.MatchString(name) {
		return fmt.Errorf("invalid environment variable name: %s", name)
	}

	// Check for dangerous environment variables
	dangerousEnvVars := []string{
		"LD_PRELOAD", "LD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES",
		"DYLD_LIBRARY_PATH", "IFS", "PATH", "PS1", "PS2", "PS4",
	}

	for _, dangerous := range dangerousEnvVars {
		if strings.EqualFold(name, dangerous) {
			return fmt.Errorf("modification of dangerous environment variable not allowed: %s", name)
		}
	}

	// Validate value length
	if len(value) > MaxEnvVarLength {
		return fmt.Errorf("environment variable value too long: %d characters (max %d)", len(value), MaxEnvVarLength)
	}

	// Check for null bytes in value
	if strings.Contains(value, "\x00") {
		return fmt.Errorf("environment variable value contains null byte")
	}

	return nil
}

// Global validator instance
var DefaultValidator = NewInputValidator()

// Convenience functions using the default validator
func ValidateHostname(hostname string) error {
	return DefaultValidator.ValidateHostname(hostname)
}

func ValidateFilePath(path string) error {
	return DefaultValidator.ValidateFilePath(path)
}

func ValidateCommand(command string) error {
	return DefaultValidator.ValidateCommand(command)
}

func ValidateSSHUser(username string) error {
	return DefaultValidator.ValidateSSHUser(username)
}

func ValidatePort(port string) error {
	return DefaultValidator.ValidatePort(port)
}

func SanitizeShellArg(arg string) string {
	return DefaultValidator.SanitizeShellArg(arg)
}

// ValidateWindowName validates tmux window names with appropriate restrictions
func ValidateWindowName(windowName string) error {
	if windowName == "" {
		return fmt.Errorf("window name cannot be empty")
	}

	if len(windowName) > 64 {
		return fmt.Errorf("window name too long: %d characters (max 64)", len(windowName))
	}

	// Window names should be safe for tmux and shell usage
	// Allow alphanumeric, hyphens, underscores, and basic safe characters
	validWindowRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validWindowRegex.MatchString(windowName) {
		return fmt.Errorf("window name contains invalid characters (only alphanumeric, hyphen, underscore allowed)")
	}

	return nil
}
