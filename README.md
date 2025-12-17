# evr-data-recorder

A comprehensive suite of tools for recording, converting, and replaying EchoVR game session and player bone data.

## Features

- **Agent**: Record session and player bone data from EchoVR game servers via HTTP API polling
- **API Server**: HTTP server for storing and retrieving session event data with MongoDB backend
- **Converter**: Convert between .echoreplay (zip) and .nevrcap (zstd compressed) file formats
- **Replayer**: HTTP server for replaying recorded session data

## Prerequisites

- Go 1.25 or later (for building from source)
- MongoDB (for API server functionality)

## Installation

### Download Pre-built Binaries

Download the latest release for your platform from the [Releases](https://github.com/EchoTools/evr-data-recorder/releases) page.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/EchoTools/evr-data-recorder.git
cd evr-data-recorder

# Build the consolidated binary
make build

# Or build for specific platforms
make linux    # Build for Linux
make windows  # Build for Windows
```

## Usage

The `evr-data-recorder` application provides a unified CLI with subcommands for different functionality.

```bash
# View available commands
evr-data-recorder --help

# Get help for a specific command
evr-data-recorder agent --help
```

### Agent - Record Game Data

Record session and player bone data from EchoVR game servers:

```bash
# Basic recording from localhost ports 6721-6730 at 30Hz
evr-data-recorder agent --frequency 30 --output ./output 127.0.0.1:6721-6730

# Record with streaming to Nakama server
evr-data-recorder agent --stream --stream-username myuser --stream-password mypass 127.0.0.1:6721

# Record with Events API enabled
evr-data-recorder agent --events --events-url http://localhost:8081 127.0.0.1:6721-6730
```

### API Server - Session Events API

Run an HTTP server for storing and retrieving session events:

```bash
# Start with default settings
evr-data-recorder apiserver

# Custom MongoDB URI and port
evr-data-recorder apiserver --mongo-uri mongodb://localhost:27017 --server-address :8081
```

### Converter - Format Conversion

Convert between replay file formats:

```bash
# Auto-detect conversion (echoreplay → nevrcap or vice versa)
evr-data-recorder converter --input game.echoreplay

# Specify output file
evr-data-recorder converter --input game.nevrcap --output converted.echoreplay

# Force specific format
evr-data-recorder converter --input game.echoreplay --format nevrcap
```

### Replayer - Replay Sessions

Replay recorded sessions via HTTP server:

```bash
# Replay a single file
evr-data-recorder replayer game.echoreplay

# Replay multiple files in sequence
evr-data-recorder replayer game1.echoreplay game2.echoreplay

# Loop playback continuously
evr-data-recorder replayer --loop game.echoreplay

# Custom bind address
evr-data-recorder replayer --bind 0.0.0.0:8080 game.echoreplay
```

## Configuration

The application supports multiple configuration methods (in order of precedence):

1. **Command-line flags** (highest priority)
2. **Environment variables** (prefix with `EVR_`)
3. **Configuration file** (YAML format)
4. **Default values** (lowest priority)

### Configuration File

Create a `evr-data-recorder.yaml` file in your working directory or specify with `--config`:

```yaml
# Global configuration
debug: false
log_level: info

# Agent configuration
agent:
  frequency: 10
  output_directory: ./output
  stream_enabled: false

# API Server configuration
apiserver:
  server_address: ":8081"
  mongo_uri: mongodb://localhost:27017
```

See [evr-data-recorder.yaml.example](evr-data-recorder.yaml.example) for a complete example.

### Environment Variables

All configuration can be set via environment variables with the `EVR_` prefix:

```bash
# Agent configuration
export EVR_AGENT_FREQUENCY=30
export EVR_AGENT_OUTPUT_DIRECTORY=./recordings

# Stream credentials
export EVR_AGENT_STREAM_USERNAME=myuser
export EVR_AGENT_STREAM_PASSWORD=mypassword

# Run the agent
evr-data-recorder agent 127.0.0.1:6721-6730
```

You can also use a `.env` file. See [.env.example](.env.example) for all available variables.

### Credential Management

Credentials (API keys, passwords, database URIs) can be managed securely:

- **Environment variables**: Set sensitive values as environment variables
- **.env file**: Store credentials in a `.env` file (never commit this file!)
- **Config file**: Use for non-sensitive configuration (can be committed)

## Development

### Building

```bash
# Build for current OS
make build

# Build all legacy individual commands
make legacy

# Run tests
make test

# Run benchmarks
make bench

# Clean build artifacts
make clean
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

## License

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
