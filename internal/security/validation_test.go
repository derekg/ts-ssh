package security

import (
	"strings"
	"testing"
)

func TestValidateHostname(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name        string
		hostname    string
		expectError bool
		errorMsg    string
	}{
		// Valid hostnames
		{"valid_simple", "example.com", false, ""},
		{"valid_subdomain", "www.example.com", false, ""},
		{"valid_deep_subdomain", "api.v2.example.com", false, ""},
		{"valid_ip4", "192.168.1.1", false, ""},
		{"valid_ip6", "2001:db8::1", false, ""},
		{"valid_single_label", "localhost", false, ""},
		{"valid_with_hyphens", "my-server.example.com", false, ""},
		{"valid_numbers", "server1.example.com", false, ""},

		// Invalid hostnames - security threats
		{"command_injection_semicolon", "server.com;rm -rf /", true, "invalid characters"},
		{"command_injection_ampersand", "server.com && rm -rf /", true, "invalid characters"},
		{"command_injection_pipe", "server.com | cat /etc/passwd", true, "invalid characters"},
		{"command_injection_backtick", "server.com`whoami`", true, "invalid characters"},
		{"command_injection_dollar", "server.com$(whoami)", true, "invalid characters"},
		{"command_injection_parens", "server.com()", true, "invalid characters"},
		{"command_injection_brackets", "server.com[test]", true, "invalid characters"},
		{"command_injection_braces", "server.com{test}", true, "invalid characters"},
		{"command_injection_redirect", "server.com>file", true, "invalid characters"},
		{"command_injection_quotes", "server.com\"test\"", true, "invalid characters"},
		{"command_injection_single_quote", "server.com'test'", true, "invalid characters"},
		{"command_injection_exclamation", "server.com!test", true, "invalid characters"},
		{"command_injection_asterisk", "server.com*", true, "invalid characters"},
		{"command_injection_question", "server.com?", true, "invalid characters"},

		// Invalid hostnames - format violations
		{"empty_hostname", "", true, "cannot be empty"},
		{"too_long", strings.Repeat("a", 254), true, "too long"},
		{"starts_with_hyphen", "-example.com", true, "cannot start or end with hyphen"},
		{"ends_with_hyphen", "example.com-", true, "cannot start or end with hyphen"},
		{"consecutive_hyphens", "ex--ample.com", true, "consecutive hyphens"},
		{"label_too_long", strings.Repeat("a", 64) + ".com", true, "label too long"},
		{"empty_label", "example..com", true, "empty label"},
		{"label_starts_hyphen", "ex-ample.-test.com", true, "cannot start or end with hyphen"},
		{"label_ends_hyphen", "example.test-.com", true, "cannot start or end with hyphen"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateHostname(tt.hostname)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for hostname %s, but got none", tt.hostname)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for hostname %s, but got: %v", tt.hostname, err)
				}
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		// Valid paths
		{"valid_absolute", "/home/user/file.txt", false, ""},
		{"valid_relative", "config/app.conf", false, ""},
		{"valid_with_dots", "/home/user/.ssh/id_rsa", false, ""},
		{"valid_with_hyphens", "/home/user/my-file.txt", false, ""},
		{"valid_with_underscores", "/home/user/my_file.txt", false, ""},

		// Invalid paths - security threats
		{"path_traversal_simple", "../../../etc/passwd", true, "path traversal"},
		{"path_traversal_absolute", "/home/user/../../../etc/passwd", true, "path traversal"},
		{"path_traversal_mixed", "/home/user/./../../etc/passwd", true, "path traversal"},
		{"command_injection_semicolon", "/home/user;rm -rf /", true, "invalid characters"},
		{"command_injection_ampersand", "/home/user && rm -rf /", true, "invalid characters"},
		{"command_injection_pipe", "/home/user | cat", true, "invalid characters"},
		{"command_injection_backtick", "/home/user`whoami`", true, "invalid characters"},
		{"command_injection_dollar", "/home/user$(whoami)", true, "invalid characters"},
		{"null_byte_injection", "/home/user\x00/etc/passwd", true, "null byte"},
		{"control_char_injection", "/home/user\x01test", true, "control character"},

		// Invalid paths - format violations
		{"empty_path", "", true, "cannot be empty"},
		{"too_long_path", "/" + strings.Repeat("a", 4100), true, "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFilePath(tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path %s, but got none", tt.path)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for path %s, but got: %v", tt.path, err)
				}
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name        string
		command     string
		expectError bool
		errorMsg    string
	}{
		// Valid commands
		{"valid_simple", "ls -la", false, ""},
		{"valid_with_args", "grep pattern file.txt", false, ""},
		{"valid_with_numbers", "ps aux", false, ""},

		// Invalid commands - injection threats
		{"command_injection_semicolon", "ls; rm -rf /", true, "dangerous pattern"},
		{"command_injection_double_amp", "ls && rm -rf /", true, "dangerous pattern"},
		{"command_injection_double_pipe", "ls || rm -rf /", true, "dangerous pattern"},
		{"command_injection_pipe", "ls | rm", true, "dangerous pattern"},
		{"command_injection_backtick", "ls `whoami`", true, "dangerous pattern"},
		{"command_injection_dollar_paren", "ls $(whoami)", true, "dangerous pattern"},
		{"command_injection_dollar_brace", "ls ${USER}", true, "dangerous pattern"},
		{"xss_html_comment", "<!-- script -->", true, "dangerous pattern"},
		{"xss_script_tag", "<script>alert(1)</script>", true, "dangerous pattern"},
		{"javascript_url", "javascript:alert(1)", true, "dangerous pattern"},
		{"vbscript_url", "vbscript:msgbox(1)", true, "dangerous pattern"},
		{"onload_event", "onload=alert(1)", true, "dangerous pattern"},
		{"onerror_event", "onerror=alert(1)", true, "dangerous pattern"},
		{"eval_function", "eval('code')", true, "dangerous pattern"},
		{"exec_function", "exec('code')", true, "dangerous pattern"},
		{"null_byte_injection", "ls\x00; rm -rf /", true, "null byte at position"},

		// Invalid commands - format violations
		{"empty_command", "", true, "cannot be empty"},
		{"too_long_command", strings.Repeat("a", 8200), true, "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCommand(tt.command)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for command %s, but got none", tt.command)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for command %s, but got: %v", tt.command, err)
				}
			}
		})
	}
}

func TestValidateSSHUser(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name        string
		username    string
		expectError bool
		errorMsg    string
	}{
		// Valid usernames
		{"valid_simple", "user", false, ""},
		{"valid_with_underscore", "user_name", false, ""},
		{"valid_with_hyphen", "user-name", false, ""},
		{"valid_with_numbers", "user123", false, ""},
		{"valid_mixed", "my_user-123", false, ""},
		{"valid_with_dot", "user.name", false, ""},
		{"valid_with_multiple_dots", "first.last.name", false, ""},

		// Invalid usernames
		{"empty_username", "", true, "cannot be empty"},
		{"too_long", strings.Repeat("a", 33), true, "too long"},
		{"starts_with_hyphen", "-user", true, "cannot start with hyphen"},
		{"starts_with_number", "1user", true, "cannot start with hyphen or number"},
		{"invalid_chars_space", "user name", true, "invalid characters"},
		{"invalid_chars_special", "user@host", true, "invalid characters"},
		{"invalid_chars_slash", "user/name", true, "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSSHUser(tt.username)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for username %s, but got none", tt.username)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for username %s, but got: %v", tt.username, err)
				}
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name        string
		port        string
		expectError bool
		errorMsg    string
	}{
		// Valid ports
		{"valid_ssh", "22", false, ""},
		{"valid_http", "80", false, ""},
		{"valid_https", "443", false, ""},
		{"valid_high", "8080", false, ""},
		{"valid_max", "65535", false, ""},

		// Invalid ports
		{"empty_port", "", true, "cannot be empty"},
		{"non_numeric", "abc", true, "must be numeric"},
		{"with_letters", "22a", true, "must be numeric"},
		{"negative", "-1", true, "must be numeric"},
		{"zero", "0", true, "out of range"},
		{"too_high", "65536", true, "out of range"},
		{"way_too_high", "99999", true, "out of range"},
		{"with_spaces", "22 ", true, "must be numeric"},
		{"float", "22.5", true, "must be numeric"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePort(tt.port)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for port %s, but got none", tt.port)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for port %s, but got: %v", tt.port, err)
				}
			}
		})
	}
}

func TestSanitizeShellArg(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple_arg", "test", `"test"`},
		{"arg_with_space", "hello world", `"hello world"`},
		{"arg_with_single_quote", "don't", `"don't"`},
		{"arg_with_double_quote", `say "hello"`, `"say \"hello\""`},
		{"empty_arg", "", `""`},
		{"arg_with_special_chars", "test;rm -rf /", `"test;rm -rf /"`},
		{"arg_with_backticks", "test`whoami`", `"test` + "`" + `whoami` + "`" + `"`},
		{"arg_with_dollar", "test$USER", `"test$USER"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.SanitizeShellArg(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeShellArg(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateEnvironmentVariable(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name        string
		envName     string
		envValue    string
		expectError bool
		errorMsg    string
	}{
		// Valid environment variables
		{"valid_simple", "MY_VAR", "value", false, ""},
		{"valid_with_underscore", "MY_LONG_VAR", "value", false, ""},
		{"valid_with_numbers", "VAR123", "value", false, ""},
		{"valid_starts_underscore", "_PRIVATE", "value", false, ""},

		// Invalid environment variable names
		{"empty_name", "", "value", true, "cannot be empty"},
		{"starts_with_number", "1VAR", "value", true, "invalid environment variable name"},
		{"contains_hyphen", "MY-VAR", "value", true, "invalid environment variable name"},
		{"contains_space", "MY VAR", "value", true, "invalid environment variable name"},
		{"contains_special", "MY@VAR", "value", true, "invalid environment variable name"},

		// Dangerous environment variables
		{"dangerous_ld_preload", "LD_PRELOAD", "/path/to/lib", true, "dangerous environment variable"},
		{"dangerous_path", "PATH", "/bin:/usr/bin", true, "dangerous environment variable"},
		{"dangerous_ifs", "IFS", " ", true, "dangerous environment variable"},

		// Invalid values
		{"value_too_long", "VALID_VAR", strings.Repeat("a", 32769), true, "value too long"},
		{"value_with_null", "VALID_VAR", "value\x00", true, "null byte"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEnvironmentVariable(tt.envName, tt.envValue)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for env var %s=%s, but got none", tt.envName, tt.envValue)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for env var %s=%s, but got: %v", tt.envName, tt.envValue, err)
				}
			}
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	// Test that the convenience functions work correctly
	tests := []struct {
		name        string
		testFunc    func() error
		expectError bool
	}{
		{"ValidateHostname_valid", func() error { return ValidateHostname("example.com") }, false},
		{"ValidateHostname_invalid", func() error { return ValidateHostname("example.com;rm -rf /") }, true},
		{"ValidateFilePath_valid", func() error { return ValidateFilePath("/home/user/file.txt") }, false},
		{"ValidateFilePath_invalid", func() error { return ValidateFilePath("../../../etc/passwd") }, true},
		{"ValidateCommand_valid", func() error { return ValidateCommand("ls -la") }, false},
		{"ValidateCommand_invalid", func() error { return ValidateCommand("ls; rm -rf /") }, true},
		{"ValidateSSHUser_valid", func() error { return ValidateSSHUser("myuser") }, false},
		{"ValidateSSHUser_invalid", func() error { return ValidateSSHUser("my user") }, true},
		{"ValidatePort_valid", func() error { return ValidatePort("22") }, false},
		{"ValidatePort_invalid", func() error { return ValidatePort("abc") }, true},
		{"ValidateWindowName_valid", func() error { return ValidateWindowName("ssh-1") }, false},
		{"ValidateWindowName_invalid", func() error { return ValidateWindowName("ssh window") }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}

	// Test SanitizeShellArg convenience function
	result := SanitizeShellArg("test arg")
	expected := `"test arg"`
	if result != expected {
		t.Errorf("SanitizeShellArg convenience function failed: got %s, expected %s", result, expected)
	}
}

func TestValidateWindowName(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name        string
		windowName  string
		expectError bool
		errorMsg    string
	}{
		// Valid window names
		{"valid_simple", "ssh-1", false, ""},
		{"valid_with_underscore", "ssh_window", false, ""},
		{"valid_with_numbers", "window123", false, ""},
		{"valid_mixed", "ssh-window_1", false, ""},

		// Invalid window names
		{"empty_window_name", "", true, "cannot be empty"},
		{"too_long", strings.Repeat("a", 65), true, "too long"},
		{"invalid_chars_space", "ssh window", true, "invalid characters"},
		{"invalid_chars_special", "ssh@window", true, "invalid characters"},
		{"invalid_chars_dot", "ssh.window", true, "invalid characters"},
		{"invalid_chars_slash", "ssh/window", true, "invalid characters"},
		{"invalid_chars_colon", "ssh:window", true, "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWindowName(tt.windowName)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for window name %s, but got none", tt.windowName)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for window name %s, but got: %v", tt.windowName, err)
				}
			}
		})
	}
}

func BenchmarkValidateHostname(b *testing.B) {
	validator := NewInputValidator()
	hostname := "www.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateHostname(hostname)
	}
}

func BenchmarkValidateFilePath(b *testing.B) {
	validator := NewInputValidator()
	path := "/home/user/documents/file.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateFilePath(path)
	}
}

func BenchmarkSanitizeShellArg(b *testing.B) {
	validator := NewInputValidator()
	arg := "complex argument with spaces and 'quotes'"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.SanitizeShellArg(arg)
	}
}
