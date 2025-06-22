package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
	"tailscale.com/tsnet"

	"github.com/derekg/ts-ssh/internal/client/scp"
	sshclient "github.com/derekg/ts-ssh/internal/client/ssh"
)

// AppConfig holds all the configuration for the application
type AppConfig struct {
	SSHUser         string
	SSHKeyPath      string
	TsnetDir        string
	TsControlURL    string
	Target          string
	Verbose         bool
	InsecureHostKey bool
	ForwardDest     string
	ShowVersion     bool
	LangFlag        string
	// Power CLI features
	ListHosts  bool
	MultiHosts string
	ExecCmd    string
	CopyFiles  string
	PickHost   bool
	Parallel   bool
	// Derived values
	RemoteCmd []string
	Logger    *log.Logger
}

// parseCommandLineArgs sets up and parses all command line arguments
func parseCommandLineArgs() *AppConfig {
	config := &AppConfig{}

	// Get current user for defaults
	currentUser, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not determine current user: %v. Using 'user' as default.\n", err)
		currentUser = &user.User{Username: "user", HomeDir: "/home/user"}
	}

	// Set up defaults
	defaultUser := currentUser.Username
	defaultKeyPath := filepath.Join(currentUser.HomeDir, ".ssh", "id_rsa")
	defaultTsnetDir := filepath.Join(currentUser.HomeDir, ".config", ClientName)

	// Initialize i18n early to support flag descriptions
	initI18n("")

	// Define flags
	flag.StringVar(&config.LangFlag, "lang", "", T("flag_lang_desc"))
	flag.StringVar(&config.SSHUser, "l", defaultUser, T("flag_user_desc"))
	flag.StringVar(&config.SSHKeyPath, "i", defaultKeyPath, T("flag_key_desc"))
	flag.StringVar(&config.TsnetDir, "tsnet-dir", defaultTsnetDir, T("flag_tsnet_desc"))
	flag.StringVar(&config.TsControlURL, "control-url", "", T("flag_control_desc"))
	flag.BoolVar(&config.Verbose, "v", false, T("flag_verbose_desc"))
	flag.BoolVar(&config.InsecureHostKey, "insecure", false, T("flag_insecure_desc"))
	flag.StringVar(&config.ForwardDest, "W", "", T("flag_forward_desc"))
	flag.BoolVar(&config.ShowVersion, "version", false, T("flag_version_desc"))

	// Power CLI features
	flag.BoolVar(&config.ListHosts, "list", false, T("flag_list_desc"))
	flag.StringVar(&config.MultiHosts, "multi", "", T("flag_multi_desc"))
	flag.StringVar(&config.ExecCmd, "exec", "", T("flag_exec_desc"))
	flag.StringVar(&config.CopyFiles, "copy", "", T("flag_copy_desc"))
	flag.BoolVar(&config.PickHost, "pick", false, T("flag_pick_desc"))
	flag.BoolVar(&config.Parallel, "parallel", false, T("flag_parallel_desc"))

	// Set up dynamic usage function for language support
	flag.Usage = createUsageFunction()

	flag.Parse()

	// Reinitialize i18n with the actual language flag after parsing
	initI18n(config.LangFlag)

	// Set up logger
	if config.Verbose {
		config.Logger = log.Default()
	} else {
		config.Logger = log.New(io.Discard, "", 0)
	}

	// Get remaining arguments as remote command
	config.RemoteCmd = flag.Args()[1:]

	return config
}

// createUsageFunction returns a usage function that supports dynamic language switching
func createUsageFunction() func() {
	return func() {
		// Parse args to get language flag before displaying help
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

		fmt.Fprint(os.Stderr, T("usage_header", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("usage_list", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("usage_multi", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("usage_exec", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("usage_copy", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("usage_pick", os.Args[0])+"\n\n")
		fmt.Fprint(os.Stderr, T("usage_description")+"\n")
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, T("examples_header")+"\n")
		fmt.Fprint(os.Stderr, T("examples_basic_ssh")+"\n")
		fmt.Fprint(os.Stderr, T("examples_interactive", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("examples_remote_cmd", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("examples_host_discovery")+"\n")
		fmt.Fprint(os.Stderr, T("examples_list_hosts", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("examples_pick_host", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("examples_multi_host")+"\n")
		fmt.Fprint(os.Stderr, T("examples_tmux", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("examples_exec_multi", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("examples_parallel", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("examples_file_transfer")+"\n")
		fmt.Fprint(os.Stderr, T("examples_scp_single", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("examples_scp_multi", os.Args[0])+"\n")
		fmt.Fprint(os.Stderr, T("examples_proxy")+"\n")
		fmt.Fprint(os.Stderr, T("examples_proxy_cmd", os.Args[0])+"\n")
	}
}

// handleVersionFlag displays version information and exits if requested
func handleVersionFlag(config *AppConfig) {
	if config.ShowVersion {
		fmt.Println(version)
		os.Exit(0)
	}
}

// isPowerCLIMode determines if we're in power CLI mode (list, multi, exec, copy, pick)
func isPowerCLIMode(config *AppConfig) bool {
	return config.ListHosts || config.MultiHosts != "" || config.ExecCmd != "" || 
		   config.CopyFiles != "" || config.PickHost
}

// handlePowerCLI handles all power CLI operations
func handlePowerCLI(config *AppConfig) error {
	srv, ctx, status, err := initTsNet(config.TsnetDir, ClientName, config.Logger, config.TsControlURL, config.Verbose)
	if err != nil {
		return fmt.Errorf("%s", T("error_init_tailscale"))
	}

	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		config.Logger.Printf("Warning: Could not determine current user: %v", err)
		currentUser = &user.User{Username: config.SSHUser}
	}

	// Handle each power CLI operation
	if config.ListHosts {
		return handleListHosts(status, config.Verbose)
	}

	if config.MultiHosts != "" {
		return handleMultiHosts(config.MultiHosts, config.Logger, config.SSHUser, 
							   config.SSHKeyPath, config.InsecureHostKey)
	}

	if config.ExecCmd != "" {
		hosts := parseHostList(flag.Args())
		return handleExecCommand(srv, ctx, config.ExecCmd, hosts, config.Logger, 
								config.SSHUser, config.SSHKeyPath, config.InsecureHostKey, 
								config.Parallel, config.Verbose)
	}

	if config.CopyFiles != "" {
		return handleCopyFiles(srv, ctx, config.CopyFiles, config.Logger, 
							  config.SSHUser, config.SSHKeyPath, config.InsecureHostKey, 
							  config.Verbose)
	}

	if config.PickHost {
		return handlePickHost(srv, ctx, status, config.Logger, config.SSHUser, 
							 config.SSHKeyPath, config.InsecureHostKey, currentUser, 
							 config.Verbose)
	}

	return nil
}

// detectSCPOperation checks if this is an SCP operation and returns parsed arguments
func detectSCPOperation(config *AppConfig) *scpArgs {
	// TODO: Implement SCP detection logic from original main.go
	return nil
}

// handleSCPOperation performs the SCP file transfer
func handleSCPOperation(scpArgs *scpArgs, config *AppConfig) error {
	srv, ctx, _, err := initTsNet(config.TsnetDir, ClientName, config.Logger, config.TsControlURL, config.Verbose)
	if err != nil {
		return fmt.Errorf("%s", T("error_init_tailscale"))
	}

	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		config.Logger.Printf("Warning: Could not determine current user: %v", err)
		currentUser = &user.User{Username: config.SSHUser}
	}

	err = scp.HandleCliScp(srv, ctx, config.Logger, scpArgs.sshUser, config.SSHKeyPath,
					  config.InsecureHostKey, currentUser, scpArgs.localPath,
					  scpArgs.remotePath, scpArgs.targetHost, true, config.Verbose)

	if err != nil {
		return fmt.Errorf("%s", T("error_scp_failed"))
	}

	fmt.Println(T("scp_success"))
	return nil
}

// handleSSHOperation performs a regular SSH connection
func handleSSHOperation(config *AppConfig) error {
	// Parse SSH target
	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	config.Target = flag.Args()[0]
	targetHost, targetPort, err := parseTarget(config.Target, DefaultSshPort)
	if err != nil {
		return fmt.Errorf("%s", T("error_parsing_target"))
	}

	// Determine SSH user
	sshSpecificUser := config.SSHUser
	if strings.Contains(targetHost, "@") {
		parts := strings.SplitN(targetHost, "@", 2)
		sshSpecificUser = parts[0]
		targetHost = parts[1]
	}

	// Initialize tsnet
	srv, nonTuiCtx, _, err := initTsNet(config.TsnetDir, ClientName, config.Logger, config.TsControlURL, config.Verbose)
	if err != nil {
		return fmt.Errorf("%s", T("error_init_ssh"))
	}

	// Handle ProxyCommand mode
	if config.ForwardDest != "" {
		return handleProxyCommand(srv, nonTuiCtx, config.ForwardDest, config.Logger)
	}

	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		config.Logger.Printf("Warning: Could not determine current user: %v", err)
		currentUser = &user.User{Username: sshSpecificUser}
	}

	// Establish SSH connection
	sshConfig := sshclient.SSHConnectionConfig{
		User:            sshSpecificUser,
		KeyPath:         config.SSHKeyPath,
		TargetHost:      targetHost,
		TargetPort:      targetPort,
		InsecureHostKey: config.InsecureHostKey,
		Verbose:         config.Verbose,
		CurrentUser:     currentUser,
		Logger:          config.Logger,
	}

	client, err := sshclient.EstablishSSHConnection(srv, nonTuiCtx, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to establish SSH connection: %w", err)
	}
	defer client.Close()

	// Handle remote command or interactive session
	if len(config.RemoteCmd) > 0 {
		return executeRemoteCommand(client, config.RemoteCmd, config.Logger)
	}

	return startInteractiveSession(client, config.Logger)
}

// handleProxyCommand handles ProxyCommand stdio forwarding
func handleProxyCommand(srv *tsnet.Server, ctx context.Context, forwardDest string, logger *log.Logger) error {
	logger.Printf("Forwarding stdio to %s via tsnet...", forwardDest)
	fwdConn, err := srv.Dial(ctx, "tcp", forwardDest)
	if err != nil {
		return fmt.Errorf("failed to dial %s via tsnet for forwarding: %w", forwardDest, err)
	}
	
	go func() {
		_, _ = io.Copy(fwdConn, os.Stdin)
		fwdConn.Close()
	}()
	
	_, _ = io.Copy(os.Stdout, fwdConn)
	return nil
}

// executeRemoteCommand executes a command on the remote host and returns
func executeRemoteCommand(client *ssh.Client, remoteCmd []string, logger *log.Logger) error {
	logger.Printf("Running remote command: %v", remoteCmd)
	session, err := sshclient.CreateSSHSession(client)
	if err != nil {
		return fmt.Errorf("failed to create SSH session for remote command: %w", err)
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	cmdStr := strings.Join(remoteCmd, " ")
	err = session.Run(cmdStr)
	if err != nil {
		if exitError, ok := err.(*ssh.ExitError); ok {
			os.Exit(exitError.ExitStatus())
		}
		return fmt.Errorf("remote command failed: %w", err)
	}

	return nil
}

// startInteractiveSession starts an interactive SSH session with PTY support
func startInteractiveSession(client *ssh.Client, logger *log.Logger) error {
	logger.Println("Starting interactive SSH session...")
	session, err := sshclient.CreateSSHSession(client)
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Set up stdin pipe
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe for SSH session: %w", err)
	}
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Set up terminal if running in one
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		err = setupTerminal(session, fd, logger)
		if err != nil {
			return fmt.Errorf("failed to setup terminal: %w", err)
		}
	}

	// Start the shell
	err = session.Shell()
	if err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Handle terminal resizing and escape sequences
	return handleInteractiveSession(session, stdinPipe, fd, logger)
}

// setupTerminal configures the terminal for interactive SSH session
func setupTerminal(session *ssh.Session, fd int, logger *log.Logger) error {
	termWidth, termHeight, err := term.GetSize(fd)
	if err != nil {
		logger.Printf("Warning: Failed to get terminal size: %v. Using default %dx%d.", err, DefaultTerminalWidth, DefaultTerminalHeight)
		termWidth = DefaultTerminalWidth
		termHeight = DefaultTerminalHeight
	}
	
	termType := os.Getenv("TERM")
	if termType == "" {
		termType = DefaultTerminalType
	}
	
	err = session.RequestPty(termType, termHeight, termWidth, ssh.TerminalModes{})
	if err != nil {
		return fmt.Errorf("failed to request pseudo-terminal: %w", err)
	}
	
	return nil
}

// handleInteractiveSession manages the interactive SSH session with proper terminal handling
func handleInteractiveSession(session *ssh.Session, stdinPipe io.WriteCloser, fd int, logger *log.Logger) error {
	termState := GetGlobalTerminalState()
	
	// Set up terminal in raw mode if we're in a terminal
	if term.IsTerminal(fd) {
		err := termState.MakeRaw(fd)
		if err != nil {
			logger.Printf("Warning: Failed to set terminal to raw mode: %v", err)
		} else {
			// Ensure terminal is restored on exit
			defer func() {
				if err := termState.Restore(); err != nil {
					logger.Printf("Warning: Failed to restore terminal: %v", err)
				}
			}()
		}
		
		fmt.Fprint(os.Stderr, T("escape_sequence")+"\n")
	}
	
	// Set up signal handling for graceful shutdown
	done := make(chan bool, 1)
	go handleInputWithTerminalState(stdinPipe, done, logger, termState)
	
	// Handle window resize signals if in terminal
	if term.IsTerminal(fd) {
		go handleSignalsAndResizeWithTerminalState(session, termState, logger)
	}
	
	// Wait for session to complete
	err := session.Wait()
	done <- true // Signal input handler to stop
	
	return err
}

// handleInputWithTerminalState handles stdin input with terminal state awareness
func handleInputWithTerminalState(stdinPipe io.WriteCloser, done chan bool, logger *log.Logger, termState *TerminalStateManager) {
	defer stdinPipe.Close()
	
	// Create a buffered reader for stdin
	input := make([]byte, 1024)
	
	for {
		select {
		case <-done:
			return
		default:
			n, err := os.Stdin.Read(input)
			if err != nil {
				if err != io.EOF {
					logger.Printf("Error reading stdin: %v", err)
				}
				return
			}
			
			// Write to SSH session
			_, writeErr := stdinPipe.Write(input[:n])
			if writeErr != nil {
				logger.Printf("Error writing to SSH session: %v", writeErr)
				return
			}
		}
	}
}

