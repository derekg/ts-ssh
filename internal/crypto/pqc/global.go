package pqc

import (
	"log"
	"strings"
	"sync"
)

var (
	// globalMonitor is a singleton monitor instance for tracking PQC usage
	globalMonitor     *Monitor
	globalMonitorOnce sync.Once
	
	// globalSelector is a singleton algorithm selector
	globalSelector     *AlgorithmSelector
	globalSelectorOnce sync.Once
)

// GetGlobalMonitor returns the global PQC monitor instance
func GetGlobalMonitor(logger *log.Logger) *Monitor {
	globalMonitorOnce.Do(func() {
		config := DefaultConfig()
		globalMonitor = NewMonitor(logger, config)
	})
	return globalMonitor
}

// GetGlobalSelector returns the global algorithm selector
func GetGlobalSelector(config *Config, logger *log.Logger) *AlgorithmSelector {
	globalSelectorOnce.Do(func() {
		if config == nil {
			config = DefaultConfig()
		}
		globalSelector = NewAlgorithmSelector(config, logger)
	})
	return globalSelector
}

// LogConnectionStatus logs the PQC status of a connection using the global monitor
func LogConnectionStatus(host string, keyExchange string, logger *log.Logger) {
	monitor := GetGlobalMonitor(logger)
	selector := GetGlobalSelector(nil, logger)
	
	status := &Status{
		Enabled:              true,
		KeyExchangeAlgorithm: keyExchange,
		IsQuantumSafe:        IsPQCKeyExchange(keyExchange),
		IsHybrid:             false, // Will be updated based on algorithm
	}
	
	// Update hybrid status
	if algo, exists := selector.GetAlgorithmInfo(keyExchange); exists {
		status.IsHybrid = algo.QuantumSafe && strings.Contains(keyExchange, "x25519")
	}
	
	status.SecurityLevel = status.GetSecurityLevel()
	monitor.LogConnectionSecurity(host, status)
}

// GenerateGlobalReport generates a report from the global monitor
func GenerateGlobalReport(logger *log.Logger) string {
	monitor := GetGlobalMonitor(logger)
	return monitor.GenerateReport()
}

// CheckGlobalQuantumReadiness checks quantum readiness using global metrics
func CheckGlobalQuantumReadiness(logger *log.Logger) (bool, string) {
	monitor := GetGlobalMonitor(logger)
	return monitor.CheckQuantumReadiness()
}

// GetGlobalRecommendations provides upgrade recommendations
func GetGlobalRecommendations(logger *log.Logger) []string {
	monitor := GetGlobalMonitor(logger)
	return monitor.RecommendUpgrade()
}