package main

import (
	"fmt"
	"os"
	// "path/filepath" // Removed as it's not used

	"context" // Keep one context import
	"errors"
	"log"
	"net"
	"os/user"
	"syscall"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term" // term is used for password prompt
	"tailscale.com/tsnet"
)
// Removed "fmt" and "os" as they are available from main package context
// Removed "golang.org/x/crypto/ssh/knownhosts" as it's not directly used here
// Removed "path/filepath" as it's not used

// performSCPTransfer handles the SCP file transfer logic.
// It uses the provided tsnet.Server to connect to the target host.
// scpDetails contains all necessary parameters for the transfer.
func performSCPTransfer(
	srv *tsnet.Server,
	appCtx context.Context,
	logger *log.Logger,
	scpDetails tuiActionResult, // This struct is defined in tui.go, ensure it's accessible or redefined/passed as individual params
	sshUser string,
	sshKeyPath string,
	insecureHostKey bool,
	currentUser *user.User, // For known_hosts
	verbose bool,
) error {
	logger.Printf("Performing SCP: Host=%s, Local=%s, Remote=%s, Upload=%t",
		scpDetails.selectedHostTarget, scpDetails.scpLocalPath, scpDetails.scpRemotePath, scpDetails.scpIsUpload)

	if scpDetails.scpLocalPath == "" || scpDetails.scpRemotePath == "" {
		return errors.New("local or remote path for SCP cannot be empty")
	}

	sshTargetAddr := net.JoinHostPort(scpDetails.selectedHostTarget, DefaultSshPort) // Use public constant

	var authMethods []ssh.AuthMethod
	if sshKeyPath != "" {
		keyAuth, keyErr := LoadPrivateKey(sshKeyPath, logger) 
		if keyErr == nil {
			authMethods = append(authMethods, keyAuth)
			logger.Printf("SCP Connect: Using public key authentication: %s", sshKeyPath)
		} else {
			logger.Printf("SCP Connect: Could not load private key %q: %v. Will attempt password auth.", sshKeyPath, keyErr)
		}
	} else {
		logger.Printf("SCP Connect: No SSH key path specified. Will attempt password auth.")
	}

	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		// This requires terminal interaction.
		fmt.Printf("Enter password for %s@%s (for SCP): ", sshUser, scpDetails.selectedHostTarget)
		bytePassword, passErr := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if passErr != nil {
			return "", fmt.Errorf("failed to read password for SCP: %w", passErr)
		}
		return string(bytePassword), nil
	}))

	var hostKeyCallback ssh.HostKeyCallback
	var hkErr error
	if insecureHostKey {
		logger.Println("SCP Connect: WARNING! Host key verification is disabled!")
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		// createKnownHostsCallback is in ssh_client.go - now public
		hostKeyCallback, hkErr = CreateKnownHostsCallback(currentUser, logger) 
		if hkErr != nil {
			return fmt.Errorf("SCP: Could not set up host key verification: %w", hkErr)
		}
		logger.Println("SCP Connect: Using known_hosts for host key verification.")
	}

	scpSSHConfig := ssh.ClientConfig{
		User:            sshUser,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second, // SCP might need longer timeouts for large files
	}

	logger.Printf("SCP Connect: Dialing %s via tsnet...", sshTargetAddr)
	dialCtx, dialCancel := context.WithTimeout(appCtx, scpSSHConfig.Timeout)
	defer dialCancel()

	conn, err := srv.Dial(dialCtx, "tcp", sshTargetAddr)
	if err != nil {
		return fmt.Errorf("SCP: tsnet dial failed for %s: %w", sshTargetAddr, err)
	}
	// ssh.NewClientConn will close 'conn' if it fails.
	// If it succeeds, the resulting ssh.Client's Close method will close 'conn'.

	logger.Printf("SCP Connect: tsnet Dial successful. Establishing SSH client for SCP...")
	sshClientConn, chans, reqs, err := ssh.NewClientConn(conn, sshTargetAddr, &scpSSHConfig)
	if err != nil {
		conn.Close() // Explicitly close on this error path
		return fmt.Errorf("SCP: failed to establish SSH client connection: %w", err)
	}
	// Wrap the ssh.ClientConn in an ssh.Client
	sshClient := ssh.NewClient(sshClientConn, chans, reqs)
	defer sshClient.Close() // This will also close the underlying sshClientConn and net.Conn

	// At this point, sshClient is an *ssh.Client
	scpClient, err := scp.NewClientBySSH(sshClient)
	if err != nil {
		return fmt.Errorf("error creating new SCP client: %w", err)
	}
	defer scpClient.Close() // This closes the SCP session, not the SSH client.

	if scpDetails.scpIsUpload {
		logger.Printf("SCP: Uploading %s to %s:%s", scpDetails.scpLocalPath, scpDetails.selectedHostTarget, scpDetails.scpRemotePath)
		localFile, errOpen := os.Open(scpDetails.scpLocalPath)
		if errOpen != nil {
			return fmt.Errorf("failed to open local file %s for upload: %w", scpDetails.scpLocalPath, errOpen)
		}
		defer localFile.Close()

		fileInfo, errStat := localFile.Stat()
		if errStat != nil {
			return fmt.Errorf("failed to get file info for local file %s: %w", scpDetails.scpLocalPath, errStat)
		}
		// SCP permissions are typically octal, e.g., "0644"
		permissions := fmt.Sprintf("0%o", fileInfo.Mode().Perm())

		// Use appCtx for the CopyFile operation for cancellability
		errCopy := scpClient.CopyFile(appCtx, localFile, scpDetails.scpRemotePath, permissions)
		if errCopy != nil {
			return fmt.Errorf("error uploading file via SCP: %w", errCopy)
		}
		logger.Println("SCP: Upload complete.")
	} else { // Download
		logger.Printf("SCP: Downloading %s:%s to %s", scpDetails.selectedHostTarget, scpDetails.scpRemotePath, scpDetails.scpLocalPath)
		// Ensure local directory exists if downloading to a nested path (optional, depends on desired behavior)
		// For now, assume os.OpenFile will handle it or fail if dir doesn't exist.
		localFile, errOpen := os.OpenFile(scpDetails.scpLocalPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if errOpen != nil {
			return fmt.Errorf("failed to open/create local file %s for download: %w", scpDetails.scpLocalPath, errOpen)
		}
		defer localFile.Close()

		// Use appCtx for the CopyFromRemote operation
		errCopy := scpClient.CopyFromRemote(appCtx, localFile, scpDetails.scpRemotePath)
		if errCopy != nil {
			// If context was cancelled, this might return a context-related error.
			if appCtx.Err() != nil {
				logger.Printf("SCP download cancelled: %v", appCtx.Err())
				return fmt.Errorf("scp download cancelled: %w", appCtx.Err())
			}
			return fmt.Errorf("error downloading file via SCP: %w", errCopy)
		}
		logger.Println("SCP: Download complete.")
	}
	return nil
}

// Placeholder for tuiActionResult if it's defined in tui.go
// If tui.go is in the same 'main' package, this isn't strictly needed here
// but helps clarify dependency if files were in different packages.
// type tuiActionResult struct {
// 	action             string
// 	selectedHostTarget string
// 	scpLocalPath       string
// 	scpRemotePath      string
// 	scpIsUpload        bool
// }

// These would be needed if ssh_client.go was a different package.
// Since it's all `package main`, direct calls are fine.
// func loadPrivateKey(path string, logger *log.Logger) (ssh.AuthMethod, error)
// func createKnownHostsCallback(currentUser *user.User, logger *log.Logger) (ssh.HostKeyCallback, error)

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
		return errors.New("local or remote path for SCP cannot be empty")
	}

	// Ensure defaultSSHPort is accessible. For now, define locally if not shared.
	// const defaultSSHPort = "22" // Already defined in this file for performSCPTransfer

	sshTargetAddr := net.JoinHostPort(targetHost, DefaultSshPort) // Use public constant

	var authMethods []ssh.AuthMethod
	if sshKeyPath != "" {
		// Call the exported function from ssh_client.go
		keyAuth, keyErr := LoadPrivateKey(sshKeyPath, logger) 
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
		fmt.Printf("Enter password for %s@%s (for SCP): ", sshUser, targetHost)
		bytePassword, passErr := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if passErr != nil {
			return "", fmt.Errorf("failed to read password for SCP: %w", passErr)
		}
		return string(bytePassword), nil
	}))

	var hostKeyCallback ssh.HostKeyCallback
	var hkErr error
	if insecureHostKey {
		logger.Println("CLI SCP: WARNING! Host key verification is disabled!")
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		// Call the exported function from ssh_client.go
		hostKeyCallback, hkErr = CreateKnownHostsCallback(currentUser, logger)
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
		logger.Println("CLI SCP: Upload complete.")
	} else { // Download
		logger.Printf("CLI SCP: Downloading %s@%s:%s to %s", sshUser, targetHost, remotePath, localPath)
		
		localFile, errOpen := os.OpenFile(localPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if errOpen != nil {
			return fmt.Errorf("CLI SCP: failed to open/create local file %s for download: %w", localPath, errOpen)
		}
		defer localFile.Close()

		errCopy := scpCl.CopyFromRemote(ctx, localFile, remotePath)
		if errCopy != nil {
			if ctx.Err() != nil {
				logger.Printf("CLI SCP download cancelled: %v", ctx.Err())
				return fmt.Errorf("CLI SCP download cancelled: %w", ctx.Err())
			}
			return fmt.Errorf("CLI SCP: error downloading file: %w", errCopy)
		}
		logger.Println("CLI SCP: Download complete.")
	}
	return nil
}
