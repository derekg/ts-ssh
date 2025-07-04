package config

// Network and connection constants
const (
	// SSH defaults
	DefaultSSHPort = "22"

	// Terminal defaults
	DefaultTerminalWidth  = 80
	DefaultTerminalHeight = 24
	DefaultTerminalType   = "xterm-256color"

	// Application constants
	ClientName = "ts-ssh"

	// Timeout constants (in seconds)
	DefaultConnectionTimeout = 30
	DefaultCommandTimeout    = 300 // 5 minutes

	// File permission constants
	SecureFilePermissions      = 0600 // -rw-------
	SecureDirectoryPermissions = 0700 // drwx------

	// SSH key discovery priority
	PreferredKeyTypes = "ed25519,ecdsa,rsa"
)

// Modern SSH key types in order of preference
var ModernKeyTypes = []string{
	"id_ed25519", // Ed25519 - fastest, most secure, smallest key size
	"id_ecdsa",   // ECDSA - good performance, secure elliptic curve
	"id_rsa",     // RSA - legacy support, discouraged for new keys
}

// SSH configuration defaults
const (
	// Known hosts file management
	KnownHostsFileName = "known_hosts"
	SSHConfigDirName   = ".ssh"

	// SSH authentication timeouts
	SSHAuthTimeout      = 30 // seconds
	SSHConnectTimeout   = 15 // seconds
	SSHHandshakeTimeout = 10 // seconds
)

// Application behavior constants
const (
	// Tmux session management
	TmuxSessionPrefix = "ts-ssh"
	MaxHostnameLength = 50 // For temp file names

	// Power CLI constants
	MaxConcurrentConnections = 10
	DefaultBatchSize         = 5

	// Logging
	MaxLogFileSize = 10 * 1024 * 1024 // 10MB
	MaxLogFiles    = 5
)

// Version and build information (will be set by build process)
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)
