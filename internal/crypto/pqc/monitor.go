package pqc

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ConnectionMetrics tracks PQC usage metrics
type ConnectionMetrics struct {
	TotalConnections      int64
	QuantumSafeConnections int64
	HybridConnections     int64
	ClassicalConnections  int64
	FailedPQCAttempts     int64
	LastUpdated          time.Time
	AlgorithmUsage       map[string]int64
}

// Monitor provides PQC monitoring and reporting
type Monitor struct {
	mu      sync.RWMutex
	logger  *log.Logger
	metrics *ConnectionMetrics
	config  *Config
}

// NewMonitor creates a new PQC monitor
func NewMonitor(logger *log.Logger, config *Config) *Monitor {
	return &Monitor{
		logger:  logger,
		config:  config,
		metrics: &ConnectionMetrics{
			AlgorithmUsage: make(map[string]int64),
			LastUpdated:    time.Now(),
		},
	}
}

// RecordConnection records a new connection with its PQC status
func (m *Monitor) RecordConnection(status *Status) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.metrics.TotalConnections++
	m.metrics.LastUpdated = time.Now()
	
	if status.IsQuantumSafe {
		m.metrics.QuantumSafeConnections++
		if status.IsHybrid {
			m.metrics.HybridConnections++
		}
	} else {
		m.metrics.ClassicalConnections++
	}
	
	// Track algorithm usage
	if status.KeyExchangeAlgorithm != "" {
		m.metrics.AlgorithmUsage[status.KeyExchangeAlgorithm]++
	}
}

// RecordFailedPQCAttempt records when PQC negotiation fails
func (m *Monitor) RecordFailedPQCAttempt() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.metrics.FailedPQCAttempts++
	m.metrics.LastUpdated = time.Now()
}

// GetMetrics returns a copy of current metrics
func (m *Monitor) GetMetrics() ConnectionMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metricsCopy := *m.metrics
	metricsCopy.AlgorithmUsage = make(map[string]int64)
	for k, v := range m.metrics.AlgorithmUsage {
		metricsCopy.AlgorithmUsage[k] = v
	}
	
	return metricsCopy
}

// GenerateReport generates a human-readable PQC usage report
func (m *Monitor) GenerateReport() string {
	metrics := m.GetMetrics()
	
	if metrics.TotalConnections == 0 {
		return "No connections recorded yet"
	}
	
	report := fmt.Sprintf("=== Post-Quantum Cryptography Report ===\n")
	report += fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339))
	
	report += fmt.Sprintf("Total Connections: %d\n", metrics.TotalConnections)
	report += fmt.Sprintf("Quantum-Safe: %d (%.1f%%)\n", 
		metrics.QuantumSafeConnections,
		float64(metrics.QuantumSafeConnections)/float64(metrics.TotalConnections)*100)
	report += fmt.Sprintf("  - Hybrid Mode: %d (%.1f%%)\n",
		metrics.HybridConnections,
		float64(metrics.HybridConnections)/float64(metrics.TotalConnections)*100)
	report += fmt.Sprintf("Classical Only: %d (%.1f%%)\n",
		metrics.ClassicalConnections,
		float64(metrics.ClassicalConnections)/float64(metrics.TotalConnections)*100)
	
	if metrics.FailedPQCAttempts > 0 {
		report += fmt.Sprintf("\nFailed PQC Attempts: %d\n", metrics.FailedPQCAttempts)
	}
	
	if len(metrics.AlgorithmUsage) > 0 {
		report += "\nAlgorithm Usage:\n"
		for algo, count := range metrics.AlgorithmUsage {
			percentage := float64(count) / float64(metrics.TotalConnections) * 100
			quantumSafe := ""
			if IsPQCKeyExchange(algo) {
				quantumSafe = " [Quantum-Safe]"
			}
			report += fmt.Sprintf("  %s: %d (%.1f%%)%s\n", algo, count, percentage, quantumSafe)
		}
	}
	
	report += fmt.Sprintf("\nLast Updated: %s\n", metrics.LastUpdated.Format(time.RFC3339))
	
	return report
}

// LogConnectionSecurity logs the security status of a connection
func (m *Monitor) LogConnectionSecurity(host string, status *Status) {
	if !m.config.LogPQCUsage {
		return
	}
	
	icon := "🔒"
	level := "Quantum-Safe"
	
	if !status.IsQuantumSafe {
		icon = "⚠️"
		level = "Classical"
	}
	
	m.logger.Printf("%s PQC Connection to %s: %s (%s)", 
		icon, host, level, status.KeyExchangeAlgorithm)
	
	// Record the connection
	m.RecordConnection(status)
}

// CheckQuantumReadiness assesses if the system is quantum-ready
func (m *Monitor) CheckQuantumReadiness() (bool, string) {
	metrics := m.GetMetrics()
	
	if metrics.TotalConnections == 0 {
		return false, "No connections to assess"
	}
	
	quantumSafeRatio := float64(metrics.QuantumSafeConnections) / float64(metrics.TotalConnections)
	
	if quantumSafeRatio >= 0.9 {
		return true, fmt.Sprintf("Excellent: %.1f%% of connections are quantum-safe", quantumSafeRatio*100)
	} else if quantumSafeRatio >= 0.5 {
		return true, fmt.Sprintf("Good: %.1f%% of connections are quantum-safe", quantumSafeRatio*100)
	} else if quantumSafeRatio > 0 {
		return false, fmt.Sprintf("Needs improvement: Only %.1f%% of connections are quantum-safe", quantumSafeRatio*100)
	}
	
	return false, "Not quantum-ready: No quantum-safe connections established"
}

// RecommendUpgrade provides upgrade recommendations based on usage
func (m *Monitor) RecommendUpgrade() []string {
	metrics := m.GetMetrics()
	recommendations := []string{}
	
	if metrics.ClassicalConnections > 0 {
		classicalRatio := float64(metrics.ClassicalConnections) / float64(metrics.TotalConnections)
		if classicalRatio > 0.5 {
			recommendations = append(recommendations, 
				fmt.Sprintf("%.1f%% of connections use classical algorithms. Consider upgrading SSH servers to support PQC.", classicalRatio*100))
		}
	}
	
	if metrics.FailedPQCAttempts > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("%d PQC connection attempts failed. Check server compatibility with sntrup761x25519-sha512@openssh.com", metrics.FailedPQCAttempts))
	}
	
	// Check for specific algorithm usage
	for algo, count := range metrics.AlgorithmUsage {
		if !IsPQCKeyExchange(algo) && count > 0 {
			percentage := float64(count) / float64(metrics.TotalConnections) * 100
			if percentage > 20 {
				recommendations = append(recommendations,
					fmt.Sprintf("%.1f%% of connections use %s. This algorithm is not quantum-safe.", percentage, algo))
			}
		}
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System is well-configured for post-quantum security")
	}
	
	return recommendations
}