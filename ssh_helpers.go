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

// createSSHAuthMethods creates authentication methods for SSH connection
// This eliminates the duplication of auth setup across multiple files
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
		fmt.Printf(T("enter_password"), sshUser, targetHost)
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		return string(bytePassword), nil
	}))

	return authMethods, nil
}

// createSSHConfig creates an SSH client configuration
// This standardizes SSH config creation across the codebase
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

// establishSSHConnection creates a complete SSH connection using tsnet
// This consolidates the connection establishment pattern used across multiple files
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
		return nil, fmt.Errorf(T("dial_failed"), sshTargetAddr, err)
	}

	// Establish SSH connection
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, sshTargetAddr, sshConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf(T("ssh_connection_failed"), sshTargetAddr, err)
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