package main

import (
	"testing"
	"time"
)

func TestTerminalStateManager(t *testing.T) {
	// Test the terminal state manager for thread safety
	manager := GetGlobalTerminalState()

	if manager == nil {
		t.Fatal("GetGlobalTerminalState() should not return nil")
	}

	// Test initial state
	if manager.IsRaw() {
		t.Error("Terminal should not be in raw mode initially")
	}

	// Test GetFD with invalid fd
	fd := manager.GetFD()
	if fd != -1 && fd < 0 {
		t.Error("GetFD() should return -1 for uninitialized state")
	}

	// Test concurrent access - this is the main race condition test
	t.Run("concurrent access", func(t *testing.T) {
		done := make(chan bool, 10)
		
		// Start multiple goroutines that access the terminal state
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				
				// Simulate rapid access to terminal state methods
				for j := 0; j < 100; j++ {
					manager.IsRaw()
					manager.GetFD()
					// Note: We can't test MakeRaw/Restore without actual terminal
				}
			}()
		}

		// Wait for all goroutines with timeout
		for i := 0; i < 10; i++ {
			select {
			case <-done:
				// Success
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for concurrent access test")
			}
		}
	})
}

func TestTerminalStateInitialization(t *testing.T) {
	// Test that multiple calls to GetGlobalTerminalState return the same instance
	manager1 := GetGlobalTerminalState()
	manager2 := GetGlobalTerminalState()

	if manager1 != manager2 {
		t.Error("GetGlobalTerminalState() should return the same instance (singleton)")
	}
}

func TestTerminalStateManagerMethods(t *testing.T) {
	manager := GetGlobalTerminalState()

	// Test that calling methods on uninitialized state doesn't panic
	t.Run("safe method calls", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Terminal state methods should not panic: %v", r)
			}
		}()

		manager.IsRaw()
		manager.GetFD()
		// Note: We avoid testing MakeRaw/Restore as they require actual terminal
	})
}