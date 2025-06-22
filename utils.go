package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
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
	
	// Validate that port is numeric
	if _, err := strconv.Atoi(port); err != nil {
		return "", "", fmt.Errorf(T("invalid_port_number"), port, err)
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
