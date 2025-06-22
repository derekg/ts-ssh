package pqc

import (
	"golang.org/x/crypto/ssh"
)

// QuantumResistanceLevel defines the level of quantum resistance required
type QuantumResistanceLevel int

const (
	// QuantumResistanceNone uses only classical algorithms
	QuantumResistanceNone QuantumResistanceLevel = iota
	// QuantumResistanceHybrid uses both classical and PQC algorithms
	QuantumResistanceHybrid
	// QuantumResistanceStrict requires PQC algorithms only (future)
	QuantumResistanceStrict
)

// Config holds post-quantum cryptography configuration
type Config struct {
	// EnablePQC enables post-quantum cryptography support
	EnablePQC bool

	// QuantumResistance sets the level of quantum resistance
	QuantumResistance QuantumResistanceLevel

	// PreferredPQCAlgos lists preferred PQC algorithms in order
	PreferredPQCAlgos []string

	// AllowClassicalFallback allows fallback to classical algorithms
	AllowClassicalFallback bool

	// LogPQCUsage enables logging of PQC algorithm usage
	LogPQCUsage bool
}

// DefaultConfig returns a default PQC configuration
func DefaultConfig() *Config {
	return &Config{
		EnablePQC:              true,
		QuantumResistance:      QuantumResistanceHybrid,
		AllowClassicalFallback: true,
		LogPQCUsage:           true,
		PreferredPQCAlgos: []string{
			"sntrup761x25519-sha512@openssh.com", // OpenSSH 9.0+ PQC
			"mlkem768x25519-sha256",              // NIST ML-KEM (future)
			"x25519-kyber768",                    // Hybrid approach (future)
		},
	}
}

// Algorithm represents a cryptographic algorithm with metadata
type Algorithm struct {
	Name             string
	Type             string // "kex", "hostkey", "cipher", "mac"
	QuantumSafe      bool
	QuantumResistant bool // Partially resistant (e.g., larger key sizes)
	SecurityBits     int  // Classical security bits
	QuantumBits      int  // Post-quantum security bits
}

// Status represents the current PQC status of a connection
type Status struct {
	// Enabled indicates if PQC is enabled for this connection
	Enabled bool

	// KeyExchangeAlgorithm is the negotiated key exchange algorithm
	KeyExchangeAlgorithm string

	// IsQuantumSafe indicates if the connection is quantum-safe
	IsQuantumSafe bool

	// IsHybrid indicates if hybrid (classical+PQC) mode is used
	IsHybrid bool

	// SecurityLevel provides a human-readable security assessment
	SecurityLevel string
}

// GetSecurityLevel returns a human-readable security level
func (s *Status) GetSecurityLevel() string {
	if s.IsQuantumSafe {
		if s.IsHybrid {
			return "Quantum-Safe (Hybrid)"
		}
		return "Quantum-Safe"
	}
	return "Classical Only"
}

// Supported PQC key exchange algorithms
var SupportedPQCKeyExchanges = []string{
	"sntrup761x25519-sha512@openssh.com", // NTRU + X25519 hybrid
	"mlkem768x25519-sha256",              // ML-KEM + X25519 hybrid
	"x25519-kyber768",                    // Kyber + X25519 hybrid
}

// Quantum-resistant signature algorithms (Ed25519 is quantum-resistant for signatures)
var QuantumResistantSignatures = []string{
	"ssh-ed25519",                           // 128-bit post-quantum security
	"ssh-ed25519-cert-v01@openssh.com",     // Ed25519 certificates
	"rsa-sha2-512",                          // Larger RSA for better resistance
	"rsa-sha2-256",                          // Larger RSA for better resistance
}

// IsPQCKeyExchange checks if an algorithm is a PQC key exchange
func IsPQCKeyExchange(algo string) bool {
	for _, pqc := range SupportedPQCKeyExchanges {
		if algo == pqc {
			return true
		}
	}
	return false
}

// IsQuantumResistantSignature checks if a signature algorithm has quantum resistance
func IsQuantumResistantSignature(algo string) bool {
	for _, sig := range QuantumResistantSignatures {
		if algo == sig {
			return true
		}
	}
	return false
}

// ConfigureSSHConfig adds PQC support to an SSH client config
func ConfigureSSHConfig(config *ssh.ClientConfig, pqcConfig *Config) {
	if !pqcConfig.EnablePQC {
		return
	}

	// Prepend PQC key exchanges to prefer them
	if pqcConfig.QuantumResistance >= QuantumResistanceHybrid {
		kexAlgos := make([]string, 0, len(pqcConfig.PreferredPQCAlgos)+len(config.KeyExchanges))
		
		// Add PQC algorithms first
		for _, algo := range pqcConfig.PreferredPQCAlgos {
			// Only add algorithms that OpenSSH currently supports
			if algo == "sntrup761x25519-sha512@openssh.com" {
				kexAlgos = append(kexAlgos, algo)
			}
		}
		
		// Add classical algorithms if fallback is allowed
		if pqcConfig.AllowClassicalFallback {
			kexAlgos = append(kexAlgos, config.KeyExchanges...)
		}
		
		config.KeyExchanges = kexAlgos
	}

	// Configure quantum-resistant host key algorithms
	if len(config.HostKeyAlgorithms) == 0 {
		config.HostKeyAlgorithms = QuantumResistantSignatures
	}
}