VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.0)
LDFLAGS = -X main.version=$(VERSION) -s -w

# Main consolidated binary
BINARY := evrtelemetry

# Legacy binaries under cmd/ (kept for compatibility)
LEGACY_CMDS := agent apiserver converter dumpevents replayer

# OS detection
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Windows-specific variables
WINDOWS_BINARY := $(BINARY).exe
WINDOWS_LEGACY_CMDS := $(addsuffix .exe,$(LEGACY_CMDS))

.PHONY: all version build windows linux legacy clean test bench

all: build

version:
	@echo $(VERSION)

# Build the main consolidated binary
build:
	@echo "Building $(BINARY) for $(GOOS)/$(GOARCH) (version=$(VERSION))"
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/evrtelemetry

# Build for Windows
windows:
	@echo "Building $(WINDOWS_BINARY) for windows/amd64 (version=$(VERSION))"
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(WINDOWS_BINARY) ./cmd/evrtelemetry

# Build for Linux
linux:
	@echo "Building $(BINARY) for linux/amd64 (version=$(VERSION))"
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/evrtelemetry

# Build legacy individual commands (for backward compatibility)
legacy: $(LEGACY_CMDS)

$(LEGACY_CMDS): %:
	@echo "Building legacy $* for $(GOOS)/$(GOARCH) (version=$(VERSION))"
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "$(LDFLAGS)" -o $* ./cmd/$*

bench:
	go test -bench=. -benchmem ./...

test:
	go test ./...

clean:
	rm -f $(BINARY) $(WINDOWS_BINARY) $(LEGACY_CMDS) $(WINDOWS_LEGACY_CMDS)
