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

	"github.com/gdamore/tcell/v2"
	"github.com/bramvdbogaerde/go-scp"
	"github.com/rivo/tview"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

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

const (
	defaultSSHPort = "22"
	clientName     = "ts-ssh-client" // How this client appears in Tailscale admin console
)

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
	defaultUser := "user" // Fallback
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
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [user@]hostname[:port] [command...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Connects to a host on your Tailscale network via SSH using tsnet.\n\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s user@host           # interactive SSH session\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s user@host ls -lah    # run a remote command non-interactively\n", os.Args[0])
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
			// If logger is discarding, this won't show, but initTsNet already logged Fatalf
			// logger.Fatalf("Failed to initialize Tailscale connection for TUI: %v", errTUI)
			os.Exit(1) // initTsNet should have logged the specific error
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
			case <-ctx.Done(): // if main context from initTsNet is cancelled
				logger.Println("Main context cancelled, initiating TUI shutdown...")
				cancelApp()
			}
		}()

		tuiResult, errTUI := startTUI(app, srv, appCtx, logger, initialStatus, sshUser, sshKeyPath, insecureHostKey, verbose)
		if errTUI != nil {
			logger.Fatalf("TUI error: %v", errTUI)
		}
		// tview app has stopped at this point
		logger.Println("TUI finished.")


		switch tuiResult.action {
		case "ssh":
			if tuiResult.selectedHostTarget != "" {
				logger.Printf("Host selected for SSH: %s. Proceeding with connection...\n", tuiResult.selectedHostTarget)
				// Ensure terminal is properly restored before SSH takes over
				// This happens because app.Stop() is called before this.
				errTUI = connectToHostFromTUI(srv, appCtx, logger, tuiResult.selectedHostTarget, sshUser, sshKeyPath, insecureHostKey, currentUser, verbose)
				if errTUI != nil {
					// Print to Stderr as logger might be io.Discard if not verbose
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
				errTUI = performSCPTransfer(srv, appCtx, logger, tuiResult, sshUser, sshKeyPath, insecureHostKey, currentUser, verbose)
				if errTUI != nil {
					fmt.Fprintf(os.Stderr, "SCP transfer from TUI failed: %v\n", errTUI)
					logger.Printf("SCP transfer from TUI failed: %v", errTUI)
				} else {
					// Also print to Stderr for non-verbose success message
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

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	target = flag.Arg(0)
	var remoteCmd []string
	if flag.NArg() > 1 {
		remoteCmd = flag.Args()[1:]
	}

	targetHost, targetPort, err := parseTarget(target)
	if err != nil {
		logger.Fatalf("Error parsing target: %v", err)
	}
	if strings.Contains(targetHost, "@") {
		parts := strings.SplitN(targetHost, "@", 2)
		sshUser = parts[0]
		targetHost = parts[1]
	}

	if verbose {
		logger.Printf("Starting %s...", clientName)
	}

	srv, ctx, _, err := initTsNet(tsnetDir, clientName, logger, tsControlURL, verbose)
	if err != nil {
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
	logger.Printf("tsnet potentially initialized. Attempting SSH connection to %s@%s:%s", sshUser, targetHost, targetPort)

	authMethods := []ssh.AuthMethod{}
	keyAuth, err := loadPrivateKey(sshKeyPath)
	if err == nil {
		authMethods = append(authMethods, keyAuth)
		logger.Printf("Using public key authentication: %s", sshKeyPath)
	} else {
		logger.Printf("Could not load private key %q: %v. Will attempt password auth.", sshKeyPath, err)
	}
	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		fmt.Printf("Enter password for %s@%s: ", sshUser, targetHost)
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
		hostKeyCallback, err = createKnownHostsCallback(currentUser)
		if err != nil {
			log.Fatalf("Could not set up host key verification (check ~/.ssh/known_hosts permissions?): %v", err)
		}
		logger.Println("Using known_hosts for host key verification.")
	}

	sshConfig := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         15 * time.Second,
	}

	sshTargetAddr := net.JoinHostPort(targetHost, targetPort)
	logger.Printf("Dialing %s via tsnet...", sshTargetAddr)
	conn, err := srv.Dial(nonTuiCtx, "tcp", sshTargetAddr)
	if err != nil {
		log.Fatalf("Failed to dial %s via tsnet (is Tailscale connection up and host reachable?): %v", sshTargetAddr, err)
	}
	logger.Printf("tsnet Dial successful. Establishing SSH connection...")

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, sshTargetAddr, sshConfig)
	if err != nil {
		if strings.Contains(err.Error(), "unable to authenticate") || strings.Contains(err.Error(), "no supported authentication methods") {
			log.Fatalf("SSH Authentication failed for user %s: %v", sshUser, err)
		}
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) {
			log.Fatalf("SSH Host key verification failed: %v", err)
		}
		log.Fatalf("Failed to establish SSH connection to %s: %v", sshTargetAddr, err)
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
		go watchWindowSize(fd, session)
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

func initTsNet(tsnetDir string, clientHostname string, logger *log.Logger, tsControlURL string, verbose bool) (*tsnet.Server, context.Context, *ipnstate.Status, error) {
	if tsnetDir == "" {
		tsnetDir = clientHostname + "-state-dir"
		logger.Printf("Warning: Using default tsnet state directory: %s", tsnetDir)
	}
	if err := os.MkdirAll(tsnetDir, 0700); err != nil && !os.IsExist(err) {
		logger.Fatalf("Failed to create tsnet state directory %q: %v", tsnetDir, err)
		return nil, nil, nil, err
	}
	srv := &tsnet.Server{
		Dir:        tsnetDir,
		Hostname:   clientHostname,
		Logf:       logger.Printf,
		ControlURL: tsControlURL,
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
		logger.Println("initTsNet: Main context cancelled, ensuring tsnet server is closed.")
		if err := srv.Close(); err != nil {
			logger.Printf("initTsNet: Error closing tsnet server: %v", err)
		}
		cancel()
	}()
	logger.Printf("Initializing tsnet in directory: %s for client %s", tsnetDir, clientHostname)
	if !verbose {
		fmt.Fprintf(os.Stderr, "Starting Tailscale connection... You may need to authenticate.\nLook for a URL printed below if needed.\n")
	}
	status, err := srv.Up(ctx)
	if err != nil {
		logger.Fatalf("Failed to bring up tsnet: %v. If authentication is required, look for a URL in the logs (run with -v if not already).", err)
		return nil, nil, nil, err
	}
	if !verbose && status != nil && status.AuthURL != "" {
		fmt.Fprintf(os.Stderr, "\nTo authenticate, visit:\n%s\n", status.AuthURL)
		fmt.Fprintf(os.Stderr, "Please authenticate in the browser. The client will then attempt to connect.\n")
	}
	logger.Println("Waiting briefly for Tailscale connection to establish...")
	time.Sleep(3 * time.Second)
	
	currentStatus := status // Default to initial status
	lc, errClient := srv.LocalClient()
	if errClient != nil {
		logger.Printf("Warning: Failed to get LocalClient to update Tailscale status: %v. Using potentially stale status.", errClient)
	} else if lc == nil {
		logger.Printf("Warning: LocalClient is nil, cannot update Tailscale status. Using potentially stale status.")
	} else {
		updatedStatus, errStatus := lc.Status(ctx)
		if errStatus != nil {
			logger.Printf("Warning: Failed to get updated Tailscale status after initial Up: %v. Using potentially stale status.", errStatus)
		} else {
			currentStatus = updatedStatus
		}
	}
	return srv, ctx, currentStatus, nil
}

func connectToHostFromTUI(
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
	logger.Printf("TUI Connect: Attempting SSH connection to %s@%s (key: %s)", sshUser, targetHost, sshKeyPath)
	targetPort := defaultSSHPort
	sshTargetAddr := net.JoinHostPort(targetHost, targetPort)
	authMethods := []ssh.AuthMethod{}
	if sshKeyPath != "" {
		keyAuth, err := loadPrivateKey(sshKeyPath)
		if err == nil {
			authMethods = append(authMethods, keyAuth)
			logger.Printf("TUI Connect: Using public key authentication: %s", sshKeyPath)
		} else {
			logger.Printf("TUI Connect: Could not load private key %q: %v. Will attempt password auth.", sshKeyPath, err)
		}
	} else {
		logger.Printf("TUI Connect: No SSH key path specified. Will attempt password auth.")
	}
	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		fmt.Printf("Enter password for %s@%s: ", sshUser, targetHost)
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		return string(bytePassword), nil
	}))
	var hostKeyCallback ssh.HostKeyCallback
	var err error
	if insecureHostKey {
		logger.Println("TUI Connect: WARNING! Host key verification is disabled!")
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		hostKeyCallback, err = createKnownHostsCallback(currentUser)
		if err != nil {
			logger.Printf("TUI Connect: Could not set up host key verification: %v", err)
			return fmt.Errorf("host key setup failed: %w", err)
		}
		logger.Println("TUI Connect: Using known_hosts for host key verification.")
	}
	sshConfig := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         15 * time.Second,
	}
	logger.Printf("TUI Connect: Dialing %s via tsnet...", sshTargetAddr)
	dialCtx, dialCancel := context.WithTimeout(appCtx, sshConfig.Timeout)
	defer dialCancel()
	conn, err := srv.Dial(dialCtx, "tcp", sshTargetAddr)
	if err != nil {
		logger.Printf("TUI Connect: Failed to dial %s via tsnet: %v", sshTargetAddr, err)
		return fmt.Errorf("tsnet dial failed for %s: %w", sshTargetAddr, err)
	}
	logger.Printf("TUI Connect: tsnet Dial successful. Establishing SSH connection to %s...", sshTargetAddr)
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, sshTargetAddr, sshConfig)
	if err != nil {
		if strings.Contains(err.Error(), "unable to authenticate") || strings.Contains(err.Error(), "no supported authentication methods") {
			logger.Printf("TUI Connect: SSH Authentication failed for user %s: %v", sshUser, err)
		} else {
			var keyErr *knownhosts.KeyError
			if errors.As(err, &keyErr) {
				logger.Printf("TUI Connect: SSH Host key verification failed: %v", keyErr)
			} else {
				logger.Printf("TUI Connect: Failed to establish SSH connection to %s: %v", sshTargetAddr, err)
			}
		}
		conn.Close()
		return fmt.Errorf("ssh connection failed: %w", err)
	}
	defer sshConn.Close()
	logger.Println("TUI Connect: SSH connection established.")
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()
	logger.Println("TUI Connect: Starting interactive SSH session...")
	session, err := client.NewSession()
	if err != nil {
		logger.Printf("TUI Connect: Failed to create SSH session: %v", err)
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()
	fd := int(os.Stdin.Fd())
	var oldState *term.State
	if term.IsTerminal(fd) {
		oldState, err = term.MakeRaw(fd)
		if err != nil {
			logger.Printf("TUI Connect: Warning: Failed to set terminal to raw mode: %v", err)
		} else {
			defer term.Restore(fd, oldState)
		}
	} else {
		logger.Println("TUI Connect: Input is not a terminal. Interactive session may not work as expected.")
	}
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	if term.IsTerminal(fd) {
		termWidth, termHeight, errSize := term.GetSize(fd)
		if errSize != nil {
			logger.Printf("TUI Connect: Warning: Failed to get terminal size: %v. Using default 80x24.", errSize)
			termWidth = 80
			termHeight = 24
		}
		termType := os.Getenv("TERM")
		if termType == "" {
			termType = "xterm-256color"
		}
		errPty := session.RequestPty(termType, termHeight, termWidth, ssh.TerminalModes{})
		if errPty != nil {
			logger.Printf("TUI Connect: Failed to request pseudo-terminal: %v", errPty)
			return fmt.Errorf("pty request failed: %w", errPty)
		}
		go watchWindowSize(fd, session)
	}
	if err := session.Shell(); err != nil {
		logger.Printf("TUI Connect: Failed to start remote shell: %v", err)
		return fmt.Errorf("shell start failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "\nEscape sequence: ~. to terminate session (standard SSH)\n")
	sessionDone := make(chan struct{})
	go func() {
		select {
		case <-appCtx.Done():
			logger.Println("TUI Connect: Application context cancelled during session, closing SSH session.")
			session.Close()
		case <-sessionDone:
		}
	}()
	err = session.Wait()
	close(sessionDone)
	if oldState != nil && term.IsTerminal(fd) { // Redundant if defer worked, but safe
		term.Restore(fd, oldState)
	}
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			if verbose {
				logger.Printf("TUI Connect: Remote command exited with status %d", exitErr.ExitStatus())
			}
			return nil
		}
		if !errors.Is(err, io.EOF) && !strings.Contains(err.Error(), "session closed") && !strings.Contains(err.Error(), "channel closed") {
			logger.Printf("TUI Connect: SSH session ended with error: %v", err)
			return fmt.Errorf("ssh session error: %w", err)
		}
	}
	logger.Println("TUI Connect: SSH session closed.")
	return nil
}

func startTUI(app *tview.Application, srv *tsnet.Server, appCtx context.Context, logger *log.Logger, initialStatus *ipnstate.Status,
	sshUser string, sshKeyPath string, insecureHostKey bool, verbose bool) (tuiActionResult, error) {
	result := tuiActionResult{}
	pages := tview.NewPages()
	hostList := tview.NewList().ShowSecondaryText(true)
	hostList.SetBorder(true).SetTitle("Select a Host (ts-ssh) - Press (q) or Ctrl+C to Quit")
	infoBox := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("Select a host and press Enter.")

	populateHostList := func() {
		hostList.Clear()
		if initialStatus == nil || len(initialStatus.Peer) == 0 {
			hostList.AddItem("No peers found", "Please check your Tailscale network or wait for connection.", 0, nil)
			return
		}
		for _, peerStatus := range initialStatus.Peer {
			var displayName, connectTarget string
			if peerStatus.DNSName != "" {
				displayName = strings.TrimSuffix(peerStatus.DNSName, ".")
				connectTarget = displayName
			} else if peerStatus.HostName != "" {
				displayName = peerStatus.HostName
				connectTarget = displayName
			} else if len(peerStatus.TailscaleIPs) > 0 {
				displayName = peerStatus.TailscaleIPs[0].String()
				connectTarget = displayName
			} else {
				displayName = "Unknown Peer"
				connectTarget = "unknown-peer"
			}
			var secondaryText string
			if len(peerStatus.TailscaleIPs) > 0 {
				secondaryText = peerStatus.TailscaleIPs[0].String()
			} else {
				secondaryText = "No IP"
			}
			secondaryText += " | " + peerStatus.OS
			if peerStatus.Online { // Online is bool
				secondaryText += " [online]"
			} else {
				secondaryText += " [offline]"
			}
			itemConnectTarget := connectTarget
			itemDisplayName := displayName
			if connectTarget == "unknown-peer" {
				hostList.AddItem(displayName, secondaryText, 0, nil)
			} else {
				hostList.AddItem(displayName, secondaryText, 0, func() {
					logger.Printf("Selected host: %s, connect target: %s", itemDisplayName, itemConnectTarget)
					result.selectedHostTarget = itemConnectTarget
					pages.ShowPage("actionChoice")
				})
			}
		}
	}
	populateHostList()

	actionList := tview.NewList().
		AddItem("Interactive SSH", "Connect via an interactive SSH shell", 's', func() {
			result.action = "ssh"
			app.Stop()
		}).
		AddItem("SCP File Transfer", "Upload or download files via SCP", 'c', func() {
			result.action = "scp"
			pages.ShowPage("scpForm")
		}).
		AddItem("Back to Host List", "Choose a different host", 'b', func() {
			pages.HidePage("actionChoice")
			pages.ShowPage("hostSelection")
		}).
		SetWrapAround(true)
	actionList.SetBorder(true).SetTitle("Choose Action")

	scpForm := tview.NewForm().
		AddInputField("Local Path:", "", 40, nil, func(text string) { result.scpLocalPath = text }).
		AddInputField("Remote Path:", "", 40, nil, func(text string) { result.scpRemotePath = text }).
		AddDropDown("Direction:", []string{"Upload (Local to Remote)", "Download (Remote to Local)"}, 0, func(option string, optionIndex int) {
			result.scpIsUpload = (optionIndex == 0)
		}).
		AddButton("Start SCP", func() {
			if result.scpLocalPath == "" || result.scpRemotePath == "" {
				logger.Println("SCP Form: Local or Remote path is empty.")
			}
			app.Stop()
		}).
		AddButton("Cancel", func() {
			result.action = ""
			pages.HidePage("scpForm")
			pages.ShowPage("actionChoice")
		})
	scpForm.SetBorder(true).SetTitle("SCP Parameters")

	pages.AddPage("hostSelection", tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(hostList, 0, 1, true).
		AddItem(infoBox, 1, 0, false), true, true)
	pages.AddPage("actionChoice", actionList, true, false)
	pages.AddPage("scpForm", scpForm, true, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			result.action = ""
			app.Stop()
			return nil
		}
		if event.Rune() == 'q' {
			currentPage, _ := pages.GetFrontPage()
			if currentPage == "hostSelection" && app.GetFocus() == hostList {
				result.action = ""
				app.Stop()
				return nil
			}
		}
		return event
	})
	go func() {
		<-appCtx.Done()
		logger.Println("TUI: Application context cancelled, stopping tview app.")
		result.action = ""
		app.Stop()
	}()
	logger.Println("Starting TUI application with Pages...")
	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		logger.Printf("Error running TUI: %v", err)
		return result, fmt.Errorf("error running TUI: %w", err)
	}
	logger.Println("TUI application stopped.")
	return result, nil
}

func performSCPTransfer(
	srv *tsnet.Server,
	appCtx context.Context,
	logger *log.Logger,
	scpDetails tuiActionResult,
	sshUser string,
	sshKeyPath string,
	insecureHostKey bool,
	currentUser *user.User,
	verbose bool,
) error {
	logger.Printf("Performing SCP: Host=%s, Local=%s, Remote=%s, Upload=%t",
		scpDetails.selectedHostTarget, scpDetails.scpLocalPath, scpDetails.scpRemotePath, scpDetails.scpIsUpload)
	if scpDetails.scpLocalPath == "" || scpDetails.scpRemotePath == "" {
		return errors.New("local or remote path for SCP cannot be empty")
	}
	targetPort := defaultSSHPort
	sshTargetAddr := net.JoinHostPort(scpDetails.selectedHostTarget, targetPort)
	var authMethods []ssh.AuthMethod
	var scpSSHConfig ssh.ClientConfig
	if sshKeyPath != "" {
		keyAuth, keyErr := loadPrivateKey(sshKeyPath)
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
		fmt.Printf("Enter password for %s@%s (for SCP): ", sshUser, scpDetails.selectedHostTarget)
		bytePassword, passErr := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if passErr != nil {
			return "", fmt.Errorf("failed to read password for SCP: %w", passErr)
		}
		return string(bytePassword), nil
	}))
	var hostKeyCallback ssh.HostKeyCallback
	if insecureHostKey {
		logger.Println("SCP Connect: WARNING! Host key verification is disabled!")
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		var hkErr error
		hostKeyCallback, hkErr = createKnownHostsCallback(currentUser)
		if hkErr != nil {
			return fmt.Errorf("SCP: Could not set up host key verification: %w", hkErr)
		}
		logger.Println("SCP Connect: Using known_hosts for host key verification.")
	}
	scpSSHConfig = ssh.ClientConfig{
		User:            sshUser,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second,
	}
	logger.Printf("SCP Connect: Dialing %s via tsnet...", sshTargetAddr)
	dialCtx, dialCancel := context.WithTimeout(appCtx, scpSSHConfig.Timeout)
	defer dialCancel()
	conn, err := srv.Dial(dialCtx, "tcp", sshTargetAddr)
	if err != nil {
		return fmt.Errorf("SCP: tsnet dial failed for %s: %w", sshTargetAddr, err)
	}
	logger.Printf("SCP Connect: tsnet Dial successful. Establishing SSH client for SCP...")
	c, chans, reqs, err := ssh.NewClientConn(conn, sshTargetAddr, &scpSSHConfig)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SCP: failed to establish SSH client connection: %w", err)
	}
	sshClient := ssh.NewClient(c, chans, reqs)
	defer sshClient.Close()
	scpClient, err := scp.NewClientBySSH(sshClient)
	if err != nil {
		return fmt.Errorf("error creating new SCP client: %w", err)
	}
	defer scpClient.Close()
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
		permissions := fmt.Sprintf("0%o", fileInfo.Mode().Perm())
		errCopy := scpClient.CopyFile(appCtx, localFile, scpDetails.scpRemotePath, permissions)
		if errCopy != nil {
			return fmt.Errorf("error uploading file via SCP: %w", errCopy)
		}
		logger.Println("SCP: Upload complete.")
	} else {
		logger.Printf("SCP: Downloading %s:%s to %s", scpDetails.selectedHostTarget, scpDetails.scpRemotePath, scpDetails.scpLocalPath)
		localFile, errOpen := os.OpenFile(scpDetails.scpLocalPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if errOpen != nil {
			return fmt.Errorf("failed to open/create local file %s for download: %w", scpDetails.scpLocalPath, errOpen)
		}
		defer localFile.Close()
		errCopy := scpClient.CopyFromRemote(appCtx, localFile, scpDetails.scpRemotePath)
		if errCopy != nil {
			return fmt.Errorf("error downloading file via SCP: %w", errCopy)
		}
		logger.Println("SCP: Download complete.")
	}
	return nil
}

func parseTarget(target string) (host, port string, err error) {
	host = target
	port = defaultSSHPort

	if strings.Contains(host, ":") {
		if host[0] == '[' { // Potential IPv6 with port
			endBracket := strings.Index(host, "]")
			if endBracket == -1 {
				return "", "", fmt.Errorf("mismatched brackets in IPv6 address: %s", host)
			}
			if len(host) > endBracket+1 && host[endBracket+1] == ':' { // Format: [ipv6]:port
				port = host[endBracket+2:]
				host = host[1:endBracket]
			} else if len(host) > endBracket+1 { // Format: [ipv6]something_else (invalid)
				return "", "", fmt.Errorf("unexpected characters after ']' in IPv6 address: %s", host)
			} else { // Format: [ipv6] (no port)
				host = host[1:endBracket]
			}
		} else { // Not starting with '['.
			h, p, errSplit := net.SplitHostPort(host)
			if errSplit == nil {
				host = h
				port = p
			} else {
				addrErr, ok := errSplit.(*net.AddrError)
				if ok && (strings.Contains(addrErr.Err, "missing port in address") || strings.Contains(addrErr.Err, "too many colons in address")) {
					lastColon := strings.LastIndex(host, ":")
					if lastColon > 0 && lastColon < len(host)-1 {
						potentialPort := host[lastColon+1:]
						isNumeric := true
						for _, char := range potentialPort {
							if char < '0' || char > '9' {
								isNumeric = false
								break
							}
						}
						if isNumeric {
							host = host[:lastColon]
							port = potentialPort
						}
					}
				} else if errSplit != nil {
					return "", "", fmt.Errorf("invalid target format: %w", errSplit)
				}
			}
		}
	}

	if host == "" {
		return "", "", errors.New("hostname cannot be empty")
	}
	return host, port, nil
}


func loadPrivateKey(path string) (ssh.AuthMethod, error) {
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
		fmt.Printf("Enter passphrase for key %s: ", path)
		bytePassword, errRead := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if errRead != nil {
			return nil, fmt.Errorf("failed to read passphrase: %w", errRead)
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, bytePassword)
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

func createKnownHostsCallback(currentUser *user.User) (ssh.HostKeyCallback, error) {
	if currentUser == nil || currentUser.HomeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ssh.InsecureIgnoreHostKey(), fmt.Errorf("cannot determine user home directory for known_hosts, disabling host key check: %w", err)
		}
		currentUser = &user.User{HomeDir: home}
		log.Printf("Warning: Could not get current user initially, found home dir %s. Proceeding with known_hosts.", home)
	}
	knownHostsPath := filepath.Join(currentUser.HomeDir, ".ssh", "known_hosts")
	sshDir := filepath.Dir(knownHostsPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return ssh.InsecureIgnoreHostKey(), fmt.Errorf("failed to create %s directory, disabling host key check: %w", sshDir, err)
	}
	f, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_RDONLY, 0600)
	if err != nil {
		return ssh.InsecureIgnoreHostKey(), fmt.Errorf("unable to create/open %s, disabling host key check: %w", knownHostsPath, err)
	}
	f.Close()
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return ssh.InsecureIgnoreHostKey(), fmt.Errorf("could not initialize known_hosts callback using %s, disabling host key check: %w", knownHostsPath, err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := hostKeyCallback(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) {
			if len(keyErr.Want) > 0 {
				fmt.Fprintf(os.Stderr, "\n@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n")
				fmt.Fprintf(os.Stderr, "@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @\n")
				fmt.Fprintf(os.Stderr, "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n")
				fmt.Fprintf(os.Stderr, "IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!\n")
				fmt.Fprintf(os.Stderr, "Someone could be eavesdropping on you right now (man-in-the-middle attack)!\n")
				fmt.Fprintf(os.Stderr, "It is also possible that a host key has just been changed.\n")
				fmt.Fprintf(os.Stderr, "The fingerprint for the %s key sent by the remote host %s is:\n%s\n", key.Type(), remote.String(), ssh.FingerprintSHA256(key))
				fmt.Fprintf(os.Stderr, "Please contact your system administrator.\n")
				fmt.Fprintf(os.Stderr, "Offending key for host %s found in %s:%d\n", hostname, keyErr.Want[0].Filename, keyErr.Want[0].Line)
				return keyErr
			} else {
				fmt.Fprintf(os.Stderr, "The authenticity of host '%s (%s)' can't be established.\n", hostname, remote.String())
				fmt.Fprintf(os.Stderr, "%s key fingerprint is %s.\n", key.Type(), ssh.FingerprintSHA256(key))
				answer, readErr := promptUserViaTTY("Are you sure you want to continue connecting (yes/no)? ")
				if readErr != nil {
					return fmt.Errorf("failed to read user confirmation: %w", readErr)
				}
				if answer == "yes" {
					return appendKnownHost(knownHostsPath, hostname, remote, key)
				} else {
					return errors.New("host key verification failed: user declined")
				}
			}
		}
		return fmt.Errorf("unexpected error during host key verification: %w", err)
	}, nil
}

func appendKnownHost(knownHostsPath, hostname string, remote net.Addr, key ssh.PublicKey) error {
	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s to append new key: %w", knownHostsPath, err)
	}
	defer f.Close()
	normalizedAddress := knownhosts.Normalize(remote.String())
	line := knownhosts.Line([]string{normalizedAddress}, key)
	if _, err := fmt.Fprintln(f, line); err != nil {
		return fmt.Errorf("failed to write host key to %s: %w", knownHostsPath, err)
	}
	fmt.Fprintf(os.Stderr, "Warning: Permanently added '%s' (%s) to the list of known hosts.\n", normalizedAddress, key.Type())
	return nil
}

func watchWindowSize(fd int, session *ssh.Session) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	if term.IsTerminal(fd) {
		termWidth, termHeight, _ := term.GetSize(fd)
		if termWidth > 0 && termHeight > 0 {
			_ = session.WindowChange(termHeight, termWidth)
		}
	}
	for range sigCh {
		if term.IsTerminal(fd) {
			termWidth, termHeight, err := term.GetSize(fd)
			if err == nil && termWidth > 0 && termHeight > 0 {
				_ = session.WindowChange(termHeight, termWidth)
			}
		}
	}
}

func promptUserViaTTY(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprint(os.Stderr, "(could not open /dev/tty, reading from stdin): ")
		reader := bufio.NewReader(os.Stdin)
		line, errRead := reader.ReadString('\n')
		if errRead != nil {
			return "", fmt.Errorf("failed to read from stdin fallback: %w", errRead)
		}
		return strings.ToLower(strings.TrimSpace(line)), nil
	}
	defer tty.Close()
	reader := bufio.NewReader(tty)
	line, errRead := reader.ReadString('\n')
	if errRead != nil {
		return "", fmt.Errorf("failed to read from tty: %w", errRead)
	}
	return strings.ToLower(strings.TrimSpace(line)), nil
}
