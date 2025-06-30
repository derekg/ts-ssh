package main

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestI18nInitialization(t *testing.T) {
	// Test that i18n can be initialized multiple times safely
	initI18n("")
	initI18n("en")
	initI18n("es")
	initI18n("invalid-lang")
}

func TestTranslationFunction(t *testing.T) {
	// Initialize with English
	initI18n("en")

	tests := []struct {
		key      string
		expectEmpty bool
	}{
		{"flag_lang_desc", false},
		{"no_peers_found", false},
		{"nonexistent_key", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := T(tt.key)
			isEmpty := result == "" || result == tt.key
			
			if tt.expectEmpty && !isEmpty {
				t.Errorf("T(%q) should return empty or key for nonexistent key, got %q", tt.key, result)
			}
			
			if !tt.expectEmpty && isEmpty {
				t.Errorf("T(%q) should return translation, got %q", tt.key, result)
			}
		})
	}
}

func TestTranslationWithArgs(t *testing.T) {
	initI18n("en")

	// Test translation with arguments
	result := T("connecting_to", "testhost")
	if result == "" || result == "connecting_to" {
		t.Errorf("T() with args should return formatted string, got %q", result)
	}

	// Should contain the argument
	if !strings.Contains(result, "testhost") {
		t.Errorf("T() result should contain argument 'testhost', got %q", result)
	}
}

func TestI18nConcurrentAccess(t *testing.T) {
	// This tests the race condition fix in i18n
	done := make(chan bool, 20)
	
	// Start multiple goroutines that access i18n functions concurrently
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			for j := 0; j < 100; j++ {
				initI18n("en")
				T("flag_lang_desc")
			}
		}()
	}

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			for j := 0; j < 100; j++ {
				initI18n("es")
				T("no_peers_found")
			}
		}()
	}

	// Wait for all goroutines with timeout
	for i := 0; i < 20; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent i18n access test")
		}
	}
}

func TestI18nLanguageSwitching(t *testing.T) {
	// Test switching between languages
	initI18n("en")
	englishResult := T("flag_lang_desc")

	initI18n("es")
	spanishResult := T("flag_lang_desc")

	// Results should be different (assuming we have Spanish translations)
	// If Spanish translations aren't available, they might be the same
	if englishResult == "" {
		t.Error("English translation should not be empty")
	}
	
	if spanishResult == "" {
		t.Error("Spanish translation should not be empty")
	}
}

func TestI18nThreadSafety(t *testing.T) {
	// Test for data races in i18n system
	var wg sync.WaitGroup
	numGoroutines := 50
	numOperations := 100

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				// Mix of operations to stress test the race condition fixes
				if j%3 == 0 {
					initI18n("en")
				} else if j%3 == 1 {
					initI18n("es")
				}
				
				// Use different translation keys
				keys := []string{"flag_lang_desc", "no_peers_found", "status_online", "status_offline"}
				key := keys[j%len(keys)]
				T(key)
			}
		}(i)
	}

	// Wait with timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(30 * time.Second):
		t.Fatal("Timeout waiting for i18n thread safety test")
	}
}

func TestI18nNewLanguages(t *testing.T) {
	// Test new language support
	testCases := []struct {
		lang string
		key  string
		shouldExist bool
	}{
		{"zh", "no_peers_found", true},
		{"hi", "no_peers_found", true},
		{"ar", "no_peers_found", true},
		{"bn", "no_peers_found", true},
		{"pt", "no_peers_found", true},
		{"ru", "no_peers_found", true},
		{"ja", "no_peers_found", true},
		{"de", "no_peers_found", true},
		{"fr", "no_peers_found", true},
		{"zh", "flag_lang_desc", true},
		{"de", "flag_lang_desc", true},
		{"fr", "flag_lang_desc", true},
	}

	for _, tc := range testCases {
		t.Run(tc.lang+"_"+tc.key, func(t *testing.T) {
			initI18n(tc.lang)
			result := T(tc.key)
			
			if tc.shouldExist {
				if result == tc.key {
					t.Errorf("Translation for key '%s' in language '%s' not found", tc.key, tc.lang)
				}
				// Verify it's different from English
				initI18n("en")
				english := T(tc.key)
				if result == english && tc.lang != "en" {
					t.Errorf("Translation for '%s' in '%s' is same as English", tc.key, tc.lang)
				}
			}
		})
	}
}

func TestI18nLanguageNormalization(t *testing.T) {
	// Test language normalization for new languages
	testCases := []struct {
		input    string
		expected string
	}{
		{"zh", "zh"},
		{"chinese", "zh"},
		{"中文", "zh"},
		{"zh-CN", "zh"},
		{"de", "de"},
		{"german", "de"},
		{"deutsch", "de"},
		{"de-DE", "de"},
		{"fr", "fr"},
		{"french", "fr"},
		{"français", "fr"},
		{"fr-FR", "fr"},
		{"pt", "pt"},
		{"portuguese", "pt"},
		{"pt-BR", "pt"},
		{"ru", "ru"},
		{"russian", "ru"},
		{"ja", "ja"},
		{"japanese", "ja"},
		{"hi", "hi"},
		{"hindi", "hi"},
		{"ar", "ar"},
		{"arabic", "ar"},
		{"bn", "bn"},
		{"bengali", "bn"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeLanguage(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeLanguage(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}