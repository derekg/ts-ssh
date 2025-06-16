package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"

	"github.com/rivo/tview"
)

// scpArgs holds parsed arguments for an SCP operation.
type scpArgs struct {
	isUpload   bool
	localPath  string
	remotePath string
	targetHost string
	sshUser    string // User from user@host:path, if present
}

// tuiActionResult defines what action the user selected in the TUI.
type tuiActionResult struct {
	action             string // "ssh", "scp", or "" if exited/cancelled
	selectedHostTarget string // Hostname or IP for the connection
	// SCP specific fields
	scpLocalPath  string
	scpRemotePath string
	scpIsUpload   bool // true for upload (local to remote), false for download (remote to local)
}

// version is set at build time via -ldflags "-X main.version=..."; default is "dev".
var version = "dev"

// DefaultSshPort is the default SSH port.
const DefaultSshPort = "22" // Made public
const clientName = "ts-ssh-client" // How this client appears in Tailscale admin console

// parseScpRemoteArg parses an SCP remote argument string (e.g., "user@host:path" or "host:path")
// It returns the host, path, and user. If user is not in the string, it returns the default SSH user.
func parseScpRemoteArg(remoteArg string, defaultSshUser string) (host, path, user string, err error) {
	user = defaultSshUser // Start with the default/flag-provided user

	parts := strings.SplitN(remoteArg, ":", 2)
	if len(parts) != 2 || parts[1] == "" { // Ensure path part exists
		return "", "", "", fmt.Errorf("invalid remote SCP argument format: %q. Must be [user@]host:path", remoteArg)
	}
	path = parts[1]
	hostPart := parts[0]

	if strings.Contains(hostPart, "@") {
		userHostParts := strings.SplitN(hostPart, "@", 2)
		if len(userHostParts) != 2 || userHostParts[0] == "" || userHostParts[1] == "" {
			return "", "", "", fmt.Errorf("invalid user@host format in SCP argument: %q", hostPart)
		}
		user = userHostParts[0]
		host = userHostParts[1]
	} else {
		host = hostPart
	}

	if host == "" {
		return "", "", "", fmt.Errorf("host cannot be empty in SCP argument: %q", remoteArg)
	}
	return host, path, user, nil
}

func main() {
	// --- Command Line Flags ---
	var (
		sshUser         string
		sshKeyPath      string
		tsnetDir        string
		tsControlURL    string
		target          string
		verbose         bool
		insecureHostKey bool
		forwardDest     string
		showVersion     bool
		tuiMode         bool
	)

	currentUser, err := user.Current()
	defaultUser := "user" 
	if err == nil {
		defaultUser = currentUser.Username
	}
	defaultKeyPath := ""
	if currentUser != nil {
		defaultKeyPath = filepath.Join(currentUser.HomeDir, ".ssh", "id_rsa")
	}
	defaultTsnetDir := ""
	if currentUser != nil {
		defaultTsnetDir = filepath.Join(currentUser.HomeDir, ".config", clientName)
	}

	flag.StringVar(&sshUser, "l", defaultUser, "SSH Username")
	flag.StringVar(&sshKeyPath, "i", defaultKeyPath, "Path to SSH private key")
	flag.StringVar(&tsnetDir, "tsnet-dir", defaultTsnetDir, "Directory to store tsnet state")
	flag.StringVar(&tsControlURL, "control-url", "", "Tailscale control plane URL (optional)")
	flag.BoolVar(&verbose, "v", false, "Verbose logging")
	flag.BoolVar(&insecureHostKey, "insecure", false, "Disable host key checking (INSECURE!)")
	flag.StringVar(&forwardDest, "W", "", "forward stdio to destination host:port (for use as ProxyCommand)")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.BoolVar(&tuiMode, "tui", false, "Enable interactive TUI mode")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [user@]hostname[:port] [command...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s [options] local_path user@hostname:remote_path\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s [options] user@hostname:remote_path local_path\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Connects to a host on your Tailscale network via SSH or SCP using tsnet.\n\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s user@host           # interactive SSH session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s user@host ls -lah    # run a remote command non-interactively\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s local.txt user@host:/remote/ # SCP upload\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s user@host:/remote/file.txt ./ # SCP download\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -W host:port        # proxy stdio to host:port via Tailscale (for ProxyCommand)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  scp -o ProxyCommand=\"%s -W %%h:%%p user@gateway\" localfile remote:/path\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -version           # print version and exit\n", os.Args[0])
	}
	flag.Parse()

	// If TUI mode is detected, redirect stderr at file descriptor level before any goroutines start
	var originalStderrFd int
	var stderrDevNull *os.File
	if tuiMode {
		// Save original stderr file descriptor
		originalStderrFd, _ = syscall.Dup(int(os.Stderr.Fd()))
		
		// Open /dev/null and redirect stderr file descriptor to it
		var err error
		stderrDevNull, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err == nil {
			syscall.Dup2(int(stderrDevNull.Fd()), int(os.Stderr.Fd()))
		}
	}

	var logger *log.Logger
	if verbose {
		logger = log.Default()
	} else {
		logger = log.New(io.Discard, "", 0)
	}

	if showVersion {
		fmt.Fprintf(os.Stdout, "%s\n", version)
		os.Exit(0)
	}

	if tuiMode {
		app := tview.NewApplication()
		// For TUI mode, always use a discarded logger to prevent output interference
		tuiLogger := log.New(io.Discard, "", 0)
		
		srv, ctx, initialStatus, errTUI := initTsNet(tsnetDir, clientName, tuiLogger, tsControlURL, verbose, true)
		if errTUI != nil {
			os.Exit(1) 
		}
		defer srv.Close()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		appCtx, cancelApp := context.WithCancel(ctx)
		defer cancelApp()

		go func() {
			select {
			case <-sigCh:
				logger.Println("Signal received, initiating TUI shutdown...")
				cancelApp()
			case <-ctx.Done(): 
				logger.Println("Main context cancelled, initiating TUI shutdown...")
				cancelApp()
			}
		}()

		tuiResult, errTUI := startTUI(app, srv, appCtx, tuiLogger, initialStatus, sshUser, sshKeyPath, insecureHostKey, verbose)
		
		// Restore stderr file descriptor after TUI operations complete
		if stderrDevNull != nil {
			stderrDevNull.Close()
			syscall.Dup2(originalStderrFd, int(os.Stderr.Fd()))
			syscall.Close(originalStderrFd)
		}
		if errTUI != nil {
			if verbose {
				logger.Fatalf("TUI error: %v", errTUI)
			} else {
				os.Exit(1)
			}
		}
		if verbose {
			logger.Println("TUI finished.")
		}

		switch tuiResult.action {
		case "ssh":
			if tuiResult.selectedHostTarget != "" {
				if verbose {
					logger.Printf("Host selected for SSH: %s. Proceeding with connection...\n", tuiResult.selectedHostTarget)
				}
				errTUI = connectToHostFromTUI(srv, appCtx, logger, tuiResult.selectedHostTarget, sshUser, sshKeyPath, insecureHostKey, currentUser, verbose)
				if errTUI != nil && verbose {
					logger.Printf("SSH connection from TUI failed or was cancelled: %v", errTUI)
				}
			} else if verbose {
				logger.Println("No host selected for SSH or action cancelled.")
			}
		case "scp":
			if tuiResult.selectedHostTarget != "" {
				if verbose {
					logger.Printf("Host selected for SCP: %s. Local: %s, Remote: %s, Upload: %t\n",
						tuiResult.selectedHostTarget, tuiResult.scpLocalPath, tuiResult.scpRemotePath, tuiResult.scpIsUpload)
				}
				
				effectiveScpUser := sshUser // User from -l flag is default
				if tuiResult.selectedHostTarget != "" { // If a host was selected
				    // Attempt to parse user from host string, if any
				    // This logic assumes selectedHostTarget might contain user@ like "user@actualhost"
				    // For TUI, selectedHostTarget is typically just the hostname/IP.
				    // If user can be part of selectedHostTarget from TUI, parse it.
				    // Otherwise, effectiveScpUser remains the one from -l flag.
				    // This example assumes selectedHostTarget is just host, not user@host for TUI.
				}

				errTUI = performSCPTransfer(srv, appCtx, logger, tuiResult, effectiveScpUser, sshKeyPath, insecureHostKey, currentUser, verbose)
				if errTUI != nil && verbose {
					logger.Printf("SCP transfer from TUI failed: %v", errTUI)
				} else if verbose {
					logger.Println("SCP transfer from TUI completed successfully.")
				}
			} else if verbose {
				logger.Println("SCP action cancelled or host not selected.")
			}
		default:
			if verbose {
				logger.Println("No action selected from TUI or TUI exited.")
			}
		}
		os.Exit(0)
	}

	// --- SCP Argument Parsing (CLI) ---
	var detectedScpArgs *scpArgs
	nonFlagArgs := flag.Args()
	
	// Use sshUser from -l flag as the default for SCP user.
	// It will be overridden if user@host is specified in the remote path.
	defaultScpUser := sshUser 

	if !tuiMode && len(nonFlagArgs) == 2 {
		arg1 := nonFlagArgs[0]
		arg2 := nonFlagArgs[1]

		arg1ContainsColon := strings.Contains(arg1, ":")
		arg2ContainsColon := strings.Contains(arg2, ":")

		if arg1ContainsColon && !arg2ContainsColon { // Potential download: user@host:remote local
			parsedHost, parsedRemotePath, parsedUser, errParse := parseScpRemoteArg(arg1, defaultScpUser)
			if errParse == nil {
				detectedScpArgs = &scpArgs{
					isUpload:   false,
					localPath:  arg2,
					remotePath: parsedRemotePath,
					targetHost: parsedHost,
					sshUser:    parsedUser, 
				}
				if verbose { logger.Printf("SCP download detected: remote %s@%s:%s to local %s", detectedScpArgs.sshUser, detectedScpArgs.targetHost, detectedScpArgs.remotePath, detectedScpArgs.localPath) }
			} else {
				if verbose { logger.Printf("Could not parse arg1 '%s' as SCP remote for download: %v", arg1, errParse) }
			}
		} else if !arg1ContainsColon && arg2ContainsColon { // Potential upload: local user@host:remote
			parsedHost, parsedRemotePath, parsedUser, errParse := parseScpRemoteArg(arg2, defaultScpUser)
			if errParse == nil {
				detectedScpArgs = &scpArgs{
					isUpload:   true,
					localPath:  arg1,
					remotePath: parsedRemotePath,
					targetHost: parsedHost,
					sshUser:    parsedUser,
				}
				if verbose { logger.Printf("SCP upload detected: local %s to remote %s@%s:%s", detectedScpArgs.localPath, detectedScpArgs.sshUser, detectedScpArgs.targetHost, detectedScpArgs.remotePath) }
			} else {
				if verbose { logger.Printf("Could not parse arg2 '%s' as SCP remote for upload: %v", arg2, errParse) }
			}
		}
	}
	// --- End SCP Argument Parsing ---

	if detectedScpArgs != nil {
		// SCP mode is active.
		srv, ctx, _, err := initTsNet(tsnetDir, clientName, logger, tsControlURL, verbose, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize Tailscale connection for SCP: %v\n", err)
			os.Exit(1)
		}
		defer srv.Close()
		
		err = HandleCliScp(srv, ctx, logger, detectedScpArgs.sshUser, sshKeyPath, insecureHostKey, currentUser,
			detectedScpArgs.localPath, detectedScpArgs.remotePath, detectedScpArgs.targetHost,
			detectedScpArgs.isUpload, verbose)

		if err != nil {
			fmt.Fprintf(os.Stderr, "SCP operation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "SCP operation completed successfully.")
		os.Exit(0)
	}

	// Fallthrough to SSH logic if not TUI and not SCP
	if !tuiMode { 
		if flag.NArg() < 1 { 
			flag.Usage()
			os.Exit(1)
		}
		target = flag.Arg(0)
		var remoteCmd []string
		if flag.NArg() > 1 {
			remoteCmd = flag.Args()[1:]
		}

		targetHost, targetPort, err := parseTarget(target, DefaultSshPort)
		if err != nil {
			logger.Fatalf("Error parsing target for SSH: %v", err)
		}
		
		sshSpecificUser := sshUser 
		if strings.Contains(targetHost, "@") { 
			parts := strings.SplitN(targetHost, "@", 2)
			sshSpecificUser = parts[0]
			targetHost = parts[1] 
			if verbose { logger.Printf("SSH target user overridden to '%s' from target string.", sshSpecificUser) }
		}
		
		if verbose {
			logger.Printf("Starting %s (SSH mode)...", clientName)
		}

		srv, ctx, _, err := initTsNet(tsnetDir, clientName, logger, tsControlURL, verbose, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize Tailscale connection for SSH: %v\n", err)
			os.Exit(1)
		}
		defer srv.Close()

		nonTuiCtx, nonTuiCancel := context.WithCancel(ctx)
		defer nonTuiCancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			select {
			case <-sigCh:
				logger.Println("Signal received, shutting down non-TUI operation...")
				nonTuiCancel()
				fd := int(os.Stdin.Fd())
				if term.IsTerminal(fd) {
					_ = term.Restore(fd, nil)
				}
				os.Exit(1)
			case <-ctx.Done():
				logger.Println("Main tsnet context cancelled, shutting down non-TUI operation...")
				nonTuiCancel()
			}
		}()

		if forwardDest != "" {
			logger.Printf("Forwarding stdio to %s via tsnet...", forwardDest)
			fwdConn, errDial := srv.Dial(nonTuiCtx, "tcp", forwardDest)
			if errDial != nil {
				log.Fatalf("Failed to dial %s via tsnet for forwarding: %v", forwardDest, errDial)
			}
			go func() {
				_, _ = io.Copy(fwdConn, os.Stdin)
				fwdConn.Close()
			}()
			_, _ = io.Copy(os.Stdout, fwdConn)
			os.Exit(0)
		}
		logger.Printf("tsnet potentially initialized. Attempting SSH connection to %s@%s:%s", sshSpecificUser, targetHost, targetPort)

		authMethods := []ssh.AuthMethod{}
		keyAuth, err := LoadPrivateKey(sshKeyPath, logger)
		if err == nil {
			authMethods = append(authMethods, keyAuth)
			logger.Printf("Using public key authentication: %s", sshKeyPath)
		} else {
			logger.Printf("Failed to load private key: %v. Will attempt password auth.", err)
		}
		authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
			fmt.Printf("Enter password for %s@%s: ", sshSpecificUser, targetHost)
			bytePassword, errRead := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if errRead != nil {
				return "", fmt.Errorf("failed to read password: %w", errRead)
			}
			return string(bytePassword), nil
		}))

		var hostKeyCallback ssh.HostKeyCallback
		if insecureHostKey {
			logger.Println("WARNING: Host key verification is disabled!")
			hostKeyCallback = ssh.InsecureIgnoreHostKey()
		} else {
			hostKeyCallback, err = CreateKnownHostsCallback(currentUser, logger)
			if err != nil {
				log.Fatalf("Could not set up host key verification: %v", err)
			}
		}

		sshConfig := &ssh.ClientConfig{
			User:            sshSpecificUser,
			Auth:            authMethods,
			HostKeyCallback: hostKeyCallback,
			Timeout:         15 * time.Second,
		}

		sshTargetAddr := net.JoinHostPort(targetHost, targetPort)
		logger.Printf("Dialing %s via tsnet...", sshTargetAddr)
		conn, err_dial := srv.Dial(nonTuiCtx, "tcp", sshTargetAddr) // Renamed err to err_dial to avoid conflict
		if err_dial != nil {
			log.Fatalf("Failed to dial %s via tsnet (is Tailscale connection up and host reachable?): %v", sshTargetAddr, err_dial)
		}
		logger.Printf("tsnet Dial successful. Establishing SSH connection...")

		sshConn, chans, reqs, err_conn := ssh.NewClientConn(conn, sshTargetAddr, sshConfig) // Renamed err to err_conn
		if err_conn != nil {
			if strings.Contains(err_conn.Error(), "unable to authenticate") || strings.Contains(err_conn.Error(), "no supported authentication methods") {
				log.Fatalf("SSH Authentication failed for user %s: %v", sshSpecificUser, err_conn)
			}
			var keyErr *knownhosts.KeyError
			if errors.As(err_conn, &keyErr) {
				log.Fatalf("SSH Host key verification failed: %v", err_conn)
			}
			log.Fatalf("Failed to establish SSH connection to %s: %v", sshTargetAddr, err_conn)
		}
		defer sshConn.Close()
		logger.Println("SSH connection established.")

		client := ssh.NewClient(sshConn, chans, reqs)
		defer client.Close()

		if len(remoteCmd) > 0 {
			logger.Printf("Running remote command: %v", remoteCmd)
			session, errSession := client.NewSession()
			if errSession != nil {
				log.Fatalf("Failed to create SSH session for remote command: %v", errSession)
			}
			defer session.Close()
			session.Stdout = os.Stdout
			session.Stderr = os.Stderr
			session.Stdin = os.Stdin
			cmd := strings.Join(remoteCmd, " ")
			if errRun := session.Run(cmd); errRun != nil {
				if exitErr, ok := errRun.(*ssh.ExitError); ok {
					os.Exit(exitErr.ExitStatus())
				}
				log.Fatalf("Remote command execution failed: %v", errRun)
			}
			os.Exit(0)
		}

		logger.Println("Starting interactive SSH session...")
		session, err := client.NewSession()
		if err != nil {
			log.Fatalf("Failed to create SSH session: %v", err)
		}
		defer session.Close()

		fd := int(os.Stdin.Fd())
		var oldState *term.State
		if term.IsTerminal(fd) {
			oldState, err = term.MakeRaw(fd)
			if err != nil {
				log.Printf("Warning: Failed to set terminal to raw mode: %v. Session might not work correctly.", err)
			} else {
				defer term.Restore(fd, oldState)
			}
		} else {
			logger.Println("Input is not a terminal, proceeding without raw mode or PTY request.")
		}

		stdinPipe, err := session.StdinPipe()
		if err != nil {
			log.Fatalf("Failed to create stdin pipe for SSH session: %v", err)
		}
		session.Stdout = os.Stdout
		session.Stderr = os.Stderr

		if term.IsTerminal(fd) {
			termWidth, termHeight, errSize := term.GetSize(fd)
			if errSize != nil {
				logger.Printf("Warning: Failed to get terminal size: %v. Using default 80x24.", errSize)
				termWidth = 80
				termHeight = 24
			}
			termType := os.Getenv("TERM")
			if termType == "" {
				termType = "xterm-256color"
			}
			errPty := session.RequestPty(termType, termHeight, termWidth, ssh.TerminalModes{})
			if errPty != nil {
				log.Fatalf("Failed to request pseudo-terminal: %v", errPty)
			}
			go watchWindowSize(fd, session, nonTuiCtx, logger)
		}

		err = session.Shell()
		if err != nil {
			log.Fatalf("Failed to start remote shell: %v", err)
		}
		fmt.Fprintf(os.Stderr, "\nEscape sequence: ~. to terminate session\n")
		go func() {
			reader := bufio.NewReader(os.Stdin)
			atLineStart := true
			for {
				b, errReadByte := reader.ReadByte()
				if errReadByte != nil {
					return
				}
				if atLineStart && b == '~' {
					next, errPeek := reader.Peek(1)
					if errPeek == nil {
						if next[0] == '.' {
							reader.ReadByte()
							if oldState != nil && term.IsTerminal(fd) {
								term.Restore(fd, oldState)
							}
							os.Exit(0)
						} else if next[0] == '~' {
							reader.ReadByte()
							stdinPipe.Write([]byte{'~'})
							atLineStart = false
							continue
						}
					}
				}
				stdinPipe.Write([]byte{b})
				atLineStart = (b == '\n' || b == '\r')
			}
		}()

		err = session.Wait()
		if oldState != nil && term.IsTerminal(fd) {
			term.Restore(fd, oldState)
		}

		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				if verbose {
					logger.Printf("Remote command exited with status %d", exitErr.ExitStatus())
				}
				os.Exit(exitErr.ExitStatus())
			}
			if !errors.Is(err, io.EOF) && !strings.Contains(err.Error(), "session closed") && !strings.Contains(err.Error(), "channel closed") {
				log.Printf("SSH session ended with error: %v", err)
			}
		}
		logger.Println("SSH session closed.")
	}
}

// Moved to tsnet_handler.go:
// func initTsNet(tsnetDir, clientHostname string, logger *log.Logger, tsControlURL string, verbose, tuiMode bool) (*tsnet.Server, context.Context, *ipnstate.Status, error)
