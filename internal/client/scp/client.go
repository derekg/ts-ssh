package scp

import (
	"fmt"
	"os"
	"context"
	"errors"
	"log"
	"net"
	"os/user"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"golang.org/x/crypto/ssh"
	"tailscale.com/tsnet"

	sshclient "github.com/derekg/ts-ssh/internal/client/ssh"
	"github.com/derekg/ts-ssh/internal/config"
	"github.com/derekg/ts-ssh/internal/security"
)

// Constants needed by SCP package
const (
	DefaultSshPort = config.DefaultSSHPort
)

// Simple T function for temporary internationalization support
// TODO: Replace with proper i18n integration
func T(key string, args ...interface{}) string {
	translations := map[string]string{
		"scp_empty_path": "SCP path cannot be empty",
		"scp_enter_password": "Enter password for %s@%s: ",
		"dial_via_tsnet": "Connecting via tsnet...",
		"dial_failed": "Connection failed",
		"scp_host_key_warning": "WARNING: SCP host key verification disabled",
	}
	
	if msg, ok := translations[key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...)
		}
		return msg
	}
	return key
}
// Removed "fmt" and "os" as they are available from main package context
// Removed "golang.org/x/crypto/ssh/knownhosts" as it's not directly used here
// Removed "path/filepath" as it's not used

// HandleCliScp performs an SCP operation based on CLI arguments.
func HandleCliScp(
	srv *tsnet.Server,
	ctx context.Context,
	logger *log.Logger,
	sshUser string, // User for the SSH connection
	sshKeyPath string,
	insecureHostKey bool,
	currentUser *user.User, // For known_hosts
	localPath string,
	remotePath string,
	targetHost string, // Host for the SCP operation
	isUpload bool,
	verbose bool,
) error {
	logger.Printf("CLI SCP: Host=%s, User=%s, LocalPath=%s, RemotePath=%s, Upload=%t, KeyPath=%s",
		targetHost, sshUser, localPath, remotePath, isUpload, sshKeyPath)

	if localPath == "" || remotePath == "" {
		return errors.New(T("scp_empty_path"))
	}

	// Ensure defaultSSHPort is accessible. For now, define locally if not shared.
	// const defaultSSHPort = "22" // Already defined in this file for performSCPTransfer

	sshTargetAddr := net.JoinHostPort(targetHost, DefaultSshPort) // Use public constant

	var authMethods []ssh.AuthMethod
	if sshKeyPath != "" {
		// Call the exported function from ssh_client.go
		keyAuth, keyErr := sshclient.LoadPrivateKey(sshKeyPath, logger) 
		if keyErr == nil {
			authMethods = append(authMethods, keyAuth)
			logger.Printf("CLI SCP: Using public key authentication: %s", sshKeyPath)
		} else {
			logger.Printf("CLI SCP: Could not load private key %q: %v. Will attempt password auth.", sshKeyPath, keyErr)
		}
	} else {
		logger.Printf("CLI SCP: No SSH key path specified. Will attempt password auth.")
	}

	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		fmt.Print(T("scp_enter_password", sshUser, targetHost))
		password, passErr := security.ReadPasswordSecurely()
		fmt.Println()
		if passErr != nil {
			return "", fmt.Errorf("failed to read password securely for SCP: %w", passErr)
		}
		return password, nil
	}))

	var hostKeyCallback ssh.HostKeyCallback
	var hkErr error
	if insecureHostKey {
		logger.Println(T("scp_host_key_warning"))
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		// Call the exported function from ssh_client.go
		hostKeyCallback, hkErr = sshclient.CreateKnownHostsCallback(currentUser, logger)
		if hkErr != nil {
			return fmt.Errorf("CLI SCP: Could not set up host key verification: %w", hkErr)
		}
		// Message about using known_hosts is logged by CreateKnownHostsCallback
	}

	cliScpSSHConfig := ssh.ClientConfig{
		User:            sshUser,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second, // SCP might need longer timeouts for large files
	}

	logger.Printf("CLI SCP: Dialing %s via tsnet...", sshTargetAddr)
	dialCtx, dialCancel := context.WithTimeout(ctx, cliScpSSHConfig.Timeout)
	defer dialCancel()

	conn, err := srv.Dial(dialCtx, "tcp", sshTargetAddr)
	if err != nil {
		return fmt.Errorf("CLI SCP: tsnet dial failed for %s: %w", sshTargetAddr, err)
	}
	
	logger.Printf("CLI SCP: tsnet Dial successful. Establishing SSH client for SCP...")
	sshClientConn, chans, reqs, err := ssh.NewClientConn(conn, sshTargetAddr, &cliScpSSHConfig)
	if err != nil {
		conn.Close() 
		return fmt.Errorf("CLI SCP: failed to establish SSH client connection: %w", err)
	}
	sshClient := ssh.NewClient(sshClientConn, chans, reqs)
	defer sshClient.Close() 

	scpCl, err := scp.NewClientBySSH(sshClient)
	if err != nil {
		return fmt.Errorf("CLI SCP: error creating new SCP client: %w", err)
	}
	defer scpCl.Close()

	if isUpload {
		logger.Printf("CLI SCP: Uploading %s to %s@%s:%s", localPath, sshUser, targetHost, remotePath)
		localFile, errOpen := os.Open(localPath)
		if errOpen != nil {
			return fmt.Errorf("CLI SCP: failed to open local file %s for upload: %w", localPath, errOpen)
		}
		defer localFile.Close()

		fileInfo, errStat := localFile.Stat()
		if errStat != nil {
			return fmt.Errorf("CLI SCP: failed to get file info for local file %s: %w", localPath, errStat)
		}
		permissions := fmt.Sprintf("0%o", fileInfo.Mode().Perm())

		errCopy := scpCl.CopyFile(ctx, localFile, remotePath, permissions)
		if errCopy != nil {
			return fmt.Errorf("CLI SCP: error uploading file: %w", errCopy)
		}
		logger.Println(T("scp_upload_complete"))
	} else { // Download
		logger.Printf("CLI SCP: Downloading %s@%s:%s to %s", sshUser, targetHost, remotePath, localPath)
		
		// Create file securely with atomic replacement to prevent race conditions
		localFile, errOpen := security.CreateSecureDownloadFileWithReplace(localPath)
		if errOpen != nil {
			return fmt.Errorf("CLI SCP: failed to create secure local file %s for download: %w", localPath, errOpen)
		}
		defer func() {
			if err := security.CompleteAtomicReplacement(localFile); err != nil {
				logger.Printf("Warning: failed to complete atomic file replacement: %v", err)
			}
		}()

		errCopy := scpCl.CopyFromRemote(ctx, localFile, remotePath)
		if errCopy != nil {
			if ctx.Err() != nil {
				logger.Printf("CLI SCP download cancelled: %v", ctx.Err())
				return fmt.Errorf("CLI SCP download cancelled: %w", ctx.Err())
			}
			return fmt.Errorf("CLI SCP: error downloading file: %w", errCopy)
		}
		logger.Println(T("scp_download_complete"))
	}
	return nil
}
