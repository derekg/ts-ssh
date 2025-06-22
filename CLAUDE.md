# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ts-ssh is a Go-based SSH and SCP client that uses Tailscale's `tsnet` library to provide userspace connectivity to Tailscale networks without requiring a full Tailscale daemon. The project enables secure SSH connections and file transfers over a Tailnet with enterprise-grade security and comprehensive cross-platform support.

## Guidance Notes

- **Quality Score Tracking**: Do not store quality scores in any artifacts, including markdown files, code comments, commit messages, or pull request descriptions. Quality metrics, including security assessments, should be reported back to the project lead but not memorialized in project artifacts.

## Common Commands

### Build
```bash
go build -o ts-ssh .
```

### Run Tests
```bash
# Run all tests (unit + integration + security)
go test ./...

# Run specific test categories
go test ./... -run "Test.*[Ss]ecure"        # Security tests only
go test ./... -run "Test.*[Ii]ntegration"   # Integration tests only
go test ./... -run "Test.*[Aa]uth"          # Authentication tests only

# Run tests with verbose output
go test ./... -v

# Run security benchmarks
go test ./... -bench="Benchmark.*[Ss]ecure"
```

### Cross-compile Examples
```bash
# Windows AMD64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ts-ssh-windows.exe .

# macOS ARM64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ts-ssh-darwin-arm64 .

# Linux AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ts-ssh-linux-amd64 .
```

### Security Assessment
```bash
# Run comprehensive security test suite
go test ./... -run "Test.*[Ss]ecure" -v

# Validate cross-platform security features
GOOS=windows go test ./... -run "Test.*[Ss]ecure"
GOOS=darwin go test ./... -run "Test.*[Ss]ecure"

# Check for race conditions
go test ./... -race
```

### Run Application
```bash
./ts-ssh [user@]hostname[:port] [command...]
./ts-ssh -h  # for help
```

[... rest of the existing file content remains unchanged ...]