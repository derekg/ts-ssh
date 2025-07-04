package security

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// version represents the application version for logging
var version = "0.4.0"

// SecurityEvent represents a security-relevant event for audit logging
type SecurityEvent struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`
	Severity  string    `json:"severity"`
	User      string    `json:"user"`
	Host      string    `json:"host"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	UserAgent string    `json:"user_agent"`
	Success   bool      `json:"success"`
	IPAddress string    `json:"ip_address,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
}

// SecurityLogger handles security audit logging
type SecurityLogger struct {
	enabled bool
	logFile *os.File
	logger  *log.Logger
}

// Global security logger instance
var securityLogger *SecurityLogger

// InitSecurityLogger initializes the security audit logging system
func InitSecurityLogger() error {
	// Check if security logging is enabled via environment variable
	enabled := os.Getenv("TS_SSH_SECURITY_AUDIT") != ""
	if !enabled {
		securityLogger = &SecurityLogger{enabled: false}
		return nil
	}

	// Determine log file path
	logPath := os.Getenv("TS_SSH_AUDIT_LOG")
	if logPath == "" {
		// Default to user's home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to determine home directory for security log: %w", err)
		}
		logPath = filepath.Join(homeDir, ".ts-ssh-security.log")
	}

	// Create secure log file with appropriate permissions
	logFile, err := CreateSecureFileForAppend(logPath, 0600)
	if err != nil {
		return fmt.Errorf("failed to create security audit log: %w", err)
	}

	// Create logger instance
	logger := log.New(logFile, "", 0) // No default prefix, we'll format our own

	securityLogger = &SecurityLogger{
		enabled: true,
		logFile: logFile,
		logger:  logger,
	}

	// Log initialization
	securityLogger.logSecurityEvent(SecurityEvent{
		EventType: "AUDIT_INIT",
		Severity:  "INFO",
		Action:    "security_audit_logging_initialized",
		Details:   fmt.Sprintf("Security audit logging enabled, log file: %s", logPath),
		Success:   true,
	})

	return nil
}

// CloseSecurityLogger safely closes the security logger
func CloseSecurityLogger() {
	if securityLogger != nil && securityLogger.enabled && securityLogger.logFile != nil {
		securityLogger.logSecurityEvent(SecurityEvent{
			EventType: "AUDIT_CLOSE",
			Severity:  "INFO",
			Action:    "security_audit_logging_closed",
			Details:   "Security audit logging session ended",
			Success:   true,
		})
		securityLogger.logFile.Close()
	}
}

// logSecurityEvent logs a security event to the audit log
func (sl *SecurityLogger) logSecurityEvent(event SecurityEvent) {
	if !sl.enabled || sl.logger == nil {
		return
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Set user agent if not provided
	if event.UserAgent == "" {
		event.UserAgent = fmt.Sprintf("ts-ssh/%s", version)
	}

	// Marshal to JSON for structured logging
	eventJSON, err := json.Marshal(event)
	if err != nil {
		// Fallback to simple text logging if JSON fails
		sl.logger.Printf("[SECURITY] %s %s %s: %s - %s",
			event.Timestamp.Format(time.RFC3339),
			event.Severity,
			event.EventType,
			event.Action,
			event.Details)
		return
	}

	// Log structured JSON event
	sl.logger.Printf("%s", string(eventJSON))
}

// LogInsecureModeUsage logs usage of insecure host key verification mode
func LogInsecureModeUsage(host, user string, forced bool, confirmed bool) {
	if securityLogger == nil {
		return
	}

	severity := "WARNING"
	if forced {
		severity = "HIGH"
	}

	action := "insecure_mode_used"
	details := fmt.Sprintf("Host key verification disabled for connection to %s", host)

	if forced {
		details += " (forced via --force-insecure flag)"
	} else if confirmed {
		details += " (user confirmed after warning)"
	} else {
		details += " (user declined after warning)"
	}

	securityLogger.logSecurityEvent(SecurityEvent{
		EventType: "HOST_KEY_BYPASS",
		Severity:  severity,
		User:      user,
		Host:      host,
		Action:    action,
		Details:   details,
		Success:   confirmed,
	})
}

// LogSSHKeyAuthentication logs SSH key authentication attempts
func LogSSHKeyAuthentication(host, user, keyPath, keyType string, success bool) {
	if securityLogger == nil {
		return
	}

	severity := "INFO"
	if !success {
		severity = "WARNING"
	}

	action := "ssh_key_authentication"
	details := fmt.Sprintf("SSH key authentication using %s (%s)", keyType, keyPath)
	if !success {
		details += " - failed"
	}

	securityLogger.logSecurityEvent(SecurityEvent{
		EventType: "SSH_AUTH",
		Severity:  severity,
		User:      user,
		Host:      host,
		Action:    action,
		Details:   details,
		Success:   success,
	})
}

// LogHostKeyVerification logs host key verification events
func LogHostKeyVerification(host, user string, action string, success bool) {
	if securityLogger == nil {
		return
	}

	severity := "INFO"
	if !success {
		severity = "HIGH"
	}

	details := fmt.Sprintf("Host key verification for %s", host)
	switch action {
	case "known_host":
		details += " - verified against known_hosts"
	case "new_host_accepted":
		details += " - new host key accepted by user"
	case "new_host_rejected":
		details += " - new host key rejected by user"
	case "verification_failed":
		details += " - verification failed"
	}

	securityLogger.logSecurityEvent(SecurityEvent{
		EventType: "HOST_KEY_VERIFICATION",
		Severity:  severity,
		User:      user,
		Host:      host,
		Action:    action,
		Details:   details,
		Success:   success,
	})
}

// LogPasswordAuthentication logs password authentication attempts
func LogPasswordAuthentication(host, user string, success bool) {
	if securityLogger == nil {
		return
	}

	severity := "INFO"
	if !success {
		severity = "WARNING"
	}

	action := "password_authentication"
	details := "SSH password authentication attempt"
	if !success {
		details += " - failed"
	}

	securityLogger.logSecurityEvent(SecurityEvent{
		EventType: "SSH_AUTH",
		Severity:  severity,
		User:      user,
		Host:      host,
		Action:    action,
		Details:   details,
		Success:   success,
	})
}

// LogSecureFileOperation logs security-relevant file operations
func LogSecureFileOperation(operation, filePath string, success bool, details string) {
	if securityLogger == nil {
		return
	}

	severity := "INFO"
	if !success {
		severity = "WARNING"
	}

	securityLogger.logSecurityEvent(SecurityEvent{
		EventType: "FILE_OPERATION",
		Severity:  severity,
		Action:    operation,
		Details:   fmt.Sprintf("File operation: %s on %s - %s", operation, filePath, details),
		Success:   success,
	})
}

// LogTTYSecurityValidation logs TTY security validation events
func LogTTYSecurityValidation(operation string, success bool, details string) {
	if securityLogger == nil {
		return
	}

	severity := "INFO"
	if !success {
		severity = "HIGH"
	}

	securityLogger.logSecurityEvent(SecurityEvent{
		EventType: "TTY_SECURITY",
		Severity:  severity,
		Action:    operation,
		Details:   fmt.Sprintf("TTY security validation: %s - %s", operation, details),
		Success:   success,
	})
}
