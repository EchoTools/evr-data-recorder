# NEVR Agent

NEVR Agent is a unified CLI for recording, streaming, converting, and replaying EchoVR game telemetry data.

## Features

- **Agent**: Capture session and player bone data from EchoVR game servers via HTTP API polling
- **API Server**: HTTP server for storing and retrieving session event data with MongoDB backend
- **Converter**: Convert between .echoreplay (zip) and .nevrcap (zstd compressed) file formats
- **Replayer**: HTTP server for replaying recorded session data
- **Dump Events**: Extract and display detected events from replay files
- **Migrate**: Run database schema migrations

## Prerequisites

- Go 1.25 or later (for building from source)
- MongoDB (for API server functionality)

## Installation

### Download Pre-built Binaries

Download the latest release for your platform from the [Releases](https://github.com/EchoTools/nevr-agent/releases) page.

### Use Container Image

The project is published to GitHub Container Registry (ghcr.io). Pull the latest image:

```bash
docker pull ghcr.io/echotools/nevr-agent:latest
```

Run the API server in a container with dependencies:

```bash
docker-compose up
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/EchoTools/nevr-agent.git
cd nevr-agent

# Build the binary
make build

# Or build for specific platforms
make linux    # Build for Linux
make windows  # Build for Windows
```

## Usage

The `agent` binary provides a unified CLI with subcommands for different functionality.

```bash
# View available commands
agent --help

# Get help for a specific command
agent agent --help
```

### Agent - Record Game Data

Record session and player bone data from EchoVR game servers:

```bash
# Basic recording from localhost ports 6721-6730 at 30Hz
agent agent --frequency 30 --output ./output 127.0.0.1:6721-6730

# Record with streaming enabled (requires JWT token)
agent agent --stream --token $JWT_TOKEN 127.0.0.1:6721

# Record with Events API enabled
agent agent --events --events-url http://localhost:8081 --token $JWT_TOKEN 127.0.0.1:6721-6730
```

### API Server - Session Events API

Run an HTTP server for storing and retrieving session events:

```bash
# Start with default settings
agent apiserver

# Custom MongoDB URI and port
agent apiserver --mongo-uri mongodb://localhost:27017 --server-address :8081
```

### Converter - Format Conversion

Convert between replay file formats:

```bash
# Auto-detect conversion (echoreplay → nevrcap or vice versa)
agent converter --input game.echoreplay

# Specify output file
agent converter --input game.nevrcap --output converted.echoreplay

# Force specific format
agent converter --input game.echoreplay --format nevrcap
```

### Replayer - Replay Sessions

Replay recorded sessions via HTTP server:

```bash
# Replay a single file
agent replayer game.echoreplay

# Replay multiple files in sequence
agent replayer game1.echoreplay game2.echoreplay

# Loop playback continuously
agent replayer --loop game.echoreplay

# Custom bind address
agent replayer --bind 0.0.0.0:8080 game.echoreplay
```

### Dump Events - Extract Events from Replay Files

Extract and display detected events from replay files:

```bash
# Output events as JSON (default)
agent dumpevents game.echoreplay

# Output as human-readable text
agent dumpevents game.nevrcap text

# Show event summary statistics
agent dumpevents game.echoreplay summary
```

### Migrate - Database Migrations

Run database schema migrations:

```bash
# Run migration with default MongoDB URI
agent migrate

# Run migration with custom MongoDB URI
agent migrate --mongo-uri mongodb://user:pass@localhost:27017/dbname
```

## Configuration

The application supports multiple configuration methods (in order of precedence):

1. **Command-line flags** (highest priority)
2. **Environment variables** (prefix with `NEVR_`)
3. **Configuration file** (YAML format)
4. **Default values** (lowest priority)

### Configuration File

Create an `agent.yaml` file in your working directory or specify with `--config`:

```yaml
# Global configuration
debug: false
log_level: info

# Agent configuration
agent:
  frequency: 10
  output_directory: ./output
  jwt_token: ""  # JWT token for API authentication
  stream_enabled: false

# API Server configuration
apiserver:
  server_address: ":8081"
  mongo_uri: mongodb://localhost:27017
```

### Environment Variables

All configuration can be set via environment variables with the `NEVR_` prefix:

```bash
# JWT token for API authentication
export NEVR_AGENT_JWT_TOKEN=your-jwt-token

# Agent configuration
export NEVR_AGENT_FREQUENCY=30
export NEVR_AGENT_OUTPUT_DIRECTORY=./recordings

# Run the agent
agent agent 127.0.0.1:6721-6730
```

You can also use a `.env` file. See [.env.example](.env.example) for all available variables.

## Development

### Building

```bash
# Build for current OS
make build

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
