.PHONY: all build test clean fmt lint deps

# Build variables
BINARY_NAME=finspect
BUILD_DIR=bin
GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_TEST=$(GO_CMD) test
GO_FMT=$(GO_CMD) fmt
GO_MOD=$(GO_CMD) mod
GO_VET=$(GO_CMD) vet

# Version information
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

all: clean deps fmt lint test build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/finspect/...

test:
	@echo "Running tests..."
	$(GO_TEST) -v -cover -race ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GO_TEST) -v -coverprofile=coverage.out ./...
	$(GO_CMD) tool cover -html=coverage.out -o coverage.html

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

fmt:
	@echo "Formatting code..."
	$(GO_FMT) ./...

lint:
	@echo "Running linter..."
	$(GO_VET) ./...

deps:
	@echo "Downloading dependencies..."
	$(GO_MOD) download
	$(GO_MOD) tidy

# Development helpers
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

watch:
	@echo "Watching for changes..."
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

# Installation
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/