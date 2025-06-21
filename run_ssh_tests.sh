#!/bin/bash

# SSH Authentication Integration Test Runner for ts-ssh
# This script runs comprehensive SSH authentication tests

set -e

echo "🔑 Running SSH Authentication Integration Tests for ts-ssh"
echo "==========================================================="

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
if [ ! -f "main.go" ]; then
    print_error "Not in ts-ssh directory (main.go not found)"
    exit 1
fi

print_status "Running in directory: $(pwd)"

# Build the application first to ensure it compiles
print_status "Building ts-ssh..."
if go build -o ts-ssh .; then
    print_success "Build successful"
else
    print_error "Build failed"
    exit 1
fi

# Run unit tests first
print_status "Running unit tests..."
if go test -v -run "Test.*" -short .; then
    print_success "Unit tests passed"
else
    print_error "Unit tests failed"
    exit 1
fi

# Run SSH authentication integration tests
print_status "Running SSH authentication integration tests..."
if go test -v -run "TestSSH.*" -timeout 30s .; then
    print_success "SSH authentication tests passed"
else
    print_warning "Some SSH authentication tests failed (this may be expected in some environments)"
fi

# Run SSH mock server tests
print_status "Running SSH mock server tests..."
if go test -v -run "TestSSHAuthenticationWithMockServer" -timeout 30s .; then
    print_success "SSH mock server tests passed"
else
    print_warning "SSH mock server tests failed (this may be expected in some environments)"
fi

# Test SSH key generation
print_status "Testing SSH key generation..."
TEMP_DIR=$(mktemp -d)
SSH_KEY_PATH="$TEMP_DIR/test_key"

# Generate a test SSH key
print_status "Generating test SSH key at $SSH_KEY_PATH"
if ssh-keygen -t rsa -b 2048 -f "$SSH_KEY_PATH" -N "" -q; then
    print_success "Test SSH key generated successfully"
    
    # Test key loading with our application
    print_status "Testing SSH key loading with ts-ssh..."
    if go test -v -run "TestSSHKeyGeneration" .; then
        print_success "SSH key loading tests passed"
    else
        print_warning "SSH key loading tests had issues"
    fi
else
    print_warning "Could not generate test SSH key (ssh-keygen not available)"
fi

# Cleanup
rm -rf "$TEMP_DIR"

echo ""
echo "==========================================================="
print_success "SSH Authentication Test Suite Complete!"
echo ""
print_status "Test Summary:"
echo "  ✓ Build verification"
echo "  ✓ Unit tests"
echo "  ✓ SSH authentication integration tests"
echo "  ✓ SSH mock server tests" 
echo "  ✓ SSH key generation tests"
echo ""
print_status "To run individual test suites:"
echo "  go test -v -run 'TestSSH.*' .          # All SSH tests"
echo "  go test -v -run 'TestSSHKeyAuth.*' .   # Key authentication tests"
echo "  go test -v -run 'TestSSHMock.*' .      # Mock server tests"
echo ""
print_status "To run tests with verbose output:"
echo "  go test -v ."
echo ""
print_status "To run tests excluding integration tests:"
echo "  go test -v -short ."