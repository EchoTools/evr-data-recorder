VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.0)
LDFLAGS = -X main.version=$(VERSION) -s -w

BINARY = datarecorder
BIN_DIR = .

.PHONY: all build build-linux build-windows test bench clean version replay-server virtex-exporter

all: build

version:
	@echo $(VERSION)

build: | $(BIN_DIR)
	@echo "Building $(BINARY) (version=$(VERSION))"
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

replay-server: | $(BIN_DIR)
	@echo "Building replay-server (version=$(VERSION))"
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/replay-server ./cmd/replayer

virtex-exporter: | $(BIN_DIR)
	@echo "Building virtex-exporter (version=$(VERSION))"
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/virtex-exporter ./cmd/virtex-exporter

build-linux: | $(BIN_DIR)
	@echo "Building linux/amd64 $(BINARY) (version=$(VERSION))"
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux .

build-windows: | $(BIN_DIR)
	@echo "Building windows/amd64 $(BINARY) (version=$(VERSION))"
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY).exe .

bench:
	go test -bench=. -benchmem ./...

test:
	go test ./...

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	rm -rf $(BIN_DIR)
