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

	// Import encoding/base64 only if manually formatting known_hosts line
	// "encoding/base64"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
	"tailscale.com/tsnet"
	// Added for potential error checking, though not strictly needed for Auth URL now
	// "tailscale.com/ipn/ipnstate"
)

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
		// forwardDest, if set, will proxy stdio to this host:port via tsnet (ProxyCommand -W)
		forwardDest string
		// showVersion prints the tool version and exits
		showVersion bool
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
	defaultTsnetDir := "" // Let tsnet decide default based on OS if empty
	if currentUser != nil {
		// Store state in a hidden directory within user's home
		defaultTsnetDir = filepath.Join(currentUser.HomeDir, ".config", clientName)
	}

	flag.StringVar(&sshUser, "l", defaultUser, "SSH Username")
	flag.StringVar(&sshKeyPath, "i", defaultKeyPath, "Path to SSH private key")
	flag.StringVar(&tsnetDir, "tsnet-dir", defaultTsnetDir, "Directory to store tsnet state")
	flag.StringVar(&tsControlURL, "control-url", "", "Tailscale control plane URL (optional)")
	flag.BoolVar(&verbose, "v", false, "Verbose logging")
	flag.BoolVar(&insecureHostKey, "insecure", false, "Disable host key checking (INSECURE!)")
	// ProxyCommand-style forwarding: direct stdio to dest via tsnet
	flag.StringVar(&forwardDest, "W", "", "forward stdio to destination host:port (for use as ProxyCommand)")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
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
	// Parse flags and arguments: require at least target, optional command
	flag.Parse()
	if showVersion {
		fmt.Fprintf(os.Stdout, "%s\n", version)
		os.Exit(0)
	}
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	target = flag.Arg(0)
	// Any additional args are treated as a remote command to execute
	var remoteCmd []string
	if flag.NArg() > 1 {
		remoteCmd = flag.Args()[1:]
	}

	// --- Parse Target: [user@]hostname[:port] ---
	targetHost, targetPort, err := parseTarget(target)
	if err != nil {
		log.Fatalf("Error parsing target: %v", err)
	}
	// Override user if specified in target string
	if strings.Contains(targetHost, "@") {
		parts := strings.SplitN(targetHost, "@", 2)
		sshUser = parts[0]
		targetHost = parts[1]
	}

	// --- Configure Logging ---
	logger := log.New(io.Discard, "", 0) // Default to no logging
	if verbose {
		logger = log.Default() // Use standard logger if verbose
		logger.Printf("Starting %s...", clientName)
	} else {
		// Ensure authentication URL is always printed if needed, even without -v
		// We achieve this by checking the error below and printing separately if needed.
		// The Logf below will handle verbose logging case.
	}

	// --- Setup tsnet Server ---
	if tsnetDir == "" {
		// Fallback if home dir wasn't found earlier
		tsnetDir = clientName + "-state"
		logger.Printf("Warning: Could not determine user home directory, using state dir: %s", tsnetDir)
	}

	if err := os.MkdirAll(tsnetDir, 0700); err != nil && !os.IsExist(err) {
		log.Fatalf("Failed to create tsnet state directory %q: %v", tsnetDir, err)
	}

	srv := &tsnet.Server{
		Dir:        tsnetDir,
		Hostname:   clientName,    // How this client identifies itself on the tailnet
		Logf:       logger.Printf, // Use our configured logger
		ControlURL: tsControlURL,  // Optional: Use custom control server
	}
	// Defer closing the tsnet server to disconnect from Tailscale on exit
	defer srv.Close()

	// Context for tsnet operations and graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT (Ctrl+C) and SIGTERM for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Println("Signal received, shutting down...")
		cancel()    // Cancel the context
		srv.Close() // Ensure tsnet is closed
		// Restore terminal potentially needed if shutdown happens during session
		fd := int(os.Stdin.Fd())
		if term.IsTerminal(fd) {
			// Attempt to restore, might fail if not in raw mode, ignore error
			_ = term.Restore(fd, nil) // Pass nil state, might not be perfect but better than nothing
		}
		os.Exit(1) // Exit after cleanup attempt
	}()

	logger.Printf("Initializing tsnet in directory: %s", tsnetDir)
	fmt.Fprintf(os.Stderr, "Starting Tailscale connection... You may need to authenticate.\nCheck logs (-v) or look for a URL printed below if needed.\n")

	// Start the tsnet server. This blocks until preferences are fetched
	// or fatally errors. It *doesn't* block until fully connected/authenticated usually.
	status, err := srv.Up(ctx)
	if err != nil {
		// *** CORRECTED ERROR HANDLING FOR AUTH ***
		// We don't look for a specific error type anymore.
		// If Up() fails, it's a problem. The auth URL should have been printed
		// by the Logf callback if needed. We add a hint here.
		log.Fatalf("Failed to bring up tsnet: %v. If authentication is required, look for a URL in the logs above (or run with -v).", err)
	}

	// Check the status right after Up for the auth URL *if* not verbose
	// The Logf callback already handles printing it if verbose.
	if !verbose && status != nil && status.AuthURL != "" {
		fmt.Fprintf(os.Stderr, "\nTo authenticate, visit:\n%s\n", status.AuthURL)
		// Give the user some time, or instruct them to restart after auth
		fmt.Fprintf(os.Stderr, "Please authenticate in the browser and potentially restart the client if connection fails.\n")
		// Consider adding a longer wait or a loop checking status again here for better UX
	}

	// Wait briefly for the connection to likely establish *after* potential auth.
	// A more robust method would involve checking srv.Status() in a loop.
	// Wait briefly for the connection to likely establish *after* potential auth.
	// A more robust method would involve checking srv.Status() in a loop.
	logger.Println("Waiting briefly for Tailscale connection to establish...")
	time.Sleep(3 * time.Second) // Adjust as needed

	// If requested, proxy raw TCP for ProxyCommand-style forwarding (-W)
	if forwardDest != "" {
		logger.Printf("Forwarding stdio to %s via tsnet...", forwardDest)
		fwdConn, err := srv.Dial(ctx, "tcp", forwardDest)
		if err != nil {
			log.Fatalf("Failed to dial %s via tsnet for forwarding: %v", forwardDest, err)
		}
		// Pipe stdin to remote, and remote to stdout
		go func() {
			_, _ = io.Copy(fwdConn, os.Stdin)
			fwdConn.Close()
		}()
		_, _ = io.Copy(os.Stdout, fwdConn)
		os.Exit(0)
	}
	logger.Printf("tsnet potentially initialized. Attempting SSH connection to %s@%s:%s", sshUser, targetHost, targetPort)

	// --- SSH Client Configuration ---
	authMethods := []ssh.AuthMethod{}

	// 1. Try Public Key Auth
	keyAuth, err := loadPrivateKey(sshKeyPath)
	if err == nil {
		authMethods = append(authMethods, keyAuth)
		logger.Printf("Using public key authentication: %s", sshKeyPath)
	} else {
		logger.Printf("Could not load private key %q: %v. Will attempt password auth.", sshKeyPath, err)
	}

	// 2. Add Password Auth (will prompt if key fails or no key given)
	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		fmt.Printf("Enter password for %s@%s: ", sshUser, targetHost)
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // Newline after password input
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		return string(bytePassword), nil
	}))

	// Host Key Verification
	var hostKeyCallback ssh.HostKeyCallback
	if insecureHostKey {
		logger.Println("WARNING: Host key verification is disabled!")
		hostKeyCallback = ssh.InsecureIgnoreHostKey() // DANGEROUS! Only for testing.
	} else {
		hostKeyCallback, err = createKnownHostsCallback(currentUser)
		if err != nil {
			// Be more specific if known_hosts can't be accessed
			log.Fatalf("Could not set up host key verification (check ~/.ssh/known_hosts permissions?): %v", err)
		}
		logger.Println("Using known_hosts for host key verification.")
	}

	sshConfig := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         15 * time.Second, // Connection timeout
	}

	// --- Establish SSH Connection via tsnet ---
	sshTargetAddr := net.JoinHostPort(targetHost, targetPort)

	// Dial using tsnet's Dial function - THIS IS THE KEY PART
	logger.Printf("Dialing %s via tsnet...", sshTargetAddr)
	conn, err := srv.Dial(ctx, "tcp", sshTargetAddr)
	if err != nil {
		log.Fatalf("Failed to dial %s via tsnet (is Tailscale connection up and host reachable?): %v", sshTargetAddr, err)
	}
	logger.Printf("tsnet Dial successful. Establishing SSH connection...")

	// Establish the SSH connection over the Tailscale tunnel
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, sshTargetAddr, sshConfig)
	if err != nil {
		// Check for specific auth errors
		if strings.Contains(err.Error(), "unable to authenticate") || strings.Contains(err.Error(), "no supported authentication methods") {
			log.Fatalf("SSH Authentication failed for user %s: %v", sshUser, err)
		}
		// Check for host key errors explicitly using errors.As
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) {
			// The callback already printed details and determined if it should fail.
			// We just need to make sure the exit status reflects the failure.
			log.Fatalf("SSH Host key verification failed: %v", err) // Log the specific key error
		}
		log.Fatalf("Failed to establish SSH connection to %s: %v", sshTargetAddr, err)
	}
	defer sshConn.Close()
	logger.Println("SSH connection established.")

	// Create an SSH client from the connection
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	// If a remote command was provided, run it non-interactively and exit
	if len(remoteCmd) > 0 {
		logger.Printf("Running remote command: %v", remoteCmd)
		session, err := client.NewSession()
		if err != nil {
			log.Fatalf("Failed to create SSH session for remote command: %v", err)
		}
		defer session.Close()
		session.Stdout = os.Stdout
		session.Stderr = os.Stderr
		session.Stdin = os.Stdin
		cmd := strings.Join(remoteCmd, " ")
		if err := session.Run(cmd); err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				os.Exit(exitErr.ExitStatus())
			}
			log.Fatalf("Remote command execution failed: %v", err)
		}
		os.Exit(0)
	}
	// --- Start Interactive SSH Session ---
	logger.Println("Starting interactive SSH session...")
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("Failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Set up terminal modes
	fd := int(os.Stdin.Fd())
	var oldState *term.State // Store old state pointer
	if term.IsTerminal(fd) { // Only make raw if it's actually a terminal
		oldState, err = term.MakeRaw(fd)
		if err != nil {
			log.Printf("Warning: Failed to set terminal to raw mode: %v. Session might not work correctly.", err)
		} else {
			// Restore terminal state on exit *if* we successfully made it raw
			defer term.Restore(fd, oldState)
		}
	} else {
		logger.Println("Input is not a terminal, proceeding without raw mode or PTY request.")
	}

	// Set up I/O with escape detection for interactive session
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to create stdin pipe for SSH session: %v", err)
	}
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Request pseudo-terminal (PTY) only if we have a terminal
	if term.IsTerminal(fd) {
		termWidth, termHeight, err := term.GetSize(fd)
		if err != nil {
			logger.Printf("Warning: Failed to get terminal size: %v. Using default 80x24.", err)
			termWidth = 80
			termHeight = 24
		}

		termType := os.Getenv("TERM")
		if termType == "" {
			termType = "xterm-256color" // A reasonable default
		}

		err = session.RequestPty(termType, termHeight, termWidth, ssh.TerminalModes{})
		if err != nil {
			log.Fatalf("Failed to request pseudo-terminal: %v", err)
		}

		// Handle window size changes only if we have a terminal and PTY
		go watchWindowSize(fd, session)
	}

	// Start the remote shell
	err = session.Shell()
	if err != nil {
		log.Fatalf("Failed to start remote shell: %v", err)
	}
	// Inform about escape sequence
	fmt.Fprintf(os.Stderr, "\nEscape sequence: ~. to terminate session\n")
	// Handle user input with escape (~.) detection
	go func() {
		reader := bufio.NewReader(os.Stdin)
		state := true // at start of line
		for {
			b, err := reader.ReadByte()
			if err != nil {
				return
			}
			if state && b == '~' {
				next, errPeek := reader.Peek(1)
				if errPeek == nil {
					if next[0] == '.' {
						// consume dot and exit
						reader.ReadByte()
						if oldState != nil && term.IsTerminal(fd) {
							term.Restore(fd, oldState)
						}
						os.Exit(0)
					} else if next[0] == '~' {
						// literal ~
						reader.ReadByte()
						stdinPipe.Write([]byte{'~'})
						state = false
						continue
					}
				}
			}
			stdinPipe.Write([]byte{b})
			if b == '\n' || b == '\r' {
				state = true
			} else {
				state = false
			}
		}
	}()
	// Wait for the session to finish
	err = session.Wait()

	// Cleanly restore terminal state *before* logging final messages or exiting
	// The defer above handles this, but doing it explicitly here is safe too.
	if oldState != nil && term.IsTerminal(fd) {
		term.Restore(fd, oldState) // Ensure terminal is restored
	}

	if err != nil {
		// Don't log fatal if it's just the remote command exiting non-zero
		if exitErr, ok := err.(*ssh.ExitError); ok {
			// logger.Printf("Remote command exited with status %d", exitErr.ExitStatus())
			// Exit silently with the same status code unless verbose
			if verbose {
				logger.Printf("Remote command exited with status %d", exitErr.ExitStatus())
			}
			os.Exit(exitErr.ExitStatus())
		}
		// Ignore "session closed" errors which are expected on normal exit
		// Also ignore EOF which can happen on clean disconnects
		if !errors.Is(err, io.EOF) && !strings.Contains(err.Error(), "session closed") && !strings.Contains(err.Error(), "channel closed") {
			log.Printf("SSH session ended with error: %v", err)
		}
	}
	logger.Println("SSH session closed.")
}

// parseTarget splits a target string like "[user@]hostname[:port]"
func parseTarget(target string) (host, port string, err error) {
	host = target
	port = defaultSSHPort

	// Check for port first, as hostname might contain '@'
	if strings.Contains(host, ":") {
		// Handle IPv6 addresses like [::1]:22
		if host[0] == '[' {
			endBracket := strings.Index(host, "]")
			if endBracket == -1 {
				return "", "", fmt.Errorf("mismatched brackets in IPv6 address")
			}
			maybePort := ""
			if len(host) > endBracket+1 {
				if host[endBracket+1] != ':' {
					return "", "", fmt.Errorf("expected ':' after ] in IPv6 address with port")
				}
				maybePort = host[endBracket+2:]
			}
			host = host[1:endBracket] // The IPv6 address itself
			if maybePort != "" {
				port = maybePort
			}
		} else {
			// Handle regular hostname:port or IPv4:port
			host, port, err = net.SplitHostPort(host)
			if err != nil {
				return "", "", fmt.Errorf("invalid host:port format: %w", err)
			}
		}
	}

	// User part is handled after flag parsing in main

	if host == "" {
		return "", "", errors.New("hostname cannot be empty")
	}

	return host, port, nil
}

// loadPrivateKey loads an SSH private key from the given path.
func loadPrivateKey(path string) (ssh.AuthMethod, error) {
	if path == "" {
		return nil, errors.New("private key path is empty")
	}
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading key file %q failed: %w", path, err)
	}

	// Try parsing without passphrase first
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err == nil {
		return ssh.PublicKeys(signer), nil
	}

	// If parsing failed, check if it's a passphrase error
	var passphraseErr *ssh.PassphraseMissingError
	if errors.As(err, &passphraseErr) {
		fmt.Printf("Enter passphrase for key %s: ", path)
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return nil, fmt.Errorf("failed to read passphrase: %w", err)
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, bytePassword)
		if err != nil {
			// Provide more context on failure
			if strings.Contains(err.Error(), "incorrect passphrase") || strings.Contains(err.Error(), "decryption failed") {
				return nil, fmt.Errorf("incorrect passphrase for key %q", path)
			}
			return nil, fmt.Errorf("parsing key %q with passphrase failed: %w", path, err)
		}
		return ssh.PublicKeys(signer), nil
	}

	// Other parsing error
	return nil, fmt.Errorf("parsing private key %q failed: %w", path, err)
}

// createKnownHostsCallback creates a HostKeyCallback using the user's known_hosts file.
func createKnownHostsCallback(currentUser *user.User) (ssh.HostKeyCallback, error) {
	if currentUser == nil || currentUser.HomeDir == "" {
		// Try getting home dir again as a fallback
		home, err := os.UserHomeDir()
		if err != nil {
			return ssh.InsecureIgnoreHostKey(), fmt.Errorf("cannot determine user home directory for known_hosts, disabling host key check: %w", err)
		}
		currentUser = &user.User{HomeDir: home} // Create a temporary user struct
		log.Printf("Warning: Could not get current user initially, found home dir %s. Proceeding with known_hosts.", home)
	}

	knownHostsPath := filepath.Join(currentUser.HomeDir, ".ssh", "known_hosts")

	// Ensure the .ssh directory exists
	sshDir := filepath.Dir(knownHostsPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return ssh.InsecureIgnoreHostKey(), fmt.Errorf("failed to create %s directory, disabling host key check: %w", sshDir, err)
	}

	// Create the file if it doesn't exist, as knownhosts.New might need it.
	f, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_RDONLY, 0600)
	if err != nil {
		// If we can't even create/open readonly, something is wrong.
		return ssh.InsecureIgnoreHostKey(), fmt.Errorf("unable to create/open %s, disabling host key check: %w", knownHostsPath, err)
	}
	f.Close() // Close immediately, knownhosts.New will reopen as needed

	// Use knownhosts.New to create the callback function
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return ssh.InsecureIgnoreHostKey(), fmt.Errorf("could not initialize known_hosts callback using %s, disabling host key check: %w", knownHostsPath, err)
	}

	// Wrap the callback to prompt the user if the host is unknown.
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Check against the actual known_hosts file using the callback from knownhosts.New
		err := hostKeyCallback(hostname, remote, key)
		if err == nil {
			return nil // Key is known and matches. Success!
		}

		// Use errors.As to check if it's a knownhosts.KeyError
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) {
			// *** CORRECTED LOGIC HERE ***
			if len(keyErr.Want) > 0 {
				// KEY MISMATCH / CHANGED
				// This means the host *was* in the file, but presented a different key.
				// This is the critical "MITM detected?" scenario.
				fmt.Fprintf(os.Stderr, "\n@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n")
				fmt.Fprintf(os.Stderr, "@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @\n")
				fmt.Fprintf(os.Stderr, "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n")
				fmt.Fprintf(os.Stderr, "IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!\n")
				fmt.Fprintf(os.Stderr, "Someone could be eavesdropping on you right now (man-in-the-middle attack)!\n")
				fmt.Fprintf(os.Stderr, "It is also possible that a host key has just been changed.\n")
				fmt.Fprintf(os.Stderr, "The fingerprint for the %s key sent by the remote host %s is:\n%s\n", key.Type(), remote.String(), ssh.FingerprintSHA256(key))
				fmt.Fprintf(os.Stderr, "Please contact your system administrator.\n")
				// keyErr.Want[0] contains details about the *expected* key found in the file
				fmt.Fprintf(os.Stderr, "Offending key for host %s found in %s:%d\n", hostname, keyErr.Want[0].Filename, keyErr.Want[0].Line)
				// Do not offer to add the key. Return the original error to prevent connection.
				return keyErr // Propagate the specific error
			} else {
				// HOST NOT FOUND (len(keyErr.Want) == 0)
				// Host is not in known_hosts, this is the first time connecting. Prompt user.
				fmt.Fprintf(os.Stderr, "The authenticity of host '%s (%s)' can't be established.\n", hostname, remote.String())
				fmt.Fprintf(os.Stderr, "%s key fingerprint is %s.\n", key.Type(), ssh.FingerprintSHA256(key))

				answer, readErr := promptUserViaTTY("Are you sure you want to continue connecting (yes/no)? ")
				if readErr != nil {
					// If we can't read the answer, safest is to deny connection.
					return fmt.Errorf("failed to read user confirmation: %w", readErr)
				}

				if answer == "yes" {
					// User accepted. Append the key to known_hosts.
					return appendKnownHost(knownHostsPath, hostname, remote, key) // Use helper func
				} else {
					// User declined.
					return errors.New("host key verification failed: user declined")
				}
			}
		}

		// If it wasn't a knownhosts.KeyError, it's some other unexpected error
		// (e.g., file permission issues that knownhosts.New didn't catch, etc.)
		return fmt.Errorf("unexpected error during host key verification: %w", err)
	}, nil
}

// appendKnownHost adds the given host key to the known_hosts file.
func appendKnownHost(knownHostsPath, hostname string, remote net.Addr, key ssh.PublicKey) error {
	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s to append new key: %w", knownHostsPath, err)
	}
	defer f.Close()

	// Use knownhosts.Normalize to create the address entry.
	// Using remote.String() includes the IP and Port as seen by the client.
	// Standard 'ssh' often uses the hostname provided by the user. Let's try that first,
	// but knownhosts.Normalize handles IP addresses correctly too.
	// We might want to add *both* hostname and normalized IP if they differ?
	// For simplicity, let's use the normalized remote address.
	normalizedAddress := knownhosts.Normalize(remote.String())
	// If hostname is different and not an IP, maybe add it too?
	// addresses := []string{normalizedAddress}
	// if host, _, err := net.SplitHostPort(normalizedAddress); err == nil && net.ParseIP(host) != nil {
	//   // If normalized is an IP, maybe add the original hostname too if it wasn't an IP
	//   if net.ParseIP(hostname) == nil && hostname != host {
	//       addresses = append(addresses, hostname)
	//   }
	// } else if hostname != normalizedAddress { // If normalized wasn't an IP host:port string
	//  addresses = append(addresses, hostname)
	// }
	// -> Sticking to just normalizedAddress for now for simplicity like standard ssh often does on first connect.

	line := knownhosts.Line([]string{normalizedAddress}, key)

	if _, err := fmt.Fprintln(f, line); err != nil {
		return fmt.Errorf("failed to write host key to %s: %w", knownHostsPath, err)
	}

	fmt.Fprintf(os.Stderr, "Warning: Permanently added '%s' (%s) to the list of known hosts.\n", normalizedAddress, key.Type())
	return nil // Key accepted and added successfully
}

// watchWindowSize monitors terminal size changes and informs the SSH session.
func watchWindowSize(fd int, session *ssh.Session) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH) // Notify on window change signal

	// Send initial size
	if term.IsTerminal(fd) {
		termWidth, termHeight, _ := term.GetSize(fd)
		if termWidth > 0 && termHeight > 0 {
			_ = session.WindowChange(termHeight, termWidth) // Ignore error, best effort
		}
	}

	// Update on signal
	for range sigCh {
		if term.IsTerminal(fd) {
			termWidth, termHeight, err := term.GetSize(fd)
			if err == nil && termWidth > 0 && termHeight > 0 {
				_ = session.WindowChange(termHeight, termWidth) // Ignore error, best effort
			}
		}
	}
}

// promptUserViaTTY reads a line directly from the controlling TTY,
// useful when stdin/stdout might be redirected (like during SSH).
func promptUserViaTTY(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt) // Print prompt to Stderr

	// Try opening /dev/tty for direct terminal interaction
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		// Fallback to Stdin if /dev/tty fails - this might not work if Stdin is already piped
		fmt.Fprint(os.Stderr, "(could not open /dev/tty, reading from stdin): ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin fallback: %w", err)
		}
		return strings.ToLower(strings.TrimSpace(line)), nil
	}
	defer tty.Close()

	reader := bufio.NewReader(tty)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read from tty: %w", err)
	}

	return strings.ToLower(strings.TrimSpace(line)), nil
}
