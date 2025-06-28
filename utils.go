package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/derekg/ts-ssh/internal/security"
)

// parseTarget takes a string like "host", "host:port", or "[ipv6host]:port"
// and returns the host and port. If no port is specified, it uses defaultSSHPort.
func parseTarget(target string, defaultPort string) (host, port string, err error) {
	host = target
	port = defaultPort 

	if strings.HasPrefix(host, "[") {
		endBracketIndex := strings.LastIndex(host, "]")
		if endBracketIndex == -1 {
			return "", "", fmt.Errorf("mismatched brackets in IPv6 address: %s", host)
		}
		if len(host) > endBracketIndex+1 && host[endBracketIndex+1] == ':' {
			port = host[endBracketIndex+2:]
			host = host[1:endBracketIndex]
		} else if len(host) > endBracketIndex+1 { 
			return "", "", fmt.Errorf("unexpected characters after ']' in IPv6 address: %s", host)
		} else { 
			host = host[1:endBracketIndex]
		}
	} else {
		h, p, errSplit := net.SplitHostPort(target) 
		if errSplit == nil {
			host = h
			port = p
		} else {
			if strings.Contains(target, ":") && !strings.HasPrefix(target, "[") {
				return "", "", fmt.Errorf(T("invalid_host_port_format"), target, errSplit)
			}
		}
	}

	if host == "" {
		return "", "", errors.New(T("hostname_cannot_be_empty"))
	}
	if port == "" { 
		port = defaultPort
	}
	
	// SECURITY: Validate extracted components
	// Handle case where host might contain user@hostname format
	actualHost := host
	if strings.Contains(host, "@") {
		// Extract just the hostname part for validation
		parts := strings.SplitN(host, "@", 2)
		if len(parts) == 2 {
			// Validate the user part
			if err := security.ValidateSSHUser(parts[0]); err != nil {
				return "", "", fmt.Errorf("SSH user validation failed: %w", err)
			}
			actualHost = parts[1]
		}
	}
	
	if err := security.ValidateHostname(actualHost); err != nil {
		return "", "", fmt.Errorf("hostname validation failed: %w", err)
	}
	
	if err := security.ValidatePort(port); err != nil {
		return "", "", fmt.Errorf("port validation failed: %w", err)
	}
	
	return host, port, nil
}

// promptUserViaTTY prompts the user for input using secure TTY validation.
func promptUserViaTTY(prompt string, logger *log.Logger) (string, error) {
	// Try secure TTY access first
	result, err := security.PromptUserSecurely(prompt)
	if err != nil {
		logger.Printf("Warning: Could not use secure TTY for prompt: %v. Falling back to stdin.", err)
		fmt.Fprint(os.Stderr, "(secure TTY unavailable, reading from stdin): ") 
		reader := bufio.NewReader(os.Stdin)
		line, errRead := reader.ReadString('\n')
		if errRead != nil {
			return "", fmt.Errorf("failed to read from stdin fallback: %w", errRead)
		}
		return strings.ToLower(strings.TrimSpace(line)), nil
	}
	return strings.ToLower(strings.TrimSpace(result)), nil
}

// parseScpRemoteArg parses an SCP remote argument string (e.g., "user@host:path" or "host:path")
// It returns the host, path, and user. If user is not in the string, it returns the default SSH user.
func parseScpRemoteArg(remoteArg string, defaultSshUser string) (host, path, user string, err error) {
	user = defaultSshUser // Start with the default/flag-provided user

	parts := strings.SplitN(remoteArg, ":", 2)
	if len(parts) != 2 || parts[1] == "" { // Ensure path part exists
		return "", "", "", fmt.Errorf("%s", T("invalid_scp_remote"))
	}
	path = parts[1]
	hostPart := parts[0]

	if strings.Contains(hostPart, "@") {
		// Split user@host
		userHostParts := strings.SplitN(hostPart, "@", 2)
		if len(userHostParts) != 2 {
			return "", "", "", fmt.Errorf("%s", T("invalid_user_host"))
		}
		user = userHostParts[0]
		host = userHostParts[1]
	} else {
		host = hostPart
	}

	if host == "" {
		return "", "", "", fmt.Errorf("%s", T("empty_host_scp"))
	}
	return host, path, user, nil
}

// validateInsecureMode validates and handles insecure host key verification mode
func validateInsecureMode(insecureHostKey, forceInsecure bool, host, user string) error {
	if !insecureHostKey {
		return nil
	}

	// Display security warnings
	fmt.Fprint(os.Stderr, "⚠️  "+T("warning_insecure_mode")+"\n")
	fmt.Fprint(os.Stderr, "⚠️  "+T("warning_mitm_vulnerability")+"\n")
	fmt.Fprint(os.Stderr, "⚠️  "+T("warning_trusted_networks_only")+"\n")
	fmt.Fprint(os.Stderr, "\n")

	if forceInsecure {
		fmt.Fprint(os.Stderr, T("insecure_mode_forced")+"\n")
		fmt.Fprint(os.Stderr, T("proceeding_with_insecure_connection")+"\n\n")
		return nil
	}

	// Get user confirmation
	response, err := promptUserViaTTY(T("confirm_insecure_connection")+" ", log.New(os.Stderr, "", 0))
	if err != nil {
		return fmt.Errorf(T("failed_read_user_input"), err)
	}

	if response != "y" && response != "yes" {
		return fmt.Errorf(T("connection_cancelled_by_user"))
	}

	fmt.Fprint(os.Stderr, T("proceeding_with_insecure_connection")+"\n\n")
	return nil
}
