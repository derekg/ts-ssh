package pqc

import (
	"log"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.EnablePQC {
		t.Error("Default config should have PQC enabled")
	}

	if config.QuantumResistance != QuantumResistanceHybrid {
		t.Errorf("Default quantum resistance should be Hybrid, got %d", config.QuantumResistance)
	}

	if !config.AllowClassicalFallback {
		t.Error("Default config should allow classical fallback")
	}

	if len(config.PreferredPQCAlgos) == 0 {
		t.Error("Default config should have preferred PQC algorithms")
	}
}

func TestIsPQCKeyExchange(t *testing.T) {
	tests := []struct {
		algo     string
		expected bool
	}{
		{"sntrup761x25519-sha512@openssh.com", true},
		{"mlkem768x25519-sha256", true},
		{"x25519-kyber768", true},
		{"curve25519-sha256@libssh.org", false},
		{"ecdh-sha2-nistp256", false},
		{"diffie-hellman-group14-sha256", false},
	}

	for _, tt := range tests {
		t.Run(tt.algo, func(t *testing.T) {
			result := IsPQCKeyExchange(tt.algo)
			if result != tt.expected {
				t.Errorf("IsPQCKeyExchange(%s) = %v, want %v", tt.algo, result, tt.expected)
			}
		})
	}
}

func TestIsQuantumResistantSignature(t *testing.T) {
	tests := []struct {
		algo     string
		expected bool
	}{
		{"ssh-ed25519", true},
		{"ssh-ed25519-cert-v01@openssh.com", true},
		{"rsa-sha2-512", true},
		{"rsa-sha2-256", true},
		{"ecdsa-sha2-nistp256", false},
		{"ssh-rsa", false},
	}

	for _, tt := range tests {
		t.Run(tt.algo, func(t *testing.T) {
			result := IsQuantumResistantSignature(tt.algo)
			if result != tt.expected {
				t.Errorf("IsQuantumResistantSignature(%s) = %v, want %v", tt.algo, result, tt.expected)
			}
		})
	}
}

func TestConfigureSSHConfig(t *testing.T) {
	tests := []struct {
		name      string
		pqcConfig *Config
		checkFunc func(t *testing.T, sshConfig *ssh.ClientConfig)
	}{
		{
			name: "PQC disabled",
			pqcConfig: &Config{
				EnablePQC: false,
			},
			checkFunc: func(t *testing.T, sshConfig *ssh.ClientConfig) {
				if len(sshConfig.KeyExchanges) != 0 {
					t.Error("KeyExchanges should be empty when PQC is disabled")
				}
			},
		},
		{
			name: "PQC hybrid mode",
			pqcConfig: &Config{
				EnablePQC:              true,
				QuantumResistance:      QuantumResistanceHybrid,
				AllowClassicalFallback: true,
				PreferredPQCAlgos: []string{
					"sntrup761x25519-sha512@openssh.com",
				},
			},
			checkFunc: func(t *testing.T, sshConfig *ssh.ClientConfig) {
				if len(sshConfig.KeyExchanges) == 0 {
					t.Error("KeyExchanges should not be empty in hybrid mode")
				}
				if sshConfig.KeyExchanges[0] != "sntrup761x25519-sha512@openssh.com" {
					t.Errorf("First key exchange should be PQC, got %s", sshConfig.KeyExchanges[0])
				}
			},
		},
		{
			name: "PQC strict mode",
			pqcConfig: &Config{
				EnablePQC:              true,
				QuantumResistance:      QuantumResistanceStrict,
				AllowClassicalFallback: false,
				PreferredPQCAlgos: []string{
					"sntrup761x25519-sha512@openssh.com",
				},
			},
			checkFunc: func(t *testing.T, sshConfig *ssh.ClientConfig) {
				if len(sshConfig.KeyExchanges) != 1 {
					t.Errorf("Strict mode should only have PQC algorithms, got %d", len(sshConfig.KeyExchanges))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sshConfig := &ssh.ClientConfig{}
			ConfigureSSHConfig(sshConfig, tt.pqcConfig)
			tt.checkFunc(t, sshConfig)
		})
	}
}

func TestAlgorithmSelector(t *testing.T) {
	logger := log.New(&strings.Builder{}, "", 0)
	config := &Config{
		EnablePQC:              true,
		QuantumResistance:      QuantumResistanceHybrid,
		AllowClassicalFallback: true,
		PreferredPQCAlgos: []string{
			"sntrup761x25519-sha512@openssh.com",
			"mlkem768x25519-sha256",
		},
	}

	selector := NewAlgorithmSelector(config, logger)

	t.Run("SelectKeyExchange with PQC support", func(t *testing.T) {
		serverAlgos := []string{
			"curve25519-sha256@libssh.org",
			"sntrup761x25519-sha512@openssh.com",
			"ecdh-sha2-nistp256",
		}

		selected, err := selector.SelectKeyExchange(serverAlgos)
		if err != nil {
			t.Fatalf("SelectKeyExchange failed: %v", err)
		}

		if selected != "sntrup761x25519-sha512@openssh.com" {
			t.Errorf("Expected PQC algorithm, got %s", selected)
		}

		status := selector.GetConnectionStatus()
		if !status.IsQuantumSafe {
			t.Error("Connection should be quantum-safe")
		}
	})

	t.Run("SelectKeyExchange without PQC support", func(t *testing.T) {
		serverAlgos := []string{
			"curve25519-sha256@libssh.org",
			"ecdh-sha2-nistp256",
		}

		selected, err := selector.SelectKeyExchange(serverAlgos)
		if err != nil {
			t.Fatalf("SelectKeyExchange failed: %v", err)
		}

		if selected != "curve25519-sha256@libssh.org" {
			t.Errorf("Expected classical algorithm, got %s", selected)
		}

		status := selector.GetConnectionStatus()
		if status.IsQuantumSafe {
			t.Error("Connection should not be quantum-safe")
		}
	})
}

func TestMonitor(t *testing.T) {
	logger := log.New(&strings.Builder{}, "", 0)
	config := DefaultConfig()
	monitor := NewMonitor(logger, config)

	t.Run("RecordConnection", func(t *testing.T) {
		// Record a quantum-safe connection
		status := &Status{
			Enabled:              true,
			KeyExchangeAlgorithm: "sntrup761x25519-sha512@openssh.com",
			IsQuantumSafe:        true,
			IsHybrid:             true,
		}
		monitor.RecordConnection(status)

		metrics := monitor.GetMetrics()
		if metrics.TotalConnections != 1 {
			t.Errorf("Expected 1 total connection, got %d", metrics.TotalConnections)
		}
		if metrics.QuantumSafeConnections != 1 {
			t.Errorf("Expected 1 quantum-safe connection, got %d", metrics.QuantumSafeConnections)
		}
		if metrics.HybridConnections != 1 {
			t.Errorf("Expected 1 hybrid connection, got %d", metrics.HybridConnections)
		}
	})

	t.Run("GenerateReport", func(t *testing.T) {
		report := monitor.GenerateReport()
		if !strings.Contains(report, "Post-Quantum Cryptography Report") {
			t.Error("Report should contain title")
		}
		if !strings.Contains(report, "Total Connections: 1") {
			t.Error("Report should show total connections")
		}
	})

	t.Run("CheckQuantumReadiness", func(t *testing.T) {
		ready, assessment := monitor.CheckQuantumReadiness()
		if !ready {
			t.Error("Should be quantum-ready with 100% quantum-safe connections")
		}
		if !strings.Contains(assessment, "100.0%") {
			t.Errorf("Assessment should mention 100%%, got: %s", assessment)
		}
	})
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{
			name: "Quantum-safe hybrid",
			status: Status{
				IsQuantumSafe: true,
				IsHybrid:      true,
			},
			expected: "Quantum-Safe (Hybrid)",
		},
		{
			name: "Quantum-safe pure",
			status: Status{
				IsQuantumSafe: true,
				IsHybrid:      false,
			},
			expected: "Quantum-Safe",
		},
		{
			name: "Classical only",
			status: Status{
				IsQuantumSafe: false,
				IsHybrid:      false,
			},
			expected: "Classical Only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.GetSecurityLevel()
			if result != tt.expected {
				t.Errorf("GetSecurityLevel() = %s, want %s", result, tt.expected)
			}
		})
	}
}
