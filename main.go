package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	osuser "os/user"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
	"tailscale.com/tsnet"

	"github.com/derekg/ts-ssh/internal/client/scp"
	sshclient "github.com/derekg/ts-ssh/internal/client/ssh"
	"github.com/derekg/ts-ssh/internal/security"
)

// version is set at build time via -ldflags
var version = "dev"

func main() {
	// Initialize security audit logging
	if err := security.InitSecurityLogger(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize security audit logging: %v\n", err)
	}
	defer security.CloseSecurityLogger()

	// Parse flags
	var (
		sshUser        = flag.String("l", currentUsername(), "SSH username")
		sshPort        = flag.String("p", "22", "SSH port")
		keyPath        = flag.String("i", defaultKeyPath(), "SSH private key path")
		tsnetDir       = flag.String("tsnet-dir", defaultTsnetDir(), "Tailscale state directory")
		controlURL     = flag.String("control-url", "", "Tailscale control server URL")
		verbose        = flag.Bool("v", false, "Verbose output")
		insecure       = flag.Bool("insecure", false, "Skip host key verification (insecure)")
		scpMode        = flag.Bool("scp", false, "SCP mode: ts-ssh -scp source dest")
		showVersion    = flag.Bool("version", false, "Show version")
		disablePTY     = flag.Bool("T", false, "Disable pseudo-terminal allocation")
		dynamicForward = flag.String("D", "", "SOCKS5 dynamic port forwarding on [bind_address:]port")
	)

	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	// Setup logger
	logger := log.New(io.Discard, "", 0)
	if *verbose {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	args := flag.Args()

	// SCP mode: ts-ssh -scp source dest
	if *scpMode {
		if len(args) != 2 {
			fmt.Fprintf(os.Stderr, "Error: SCP mode requires exactly 2 arguments (source dest)\n")
			os.Exit(1)
		}
		if err := runSCP(args[0], args[1], *sshUser, *keyPath, *tsnetDir, *controlURL, *insecure, *verbose, logger); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// SSH mode: ts-ssh [user@]host[:port] [command...]
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: target hostname required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	target := args[0]
	var remoteCmd []string
	if len(args) > 1 {
		remoteCmd = args[1:]
	}

	if err := runSSH(target, remoteCmd, *sshUser, *sshPort, *keyPath, *tsnetDir, *controlURL, *insecure, *disablePTY, *dynamicForward, *verbose, logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] [user@]host[:port] [command...]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "       %s -scp source dest\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "SSH over Tailscale without requiring a full Tailscale daemon\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s hostname                    # Interactive SSH\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s user@hostname uptime        # Execute command\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s hostname:2222               # Custom port\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -scp file.txt host:/tmp/    # Copy file\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -v hostname                 # Verbose mode\n", os.Args[0])
}

// runSSH handles the SSH connection
func runSSH(target string, remoteCmd []string, defaultUser, defaultPort, keyPath, tsnetDir, controlURL string, insecure, disablePTY bool, dynamicForward string, verbose bool, logger *log.Logger) error {
	// Parse target: [user@]host[:port]
	sshUser, host, port, err := parseSSHTarget(target, defaultUser, defaultPort)
	if err != nil {
		return err
	}

	// Validate inputs
	if err := security.ValidateSSHUser(sshUser); err != nil {
		return fmt.Errorf("invalid SSH user: %w", err)
	}
	if err := security.ValidateHostname(host); err != nil {
		return fmt.Errorf("invalid hostname: %w", err)
	}
	if err := security.ValidatePort(port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	// Initialize tsnet
	srv, ctx, err := initTailscale(tsnetDir, controlURL, verbose, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize Tailscale: %w", err)
	}

	// Establish SSH connection
	client, err := connectSSH(srv, ctx, sshUser, host, port, keyPath, insecure, verbose, logger)
	if err != nil {
		return fmt.Errorf("failed to connect via SSH: %w", err)
	}
	defer client.Close()

	// Setup dynamic port forwarding if requested
	if dynamicForward != "" {
		if err := setupDynamicForward(client, dynamicForward, verbose, logger); err != nil {
			return fmt.Errorf("failed to setup dynamic forwarding: %w", err)
		}
	}

	// Execute command or start interactive session
	if len(remoteCmd) > 0 {
		return execRemoteCommand(client, remoteCmd, logger)
	}

	return interactiveSession(client, disablePTY, logger)
}

// runSCP handles SCP file transfer
func runSCP(source, dest, defaultUser, keyPath, tsnetDir, controlURL string, insecure, verbose bool, logger *log.Logger) error {
	// Determine which is local and which is remote
	srcHost, srcPath, srcIsRemote := parseSCPArg(source)
	dstHost, dstPath, dstIsRemote := parseSCPArg(dest)

	// Exactly one must be remote
	if srcIsRemote == dstIsRemote {
		return fmt.Errorf("exactly one of source or destination must be remote (host:path)")
	}

	var targetHost, remotePath, localPath, sshUser string
	var upload bool

	if srcIsRemote {
		// Download: remote -> local
		targetHost = srcHost
		remotePath = srcPath
		localPath = dstPath
		upload = false
	} else {
		// Upload: local -> remote
		targetHost = dstHost
		remotePath = dstPath
		localPath = srcPath
		upload = true
	}

	// Parse target host for user@host[:port]
	sshUser, host, port, err := parseSSHTarget(targetHost, defaultUser, "22")
	if err != nil {
		return err
	}

	// Validate inputs
	if err := security.ValidateSSHUser(sshUser); err != nil {
		return fmt.Errorf("invalid SSH user: %w", err)
	}
	if err := security.ValidateHostname(host); err != nil {
		return fmt.Errorf("invalid hostname: %w", err)
	}

	// Initialize tsnet
	srv, ctx, err := initTailscale(tsnetDir, controlURL, verbose, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize Tailscale: %w", err)
	}

	// Get current user for SCP client
	currentUser, err := osuser.Current()
	if err != nil {
		currentUser = &osuser.User{Username: sshUser}
	}

	// Perform SCP operation
	addr := host + ":" + port
	if err := scp.HandleCliScp(srv, ctx, logger, sshUser, keyPath, insecure, currentUser,
		localPath, remotePath, addr, upload, verbose); err != nil {
		return fmt.Errorf("SCP failed: %w", err)
	}

	if verbose {
		logger.Println("SCP transfer completed successfully")
	}
	return nil
}

// parseSSHTarget parses [user@]host[:port] and returns user, host, port
func parseSSHTarget(target, defaultUser, defaultPort string) (user, host, port string, err error) {
	user = defaultUser
	host = target
	port = defaultPort

	// Extract user if present
	if strings.Contains(host, "@") {
		parts := strings.SplitN(host, "@", 2)
		user = parts[0]
		host = parts[1]
	}

	// Extract port if present
	if strings.Contains(host, ":") {
		// Handle IPv6 addresses [::1]:port
		if strings.HasPrefix(host, "[") {
			endBracket := strings.Index(host, "]")
			if endBracket == -1 {
				return "", "", "", fmt.Errorf("invalid IPv6 address format")
			}
			if len(host) > endBracket+1 && host[endBracket+1] == ':' {
				port = host[endBracket+2:]
				host = host[1:endBracket]
			} else {
				host = host[1:endBracket]
			}
		} else {
			parts := strings.Split(host, ":")
			if len(parts) == 2 {
				host = parts[0]
				port = parts[1]
			}
		}
	}

	if host == "" {
		return "", "", "", fmt.Errorf("hostname cannot be empty")
	}

	return user, host, port, nil
}

// parseSCPArg parses SCP argument (either local path or host:path)
func parseSCPArg(arg string) (host, path string, isRemote bool) {
	// Check if it contains : (remote path)
	// But not C:\ on Windows
	if idx := strings.Index(arg, ":"); idx > 0 && idx < len(arg)-1 {
		// Make sure it's not a Windows drive letter
		if idx == 1 && len(arg) > 2 {
			// Could be C:\path on Windows, treat as local
			return "", arg, false
		}
		host = arg[:idx]
		path = arg[idx+1:]
		return host, path, true
	}
	return "", arg, false
}

// initTailscale initializes tsnet and returns server and context
func initTailscale(tsnetDir, controlURL string, verbose bool, logger *log.Logger) (*tsnet.Server, context.Context, error) {
	// Ensure directory exists
	if err := os.MkdirAll(tsnetDir, 0700); err != nil {
		return nil, nil, fmt.Errorf("failed to create tsnet directory: %w", err)
	}

	srv := &tsnet.Server{
		Dir:        tsnetDir,
		Hostname:   ClientName,
		ControlURL: controlURL,
	}

	// Configure logging
	if verbose {
		srv.Logf = logger.Printf
		srv.UserLogf = logger.Printf
	} else {
		// Silent mode - only show auth URLs
		srv.Logf = func(string, ...interface{}) {}
		srv.UserLogf = func(format string, args ...interface{}) {
			msg := fmt.Sprintf(format, args...)
			if strings.Contains(msg, "https://login.tailscale.com/") {
				fmt.Fprintf(os.Stderr, "\nTo authenticate, visit:\n%s\n\n", extractURL(msg))
			}
		}
	}

	ctx := context.Background()

	if !verbose {
		fmt.Fprintf(os.Stderr, "Connecting to Tailscale...\n")
	}

	status, err := srv.Up(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to bring up Tailscale: %w", err)
	}

	// Show auth URL if needed
	if status != nil && status.AuthURL != "" {
		fmt.Fprintf(os.Stderr, "\nTo authenticate, visit:\n%s\n\n", status.AuthURL)
	}

	return srv, ctx, nil
}

// connectSSH establishes SSH connection
func connectSSH(srv *tsnet.Server, ctx context.Context, user, host, port, keyPath string, insecure, verbose bool, logger *log.Logger) (*ssh.Client, error) {
	currentUser, err := osuser.Current()
	if err != nil {
		currentUser = &osuser.User{Username: user}
	}

	config := sshclient.SSHConnectionConfig{
		User:            user,
		KeyPath:         keyPath,
		TargetHost:      host,
		TargetPort:      port,
		InsecureHostKey: insecure,
		Verbose:         verbose,
		CurrentUser:     currentUser,
		Logger:          logger,
	}

	return sshclient.EstablishSSHConnection(srv, ctx, config)
}

// execRemoteCommand executes a remote command
func execRemoteCommand(client *ssh.Client, cmd []string, logger *log.Logger) error {
	logger.Printf("Executing remote command: %v\n", cmd)

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	cmdStr := strings.Join(cmd, " ")
	if err := session.Run(cmdStr); err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			os.Exit(exitErr.ExitStatus())
		}
		return fmt.Errorf("remote command failed: %w", err)
	}

	return nil
}

// interactiveSession starts an interactive SSH session
func interactiveSession(client *ssh.Client, disablePTY bool, logger *log.Logger) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Setup I/O
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to setup stdin: %w", err)
	}
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Setup PTY if we're in a terminal and PTY is not disabled
	fd := int(os.Stdin.Fd())
	if !disablePTY && term.IsTerminal(fd) {
		// Get terminal size
		width, height, err := term.GetSize(fd)
		if err != nil {
			width, height = 80, 24
		}

		termType := os.Getenv("TERM")
		if termType == "" {
			termType = "xterm-256color"
		}

		if err := session.RequestPty(termType, height, width, ssh.TerminalModes{}); err != nil {
			return fmt.Errorf("failed to request PTY: %w", err)
		}

		// Put terminal in raw mode
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			logger.Printf("Warning: failed to set raw mode: %v\n", err)
		} else {
			defer term.Restore(fd, oldState)
		}
	}

	// Start shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Copy stdin to session
	go func() {
		io.Copy(stdinPipe, os.Stdin)
		stdinPipe.Close()
	}()

	// Wait for session to finish
	return session.Wait()
}

// Helper functions for defaults
func currentUsername() string {
	if u, err := osuser.Current(); err == nil {
		return u.Username
	}
	return "root"
}

func defaultKeyPath() string {
	if u, err := osuser.Current(); err == nil {
		return filepath.Join(u.HomeDir, ".ssh", "id_rsa")
	}
	return "~/.ssh/id_rsa"
}

func defaultTsnetDir() string {
	if u, err := osuser.Current(); err == nil {
		return filepath.Join(u.HomeDir, ".config", ClientName)
	}
	return "~/.config/" + ClientName
}

func extractURL(msg string) string {
	if idx := strings.Index(msg, "https://"); idx != -1 {
		url := msg[idx:]
		if endIdx := strings.IndexAny(url, " \n\r\t"); endIdx != -1 {
			url = url[:endIdx]
		}
		return url
	}
	return msg
}

// setupDynamicForward sets up SOCKS5 dynamic port forwarding
func setupDynamicForward(client *ssh.Client, forwardSpec string, verbose bool, logger *log.Logger) error {
	// Parse bind address and port from forwardSpec
	// Format can be: "port" or "bind_address:port"
	bindAddr := "localhost"
	port := forwardSpec

	if strings.Contains(forwardSpec, ":") {
		parts := strings.Split(forwardSpec, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid dynamic forward specification: %s", forwardSpec)
		}
		bindAddr = parts[0]
		port = parts[1]
	}

	// Validate port
	if err := security.ValidatePort(port); err != nil {
		return fmt.Errorf("invalid port for dynamic forwarding: %w", err)
	}

	// Validate bind address for security
	// Allow localhost, 127.0.0.1, ::1, and empty (defaults to all interfaces)
	// Warn on binding to non-localhost addresses as they expose the proxy to network
	if bindAddr != "" && bindAddr != "localhost" && bindAddr != "127.0.0.1" && bindAddr != "::1" {
		// Parse to check if it's a valid IP
		ip := net.ParseIP(bindAddr)
		if ip == nil && bindAddr != "0.0.0.0" && bindAddr != "::" {
			return fmt.Errorf("invalid bind address: %s", bindAddr)
		}
		if verbose {
			logger.Printf("Warning: Binding SOCKS5 proxy to %s exposes it to the network\n", bindAddr)
		}
	}

	listenAddr := net.JoinHostPort(bindAddr, port)

	// Start listening on local port
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
	}

	if verbose {
		logger.Printf("SOCKS5 dynamic forwarding listening on %s\n", listenAddr)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Handle incoming SOCKS5 connections in background
	go func() {
		defer listener.Close()
		defer cancel()
		for {
			localConn, err := listener.Accept()
			if err != nil {
				// Check if this is a normal shutdown or an error
				select {
				case <-ctx.Done():
					// Context cancelled, normal shutdown
					return
				default:
					// Check if listener was closed
					if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
						return
					}
					if verbose {
						logger.Printf("Error accepting SOCKS5 connection: %v\n", err)
					}
					return
				}
			}
			go handleSOCKS5(client, localConn, verbose, logger)
		}
	}()

	return nil
}

// handleSOCKS5 handles a SOCKS5 connection
func handleSOCKS5(client *ssh.Client, localConn net.Conn, verbose bool, logger *log.Logger) {
	defer localConn.Close()

	// SOCKS5 handshake
	buf := make([]byte, 256)

	// Read version and methods
	n, err := localConn.Read(buf)
	if err != nil || n < 2 {
		if verbose {
			logger.Printf("SOCKS5 handshake failed: %v\n", err)
		}
		return
	}

	// Check SOCKS version
	if buf[0] != 0x05 {
		if verbose {
			logger.Printf("Not SOCKS5 protocol: version=%d\n", buf[0])
		}
		return
	}

	// Send "no authentication required" response
	if _, err := localConn.Write([]byte{0x05, 0x00}); err != nil {
		if verbose {
			logger.Printf("Failed to send auth response: %v\n", err)
		}
		return
	}

	// Read connection request
	n, err = localConn.Read(buf)
	if err != nil || n < 7 {
		if verbose {
			logger.Printf("Failed to read connection request: %v\n", err)
		}
		return
	}

	// Check version and command
	if buf[0] != 0x05 || buf[1] != 0x01 {
		if verbose {
			logger.Printf("Invalid SOCKS5 request: version=%d, cmd=%d\n", buf[0], buf[1])
		}
		// Send failure response
		localConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}

	// Parse address
	addrType := buf[3]
	var host string
	var port uint16

	switch addrType {
	case 0x01: // IPv4
		if n < 10 {
			if verbose {
				logger.Printf("Invalid IPv4 address length\n")
			}
			localConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
			return
		}
		host = fmt.Sprintf("%d.%d.%d.%d", buf[4], buf[5], buf[6], buf[7])
		port = uint16(buf[8])<<8 | uint16(buf[9])
	case 0x03: // Domain name
		addrLen := int(buf[4])
		if n < 5+addrLen+2 {
			if verbose {
				logger.Printf("Invalid domain name length\n")
			}
			localConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
			return
		}
		host = string(buf[5 : 5+addrLen])
		port = uint16(buf[5+addrLen])<<8 | uint16(buf[5+addrLen+1])
	case 0x04: // IPv6
		if n < 22 {
			if verbose {
				logger.Printf("Invalid IPv6 address length\n")
			}
			localConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
			return
		}
		host = net.IP(buf[4:20]).String()
		port = uint16(buf[20])<<8 | uint16(buf[21])
	default:
		if verbose {
			logger.Printf("Unsupported address type: %d\n", addrType)
		}
		localConn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}

	targetAddr := fmt.Sprintf("%s:%d", host, port)
	if verbose {
		logger.Printf("SOCKS5 forwarding to: %s\n", targetAddr)
	}

	// Dial through SSH
	remoteConn, err := client.Dial("tcp", targetAddr)
	if err != nil {
		if verbose {
			logger.Printf("Failed to dial %s: %v\n", targetAddr, err)
		}
		// Send connection refused
		localConn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}
	defer remoteConn.Close()

	// Send success response
	if _, err := localConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
		if verbose {
			logger.Printf("Failed to send success response: %v\n", err)
		}
		return
	}

	// Bidirectional copy
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()

	<-done
}
