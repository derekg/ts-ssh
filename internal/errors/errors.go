package errors

import (
	"fmt"
	"log"
	"os"
)

// ErrorCode represents different types of errors in ts-ssh
type ErrorCode int

const (
	// Application errors
	ErrCodeUnknown ErrorCode = iota
	ErrCodeTargetParsing
	ErrCodeSSHConnection
	ErrCodeSSHAuth
	ErrCodeTsnetInit
	ErrCodeHostKeyVerification
	ErrCodeFileOperation
	ErrCodeSecurityValidation
	ErrCodeUserInput
	ErrCodeConfiguration
	ErrCodeNetworking
	ErrCodeTerminal
	ErrCodeTmux
	ErrCodeSCP
)

// TSError represents a structured error with operation context and error code
type TSError struct {
	Op      string    // Operation that failed (e.g., "ssh_connect", "parse_target")
	Code    ErrorCode // Error classification
	Err     error     // Underlying error
	Context string    // Additional context (optional)
	Fatal   bool      // Whether this error should cause program exit
}

// Error implements the error interface
func (e *TSError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Context, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error for error wrapping support
func (e *TSError) Unwrap() error {
	return e.Err
}

// IsFatal returns whether this error should cause program termination
func (e *TSError) IsFatal() bool {
	return e.Fatal
}

// GetCode returns the error classification code
func (e *TSError) GetCode() ErrorCode {
	return e.Code
}

// ErrorHandler provides standardized error handling across the application
type ErrorHandler struct {
	logger *log.Logger
	debug  bool
}

// NewErrorHandler creates a new error handler with the given logger
func NewErrorHandler(logger *log.Logger, debug bool) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
		debug:  debug,
	}
}

// Handle processes an error according to its type and severity
func (eh *ErrorHandler) Handle(err error) {
	if err == nil {
		return
	}

	var tsErr *TSError
	if e, ok := err.(*TSError); ok {
		tsErr = e
	} else {
		// Wrap unknown errors
		tsErr = &TSError{
			Op:   "unknown_operation",
			Code: ErrCodeUnknown,
			Err:  err,
		}
	}

	// Log the error with appropriate level
	if eh.debug {
		eh.logger.Printf("[%s] %s", eh.codeToString(tsErr.Code), tsErr.Error())
	} else {
		eh.logger.Printf("Error: %s", tsErr.Error())
	}

	// Handle fatal errors
	if tsErr.IsFatal() {
		eh.logger.Printf("Fatal error encountered, exiting...")
		os.Exit(1)
	}
}

// HandleWithExit is a convenience function for fatal errors
func (eh *ErrorHandler) HandleWithExit(err error) {
	if err == nil {
		return
	}

	var tsErr *TSError
	if e, ok := err.(*TSError); ok {
		tsErr = e
	} else {
		tsErr = &TSError{
			Op:    "unknown_operation",
			Code:  ErrCodeUnknown,
			Err:   err,
			Fatal: true,
		}
	}

	tsErr.Fatal = true
	eh.Handle(tsErr)
}

// codeToString converts error codes to readable strings
func (eh *ErrorHandler) codeToString(code ErrorCode) string {
	switch code {
	case ErrCodeTargetParsing:
		return "TARGET_PARSING"
	case ErrCodeSSHConnection:
		return "SSH_CONNECTION"
	case ErrCodeSSHAuth:
		return "SSH_AUTH"
	case ErrCodeTsnetInit:
		return "TSNET_INIT"
	case ErrCodeHostKeyVerification:
		return "HOST_KEY_VERIFICATION"
	case ErrCodeFileOperation:
		return "FILE_OPERATION"
	case ErrCodeSecurityValidation:
		return "SECURITY_VALIDATION"
	case ErrCodeUserInput:
		return "USER_INPUT"
	case ErrCodeConfiguration:
		return "CONFIGURATION"
	case ErrCodeNetworking:
		return "NETWORKING"
	case ErrCodeTerminal:
		return "TERMINAL"
	case ErrCodeTmux:
		return "TMUX"
	case ErrCodeSCP:
		return "SCP"
	default:
		return "UNKNOWN"
	}
}

// Helper functions for creating common error types

// NewTargetParsingError creates a target parsing error
func NewTargetParsingError(target string, err error) *TSError {
	return &TSError{
		Op:      "parse_target",
		Code:    ErrCodeTargetParsing,
		Err:     err,
		Context: fmt.Sprintf("target: %s", target),
		Fatal:   true,
	}
}

// NewSSHConnectionError creates an SSH connection error
func NewSSHConnectionError(host string, err error) *TSError {
	return &TSError{
		Op:      "ssh_connect",
		Code:    ErrCodeSSHConnection,
		Err:     err,
		Context: fmt.Sprintf("host: %s", host),
	}
}

// NewSSHAuthError creates an SSH authentication error
func NewSSHAuthError(user, host string, err error) *TSError {
	return &TSError{
		Op:      "ssh_auth",
		Code:    ErrCodeSSHAuth,
		Err:     err,
		Context: fmt.Sprintf("user: %s, host: %s", user, host),
	}
}

// NewTsnetInitError creates a tsnet initialization error
func NewTsnetInitError(err error) *TSError {
	return &TSError{
		Op:    "tsnet_init",
		Code:  ErrCodeTsnetInit,
		Err:   err,
		Fatal: true,
	}
}

// NewSecurityValidationError creates a security validation error
func NewSecurityValidationError(operation string, err error) *TSError {
	return &TSError{
		Op:      "security_validation",
		Code:    ErrCodeSecurityValidation,
		Err:     err,
		Context: operation,
		Fatal:   true, // Security errors are typically fatal
	}
}

// NewFileOperationError creates a file operation error
func NewFileOperationError(operation, path string, err error) *TSError {
	return &TSError{
		Op:      "file_operation",
		Code:    ErrCodeFileOperation,
		Err:     err,
		Context: fmt.Sprintf("operation: %s, path: %s", operation, path),
	}
}

// NewUserInputError creates a user input error
func NewUserInputError(prompt string, err error) *TSError {
	return &TSError{
		Op:      "user_input",
		Code:    ErrCodeUserInput,
		Err:     err,
		Context: fmt.Sprintf("prompt: %s", prompt),
	}
}

// NewTerminalError creates a terminal operation error
func NewTerminalError(operation string, err error) *TSError {
	return &TSError{
		Op:      "terminal_operation",
		Code:    ErrCodeTerminal,
		Err:     err,
		Context: operation,
	}
}

// NewTmuxError creates a tmux operation error
func NewTmuxError(operation string, err error) *TSError {
	return &TSError{
		Op:      "tmux_operation",
		Code:    ErrCodeTmux,
		Err:     err,
		Context: operation,
	}
}

// NewSCPError creates an SCP operation error
func NewSCPError(operation, path string, err error) *TSError {
	return &TSError{
		Op:      "scp_operation",
		Code:    ErrCodeSCP,
		Err:     err,
		Context: fmt.Sprintf("operation: %s, path: %s", operation, path),
	}
}
