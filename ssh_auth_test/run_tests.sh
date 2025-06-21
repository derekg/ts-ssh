#!/bin/bash

# SSH Authentication Test Runner
# Comprehensive test suite for SSH key authentication functionality

set -e

echo "🔑 SSH Authentication Integration Test Suite"
echo "==========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is available
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

print_status "Go version: $(go version)"

# Check if we're in the right directory
if [ ! -f "ssh_auth_test.go" ]; then
    print_error "Not in ssh_auth_test directory (ssh_auth_test.go not found)"
    exit 1
fi

print_status "Running in directory: $(pwd)"

# Initialize dependencies
print_status "Initializing Go module dependencies..."
if go mod tidy; then
    print_success "Dependencies initialized"
else
    print_error "Failed to initialize dependencies"
    exit 1
fi

# Run SSH key generation tests
print_status "Running SSH key generation tests..."
if go test -v -run "TestSSHKeyGeneration" -timeout 30s .; then
    print_success "SSH key generation tests passed"
else
    print_error "SSH key generation tests failed"
    exit 1
fi

# Run SSH authentication integration tests
print_status "Running SSH authentication integration tests..."
if go test -v -run "TestSSHKeyAuthentication" -timeout 30s .; then
    print_success "SSH authentication integration tests passed"
else
    print_error "SSH authentication integration tests failed"
    exit 1
fi

# Run modern key discovery integration tests
print_status "Running modern key discovery integration tests..."
if go test -v -run "TestModernKeyDiscoveryIntegration" -timeout 60s .; then
    print_success "Modern key discovery integration tests passed"
else
    print_error "Modern key discovery integration tests failed"
    exit 1
fi

# Run Ed25519 specific integration tests
print_status "Running Ed25519 specific integration tests..."
if go test -v -run "TestEd25519SpecificIntegration" -timeout 30s .; then
    print_success "Ed25519 specific tests passed"
else
    print_error "Ed25519 specific tests failed"
    exit 1
fi

# Run enhanced key types support tests
print_status "Running enhanced key types support tests..."
if go test -v -run "TestEnhancedKeyTypesSupport" -timeout 60s .; then
    print_success "Enhanced key types support tests passed"
else
    print_error "Enhanced key types support tests failed"
    exit 1
fi

# Run key format compatibility tests
print_status "Running key format compatibility tests..."
if go test -v -run "TestKeyFormatCompatibility" -timeout 30s .; then
    print_success "Key format compatibility tests passed"
else
    print_error "Key format compatibility tests failed"
    exit 1
fi

# Run security validation integration tests
print_status "Running security validation integration tests..."
if go test -v -run "TestSecurityValidationIntegration" -timeout 30s .; then
    print_success "Security validation integration tests passed"
else
    print_error "Security validation integration tests failed"
    exit 1
fi

# Run backward compatibility tests
print_status "Running backward compatibility tests..."
if go test -v -run "TestBackwardCompatibilityIntegration" -timeout 30s .; then
    print_success "Backward compatibility tests passed"
else
    print_error "Backward compatibility tests failed"
    exit 1
fi

# Run all tests together
print_status "Running complete test suite..."
if go test -v -timeout 60s .; then
    print_success "Complete test suite passed"
else
    print_error "Some tests in the complete suite failed"
    exit 1
fi

echo ""
echo "==========================================="
print_success "SSH Authentication Test Suite Complete!"
echo ""
print_status "Test Summary:"
echo "  ✓ SSH key generation (protected and unprotected)"
echo "  ✓ SSH key loading and validation"
echo "  ✓ SSH authentication with mock server"
echo "  ✓ SSH session establishment"
echo "  ✓ End-to-end authentication flow"
echo "  ✓ Modern key discovery (Ed25519, ECDSA, RSA priority)"
echo "  ✓ Ed25519 specific functionality"
echo "  ✓ Enhanced key types support"
echo "  ✓ Key format compatibility (PKCS#1, PKCS#8)"
echo "  ✓ Security validation integration"
echo "  ✓ Backward compatibility with legacy setups"
echo ""
print_status "Coverage includes:"
echo "  - Ed25519 key generation and authentication (modern)"
echo "  - ECDSA key generation and authentication (P-256)"
echo "  - RSA key generation (2048-bit, 4096-bit, legacy)"
echo "  - Key discovery priority (Ed25519 > ECDSA > RSA)"
echo "  - PEM encoding/decoding (PKCS#1, PKCS#8)"
echo "  - Passphrase protection"
echo "  - SSH public key creation for all types"
echo "  - SSH client configuration with auto-discovery"
echo "  - Mock SSH server implementation"
echo "  - Authentication method testing"
echo "  - Security permission validation"
echo "  - Legacy compatibility testing"
echo ""
print_status "To run individual test suites:"
echo "  go test -v -run 'TestSSHKeyGeneration' .              # Basic key generation tests"
echo "  go test -v -run 'TestSSHKeyAuthentication' .          # Basic authentication tests"
echo "  go test -v -run 'TestModernKeyDiscoveryIntegration' . # Modern key discovery tests"
echo "  go test -v -run 'TestEd25519SpecificIntegration' .    # Ed25519 specific tests"
echo "  go test -v -run 'TestEnhancedKeyTypesSupport' .       # All key types support tests"
echo "  go test -v -run 'TestKeyFormatCompatibility' .        # Key format tests"
echo "  go test -v -run 'TestSecurityValidationIntegration' . # Security validation tests"
echo "  go test -v -run 'TestBackwardCompatibilityIntegration' . # Legacy compatibility tests"
echo ""
print_status "To run by category:"
echo "  go test -v -run 'Integration' .    # All integration tests"
echo "  go test -v -run 'Ed25519' .        # All Ed25519 tests"
echo "  go test -v -run 'Security' .       # All security tests"
echo "  go test -v -run 'Discovery' .      # All key discovery tests"
echo ""
print_status "To run with verbose output:"
echo "  go test -v ."
echo ""
print_status "To run in short mode (skip integration tests):"
echo "  go test -v -short ."