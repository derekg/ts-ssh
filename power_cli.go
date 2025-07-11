package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"sort"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"

	"github.com/derekg/ts-ssh/internal/client/scp"
	sshclient "github.com/derekg/ts-ssh/internal/client/ssh"
	"github.com/derekg/ts-ssh/internal/security"
)

// handleListHosts lists all available Tailscale hosts
func handleListHosts(status *ipnstate.Status, verbose bool) error {
	if status == nil || len(status.Peer) == 0 {
		fmt.Println(T("no_peers_found"))
		return nil
	}

	// Collect and sort hosts
	type hostInfo struct {
		name   string
		ip     string
		online bool
		os     string
	}

	var hosts []hostInfo
	for _, peer := range status.Peer {
		name := getHostDisplayName(peer)
		ip := ""
		if len(peer.TailscaleIPs) > 0 {
			ip = peer.TailscaleIPs[0].String()
		}

		hosts = append(hosts, hostInfo{
			name:   name,
			ip:     ip,
			online: peer.Online,
			os:     peer.OS,
		})
	}

	// Sort by name
	sort.Slice(hosts, func(i, j int) bool {
		return hosts[i].name < hosts[j].name
	})

	// Print results
	if verbose {
		// Get individual labels for the header
		labels := strings.Split(T("host_list_labels"), ",")
		separators := strings.Split(T("host_list_separator"), ",")

		// Default fallback if translation fails
		if len(labels) < 4 {
			labels = []string{"HOST", "IP", "STATUS", "OS"}
		}
		if len(separators) < 4 {
			separators = []string{"----", "--", "------", "--"}
		}

		fmt.Printf("%-25s %-15s %-8s %s\n", labels[0], labels[1], labels[2], labels[3])
		fmt.Printf("%-25s %-15s %-8s %s\n", separators[0], separators[1], separators[2], separators[3])

		for _, host := range hosts {
			status := T("status_offline")
			if host.online {
				status = T("status_online")
			}
			fmt.Printf("%-25s %-15s %-8s %s\n", host.name, host.ip, status, host.os)
		}
	} else {
		// Simple format - just online hosts
		for _, host := range hosts {
			if host.online {
				fmt.Println(host.name)
			}
		}
	}

	return nil
}

// handlePickHost provides simple interactive host selection
func handlePickHost(srv *tsnet.Server, ctx context.Context, status *ipnstate.Status, logger *log.Logger,
	sshUser, sshKeyPath string, insecureHostKey bool, currentUser *user.User, verbose bool) error {

	if status == nil || len(status.Peer) == 0 {
		return fmt.Errorf("%s", T("no_peers_found"))
	}

	// Collect online hosts
	var onlineHosts []string
	for _, peer := range status.Peer {
		if peer.Online {
			onlineHosts = append(onlineHosts, getHostDisplayName(peer))
		}
	}

	if len(onlineHosts) == 0 {
		return fmt.Errorf("%s", T("no_online_hosts"))
	}

	sort.Strings(onlineHosts)

	// Simple selection interface
	fmt.Println(T("available_hosts"))
	for i, host := range onlineHosts {
		fmt.Printf("  %d) %s\n", i+1, host)
	}
	fmt.Print(T("select_host", len(onlineHosts)))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	var selection int
	if _, err := fmt.Sscanf(strings.TrimSpace(input), "%d", &selection); err != nil {
		return fmt.Errorf("%s", T("invalid_selection"))
	}

	if selection < 1 || selection > len(onlineHosts) {
		return fmt.Errorf("%s", T("selection_out_of_range"))
	}

	selectedHost := onlineHosts[selection-1]
	fmt.Println(T("connecting_to", selectedHost))

	// Connect to selected host
	return sshclient.ConnectToHost(srv, ctx, logger, selectedHost, sshUser, sshKeyPath, insecureHostKey, currentUser, verbose)
}

// handleMultiHosts starts a tmux session with multiple hosts
func handleMultiHosts(multiHosts string, logger *log.Logger, sshUser, sshKeyPath string, insecureHostKey bool) error {
	hosts := strings.Split(multiHosts, ",")
	if len(hosts) == 0 {
		return fmt.Errorf("%s", T("no_hosts_specified"))
	}

	// Clean up host names
	for i, host := range hosts {
		hosts[i] = strings.TrimSpace(host)
	}

	if logger != nil {
		logger.Printf("Starting tmux session with hosts: %v", hosts)
	}

	tmuxManager := NewTmuxManager(logger, sshUser, sshKeyPath, insecureHostKey)

	// Ensure cleanup happens on exit
	defer tmuxManager.cleanupTempConfigFiles()

	return tmuxManager.StartMultiSession(hosts)
}

// handleExecCommand executes a command on multiple hosts
func handleExecCommand(srv *tsnet.Server, ctx context.Context, execCmd string, hosts []string,
	logger *log.Logger, sshUser, sshKeyPath string, insecureHostKey bool, parallel, verbose bool) error {

	if len(hosts) == 0 {
		return fmt.Errorf("%s", T("no_hosts_for_exec"))
	}

	if parallel {
		return executeParallel(srv, ctx, execCmd, hosts, logger, sshUser, sshKeyPath, insecureHostKey, verbose)
	} else {
		return executeSequential(srv, ctx, execCmd, hosts, logger, sshUser, sshKeyPath, insecureHostKey, verbose)
	}
}

// handleCopyFiles copies files to multiple hosts
func handleCopyFiles(srv *tsnet.Server, ctx context.Context, copyFiles string, logger *log.Logger,
	sshUser, sshKeyPath string, insecureHostKey bool, verbose bool) error {

	// Parse format: localfile host1,host2:/path/
	parts := strings.Split(copyFiles, " ")
	if len(parts) != 2 {
		return fmt.Errorf("%s", T("invalid_copy_format"))
	}

	localFile := parts[0]
	remoteSpec := parts[1]

	// Split remote spec into hosts and path
	if !strings.Contains(remoteSpec, ":") {
		return fmt.Errorf("%s", T("invalid_remote_spec"))
	}

	colonIdx := strings.LastIndex(remoteSpec, ":")
	hostsStr := remoteSpec[:colonIdx]
	remotePath := remoteSpec[colonIdx+1:]

	hosts := strings.Split(hostsStr, ",")
	for i, host := range hosts {
		hosts[i] = strings.TrimSpace(host)
	}

	// Copy to each host sequentially
	for _, host := range hosts {
		fmt.Println(T("copying_to", localFile, host, remotePath))

		// Use our existing SCP logic
		err := scp.HandleCliScp(srv, ctx, logger, sshUser, sshKeyPath, insecureHostKey, nil,
			localFile, remotePath, host, true, verbose)

		if err != nil {
			fmt.Println(T("copy_failed", host, err))
			continue
		}

		if verbose {
			fmt.Println(T("copy_success", host))
		}
	}

	return nil
}

// executeParallel runs commands on multiple hosts in parallel with race condition protection
func executeParallel(srv *tsnet.Server, ctx context.Context, execCmd string, hosts []string,
	logger *log.Logger, sshUser, sshKeyPath string, insecureHostKey bool, verbose bool) error {

	var wg sync.WaitGroup
	results := make(chan string, len(hosts))

	// Create a mutex to protect against concurrent password prompts
	var authMutex sync.Mutex

	for _, host := range hosts {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()

			// Create a host-specific logger to avoid concurrent access to shared logger
			hostLogger := log.New(logger.Writer(), fmt.Sprintf("[%s] ", h), logger.Flags())

			output, err := executeOnHostSafe(srv, ctx, execCmd, h, hostLogger, sshUser, sshKeyPath, insecureHostKey, verbose, &authMutex)
			if err != nil {
				results <- fmt.Sprintf("[%s] ERROR: %v", h, err)
			} else {
				results <- fmt.Sprintf("[%s]\n%s", h, output)
			}
		}(host)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Print results as they come in
	for result := range results {
		fmt.Printf("%s\n", result)
	}

	return nil
}

// executeSequential runs commands on hosts one by one
func executeSequential(srv *tsnet.Server, ctx context.Context, execCmd string, hosts []string,
	logger *log.Logger, sshUser, sshKeyPath string, insecureHostKey bool, verbose bool) error {

	for _, host := range hosts {
		fmt.Printf("=== %s ===\n", host)

		output, err := executeOnHost(srv, ctx, execCmd, host, logger, sshUser, sshKeyPath, insecureHostKey, verbose)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}

		fmt.Printf("%s\n", output)
	}

	return nil
}

// executeCommandOnHost executes a command on a remote host using SSH helpers
func executeCommandOnHost(srv *tsnet.Server, ctx context.Context, execCmd, host string,
	logger *log.Logger, sshUser, sshKeyPath string, insecureHostKey bool, authMutex *sync.Mutex) (string, error) {

	// SECURITY: Validate command and hostname to prevent injection attacks
	if err := security.ValidateCommand(execCmd); err != nil {
		return "", fmt.Errorf("command validation failed: %w", err)
	}

	if err := security.ValidateHostname(host); err != nil {
		return "", fmt.Errorf("hostname validation failed: %w", err)
	}

	// Parse target and user
	targetHost, targetPort, err := parseTarget(host, DefaultSshPort)
	if err != nil {
		return "", fmt.Errorf("error parsing target %s: %w", host, err)
	}

	effectiveUser := sshUser
	if strings.Contains(targetHost, "@") {
		parts := strings.SplitN(targetHost, "@", 2)
		effectiveUser = parts[0]
		targetHost = parts[1]

		// SECURITY: Validate extracted SSH user
		if err := security.ValidateSSHUser(effectiveUser); err != nil {
			return "", fmt.Errorf("SSH user validation failed: %w", err)
		}

		// SECURITY: Re-validate hostname after extraction
		if err := security.ValidateHostname(targetHost); err != nil {
			return "", fmt.Errorf("extracted hostname validation failed: %w", err)
		}
	}

	// SECURITY: Validate SSH user in all cases
	if err := security.ValidateSSHUser(effectiveUser); err != nil {
		return "", fmt.Errorf("SSH user validation failed: %w", err)
	}

	// SECURITY: Validate port
	if err := security.ValidatePort(targetPort); err != nil {
		return "", fmt.Errorf("port validation failed: %w", err)
	}

	// Create SSH config using standard helpers
	sshConfig := sshclient.SSHConnectionConfig{
		User:            effectiveUser,
		KeyPath:         sshKeyPath,
		TargetHost:      targetHost,
		TargetPort:      targetPort,
		InsecureHostKey: insecureHostKey,
		Verbose:         false,
		CurrentUser:     nil, // Not needed for command execution
		Logger:          logger,
	}

	// Override auth methods for thread-safe password prompts if needed
	if authMutex != nil {
		authMethods, err := createSSHAuthMethodsWithMutex(sshKeyPath, effectiveUser, targetHost, logger, authMutex)
		if err != nil {
			return "", fmt.Errorf("failed to create auth methods: %w", err)
		}

		// Create client config manually for custom auth
		clientConfig := &ssh.ClientConfig{
			User:            effectiveUser,
			Auth:            authMethods,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Simplified for parallel execution
			Timeout:         DefaultSSHTimeout,
		}

		return executeSSHCommandWithConfig(srv, ctx, clientConfig, targetHost, targetPort, execCmd)
	}

	// Use standard SSH helper for non-parallel execution
	client, err := sshclient.EstablishSSHConnection(srv, ctx, sshConfig)
	if err != nil {
		return "", fmt.Errorf("failed to establish SSH connection: %w", err)
	}
	defer client.Close()

	return executeSSHCommand(client, execCmd)
}

// createSSHAuthMethodsWithMutex creates auth methods with thread-safe password prompts
func createSSHAuthMethodsWithMutex(keyPath, user, targetHost string, logger *log.Logger, authMutex *sync.Mutex) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	// Try to load SSH key if provided
	if keyPath != "" {
		keyAuth, err := sshclient.LoadPrivateKey(keyPath, logger)
		if err == nil {
			authMethods = append(authMethods, keyAuth)
		}
	}

	// Add thread-safe password authentication using secure TTY
	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		authMutex.Lock()
		defer authMutex.Unlock()

		fmt.Print(T("enter_password", user, targetHost))
		password, err := security.ReadPasswordSecurely()
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("failed to read password securely: %w", err)
		}
		return password, nil
	}))

	return authMethods, nil
}

// executeSSHCommand runs a command on an established SSH connection
func executeSSHCommand(client *ssh.Client, execCmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(execCmd)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// executeSSHCommandWithConfig runs a command using low-level SSH connection
func executeSSHCommandWithConfig(srv *tsnet.Server, ctx context.Context, sshConfig *ssh.ClientConfig, targetHost, targetPort, execCmd string) (string, error) {
	sshTargetAddr := net.JoinHostPort(targetHost, targetPort)

	conn, err := srv.Dial(ctx, "tcp", sshTargetAddr)
	if err != nil {
		return "", fmt.Errorf("failed to dial %s via tsnet: %w", sshTargetAddr, err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, sshTargetAddr, sshConfig)
	if err != nil {
		conn.Close()
		return "", fmt.Errorf("failed to establish SSH connection: %w", err)
	}
	defer sshConn.Close()

	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	return executeSSHCommand(client, execCmd)
}

// executeOnHost executes a command on a single host and returns the output
func executeOnHost(srv *tsnet.Server, ctx context.Context, execCmd, host string,
	logger *log.Logger, sshUser, sshKeyPath string, insecureHostKey bool, verbose bool) (string, error) {

	return executeCommandOnHost(srv, ctx, execCmd, host, logger, sshUser, sshKeyPath, insecureHostKey, nil)
}

// executeOnHostSafe executes a command on a single host with thread-safe authentication
func executeOnHostSafe(srv *tsnet.Server, ctx context.Context, execCmd, host string,
	logger *log.Logger, sshUser, sshKeyPath string, insecureHostKey bool, verbose bool, authMutex *sync.Mutex) (string, error) {

	return executeCommandOnHost(srv, ctx, execCmd, host, logger, sshUser, sshKeyPath, insecureHostKey, authMutex)
}

// parseHostList parses comma-separated host list from args
func parseHostList(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	// Pre-calculate capacity to reduce allocations
	// Estimate based on comma count + number of args
	capacity := len(args)
	for _, arg := range args {
		capacity += strings.Count(arg, ",")
	}

	hosts := make([]string, 0, capacity)
	for _, arg := range args {
		if strings.Contains(arg, ",") {
			parts := strings.Split(arg, ",")
			// Trim spaces in place to avoid additional allocation
			for i, part := range parts {
				parts[i] = strings.TrimSpace(part)
			}
			hosts = append(hosts, parts...)
		} else {
			hosts = append(hosts, strings.TrimSpace(arg))
		}
	}

	return hosts
}

// getHostDisplayName extracts the best display name for a host
func getHostDisplayName(peer *ipnstate.PeerStatus) string {
	if peer.DNSName != "" {
		return strings.TrimSuffix(peer.DNSName, ".")
	}
	if peer.HostName != "" {
		return peer.HostName
	}
	if len(peer.TailscaleIPs) > 0 {
		return peer.TailscaleIPs[0].String()
	}
	return fmt.Sprintf("unknown-%s", peer.ID)
}
