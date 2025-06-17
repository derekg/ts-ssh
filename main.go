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
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

// scpArgs holds parsed arguments for an SCP operation.
type scpArgs struct {
	isUpload   bool
	localPath  string
	remotePath string
	targetHost string
	sshUser    string // User from user@host:path, if present
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
		return "", "", "", fmt.Errorf(T("invalid_scp_remote"), remoteArg)
	}
	path = parts[1]
	hostPart := parts[0]

	if strings.Contains(hostPart, "@") {
		userHostParts := strings.SplitN(hostPart, "@", 2)
		if len(userHostParts) != 2 || userHostParts[0] == "" || userHostParts[1] == "" {
			return "", "", "", fmt.Errorf(T("invalid_user_host"), hostPart)
		}
		user = userHostParts[0]
		host = userHostParts[1]
	} else {
		host = hostPart
	}

	if host == "" {
		return "", "", "", fmt.Errorf(T("empty_host_scp"), remoteArg)
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
		langFlag        string
		// Power CLI features
		listHosts       bool
		multiHosts      string
		execCmd         string
		copyFiles       string
		pickHost        bool
		parallel        bool
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

	// Initialize i18n early to support flag descriptions
	// We'll do a basic initialization here and reinitialize after parsing flags
	initI18n("")
	
	flag.StringVar(&langFlag, "lang", "", T("flag_lang_desc"))
	flag.StringVar(&sshUser, "l", defaultUser, T("flag_user_desc"))
	flag.StringVar(&sshKeyPath, "i", defaultKeyPath, T("flag_key_desc"))
	flag.StringVar(&tsnetDir, "tsnet-dir", defaultTsnetDir, T("flag_tsnet_desc"))
	flag.StringVar(&tsControlURL, "control-url", "", T("flag_control_desc"))
	flag.BoolVar(&verbose, "v", false, T("flag_verbose_desc"))
	flag.BoolVar(&insecureHostKey, "insecure", false, T("flag_insecure_desc"))
	flag.StringVar(&forwardDest, "W", "", T("flag_forward_desc"))
	flag.BoolVar(&showVersion, "version", false, T("flag_version_desc"))
	
	// Power CLI features
	flag.BoolVar(&listHosts, "list", false, T("flag_list_desc"))
	flag.StringVar(&multiHosts, "multi", "", T("flag_multi_desc"))
	flag.StringVar(&execCmd, "exec", "", T("flag_exec_desc"))
	flag.StringVar(&copyFiles, "copy", "", T("flag_copy_desc"))
	flag.BoolVar(&pickHost, "pick", false, T("flag_pick_desc"))
	flag.BoolVar(&parallel, "parallel", false, T("flag_parallel_desc"))
	flag.Usage = func() {
		// Parse args to get language flag before displaying help
		// This is a bit hacky but necessary for dynamic language in help
		tempLang := ""
		for i, arg := range os.Args[1:] {
			if arg == "--lang" && i+1 < len(os.Args[1:]) {
				tempLang = os.Args[i+2]
				break
			} else if strings.HasPrefix(arg, "--lang=") {
				tempLang = strings.SplitN(arg, "=", 2)[1]
				break
			}
		}
		
		// Temporarily reinitialize i18n for help display
		if tempLang != "" {
			initI18n(tempLang)
		}
		
		fmt.Fprintf(os.Stderr, T("usage_header", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("usage_list", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("usage_multi", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("usage_exec", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("usage_copy", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("usage_pick", os.Args[0])+"\n\n")
		fmt.Fprintf(os.Stderr, T("usage_description")+"\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, T("examples_header")+"\n")
		fmt.Fprintf(os.Stderr, T("examples_basic_ssh")+"\n")
		fmt.Fprintf(os.Stderr, T("examples_interactive", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("examples_remote_cmd", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("examples_host_discovery")+"\n")
		fmt.Fprintf(os.Stderr, T("examples_list_hosts", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("examples_pick_host", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("examples_multi_host")+"\n")
		fmt.Fprintf(os.Stderr, T("examples_tmux", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("examples_exec_multi", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("examples_parallel", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("examples_file_transfer")+"\n")
		fmt.Fprintf(os.Stderr, T("examples_scp_single", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("examples_scp_multi", os.Args[0])+"\n")
		fmt.Fprintf(os.Stderr, T("examples_proxy")+"\n")
		fmt.Fprintf(os.Stderr, T("examples_proxy_cmd", os.Args[0])+"\n")
	}
	flag.Parse()

	// Reinitialize i18n with the actual language flag after parsing
	initI18n(langFlag)

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

	// Handle power CLI features
	if listHosts || pickHost || multiHosts != "" || execCmd != "" || copyFiles != "" {
		srv, ctx, status, err := initTsNet(tsnetDir, clientName, logger, tsControlURL, verbose, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, T("error_init_tailscale")+"\n", err)
			os.Exit(1)
		}
		defer srv.Close()

		if listHosts {
			err = handleListHosts(status, verbose)
		} else if pickHost {
			err = handlePickHost(srv, ctx, status, logger, sshUser, sshKeyPath, insecureHostKey, currentUser, verbose)
		} else if multiHosts != "" {
			err = handleMultiHosts(multiHosts, logger, sshUser, sshKeyPath, insecureHostKey)
		} else if execCmd != "" {
			hosts := parseHostList(flag.Args())
			err = handleExecCommand(srv, ctx, execCmd, hosts, logger, sshUser, sshKeyPath, insecureHostKey, parallel, verbose)
		} else if copyFiles != "" {
			err = handleCopyFiles(srv, ctx, copyFiles, logger, sshUser, sshKeyPath, insecureHostKey, verbose)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// --- SCP Argument Parsing (CLI) ---
	var detectedScpArgs *scpArgs
	nonFlagArgs := flag.Args()
	
	// Use sshUser from -l flag as the default for SCP user.
	// It will be overridden if user@host is specified in the remote path.
	defaultScpUser := sshUser 

	if len(nonFlagArgs) == 2 {
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
			fmt.Fprintf(os.Stderr, T("error_init_tailscale")+"\n", err)
			os.Exit(1)
		}
		defer srv.Close()
		
		err = HandleCliScp(srv, ctx, logger, detectedScpArgs.sshUser, sshKeyPath, insecureHostKey, currentUser,
			detectedScpArgs.localPath, detectedScpArgs.remotePath, detectedScpArgs.targetHost,
			detectedScpArgs.isUpload, verbose)

		if err != nil {
			fmt.Fprintf(os.Stderr, T("error_scp_failed")+"\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, T("scp_success"))
		os.Exit(0)
	}

	// Fallthrough to SSH logic if not SCP 
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
		logger.Fatalf(T("error_parsing_target"), err)
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
		fmt.Fprintf(os.Stderr, T("error_init_ssh")+"\n", err)
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
		logger.Printf(T("using_key_auth"), sshKeyPath)
	} else {
		logger.Printf(T("key_auth_failed"), err)
	}
	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		fmt.Printf(T("enter_password"), sshSpecificUser, targetHost)
		bytePassword, errRead := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if errRead != nil {
			return "", fmt.Errorf("failed to read password: %w", errRead)
		}
		return string(bytePassword), nil
	}))

	var hostKeyCallback ssh.HostKeyCallback
	if insecureHostKey {
		logger.Println(T("host_key_warning"))
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
	logger.Printf(T("dial_via_tsnet"), sshTargetAddr)
	conn, err_dial := srv.Dial(nonTuiCtx, "tcp", sshTargetAddr) // Renamed err to err_dial to avoid conflict
	if err_dial != nil {
		log.Fatalf(T("dial_failed"), sshTargetAddr, err_dial)
	}
	logger.Printf("tsnet Dial successful. Establishing SSH connection...")

	sshConn, chans, reqs, err_conn := ssh.NewClientConn(conn, sshTargetAddr, sshConfig) // Renamed err to err_conn
	if err_conn != nil {
		if strings.Contains(err_conn.Error(), "unable to authenticate") || strings.Contains(err_conn.Error(), "no supported authentication methods") {
			log.Fatalf(T("ssh_auth_failed"), sshSpecificUser, err_conn)
		}
		var keyErr *knownhosts.KeyError
		if errors.As(err_conn, &keyErr) {
			log.Fatalf(T("host_key_failed"), err_conn)
		}
		log.Fatalf(T("ssh_connection_failed"), sshTargetAddr, err_conn)
	}
	defer sshConn.Close()
	logger.Println(T("ssh_connection_established"))

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
		if runtime.GOOS != "windows" {
			go watchWindowSize(fd, session, nonTuiCtx, logger)
		}
	}

	err = session.Shell()
	if err != nil {
		log.Fatalf("Failed to start remote shell: %v", err)
	}
	fmt.Fprintf(os.Stderr, T("escape_sequence")+"\n")
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
	logger.Println(T("ssh_session_closed"))
}

// Moved to tsnet_handler.go:
// func initTsNet(tsnetDir, clientHostname string, logger *log.Logger, tsControlURL string, verbose, tuiMode bool) (*tsnet.Server, context.Context, *ipnstate.Status, error)
