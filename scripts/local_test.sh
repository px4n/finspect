#!/bin/bash
# Local test script for finspect

set -e

echo "Running local tests for finspect..."
echo "=========================================="

# Run go fmt
echo "1. Checking code formatting..."
if ! go fmt ./... | grep -q .; then
    echo "✓ Code formatting OK"
else
    echo "✗ Code needs formatting (run: go fmt ./...)"
    exit 1
fi

# Run go vet
echo "2. Running go vet..."
if go vet ./...; then
    echo "✓ go vet passed"
else
    echo "✗ go vet failed"
    exit 1
fi

# Run golangci-lint
echo "3. Running golangci-lint..."
if golangci-lint run; then
    echo "✓ golangci-lint passed"
else
    echo "✗ golangci-lint failed"
    exit 1
fi

# Run gosec
echo "4. Running security scan with gosec..."
if command -v gosec &>/dev/null; then
    if gosec -fmt json -severity medium ./... 2>/dev/null | jq -e '.Issues | length == 0' >/dev/null; then
        echo "✓ gosec security scan passed"
    else
        echo "✗ gosec found security issues"
        gosec ./...
        exit 1
    fi
else
    echo "⚠ gosec not installed, skipping security scan"
fi

# Run tests
echo "5. Running unit tests..."
if go test -race ./...; then
    echo "✓ All tests passed"
else
    echo "✗ Tests failed"
    exit 1
fi

# Build
echo "6. Building project..."
if go build ./...; then
    echo "✓ Build successful"
else
    echo "✗ Build failed"
    exit 1
fi

echo ""
echo "=========================================="
echo "✓ All checks passed successfully!"
