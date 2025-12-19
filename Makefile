VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.0)
LDFLAGS = -X main.version=$(VERSION) -s -w

# Main consolidated binary
BINARY := agent

# OS detection
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Windows-specific variables
WINDOWS_BINARY := $(BINARY).exe

.PHONY: all help version build windows linux clean test bench

# Default target - show help
.DEFAULT_GOAL := help

help:
	@echo "EVR Data Recorder - Available targets:"
	@echo ""
	@echo "  make help              Show this help message"
	@echo "  make all               Clean, test, and build for all platforms"
	@echo "  make build             Build for current OS/architecture ($(GOOS)/$(GOARCH))"
	@echo "  make windows           Build Windows binary (windows/amd64)"
	@echo "  make linux             Build Linux binary (linux/amd64)"
	@echo "  make test              Run tests"
	@echo "  make bench             Run benchmarks"
	@echo "  make clean             Remove built binaries"
	@echo "  make version           Display version"
	@echo ""

all: clean test build windows linux
	@echo "✓ All build targets completed successfully"

version:
	@echo $(VERSION)

# Build the main consolidated binary
build:
	@echo "Building $(BINARY) for $(GOOS)/$(GOARCH) (version=$(VERSION))"
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/agent

# Build for Windows
windows:
	@echo "Building $(WINDOWS_BINARY) for windows/amd64 (version=$(VERSION))"
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(WINDOWS_BINARY) ./cmd/agent

# Build for Linux
linux:
	@echo "Building $(BINARY) for linux/amd64 (version=$(VERSION))"
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/agent

bench:
	go test -bench=. -benchmem ./...

test:
	go test ./...

clean:
	rm -f $(BINARY) $(WINDOWS_BINARY)
