# Migration Guide: Individual Commands to Consolidated CLI

This guide helps you migrate from the old individual command binaries to the new consolidated `evrtelemetry` CLI.

## What Changed?

Previously, you had separate binaries for each function:
- `agent` → `evrtelemetry stream`
- `apiserver` → `evrtelemetry serve`
- `converter` → `evrtelemetry convert`
- `replayer` → `evrtelemetry replay`

## Quick Migration

### Agent Command

**Old:**
```bash
./agent -debug -frequency 30 -log agent.log -output ./output 127.0.0.1:6721-6730
```

**New:**
```bash
./evrtelemetry stream --debug --frequency 30 --log-file agent.log --output ./output 127.0.0.1:6721-6730
```

### API Server Command

**Old:**
```bash
./apiserver
```

**New:**
```bash
./evrtelemetry serve
```

With environment variables:
```bash
export EVR_APISERVER_MONGO_URI=mongodb://localhost:27017
export EVR_APISERVER_SERVER_ADDRESS=:8081
./evrtelemetry serve
```

### Converter Command

**Old:**
```bash
./converter -input game.echoreplay -output game.nevrcap
```

**New:**
```bash
./evrtelemetry convert --input game.echoreplay --output game.nevrcap
```

### Replayer Command

**Old:**
```bash
./replayer -loop -bind 127.0.0.1:6721 game.echoreplay
```

**New:**
```bash
./evrtelemetry replay --loop --bind 127.0.0.1:6721 game.echoreplay
```

## New Features

### 1. Configuration Files

Create a `evrtelemetry.yaml` file:

```yaml
debug: true
log_level: info

agent:
  frequency: 30
  output_directory: ./recordings
  
apiserver:
  server_address: ":8081"
  mongo_uri: mongodb://localhost:27017
```

Then simply run:
```bash
./evrtelemetry stream -c evrtelemetry.yaml 127.0.0.1:6721-6730
```

### 2. Environment Variables

All configuration can be set via environment variables with `EVR_` prefix:

```bash
export EVR_AGENT_FREQUENCY=30
export EVR_AGENT_OUTPUT_DIRECTORY=./recordings
./evrtelemetry stream 127.0.0.1:6721-6730
```

### 3. .env File Support

Create a `.env` file for credentials:

```bash
EVR_AGENT_STREAM_USERNAME=myuser
EVR_AGENT_STREAM_PASSWORD=mypassword
EVR_APISERVER_MONGO_URI=mongodb://user:pass@localhost:27017
```

The application automatically loads `.env` files.

## Configuration Precedence

Configuration sources are applied in this order (highest to lowest priority):

1. Command-line flags
2. Environment variables
3. Configuration file (YAML)
4. Default values

## Credential Management

**Important Security Note:**

- Store sensitive credentials in `.env` file (never commit this!)
- Use environment variables for CI/CD
- Keep non-sensitive config in YAML files (safe to commit)

## Building Legacy Binaries

If you need the old individual binaries:

```bash
make legacy
```

This will build: `agent`, `apiserver`, `converter`, `replayer`

## Docker/Container Usage

Update your Dockerfiles:

**Old:**
```dockerfile
COPY agent /app/agent
CMD ["/app/agent", "-frequency", "30", "127.0.0.1:6721-6730"]
```

**New:**
```dockerfile
COPY evrtelemetry /app/evrtelemetry
COPY evrtelemetry.yaml /app/
CMD ["/app/evrtelemetry", "stream", "-c", "/app/evrtelemetry.yaml", "127.0.0.1:6721-6730"]
```

Or using environment variables:
```dockerfile
COPY evrtelemetry /app/evrtelemetry
ENV EVR_AGENT_FREQUENCY=30
ENV EVR_AGENT_OUTPUT_DIRECTORY=/data
CMD ["/app/evrtelemetry", "stream", "127.0.0.1:6721-6730"]
```

## Getting Help

View all available commands:
```bash
./evrtelemetry --help
```

Get help for a specific command:
```bash
./evrtelemetry stream --help
./evrtelemetry convert --help
```

## Troubleshooting

### Command not found

Make sure you're using the new binary name:
```bash
# Wrong
./agent --help

# Correct
./evrtelemetry stream --help
```

### Configuration not loading

Check the order of precedence. Command-line flags override everything:
```bash
# Config file says frequency=10, but this overrides it to 30
./evrtelemetry stream -c config.yaml --frequency 30 127.0.0.1:6721
```

### Environment variables not working

Ensure you're using the correct prefix:
```bash
# Wrong
export AGENT_FREQUENCY=30

# Correct
export EVR_AGENT_FREQUENCY=30
```

## Need More Help?

- Check the [README.md](README.md) for comprehensive documentation
- See [evrtelemetry.yaml.example](evrtelemetry.yaml.example) for config file examples
- See [.env.example](.env.example) for environment variable examples
