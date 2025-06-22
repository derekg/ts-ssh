package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
	"tailscale.com/tsnet"
)

// defaultSSHPort is defined in main.go (or should be made accessible globally)
// For now, we assume it's accessible or HandleCliScp will use its own.

// connectToHost handles SSH connection and starts an interactive session
// This replaces the old TUI-specific connection logic with standardized SSH helpers
func connectToHost(
	srv *tsnet.Server,
	appCtx context.Context, 
	logger *log.Logger,
	targetHost string, 
	sshUser string,
	sshKeyPath string,
	insecureHostKey bool,
	currentUser *user.User, 
	verbose bool,
) error {
	// Use the standard SSH helper configuration
	sshConfig := SSHConnectionConfig{
		User:            sshUser,
		KeyPath:         sshKeyPath,
		TargetHost:      targetHost,
		TargetPort:      DefaultSshPort,
		InsecureHostKey: insecureHostKey,
		Verbose:         verbose,
		CurrentUser:     currentUser,
		Logger:          logger,
	}

	// Establish SSH connection using standardized helper
	client, err := establishSSHConnection(srv, appCtx, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to establish SSH connection: %w", err)
	}
	defer client.Close()

	// Start interactive session using standardized helper
	return startInteractiveSession(client, logger)
}

// LoadPrivateKey loads an SSH private key from the given path.
// It supports unencrypted keys and keys encrypted with a passphrase, prompting for it if needed.
func LoadPrivateKey(path string, logger *log.Logger) (ssh.AuthMethod, error) {
	if path == "" {
		return nil, errors.New("private key path is empty")
	}
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading key file %q failed: %w", path, err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err == nil {
		return ssh.PublicKeys(signer), nil
	}

	var passphraseErr *ssh.PassphraseMissingError
	if errors.As(err, &passphraseErr) {
		logger.Printf("SSH key %s is passphrase protected.", path)
		fmt.Printf("Enter passphrase for key %s: ", path)
		password, errRead := readPasswordSecurely()
		fmt.Println()
		if errRead != nil {
			return nil, fmt.Errorf("failed to read passphrase securely: %w", errRead)
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(password))
		if err != nil {
			if strings.Contains(err.Error(), "incorrect passphrase") || strings.Contains(err.Error(), "decryption failed") {
				return nil, fmt.Errorf("incorrect passphrase for key %q", path)
			}
			return nil, fmt.Errorf("parsing key %q with passphrase failed: %w", path, err)
		}
		return ssh.PublicKeys(signer), nil
	}

	return nil, fmt.Errorf("parsing private key %q failed: %w", path, err)
}

// CreateKnownHostsCallback returns a ssh.HostKeyCallback that uses a known_hosts file.
// It will prompt the user to add new host keys if the host is not found.
func CreateKnownHostsCallback(currentUser *user.User, logger *log.Logger) (ssh.HostKeyCallback, error) {
	if currentUser == nil || currentUser.HomeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Printf("Warning: Cannot determine user home directory for known_hosts: %v. Host key checking may be impaired or prompt.", err)
			return nil, fmt.Errorf("user home directory unknown, cannot reliably manage known_hosts: %w", err)
		}
		currentUser = &user.User{HomeDir: home} 
		logger.Printf("Warning: currentUser was nil or HomeDir empty. Deduced home as %s for known_hosts.", home)
	}

	knownHostsPath := filepath.Join(currentUser.HomeDir, ".ssh", "known_hosts")

	// Create known_hosts file securely to prevent race conditions
	if err := createSecureKnownHostsFile(knownHostsPath); err != nil {
		logger.Printf("Unable to create secure known_hosts file %s: %v. Host key management will be impaired.", knownHostsPath, err)
	}

	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		logger.Printf("Could not initialize known_hosts callback using %s: %v. Host key verification will prompt for every new host without persistence.", knownHostsPath, err)
		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return handleHostKey(hostname, remote, key, "", logger) 
		}, nil
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := hostKeyCallback(hostname, remote, key)
		if err == nil {
			return nil 
		}
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) {
			return handleHostKey(hostname, remote, key, knownHostsPath, logger, keyErr)
		}
		logger.Printf("Unexpected error during host key verification for %s: %v", hostname, err)
		return fmt.Errorf("unexpected error during host key verification: %w", err)
	}, nil
}

func handleHostKey(hostname string, remote net.Addr, key ssh.PublicKey, knownHostsPath string, logger *log.Logger, keyErr ...*knownhosts.KeyError) error {
	var specificKeyError *knownhosts.KeyError
	if len(keyErr) > 0 {
		specificKeyError = keyErr[0]
	}

	if specificKeyError != nil && len(specificKeyError.Want) > 0 {
		logger.Printf("WARNING: Remote host identification has changed for %s!", hostname)
		fmt.Fprintf(os.Stderr, "\n@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n")
		fmt.Fprintf(os.Stderr, "@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @\n")
		fmt.Fprintf(os.Stderr, "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n")
		fmt.Fprintf(os.Stderr, "IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!\n")
		fmt.Fprintf(os.Stderr, "Someone could be eavesdropping on you right now (man-in-the-middle attack)!\n")
		fmt.Fprintf(os.Stderr, "It is also possible that a host key has just been changed.\n")
		fmt.Fprintf(os.Stderr, "The fingerprint for the %s key sent by the remote host %s is:\n%s\n", key.Type(), remote.String(), ssh.FingerprintSHA256(key))
		fmt.Fprintf(os.Stderr, "Please contact your system administrator.\n")
		for _, kh := range specificKeyError.Want {
			fmt.Fprintf(os.Stderr, "Offending ECDSA key in %s:%d\n", kh.Filename, kh.Line)
		}
		return specificKeyError 
	} else {
		fmt.Fprintf(os.Stderr, "The authenticity of host '%s (%s)' can't be established.\n", hostname, remote.String())
		fmt.Fprintf(os.Stderr, "%s key fingerprint is %s.\n", key.Type(), ssh.FingerprintSHA256(key))

		answer, readErr := promptUserViaTTY(fmt.Sprintf("Are you sure you want to continue connecting (yes/no/[fingerprint])? "), logger)
		if readErr != nil {
			return fmt.Errorf("failed to read user confirmation: %w", readErr)
		}

		if strings.ToLower(answer) == "yes" {
			if knownHostsPath == "" {
				logger.Printf("Warning: Host key for %s accepted but known_hosts path is not available. Key not persisted.", hostname)
				return nil 
			}
			return appendKnownHost(knownHostsPath, hostname, remote, key, logger)
		} else if strings.ToLower(answer) == "fingerprint" {
			fmt.Fprintf(os.Stderr, "Re-displaying fingerprint for verification: %s\n", ssh.FingerprintSHA256(key))
			answer, readErr = promptUserViaTTY("Are you sure you want to continue connecting (yes/no)? ", logger)
			if readErr != nil {
				return fmt.Errorf("failed to read user re-confirmation: %w", readErr)
			}
			if strings.ToLower(answer) == "yes" {
				if knownHostsPath == "" {
					logger.Printf("Warning: Host key for %s accepted but known_hosts path is not available. Key not persisted.", hostname)
					return nil
				}
				return appendKnownHost(knownHostsPath, hostname, remote, key, logger)
			}
			return errors.New("host key verification failed: user declined after fingerprint display")
		} else {
			return errors.New("host key verification failed: user declined")
		}
	}
}

func appendKnownHost(knownHostsPath, hostname string, remote net.Addr, key ssh.PublicKey, logger *log.Logger) error {
	if knownHostsPath == "" {
		return errors.New("cannot append known host: path is empty")
	}

	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s to append new key: %w", knownHostsPath, err)
	}
	defer f.Close()
	
	var addresses []string
	normalizedRemoteAddr := knownhosts.Normalize(remote.String())
	addresses = append(addresses, hostname)
	if hostname != normalizedRemoteAddr && !strings.Contains(normalizedRemoteAddr, "[") { 
		isDuplicate := false
		for _, addr := range addresses {
			if addr == normalizedRemoteAddr {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			addresses = append(addresses, normalizedRemoteAddr)
		}
	}

	line := knownhosts.Line(addresses, key)
	if _, err := f.WriteString(line + "\n"); err != nil { 
		return fmt.Errorf("failed to write host key to %s: %w", knownHostsPath, err)
	}
	logger.Printf("Host key for %s (%s) added to %s.", hostname, key.Type(), knownHostsPath)
	fmt.Fprintf(os.Stderr, "Warning: Permanently added '%s' (%s) to the list of known hosts.\n", hostname, key.Type())
	return nil
}

// getSigWinch returns SIGWINCH on Unix platforms, nil on Windows
func getSigWinch() os.Signal {
	if runtime.GOOS == "windows" {
		return nil
	}
	// This will only compile on Unix platforms
	return syscall.Signal(0x1c) // SIGWINCH value on most Unix systems
}

func watchWindowSize(fd int, session *ssh.Session, ctx context.Context, logger *log.Logger) {
	// Window resize monitoring is limited on some platforms
	if runtime.GOOS == "windows" {
		logger.Println("Window resize monitoring not supported on Windows")
		return
	}
	
	sigCh := make(chan os.Signal, 1)
	// Use reflection to access SIGWINCH on Unix platforms only
	if sigWinch := getSigWinch(); sigWinch != nil {
		signal.Notify(sigCh, sigWinch)
	}
	defer signal.Stop(sigCh) 

	if term.IsTerminal(fd) {
		termWidth, termHeight, err := term.GetSize(fd)
		if err == nil && termWidth > 0 && termHeight > 0 {
			if err := session.WindowChange(termHeight, termWidth); err != nil {
				logger.Printf("watchWindowSize: Error sending initial window size: %v", err)
			}
		} else if err != nil {
			logger.Printf("watchWindowSize: Error getting initial terminal size: %v", err)
		}
	}

	for {
		select {
		case <-sigCh:
			if term.IsTerminal(fd) {
				termWidth, termHeight, err := term.GetSize(fd)
				if err == nil && termWidth > 0 && termHeight > 0 {
					if err := session.WindowChange(termHeight, termWidth); err != nil {
						logger.Printf("watchWindowSize: Error sending window change: %v", err)
						if strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "closed") {
							logger.Println("watchWindowSize: Session appears to be closed, exiting.")
							return
						}
					}
				} else if err != nil {
					logger.Printf("watchWindowSize: Error getting terminal size on SIGWINCH: %v", err)
				}
			}
		case <-ctx.Done():
			logger.Println("watchWindowSize: Context cancelled, stopping window size watcher.")
			return
		}
	}
}
