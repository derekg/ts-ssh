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
const DefaultSshPort = "22" 
const clientName = "ts-ssh-client" 

// parseScpRemoteArg parses an SCP remote argument string (e.g., "user@host:path" or "host:path")
// It returns the host, path, and user. If user is not in the string, it returns the default SSH user.
func parseScpRemoteArg(remoteArg string, defaultSshUser string) (host, path, user string, err error) {
	user = defaultSshUser 

	parts := strings.SplitN(remoteArg, ":", 2)
	if len(parts) != 2 || parts[1] == "" { 
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

// tryParseScpArgs attempts to parse command line arguments for SCP operations.
// It returns scpArgs if SCP operation is detected, isScpOp true, and error nil.
// If args don't match SCP syntax, it returns nil, false, nil.
// If args match SCP syntax but are invalid (e.g. bad remote format), it returns nil, false, and an error.
func tryParseScpArgs(nonFlagArgs []string, defaultSshUser string, logger *log.Logger, verbose bool) (args *scpArgs, isScpOp bool, err error) {
	if len(nonFlagArgs) != 2 {
		return nil, false, nil // Not an SCP operation based on arg count
	}

	arg1 := nonFlagArgs[0]
	arg2 := nonFlagArgs[1]
	arg1ContainsColon := strings.Contains(arg1, ":")
	arg2ContainsColon := strings.Contains(arg2, ":")

	if arg1ContainsColon && !arg2ContainsColon { // Potential download: user@host:remote local
		parsedHost, parsedRemotePath, parsedUser, errParse := parseScpRemoteArg(arg1, defaultSshUser)
		if errParse == nil {
			details := &scpArgs{
				isUpload:   false,
				localPath:  arg2,
				remotePath: parsedRemotePath,
				targetHost: parsedHost,
				sshUser:    parsedUser,
			}
			if verbose && logger != nil { 
				logger.Printf("SCP download detected: remote %s@%s:%s to local %s", details.sshUser, details.targetHost, details.remotePath, details.localPath)
			}
			return details, true, nil
		}
		if verbose && logger != nil {
			logger.Printf("Could not parse arg1 '%s' as SCP remote for download: %v", arg1, errParse)
		}
		return nil, false, fmt.Errorf("failed to parse remote argument %s: %w", arg1, errParse)
	} else if !arg1ContainsColon && arg2ContainsColon { // Potential upload: local user@host:remote
		parsedHost, parsedRemotePath, parsedUser, errParse := parseScpRemoteArg(arg2, defaultSshUser)
		if errParse == nil {
			details := &scpArgs{
				isUpload:   true,
				localPath:  arg1,
				remotePath: parsedRemotePath,
				targetHost: parsedHost,
				sshUser:    parsedUser,
			}
			if verbose && logger != nil {
				logger.Printf("SCP upload detected: local %s to remote %s@%s:%s", details.localPath, details.sshUser, details.targetHost, details.remotePath)
			}
			return details, true, nil
		}
		if verbose && logger != nil {
			logger.Printf("Could not parse arg2 '%s' as SCP remote for upload: %v", arg2, errParse)
		}
		return nil, false, fmt.Errorf("failed to parse remote argument %s: %w", arg2, errParse)
	}
	return nil, false, nil // Doesn't match SCP pattern
}


func main() {
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

	currentUserBody, err_user := user.Current() // renamed err to avoid conflict
	defaultUser := "user" 
	if err_user == nil {
		defaultUser = currentUserBody.Username
	}
	defaultKeyPath := ""
	if currentUserBody != nil {
		defaultKeyPath = filepath.Join(currentUserBody.HomeDir, ".ssh", "id_rsa")
	}
	defaultTsnetDir := ""
	if currentUserBody != nil {
		defaultTsnetDir = filepath.Join(currentUserBody.HomeDir, ".config", clientName)
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
		srv, ctx, initialStatus, errTUI := initTsNet(tsnetDir, clientName, logger, tsControlURL, verbose)
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

		tuiResult, errTUI := startTUI(app, srv, appCtx, logger, initialStatus, sshUser, sshKeyPath, insecureHostKey, verbose)
		if errTUI != nil {
			logger.Fatalf("TUI error: %v", errTUI)
		}
		logger.Println("TUI finished.")

		switch tuiResult.action {
		case "ssh":
			if tuiResult.selectedHostTarget != "" {
				logger.Printf("Host selected for SSH: %s. Proceeding with connection...\n", tuiResult.selectedHostTarget)
				errTUI = connectToHostFromTUI(srv, appCtx, logger, tuiResult.selectedHostTarget, sshUser, sshKeyPath, insecureHostKey, currentUserBody, verbose)
				if errTUI != nil {
					fmt.Fprintf(os.Stderr, "SSH connection from TUI failed or was cancelled: %v\n", errTUI)
					logger.Printf("SSH connection from TUI failed or was cancelled: %v", errTUI)
				}
			} else {
				logger.Println("No host selected for SSH or action cancelled.")
			}
		case "scp":
			if tuiResult.selectedHostTarget != "" {
				logger.Printf("Host selected for SCP: %s. Local: %s, Remote: %s, Upload: %t\n",
					tuiResult.selectedHostTarget, tuiResult.scpLocalPath, tuiResult.scpRemotePath, tuiResult.scpIsUpload)
				
				effectiveScpUser := sshUser 
				
				errTUI = performSCPTransfer(srv, appCtx, logger, tuiResult, effectiveScpUser, sshKeyPath, insecureHostKey, currentUserBody, verbose)
				if errTUI != nil {
					fmt.Fprintf(os.Stderr, "SCP transfer from TUI failed: %v\n", errTUI)
					logger.Printf("SCP transfer from TUI failed: %v", errTUI)
				} else {
					fmt.Fprintln(os.Stderr, "SCP transfer from TUI completed successfully.")
					logger.Println("SCP transfer from TUI completed successfully.")
				}
			} else {
				logger.Println("SCP action cancelled or host not selected.")
			}
		default:
			logger.Println("No action selected from TUI or TUI exited.")
		}
		os.Exit(0)
	}

	nonFlagArgs := flag.Args()
	detectedScpArgs, isScpOp, errScpParse := tryParseScpArgs(nonFlagArgs, sshUser, logger, verbose)
	if errScpParse != nil {
		fmt.Fprintf(os.Stderr, "Error parsing SCP arguments: %v\n", errScpParse)
		flag.Usage()
		os.Exit(1)
	}

	if isScpOp {
		srv, ctx, _, err_tsnet := initTsNet(tsnetDir, clientName, logger, tsControlURL, verbose) 
		if err_tsnet != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize Tailscale connection for SCP: %v\n", err_tsnet)
			os.Exit(1)
		}
		defer srv.Close()
		
		err_scp := HandleCliScp(srv, ctx, logger, detectedScpArgs.sshUser, sshKeyPath, insecureHostKey, currentUserBody,
			detectedScpArgs.localPath, detectedScpArgs.remotePath, detectedScpArgs.targetHost,
			detectedScpArgs.isUpload, verbose) 

		if err_scp != nil {
			fmt.Fprintf(os.Stderr, "SCP operation failed: %v\n", err_scp)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "SCP operation completed successfully.")
		os.Exit(0)
	}

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

		targetHost, targetPort, err_parse := parseTarget(target, DefaultSshPort) 
		if err_parse != nil {
			logger.Fatalf("Error parsing target for SSH: %v", err_parse)
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

		srv, ctx, _, err_tsnet_ssh := initTsNet(tsnetDir, clientName, logger, tsControlURL, verbose) 
		if err_tsnet_ssh != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize Tailscale connection for SSH: %v\n", err_tsnet_ssh)
			os.Exit(1)
		}
		defer srv.Close()

		nonTuiCtx, nonTuiCancel := context.WithCancel(ctx)
		defer nonTuiCancel()

		sigCh_ssh := make(chan os.Signal, 1) 
		signal.Notify(sigCh_ssh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			select {
			case <-sigCh_ssh:
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
		keyAuth, err_key := LoadPrivateKey(sshKeyPath, logger) 
		if err_key == nil {
			authMethods = append(authMethods, keyAuth)
			logger.Printf("Using public key authentication: %s", sshKeyPath)
		} else {
			logger.Printf("Failed to load private key: %v. Will attempt password auth.", err_key)
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
			var err_hkc error 
			hostKeyCallback, err_hkc = CreateKnownHostsCallback(currentUserBody, logger) 
			if err_hkc != nil {
				log.Fatalf("Could not set up host key verification: %v", err_hkc)
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
		conn, err_dial_ssh := srv.Dial(nonTuiCtx, "tcp", sshTargetAddr) 
		if err_dial_ssh != nil {
			log.Fatalf("Failed to dial %s via tsnet (is Tailscale connection up and host reachable?): %v", sshTargetAddr, err_dial_ssh)
		}
		logger.Printf("tsnet Dial successful. Establishing SSH connection...")

		sshConn, chans, reqs, err_conn_ssh := ssh.NewClientConn(conn, sshTargetAddr, sshConfig) 
		if err_conn_ssh != nil {
			if strings.Contains(err_conn_ssh.Error(), "unable to authenticate") || strings.Contains(err_conn_ssh.Error(), "no supported authentication methods") {
				log.Fatalf("SSH Authentication failed for user %s: %v", sshSpecificUser, err_conn_ssh)
			}
			var keyErr *knownhosts.KeyError
			if errors.As(err_conn_ssh, &keyErr) {
				log.Fatalf("SSH Host key verification failed: %v", err_conn_ssh)
			}
			log.Fatalf("Failed to establish SSH connection to %s: %v", sshTargetAddr, err_conn_ssh)
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
		session, err_session := client.NewSession() 
		if err_session != nil {
			log.Fatalf("Failed to create SSH session: %v", err_session)
		}
		defer session.Close()

		fd := int(os.Stdin.Fd())
		var oldState *term.State
		if term.IsTerminal(fd) {
			var err_raw error 
			oldState, err_raw = term.MakeRaw(fd)
			if err_raw != nil {
				log.Printf("Warning: Failed to set terminal to raw mode: %v. Session might not work correctly.", err_raw)
			} else {
				defer term.Restore(fd, oldState)
			}
		} else {
			logger.Println("Input is not a terminal, proceeding without raw mode or PTY request.")
		}

		stdinPipe, err_pipe := session.StdinPipe() 
		if err_pipe != nil {
			log.Fatalf("Failed to create stdin pipe for SSH session: %v", err_pipe)
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

		err_shell := session.Shell() 
		if err_shell != nil {
			log.Fatalf("Failed to start remote shell: %v", err_shell)
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

		err_wait := session.Wait() 
		if oldState != nil && term.IsTerminal(fd) {
			term.Restore(fd, oldState)
		}

		if err_wait != nil {
			if exitErr, ok := err_wait.(*ssh.ExitError); ok {
				if verbose {
					logger.Printf("Remote command exited with status %d", exitErr.ExitStatus())
				}
				os.Exit(exitErr.ExitStatus())
			}
			if !errors.Is(err_wait, io.EOF) && !strings.Contains(err_wait.Error(), "session closed") && !strings.Contains(err_wait.Error(), "channel closed") {
				log.Printf("SSH session ended with error: %v", err_wait)
			}
		}
		logger.Println("SSH session closed.")
	}
}
