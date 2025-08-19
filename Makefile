.PHONY: build test clean help converter

# Build targets
build: ## Build the main recorder application
	go build -o bin/evr-data-recorder .

converter: ## Build the echoreplay converter tool
	go build -o bin/echoreplay-converter ./cmd/echoreplay-converter

all: build converter ## Build all applications

# Test targets
test: ## Run all tests
	go test ./...

test-verbose: ## Run tests with verbose output
	go test -v ./...

test-converter: ## Run converter tests only
	go test -v ./converter

# Development targets
clean: ## Clean build artifacts
	rm -rf bin/
	go clean

deps: ## Download dependencies
	go mod download
	go mod tidy

# Utility targets
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Create bin directory
bin:
	mkdir -p bin

# Ensure bin directory exists for build targets
build converter: | bin