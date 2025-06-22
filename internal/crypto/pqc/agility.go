package pqc

import (
	"fmt"
	"log"
	"strings"
)

// AlgorithmSelector provides algorithm agility for PQC migration
type AlgorithmSelector struct {
	config           *Config
	logger           *log.Logger
	supportedAlgos   map[string]*Algorithm
	connectionStatus *Status
}

// NewAlgorithmSelector creates a new algorithm selector
func NewAlgorithmSelector(config *Config, logger *log.Logger) *AlgorithmSelector {
	as := &AlgorithmSelector{
		config:         config,
		logger:         logger,
		supportedAlgos: initializeAlgorithms(),
		connectionStatus: &Status{
			Enabled: config.EnablePQC,
		},
	}
	return as
}

// initializeAlgorithms creates the algorithm database
func initializeAlgorithms() map[string]*Algorithm {
	return map[string]*Algorithm{
		// Post-Quantum Key Exchange Algorithms
		"sntrup761x25519-sha512@openssh.com": {
			Name:             "sntrup761x25519-sha512@openssh.com",
			Type:             "kex",
			QuantumSafe:      true,
			QuantumResistant: true,
			SecurityBits:     128, // Classical
			QuantumBits:      128, // Post-quantum
		},
		"mlkem768x25519-sha256": {
			Name:             "mlkem768x25519-sha256",
			Type:             "kex",
			QuantumSafe:      true,
			QuantumResistant: true,
			SecurityBits:     128,
			QuantumBits:      128,
		},
		
		// Classical Key Exchange Algorithms
		"curve25519-sha256@libssh.org": {
			Name:             "curve25519-sha256@libssh.org",
			Type:             "kex",
			QuantumSafe:      false,
			QuantumResistant: false,
			SecurityBits:     128,
			QuantumBits:      0,
		},
		"ecdh-sha2-nistp256": {
			Name:             "ecdh-sha2-nistp256",
			Type:             "kex",
			QuantumSafe:      false,
			QuantumResistant: false,
			SecurityBits:     128,
			QuantumBits:      0,
		},
		
		// Signature Algorithms
		"ssh-ed25519": {
			Name:             "ssh-ed25519",
			Type:             "hostkey",
			QuantumSafe:      true,  // Ed25519 signatures are quantum-resistant
			QuantumResistant: true,
			SecurityBits:     128,
			QuantumBits:      128,
		},
		"rsa-sha2-512": {
			Name:             "rsa-sha2-512",
			Type:             "hostkey",
			QuantumSafe:      false,
			QuantumResistant: true, // Partially resistant with larger keys
			SecurityBits:     128,
			QuantumBits:      64, // Degraded by quantum attacks
		},
		"ecdsa-sha2-nistp256": {
			Name:             "ecdsa-sha2-nistp256",
			Type:             "hostkey",
			QuantumSafe:      false,
			QuantumResistant: false,
			SecurityBits:     128,
			QuantumBits:      0,
		},
	}
}

// SelectKeyExchange selects the best key exchange algorithm based on server support
func (as *AlgorithmSelector) SelectKeyExchange(serverAlgos []string) (string, error) {
	as.logger.Printf("PQC: Selecting key exchange from server algorithms: %v", serverAlgos)
	
	// Build our preference list based on configuration
	var preferences []string
	
	switch as.config.QuantumResistance {
	case QuantumResistanceStrict:
		// Only PQC algorithms
		preferences = as.config.PreferredPQCAlgos
		
	case QuantumResistanceHybrid:
		// Prefer PQC, but allow classical fallback
		preferences = append(preferences, as.config.PreferredPQCAlgos...)
		if as.config.AllowClassicalFallback {
			preferences = append(preferences, 
				"curve25519-sha256@libssh.org",
				"ecdh-sha2-nistp256",
				"ecdh-sha2-nistp384",
				"ecdh-sha2-nistp521",
			)
		}
		
	case QuantumResistanceNone:
		// Classical algorithms only
		preferences = []string{
			"curve25519-sha256@libssh.org",
			"ecdh-sha2-nistp256",
			"ecdh-sha2-nistp384",
			"ecdh-sha2-nistp521",
		}
	}
	
	// Find the first matching algorithm
	for _, pref := range preferences {
		for _, server := range serverAlgos {
			if pref == server {
				as.updateConnectionStatus(pref)
				return pref, nil
			}
		}
	}
	
	// No match found
	if as.config.QuantumResistance == QuantumResistanceStrict {
		return "", fmt.Errorf("no quantum-safe algorithms supported by server")
	}
	
	return "", fmt.Errorf("no supported key exchange algorithms found")
}

// SelectHostKeyAlgorithm selects the best host key algorithm
func (as *AlgorithmSelector) SelectHostKeyAlgorithm(serverAlgos []string) (string, error) {
	as.logger.Printf("PQC: Selecting host key algorithm from: %v", serverAlgos)
	
	// Prefer quantum-resistant signatures
	preferences := []string{
		"ssh-ed25519",                       // Quantum-resistant
		"ssh-ed25519-cert-v01@openssh.com", // Quantum-resistant certificates
		"rsa-sha2-512",                      // Better resistance with larger keys
		"rsa-sha2-256",
	}
	
	// Add ECDSA only if classical fallback is allowed
	if as.config.AllowClassicalFallback {
		preferences = append(preferences,
			"ecdsa-sha2-nistp256",
			"ecdsa-sha2-nistp384",
			"ecdsa-sha2-nistp521",
		)
	}
	
	// Find the first matching algorithm
	for _, pref := range preferences {
		for _, server := range serverAlgos {
			if pref == server {
				return pref, nil
			}
		}
	}
	
	return "", fmt.Errorf("no supported host key algorithms found")
}

// updateConnectionStatus updates the connection status based on negotiated algorithm
func (as *AlgorithmSelector) updateConnectionStatus(kexAlgo string) {
	as.connectionStatus.KeyExchangeAlgorithm = kexAlgo
	
	if algo, exists := as.supportedAlgos[kexAlgo]; exists {
		as.connectionStatus.IsQuantumSafe = algo.QuantumSafe
		as.connectionStatus.IsHybrid = strings.Contains(kexAlgo, "x25519") && 
			(strings.Contains(kexAlgo, "sntrup") || strings.Contains(kexAlgo, "mlkem") || strings.Contains(kexAlgo, "kyber"))
	}
	
	as.connectionStatus.SecurityLevel = as.connectionStatus.GetSecurityLevel()
	
	// Log PQC status if enabled
	if as.config.LogPQCUsage {
		as.logPQCStatus()
	}
}

// logPQCStatus logs the current PQC status
func (as *AlgorithmSelector) logPQCStatus() {
	status := as.connectionStatus
	
	if status.IsQuantumSafe {
		as.logger.Printf("🔒 PQC: Quantum-safe connection established using %s", status.KeyExchangeAlgorithm)
		if status.IsHybrid {
			as.logger.Printf("🔒 PQC: Hybrid mode active (classical + post-quantum security)")
		}
	} else {
		as.logger.Printf("⚠️  PQC: Classical-only connection using %s", status.KeyExchangeAlgorithm)
		if as.config.QuantumResistance >= QuantumResistanceHybrid {
			as.logger.Printf("⚠️  PQC: Server does not support quantum-safe algorithms")
		}
	}
}

// GetConnectionStatus returns the current connection PQC status
func (as *AlgorithmSelector) GetConnectionStatus() *Status {
	return as.connectionStatus
}

// GetAlgorithmInfo returns information about a specific algorithm
func (as *AlgorithmSelector) GetAlgorithmInfo(name string) (*Algorithm, bool) {
	algo, exists := as.supportedAlgos[name]
	return algo, exists
}

// AssessSecurityLevel provides a detailed security assessment
func (as *AlgorithmSelector) AssessSecurityLevel() string {
	status := as.connectionStatus
	
	if !status.Enabled {
		return "Post-quantum cryptography is disabled"
	}
	
	algo, exists := as.supportedAlgos[status.KeyExchangeAlgorithm]
	if !exists {
		return fmt.Sprintf("Unknown algorithm: %s", status.KeyExchangeAlgorithm)
	}
	
	assessment := fmt.Sprintf("Algorithm: %s\n", algo.Name)
	assessment += fmt.Sprintf("Type: %s\n", algo.Type)
	assessment += fmt.Sprintf("Classical Security: %d bits\n", algo.SecurityBits)
	assessment += fmt.Sprintf("Quantum Security: %d bits\n", algo.QuantumBits)
	
	if algo.QuantumSafe {
		assessment += "Status: ✅ Quantum-Safe\n"
	} else if algo.QuantumResistant {
		assessment += "Status: ⚠️  Partially Quantum-Resistant\n"
	} else {
		assessment += "Status: ❌ Not Quantum-Safe\n"
	}
	
	return assessment
}