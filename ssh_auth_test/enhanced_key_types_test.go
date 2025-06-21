package main

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

// TestEnhancedKeyTypesSupport tests all modern SSH key types
func TestEnhancedKeyTypesSupport(t *testing.T) {
	keyTypeTests := []struct {
		name        string
		keyType     string
		genFunc     func() (interface{}, ssh.PublicKey, error)
		description string
	}{
		{
			name:        "ed25519_key_support",
			keyType:     "Ed25519",
			genFunc:     generateEd25519KeyPair,
			description: "Ed25519 keys should be fully supported (most secure)",
		},
		{
			name:        "ecdsa_p256_key_support", 
			keyType:     "ECDSA",
			genFunc:     generateECDSAKeyPair,
			description: "ECDSA P-256 keys should be fully supported",
		},
		{
			name:        "rsa_2048_key_support",
			keyType:     "RSA",
			genFunc:     generateRSA2048KeyPair,
			description: "RSA 2048-bit keys should be supported (legacy)",
		},
		{
			name:        "rsa_4096_key_support",
			keyType:     "RSA",
			genFunc:     generateRSA4096KeyPair,
			description: "RSA 4096-bit keys should be supported (legacy but strong)",
		},
	}

	for _, tt := range keyTypeTests {
		t.Run(tt.name, func(t *testing.T) {
			testKeyTypeSupport(t, tt.keyType, tt.genFunc, tt.description)
		})
	}
}

// testKeyTypeSupport tests a specific key type through the complete flow
func testKeyTypeSupport(t *testing.T, keyType string, genFunc func() (interface{}, ssh.PublicKey, error), description string) {
	t.Logf("Testing: %s", description)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("key-type-test-%s-*", keyType))
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate key pair
	privateKey, publicKey, err := genFunc()
	if err != nil {
		t.Fatalf("Failed to generate %s key pair: %v", keyType, err)
	}

	// Write private key to file
	keyPath := filepath.Join(tempDir, "test_key")
	if err := writePrivateKeyToFile(privateKey, keyPath); err != nil {
		t.Fatalf("Failed to write %s private key: %v", keyType, err)
	}

	// Test key loading
	authMethod, err := loadPrivateKey(keyPath)
	if err != nil {
		t.Fatalf("Failed to load %s key: %v", keyType, err)
	}
	if authMethod == nil {
		t.Fatalf("%s auth method is nil", keyType)
	}

	// Test authentication with mock server
	testAuthenticationWithKeyType(t, publicKey, keyPath, keyType)

	t.Logf("✓ %s: %s key type fully supported", keyType, description)
}

// TestKeyFormatCompatibility tests different key formats and encodings
func TestKeyFormatCompatibility(t *testing.T) {
	formatTests := []struct {
		name         string
		keyType      string
		pemType      string
		marshalFunc  func(interface{}) ([]byte, error)
		description  string
	}{
		{
			name:        "rsa_pkcs1_format",
			keyType:     "RSA",
			pemType:     "RSA PRIVATE KEY",
			marshalFunc: marshalRSAPKCS1,
			description: "Traditional RSA PKCS#1 format should work",
		},
		{
			name:        "rsa_pkcs8_format",
			keyType:     "RSA", 
			pemType:     "PRIVATE KEY",
			marshalFunc: marshalPKCS8,
			description: "Modern PKCS#8 format should work for RSA",
		},
		{
			name:        "ed25519_pkcs8_format",
			keyType:     "Ed25519",
			pemType:     "PRIVATE KEY",
			marshalFunc: marshalPKCS8,
			description: "Ed25519 in PKCS#8 format should work",
		},
		{
			name:        "ecdsa_pkcs8_format",
			keyType:     "ECDSA",
			pemType:     "PRIVATE KEY", 
			marshalFunc: marshalPKCS8,
			description: "ECDSA in PKCS#8 format should work",
		},
	}

	for _, tt := range formatTests {
		t.Run(tt.name, func(t *testing.T) {
			testKeyFormat(t, tt.keyType, tt.pemType, tt.marshalFunc, tt.description)
		})
	}
}

// testKeyFormat tests a specific key format
func testKeyFormat(t *testing.T, keyType, pemType string, marshalFunc func(interface{}) ([]byte, error), description string) {
	t.Logf("Testing: %s", description)

	// Generate appropriate key
	var privateKey interface{}
	var publicKey ssh.PublicKey
	var err error

	switch keyType {
	case "RSA":
		privateKey, publicKey, err = generateRSA2048KeyPair()
	case "Ed25519":
		privateKey, publicKey, err = generateEd25519KeyPair()
	case "ECDSA":
		privateKey, publicKey, err = generateECDSAKeyPair()
	default:
		t.Fatalf("Unknown key type: %s", keyType)
	}

	if err != nil {
		t.Fatalf("Failed to generate %s key: %v", keyType, err)
	}

	// Marshal key in specified format
	keyBytes, err := marshalFunc(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal %s key: %v", keyType, err)
	}

	// Create PEM block
	pemBlock := &pem.Block{
		Type:  pemType,
		Bytes: keyBytes,
	}

	// Write to temporary file
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("format-test-%s-*", keyType))
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	keyPath := filepath.Join(tempDir, "test_key")
	pemData := pem.EncodeToMemory(pemBlock)
	if err := os.WriteFile(keyPath, pemData, 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Test loading
	authMethod, err := loadPrivateKey(keyPath)
	if err != nil {
		t.Fatalf("Failed to load %s key in %s format: %v", keyType, pemType, err)
	}
	if authMethod == nil {
		t.Fatalf("Auth method is nil for %s key", keyType)
	}

	// Test authentication
	testAuthenticationWithKeyType(t, publicKey, keyPath, fmt.Sprintf("%s-%s", keyType, pemType))

	t.Logf("✓ %s format support verified", description)
}

// TestKeyStrengthValidation tests different key strengths
func TestKeyStrengthValidation(t *testing.T) {
	strengthTests := []struct {
		name        string
		genFunc     func() (interface{}, ssh.PublicKey, error)
		expectValid bool
		description string
	}{
		{
			name:        "ed25519_always_strong",
			genFunc:     generateEd25519KeyPair,
			expectValid: true,
			description: "Ed25519 keys are always considered strong",
		},
		{
			name:        "rsa_2048_acceptable",
			genFunc:     generateRSA2048KeyPair,
			expectValid: true,
			description: "RSA 2048-bit should be acceptable",
		},
		{
			name:        "rsa_4096_strong",
			genFunc:     generateRSA4096KeyPair,
			expectValid: true,
			description: "RSA 4096-bit should be strong",
		},
		{
			name:        "ecdsa_p256_strong",
			genFunc:     generateECDSAKeyPair,
			expectValid: true,
			description: "ECDSA P-256 should be strong",
		},
	}

	for _, tt := range strengthTests {
		t.Run(tt.name, func(t *testing.T) {
			testKeyStrength(t, tt.genFunc, tt.expectValid, tt.description)
		})
	}
}

// testKeyStrength tests key strength validation
func testKeyStrength(t *testing.T, genFunc func() (interface{}, ssh.PublicKey, error), expectValid bool, description string) {
	t.Logf("Testing: %s", description)

	privateKey, publicKey, err := genFunc()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Test key can be used for authentication
	tempDir, err := os.MkdirTemp("", "strength-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	keyPath := filepath.Join(tempDir, "test_key")
	if err := writePrivateKeyToFile(privateKey, keyPath); err != nil {
		t.Fatalf("Failed to write key: %v", err)
	}

	authMethod, err := loadPrivateKey(keyPath)
	
	if expectValid {
		if err != nil {
			t.Errorf("Expected key to be valid, but got error: %v", err)
		}
		if authMethod == nil {
			t.Error("Expected valid auth method")
		}
		
		// Test actual authentication
		testAuthenticationWithKeyType(t, publicKey, keyPath, "strength-test")
		
		t.Logf("✓ %s - key strength validated", description)
	} else {
		if err == nil {
			t.Errorf("Expected key to be rejected due to insufficient strength")
		}
		t.Logf("✓ %s - weak key correctly rejected", description)
	}
}

// Helper functions for key generation

// generateEd25519KeyPair generates an Ed25519 key pair
func generateEd25519KeyPair() (interface{}, ssh.PublicKey, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, nil, err
	}

	return privKey, sshPubKey, nil
}

// generateECDSAKeyPair generates an ECDSA key pair using P-256 curve
func generateECDSAKeyPair() (interface{}, ssh.PublicKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	sshPubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, sshPubKey, nil
}

// generateRSA2048KeyPair generates a 2048-bit RSA key pair
func generateRSA2048KeyPair() (interface{}, ssh.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	sshPubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, sshPubKey, nil
}

// generateRSA4096KeyPair generates a 4096-bit RSA key pair
func generateRSA4096KeyPair() (interface{}, ssh.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	sshPubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, sshPubKey, nil
}

// Helper functions for key marshaling

// marshalRSAPKCS1 marshals RSA key in PKCS#1 format
func marshalRSAPKCS1(privateKey interface{}) ([]byte, error) {
	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA key")
	}
	return x509.MarshalPKCS1PrivateKey(rsaKey), nil
}

// marshalPKCS8 marshals any key in PKCS#8 format
func marshalPKCS8(privateKey interface{}) ([]byte, error) {
	return x509.MarshalPKCS8PrivateKey(privateKey)
}

// writePrivateKeyToFile writes a private key to file in appropriate format
func writePrivateKeyToFile(privateKey interface{}, filePath string) error {
	var keyBytes []byte
	var pemType string
	var err error

	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		keyBytes = x509.MarshalPKCS1PrivateKey(key)
		pemType = "RSA PRIVATE KEY"
	case ed25519.PrivateKey:
		keyBytes, err = x509.MarshalPKCS8PrivateKey(key)
		pemType = "PRIVATE KEY"
	case *ecdsa.PrivateKey:
		keyBytes, err = x509.MarshalPKCS8PrivateKey(key)
		pemType = "PRIVATE KEY"
	default:
		return fmt.Errorf("unsupported key type: %T", privateKey)
	}

	if err != nil {
		return err
	}

	pemBlock := &pem.Block{
		Type:  pemType,
		Bytes: keyBytes,
	}

	pemData := pem.EncodeToMemory(pemBlock)
	return os.WriteFile(filePath, pemData, 0600)
}

// testAuthenticationWithKeyType tests authentication with a specific key type
func testAuthenticationWithKeyType(t *testing.T, publicKey ssh.PublicKey, keyPath, keyType string) {
	// Start mock SSH server that accepts this public key
	serverAddr, cleanup := startMockSSHServer(t, publicKey)
	defer cleanup()

	// Test authentication
	testSSHConnection(t, serverAddr, keyPath, true)
	
	t.Logf("✓ Authentication successful with %s key", keyType)
}