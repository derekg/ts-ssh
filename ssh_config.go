package main

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// SSHConfigOptions holds SSH configuration options parsed from config file
type SSHConfigOptions struct {
	User            string
	IdentityFile    string
	HostKeyChecking string
	KnownHostsFile  string
}

// parseSSHConfig reads and parses an SSH config file for the specified host
// This is a minimal implementation focused on our security needs
func parseSSHConfig(configPath, hostname string) (*SSHConfigOptions, error) {
	if configPath == "" {
		return nil, fmt.Errorf("no SSH config file specified")
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SSH config file %s: %w", configPath, err)
	}
	defer file.Close()

	options := &SSHConfigOptions{}
	inHostSection := false
	matchesGlobal := true

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "host":
			// Check if this host section matches our target
			hostPattern := value
			inHostSection = (hostPattern == hostname || hostPattern == "*")
			continue
		}

		// Only process options if we're in a matching host section or global section
		if !inHostSection && !matchesGlobal {
			continue
		}

		switch key {
		case "user":
			if options.User == "" { // First match wins
				options.User = value
			}
		case "identityfile":
			if options.IdentityFile == "" {
				// Expand ~ to home directory
				if strings.HasPrefix(value, "~/") {
					currentUser, err := user.Current()
					if err == nil {
						value = filepath.Join(currentUser.HomeDir, value[2:])
					}
				}
				options.IdentityFile = value
			}
		case "stricthostkeychecking":
			if options.HostKeyChecking == "" {
				options.HostKeyChecking = strings.ToLower(value)
			}
		case "userknownhostsfile":
			if options.KnownHostsFile == "" {
				// Expand ~ to home directory
				if strings.HasPrefix(value, "~/") {
					currentUser, err := user.Current()
					if err == nil {
						value = filepath.Join(currentUser.HomeDir, value[2:])
					}
				}
				options.KnownHostsFile = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SSH config file: %w", err)
	}

	return options, nil
}

// applySSHConfigToConnection applies SSH config file options to connection parameters
func applySSHConfigToConnection(configFile, hostname string, sshUser, sshKeyPath *string, insecureHostKey *bool) error {
	if configFile == "" {
		return nil // No config file specified
	}

	options, err := parseSSHConfig(configFile, hostname)
	if err != nil {
		return err
	}

	// Apply config values if not already set via command line
	if *sshUser == "" && options.User != "" {
		*sshUser = options.User
	}

	if *sshKeyPath == "" && options.IdentityFile != "" {
		*sshKeyPath = options.IdentityFile
	}

	// Apply host key checking settings
	if options.HostKeyChecking == "no" {
		*insecureHostKey = true
	}

	return nil
}