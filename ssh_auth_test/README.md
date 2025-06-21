# SSH Authentication Integration Test Suite

This directory contains comprehensive integration tests for SSH key authentication functionality in ts-ssh.

## Overview

The test suite verifies that SSH key authentication works correctly with authorized_keys, covering:

- SSH key generation (RSA 2048-bit)
- Protected and unprotected keys
- PEM encoding/decoding
- SSH client configuration
- Mock SSH server implementation
- End-to-end authentication flow

## Test Coverage

### TestSSHKeyGeneration
- ✅ Generates unprotected RSA SSH keys
- ✅ Generates passphrase-protected SSH keys  
- ✅ Verifies key files are created with correct permissions
- ✅ Tests key loading and validation
- ✅ Ensures passphrase-protected keys require authentication

### TestSSHKeyAuthentication
- ✅ Tests successful SSH authentication with valid keys
- ✅ Tests authentication failure with missing keys
- ✅ Verifies SSH session establishment
- ✅ Uses mock SSH server for isolated testing
- ✅ Validates end-to-end authentication flow

## Running Tests

### Quick Start
```bash
./run_tests.sh
```

### Individual Test Suites
```bash
# SSH key generation tests
go test -v -run 'TestSSHKeyGeneration' .

# SSH authentication tests  
go test -v -run 'TestSSHKeyAuthentication' .

# All tests
go test -v .

# Skip integration tests (short mode)
go test -v -short .
```

## Test Architecture

### Key Components

1. **SSH Key Generation**: Creates RSA keys in both protected and unprotected formats
2. **Mock SSH Server**: Implements a minimal SSH server for testing authentication
3. **Authentication Flow**: Tests the complete SSH handshake and session creation
4. **Key Loading**: Validates PEM parsing and SSH signer creation

### Dependencies

- `golang.org/x/crypto/ssh`: SSH protocol implementation
- Standard Go crypto libraries for RSA key generation
- Go testing framework

### Test Environment

- Creates temporary directories for test keys
- Generates unique key pairs for each test
- Uses loopback networking for mock server
- Cleans up all test artifacts

## Security Testing

The test suite validates several security aspects:

- ✅ Passphrase protection enforcement
- ✅ Key format validation (PEM)
- ✅ SSH protocol compliance
- ✅ Authentication method verification
- ✅ Secure key generation (2048-bit RSA)

## Integration with ts-ssh

This test suite is designed to validate the SSH authentication mechanisms used by ts-ssh:

1. **Key Loading**: Tests the same key loading logic used in production
2. **Authentication Methods**: Validates public key and password auth
3. **SSH Configuration**: Tests client configuration setup
4. **Session Management**: Verifies SSH session creation and management

## Continuous Integration

The test suite is designed to run in CI environments:

- No external dependencies required
- Uses mock servers instead of real SSH daemons
- Comprehensive error reporting
- Colored output for easy debugging
- Timeout protection for hanging tests

## Troubleshooting

### Common Issues

1. **Permission Errors**: Ensure temp directories are writable
2. **Network Errors**: Mock server uses loopback, should work in all environments
3. **Key Generation Failures**: Requires sufficient entropy for RSA key generation
4. **Timeout Issues**: Increase timeout values if running on slow systems

### Debugging

Enable verbose output for detailed test information:
```bash
go test -v -timeout 60s .
```

## Future Enhancements

Potential additions to the test suite:

- [ ] ECDSA key support testing
- [ ] Ed25519 key support testing
- [ ] SSH agent integration testing
- [ ] Multiple key format testing (OpenSSH, PKCS#8)
- [ ] SSH certificate authentication testing
- [ ] Real SSH server integration tests