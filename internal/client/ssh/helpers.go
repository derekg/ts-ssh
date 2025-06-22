package ssh

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/user"
	"time"

	"golang.org/x/crypto/ssh"
	"tailscale.com/tsnet"
	
	"github.com/derekg/ts-ssh/internal/config"
	"github.com/derekg/ts-ssh/internal/crypto/pqc"
)

// Constants needed by SSH package

const (
	DefaultSshPort = config.DefaultSSHPort
)

// SSH key discovery constants (imported from config)
var (
	ModernKeyTypes = config.ModernKeyTypes
)

// Timeout constants
const (
	DefaultSSHTimeout = 15 * time.Second
)

// Simple T function for temporary internationalization support
// TODO: Replace with proper i18n integration
func T(key string, args ...interface{}) string {
	translations := map[string]string{
		"host_key_warning": "WARNING: Host key verification is disabled",
		"dial_via_tsnet": "Connecting via tsnet...",
		"ssh_handshake": "Performing SSH handshake...",
	}
	
	if msg, ok := translations[key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...)
		}
		return msg
	}
	return key
}

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
	PQCConfig       *pqc.Config // Post-quantum cryptography configuration
}

// createSSHAuthMethods creates authentication methods for SSH connection.
// It uses modern key discovery to automatically find the best available SSH key,
// prioritizing Ed25519 over legacy RSA keys. Always includes password auth as fallback.
//
// Parameters:
//   - keyPath: path to SSH private key file (optional, if empty uses auto-discovery)
//   - sshUser: username for SSH connection
//   - targetHost: hostname for password prompts
//   - logger: logger instance for debug output
//
// Returns a slice of ssh.AuthMethod and any error that occurred.
func createSSHAuthMethods(keyPath, sshUser, targetHost string, logger *log.Logger) ([]ssh.AuthMethod, error) {
	// Get current user for key discovery
	currentUser, err := user.Current()
	if err != nil && logger != nil {
		logger.Printf("Warning: Could not get current user for SSH key discovery: %v", err)
	}

	// Use the modern key discovery system
	return createModernSSHAuthMethods(keyPath, sshUser, targetHost, currentUser, logger)
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
			config.Logger.Printf("%s", T("host_key_warning"))
		}
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		var err error
		hostKeyCallback, err = CreateKnownHostsCallback(config.CurrentUser, config.Logger)
		if err != nil {
			return nil, fmt.Errorf("could not set up host key verification: %w", err)
		}
	}

	sshConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         DefaultSSHTimeout,
	}
	
	// Apply PQC configuration if provided
	if config.PQCConfig != nil {
		pqc.ConfigureSSHConfig(sshConfig, config.PQCConfig)
		if config.Logger != nil && config.PQCConfig.EnablePQC {
			config.Logger.Printf("PQC: Post-quantum cryptography enabled (level: %d)", config.PQCConfig.QuantumResistance)
		}
	}
	
	return sshConfig, nil
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
func EstablishSSHConnection(srv *tsnet.Server, ctx context.Context, config SSHConnectionConfig) (*ssh.Client, error) {
	// Create SSH configuration
	sshConfig, err := createSSHConfig(config)
	if err != nil {
		return nil, err
	}

	// Create connection address
	sshTargetAddr := net.JoinHostPort(config.TargetHost, config.TargetPort)
	
	if config.Logger != nil {
		config.Logger.Printf("%s", T("dial_via_tsnet"))
	}

	// Dial via tsnet
	conn, err := srv.Dial(ctx, "tcp", sshTargetAddr)
	if err != nil {
		return nil, fmt.Errorf("%s", T("dial_failed"))
	}

	// Establish SSH connection
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, sshTargetAddr, sshConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("%s", T("ssh_connection_failed"))
	}

	client := ssh.NewClient(sshConn, chans, reqs)
	
	if config.Logger != nil {
		config.Logger.Printf("%s", T("ssh_connection_established"))
	}

	return client, nil
}

// CreateSSHSession creates an SSH session with standard configuration
// This standardizes session creation across different use cases
func CreateSSHSession(client *ssh.Client) (*ssh.Session, error) {
	if client == nil {
		return nil, fmt.Errorf("SSH client cannot be nil")
	}
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	return session, nil
}