# GitHub Copilot Instructions for nevr-agent

## Project Overview

nevr-agent is a Go CLI application for recording, converting, and replaying EchoVR game session telemetry data. It uses a unified binary (`agent`) with git-style subcommands.

## Commit Strategy

**IMPORTANT**: Always break changes into many small, focused commits. Each commit should:

- Address a single concern or change
- Have a clear, descriptive commit message
- Be independently reviewable

The maintainer will **always squash merge** pull requests, so many commits help with:
- Easier code review
- Better change tracking during development
- Ability to revert specific changes if needed

Example commit breakdown for a feature:
1. "Add new config field for feature X"
2. "Implement core logic for feature X"
3. "Add CLI flags for feature X"
4. "Add tests for feature X"
5. "Update documentation for feature X"

## Code Style & Conventions

### Go Conventions

- Use `go fmt` for formatting (run `make lint` before committing)
- Follow standard Go naming conventions (camelCase for private, PascalCase for exported)
- Use meaningful variable names; avoid single-letter names except in short loops
- Prefer early returns to reduce nesting
- Always handle errors explicitly; don't ignore them with `_`

### Project Structure

```
cmd/agent/           # CLI commands (cobra-based)
internal/agent/      # Core agent logic (writers, pollers)
internal/api/        # HTTP API server implementation
  ├── service.go         # Main API service
  ├── storage_manager.go # Nevrcap file storage with retention
  ├── match_retrieval.go # Match download API
  ├── metrics.go         # Prometheus metrics
  ├── player_lookup.go   # Player info lookup with caching
  └── stream_api.go      # Real-time WebSocket streaming
internal/config/     # Configuration loading and validation
internal/amqp/       # AMQP/RabbitMQ integration
```

### CLI Commands (Cobra)

When adding new commands:

```go
func newMyCommand() *cobra.Command {
    var (
        // Define local flag variables here, not package-level
        myFlag string
    )

    cmd := &cobra.Command{
        Use:   "mycommand [flags] <required-arg>",  // flags before args
        Short: "Brief description",
        Long:  `Longer description with details.`,
        Example: `  # Example usage
  agent mycommand --flag value arg`,
        Args: cobra.MinimumNArgs(1),  // Use cobra's arg validation
        RunE: func(cmd *cobra.Command, args []string) error {
            return runMyCommand(cmd, args, myFlag)
        },
    }

    // Use local variables with *Var functions, not viper.BindPFlags
    cmd.Flags().StringVar(&myFlag, "flag", "default", "Flag description")

    return cmd
}
```

**Important**: Use local flag variables (`cmd.Flags().StringVar(&localVar, ...)`) instead of `viper.BindPFlags()` to avoid conflicts between subcommands.

### Configuration

- Configuration is loaded via `internal/config/config.go`
- Support hierarchy: CLI flags > environment variables > config file > defaults
- Environment variables use `NEVR_` prefix
- Add new config fields to appropriate struct in `config.go` with proper tags:

```go
type MyConfig struct {
    Field string `yaml:"field" mapstructure:"field"`
}
```

### Logging

Use `go.uber.org/zap` for structured logging:

```go
logger.Info("Operation completed",
    zap.String("key", value),
    zap.Int("count", count),
    zap.Error(err))

logger.Debug("Verbose info")  // Only shown with --debug flag
logger.Warn("Warning message")
logger.Error("Error occurred", zap.Error(err))
```

### Interfaces

The project uses interfaces for flexibility:

```go
// FrameWriter is implemented by all output destinations
type FrameWriter interface {
    Context() context.Context
    WriteFrame(*telemetry.LobbySessionStateFrame) error
    Close()
    IsStopped() bool
}
```

When adding new writers, implement the `FrameWriter` interface.

### Testing

- Write tests in `*_test.go` files alongside the code
- Use table-driven tests for multiple cases
- Smoke tests in `cmd/agent/smoke_test.go` test CLI behavior
- Run tests with `make test` or `go test ./...`
- Run smoke tests with `make smoke-test`

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case1", "input1", "expected1"},
        {"case2", "input2", "expected2"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := MyFunction(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Pre-commit Hooks

A pre-commit hook runs automatically before commits. To install:

```bash
make install-hooks
```

The hook runs:
1. `gofmt` check
2. `go vet`
3. Smoke tests
4. Unit tests

### Dependencies

- **cobra**: CLI framework
- **viper**: Configuration management (use sparingly, prefer local flag vars)
- **zap**: Structured logging
- **protobuf**: Data serialization (telemetry data)
- **gorilla/mux**: HTTP routing
- **gorilla/websocket**: WebSocket connections
- **mongo-driver**: MongoDB client
- **prometheus/client_golang**: Prometheus metrics
- **schollz/progressbar**: Terminal progress bars

### Related Repositories

- `nevr-common`: Shared protobuf definitions and types
- `nevrcap`: File codec implementations (.echoreplay, .nevrcap formats)

These are linked via `go.work` for local development.

## Common Tasks

### Adding a New Subcommand

1. Create `cmd/agent/mycommand.go`
2. Implement `newMyCommand() *cobra.Command`
3. Add to `rootCmd.AddCommand()` in `main.go`
4. Add config struct if needed in `internal/config/config.go`
5. Add tests in `cmd/agent/mycommand_test.go`

### Adding a New Writer

1. Create `internal/agent/writer_mytype.go`
2. Implement `FrameWriter` interface
3. Add factory function `NewMyTypeWriter(...)`
4. Integrate in `cmd/agent/agent.go` writer creation logic

### Adding New Configuration

1. Add field to appropriate struct in `internal/config/config.go`
2. Add default value in `DefaultConfig()`
3. Add CLI flag in the relevant command
4. Update validation if needed
5. Update `.env.example` and `agent.yaml.example`

## Build & Release

```bash
make build          # Build for current OS
make linux          # Build for Linux
make windows        # Build for Windows
make test           # Run all tests
make bench          # Run benchmarks
make clean          # Remove build artifacts
```

Version is injected via `-ldflags` from git tags.

## GitHub Actions

- `release-artifacts.yml`: Builds binaries when draft release is created
- `benchmarks.yml`: Runs performance benchmarks
- `build-and-push.yml`: Docker image builds

## Error Handling

- Return errors with context: `fmt.Errorf("failed to do X: %w", err)`
- Use sentinel errors for specific cases: `var ErrAPIAccessDisabled = errors.New("API access disabled")`
- Log errors at the point of handling, not at every level
