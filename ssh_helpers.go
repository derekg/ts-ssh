package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/user"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
	"tailscale.com/tsnet"
)

// SSHConnectionConfig holds all the parameters needed for SSH connection setup
type SSHConnectionConfig struct {
	User            string
	KeyPath         string
	TargetHost      string
	TargetPort      string
	InsecureHostKey bool
	Verbose         bool
	CurrentUser     *user.User
	Logger          *log.Logger
}

// createSSHAuthMethods creates authentication methods for SSH connection.
// It attempts to load an SSH private key from keyPath if provided, and always
// includes password authentication as a fallback. This function eliminates
// duplication of auth setup across multiple files.
//
// Parameters:
//   - keyPath: path to SSH private key file (optional)
//   - sshUser: username for SSH connection
//   - targetHost: hostname for password prompts
//   - logger: logger instance for debug output
//
// Returns a slice of ssh.AuthMethod and any error that occurred.
func createSSHAuthMethods(keyPath, sshUser, targetHost string, logger *log.Logger) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	// Try to load SSH key if provided
	if keyPath != "" {
		keyAuth, err := LoadPrivateKey(keyPath, logger)
		if err == nil {
			authMethods = append(authMethods, keyAuth)
			if logger != nil {
				logger.Printf(T("using_key_auth"), keyPath)
			}
		} else {
			if logger != nil {
				logger.Printf(T("key_auth_failed"), err)
			}
		}
	}

	// Add password authentication as fallback
	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		fmt.Print(T("enter_password", sshUser, targetHost))
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		return string(bytePassword), nil
	}))

	return authMethods, nil
}

// createSSHConfig creates an SSH client configuration from the provided parameters.
// This function standardizes SSH configuration creation across the codebase,
// handling authentication methods and host key verification consistently.
//
// The function sets up:
//   - Authentication methods (key-based and password)
//   - Host key verification (secure or insecure mode)
//   - Connection timeout settings
//
// Returns a configured ssh.ClientConfig ready for connection establishment.
func createSSHConfig(config SSHConnectionConfig) (*ssh.ClientConfig, error) {
	// Create authentication methods
	authMethods, err := createSSHAuthMethods(config.KeyPath, config.User, config.TargetHost, config.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth methods: %w", err)
	}

	// Set up host key callback
	var hostKeyCallback ssh.HostKeyCallback
	if config.InsecureHostKey {
		if config.Logger != nil {
			config.Logger.Printf(T("host_key_warning"))
		}
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		var err error
		hostKeyCallback, err = CreateKnownHostsCallback(config.CurrentUser, config.Logger)
		if err != nil {
			return nil, fmt.Errorf("could not set up host key verification: %w", err)
		}
	}

	return &ssh.ClientConfig{
		User:            config.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         DefaultSSHTimeout,
	}, nil
}

// establishSSHConnection creates a complete SSH connection using tsnet.
// This function consolidates the connection establishment pattern used across
// multiple files, providing a standardized way to connect to SSH hosts via Tailscale.
//
// The connection process includes:
//   1. Creating SSH client configuration
//   2. Establishing TCP connection via tsnet
//   3. Performing SSH handshake
//   4. Returning ready-to-use SSH client
//
// Returns an active ssh.Client that must be closed by the caller.
func establishSSHConnection(srv *tsnet.Server, ctx context.Context, config SSHConnectionConfig) (*ssh.Client, error) {
	// Create SSH configuration
	sshConfig, err := createSSHConfig(config)
	if err != nil {
		return nil, err
	}

	// Create connection address
	sshTargetAddr := net.JoinHostPort(config.TargetHost, config.TargetPort)
	
	if config.Logger != nil {
		config.Logger.Printf(T("dial_via_tsnet"), sshTargetAddr)
	}

	// Dial via tsnet
	conn, err := srv.Dial(ctx, "tcp", sshTargetAddr)
	if err != nil {
		return nil, fmt.Errorf(T("dial_failed", sshTargetAddr, err))
	}

	// Establish SSH connection
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, sshTargetAddr, sshConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf(T("ssh_connection_failed", sshTargetAddr, err))
	}

	client := ssh.NewClient(sshConn, chans, reqs)
	
	if config.Logger != nil {
		config.Logger.Printf(T("ssh_connection_established"))
	}

	return client, nil
}

// createSSHSession creates an SSH session with standard configuration
// This standardizes session creation across different use cases
func createSSHSession(client *ssh.Client) (*ssh.Session, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	return session, nil
}