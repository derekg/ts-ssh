package errors

import (
	"errors"
	"log"
	"os"
	"strings"
	"testing"
)

// TestErrorCode tests error code constants
func TestErrorCode(t *testing.T) {
	codes := []ErrorCode{
		ErrCodeUnknown,
		ErrCodeTargetParsing,
		ErrCodeSSHConnection,
		ErrCodeSSHAuth,
		ErrCodeTsnetInit,
		ErrCodeHostKeyVerification,
		ErrCodeFileOperation,
		ErrCodeSecurityValidation,
		ErrCodeUserInput,
		ErrCodeConfiguration,
		ErrCodeNetworking,
		ErrCodeTerminal,
		ErrCodeTmux,
		ErrCodeSCP,
	}
	
	// Verify all codes are unique
	seen := make(map[ErrorCode]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("Duplicate error code: %d", code)
		}
		seen[code] = true
	}
	
	// Verify starting from 0
	if ErrCodeUnknown != 0 {
		t.Errorf("ErrCodeUnknown should be 0, got %d", ErrCodeUnknown)
	}
}

// TestTSError tests TSError structure and methods
func TestTSError(t *testing.T) {
	tests := []struct {
		name    string
		tsErr   *TSError
		wantErr string
		wantCode ErrorCode
		wantFatal bool
	}{
		{
			name: "basic error",
			tsErr: &TSError{
				Op:   "test_op",
				Code: ErrCodeUnknown,
				Err:  errors.New("test error"),
			},
			wantErr: "test_op: test error",
			wantCode: ErrCodeUnknown,
			wantFatal: false,
		},
		{
			name: "error with context",
			tsErr: &TSError{
				Op:      "test_op",
				Code:    ErrCodeSSHConnection,
				Err:     errors.New("connection failed"),
				Context: "host: example.com",
			},
			wantErr: "test_op: host: example.com: connection failed",
			wantCode: ErrCodeSSHConnection,
			wantFatal: false,
		},
		{
			name: "fatal error",
			tsErr: &TSError{
				Op:    "critical_op",
				Code:  ErrCodeSecurityValidation,
				Err:   errors.New("security breach"),
				Fatal: true,
			},
			wantErr: "critical_op: security breach",
			wantCode: ErrCodeSecurityValidation,
			wantFatal: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Error() method
			if got := tt.tsErr.Error(); got != tt.wantErr {
				t.Errorf("Error() = %v, want %v", got, tt.wantErr)
			}
			
			// Test GetCode() method
			if got := tt.tsErr.GetCode(); got != tt.wantCode {
				t.Errorf("GetCode() = %v, want %v", got, tt.wantCode)
			}
			
			// Test IsFatal() method
			if got := tt.tsErr.IsFatal(); got != tt.wantFatal {
				t.Errorf("IsFatal() = %v, want %v", got, tt.wantFatal)
			}
		})
	}
}

// TestTSErrorUnwrap tests error unwrapping
func TestTSErrorUnwrap(t *testing.T) {
	originalErr := errors.New("original error")
	tsErr := &TSError{
		Op:  "test_op",
		Err: originalErr,
	}
	
	unwrapped := tsErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, originalErr)
	}
	
	// Test errors.Is() works with unwrapping
	if !errors.Is(tsErr, originalErr) {
		t.Error("errors.Is() should work with TSError")
	}
}

// TestErrorHandler tests error handler functionality
func TestErrorHandler(t *testing.T) {
	// Create a test logger that captures output
	var logOutput strings.Builder
	logger := log.New(&logOutput, "", 0)
	
	tests := []struct {
		name      string
		debug     bool
		err       error
		wantLog   string
	}{
		{
			name:    "nil error",
			debug:   false,
			err:     nil,
			wantLog: "",
		},
		{
			name:    "ts error in debug mode",
			debug:   true,
			err:     &TSError{Op: "test_op", Code: ErrCodeSSHConnection, Err: errors.New("test")},
			wantLog: "[SSH_CONNECTION] test_op: test",
		},
		{
			name:    "ts error in normal mode",
			debug:   false,
			err:     &TSError{Op: "test_op", Code: ErrCodeSSHConnection, Err: errors.New("test")},
			wantLog: "Error: test_op: test",
		},
		{
			name:    "unknown error wrapped",
			debug:   true,
			err:     errors.New("unknown error"),
			wantLog: "[UNKNOWN] unknown_operation: unknown error",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logOutput.Reset()
			eh := NewErrorHandler(logger, tt.debug)
			
			eh.Handle(tt.err)
			
			if tt.wantLog == "" {
				if logOutput.Len() > 0 {
					t.Errorf("Expected no log output, got: %s", logOutput.String())
				}
			} else {
				if !strings.Contains(logOutput.String(), tt.wantLog) {
					t.Errorf("Log output %q does not contain %q", logOutput.String(), tt.wantLog)
				}
			}
		})
	}
}

// TestErrorHandlerCodeToString tests error code string conversion
func TestErrorHandlerCodeToString(t *testing.T) {
	eh := &ErrorHandler{}
	
	tests := []struct {
		code ErrorCode
		want string
	}{
		{ErrCodeTargetParsing, "TARGET_PARSING"},
		{ErrCodeSSHConnection, "SSH_CONNECTION"},
		{ErrCodeSSHAuth, "SSH_AUTH"},
		{ErrCodeTsnetInit, "TSNET_INIT"},
		{ErrCodeHostKeyVerification, "HOST_KEY_VERIFICATION"},
		{ErrCodeFileOperation, "FILE_OPERATION"},
		{ErrCodeSecurityValidation, "SECURITY_VALIDATION"},
		{ErrCodeUserInput, "USER_INPUT"},
		{ErrCodeConfiguration, "CONFIGURATION"},
		{ErrCodeNetworking, "NETWORKING"},
		{ErrCodeTerminal, "TERMINAL"},
		{ErrCodeTmux, "TMUX"},
		{ErrCodeSCP, "SCP"},
		{ErrCodeUnknown, "UNKNOWN"},
		{ErrorCode(999), "UNKNOWN"}, // Unknown code
	}
	
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := eh.codeToString(tt.code); got != tt.want {
				t.Errorf("codeToString(%v) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

// TestErrorHelperFunctions tests the helper functions for creating specific errors
func TestErrorHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		createFn func() *TSError
		wantCode ErrorCode
		wantOp   string
		wantFatal bool
	}{
		{
			name: "NewTargetParsingError",
			createFn: func() *TSError {
				return NewTargetParsingError("invalid-target", errors.New("parse failed"))
			},
			wantCode: ErrCodeTargetParsing,
			wantOp:   "parse_target",
			wantFatal: true,
		},
		{
			name: "NewSSHConnectionError",
			createFn: func() *TSError {
				return NewSSHConnectionError("example.com", errors.New("connection refused"))
			},
			wantCode: ErrCodeSSHConnection,
			wantOp:   "ssh_connect",
			wantFatal: false,
		},
		{
			name: "NewSSHAuthError",
			createFn: func() *TSError {
				return NewSSHAuthError("user", "host", errors.New("auth failed"))
			},
			wantCode: ErrCodeSSHAuth,
			wantOp:   "ssh_auth",
			wantFatal: false,
		},
		{
			name: "NewTsnetInitError",
			createFn: func() *TSError {
				return NewTsnetInitError(errors.New("tsnet failed"))
			},
			wantCode: ErrCodeTsnetInit,
			wantOp:   "tsnet_init",
			wantFatal: true,
		},
		{
			name: "NewSecurityValidationError",
			createFn: func() *TSError {
				return NewSecurityValidationError("key_check", errors.New("invalid key"))
			},
			wantCode: ErrCodeSecurityValidation,
			wantOp:   "security_validation",
			wantFatal: true,
		},
		{
			name: "NewFileOperationError",
			createFn: func() *TSError {
				return NewFileOperationError("read", "/tmp/test", errors.New("permission denied"))
			},
			wantCode: ErrCodeFileOperation,
			wantOp:   "file_operation",
			wantFatal: false,
		},
		{
			name: "NewUserInputError",
			createFn: func() *TSError {
				return NewUserInputError("password prompt", errors.New("input failed"))
			},
			wantCode: ErrCodeUserInput,
			wantOp:   "user_input",
			wantFatal: false,
		},
		{
			name: "NewTerminalError",
			createFn: func() *TSError {
				return NewTerminalError("tty_setup", errors.New("no tty"))
			},
			wantCode: ErrCodeTerminal,
			wantOp:   "terminal_operation",
			wantFatal: false,
		},
		{
			name: "NewTmuxError",
			createFn: func() *TSError {
				return NewTmuxError("session_create", errors.New("tmux not found"))
			},
			wantCode: ErrCodeTmux,
			wantOp:   "tmux_operation",
			wantFatal: false,
		},
		{
			name: "NewSCPError",
			createFn: func() *TSError {
				return NewSCPError("upload", "/local/file", errors.New("transfer failed"))
			},
			wantCode: ErrCodeSCP,
			wantOp:   "scp_operation",
			wantFatal: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createFn()
			
			if err.GetCode() != tt.wantCode {
				t.Errorf("GetCode() = %v, want %v", err.GetCode(), tt.wantCode)
			}
			
			if err.Op != tt.wantOp {
				t.Errorf("Op = %v, want %v", err.Op, tt.wantOp)
			}
			
			if err.IsFatal() != tt.wantFatal {
				t.Errorf("IsFatal() = %v, want %v", err.IsFatal(), tt.wantFatal)
			}
			
			// Verify error message is not empty
			if err.Error() == "" {
				t.Error("Error() should not return empty string")
			}
		})
	}
}

// TestNewErrorHandler tests error handler creation
func TestNewErrorHandler(t *testing.T) {
	logger := log.New(os.Stderr, "", 0)
	
	tests := []struct {
		name  string
		debug bool
	}{
		{"debug mode", true},
		{"normal mode", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eh := NewErrorHandler(logger, tt.debug)
			
			if eh.logger != logger {
				t.Error("Logger not set correctly")
			}
			
			if eh.debug != tt.debug {
				t.Errorf("Debug flag = %v, want %v", eh.debug, tt.debug)
			}
		})
	}
}

// TestHandleWithExit tests the HandleWithExit method (without actually exiting)
func TestHandleWithExit(t *testing.T) {
	var logOutput strings.Builder
	logger := log.New(&logOutput, "", 0)
	eh := NewErrorHandler(logger, false)
	
	// Test with nil error
	eh.HandleWithExit(nil)
	if logOutput.Len() > 0 {
		t.Error("HandleWithExit(nil) should not log anything")
	}
	
	// Note: We can't test the actual exit behavior without modifying the code
	// or using build tags, but we can test that the error is properly formatted
	// The actual os.Exit() call would be tested in integration tests
}

// TestTSErrorInterfaceCompliance tests that TSError implements the error interface
func TestTSErrorInterfaceCompliance(t *testing.T) {
	var _ error = &TSError{} // Compile-time check
	
	tsErr := &TSError{
		Op:  "test",
		Err: errors.New("test error"),
	}
	
	// Test that it can be used as an error
	err := error(tsErr)
	if err.Error() == "" {
		t.Error("Error interface not properly implemented")
	}
}