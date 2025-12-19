package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/echotools/nevr-agent/v4/internal/agent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func newAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stream [host:port[-endPort]] [host:port[-endPort]...]",
		Short: "Record session and player bone data from EchoVR game servers",
		Long: `The stream command regularly scans specified ports and starts polling 
the HTTP API at the configured frequency, storing output to files.`,
		Example: `  # Record from ports 6721-6730 on localhost at 30Hz
	  agent stream --frequency 30 --output ./output 127.0.0.1:6721-6730

  # Record with streaming enabled
	  agent stream --stream --stream-username myuser 127.0.0.1:6721-6730

  # Use a config file
	  agent stream -c config.yaml 127.0.0.1:6721`,
		RunE: runAgent,
	}

	// Agent-specific flags
	cmd.Flags().Int("frequency", 10, "Polling frequency in Hz")
	cmd.Flags().String("format", "replay", "Output format (replay, stream, or comma-separated)")
	cmd.Flags().String("output", "output", "Output directory")

	// Stream options
	// (Removed)

	// Events API options
	cmd.Flags().Bool("events", false, "Enable sending frames to events API")
	cmd.Flags().Bool("events-stream", false, "Enable streaming frames to events API via WebSocket")
	cmd.Flags().String("events-url", "http://localhost:8081", "Base URL of the events API")
	cmd.Flags().String("events-user-id", "", "Optional user ID header for events API")
	cmd.Flags().String("events-node-id", "default-node", "Node ID header for events API")

	// Bind flags to viper
	viper.BindPFlags(cmd.Flags())

	return cmd
}

func runAgent(cmd *cobra.Command, args []string) error {
	// Override config with command flags
	cfg.Agent.Frequency = viper.GetInt("frequency")
	cfg.Agent.Format = viper.GetString("format")
	cfg.Agent.OutputDirectory = viper.GetString("output")
	cfg.Agent.EventsEnabled = viper.GetBool("events")
	cfg.Agent.EventsURL = viper.GetString("events-url")

	// Parse targets from arguments
	if len(args) == 0 {
		return fmt.Errorf("at least one host:port target must be specified")
	}

	targets := make(map[string][]int)
	for _, hostPort := range args {
		host, ports, err := parseHostPort(hostPort)
		if err != nil {
			return fmt.Errorf("failed to parse host:port %q: %w", hostPort, err)
		}
		targets[host] = ports
	}

	// Validate configuration
	if err := cfg.ValidateAgentConfig(); err != nil {
		return err
	}

	logger.Info("Starting agent",
		zap.Int("frequency", cfg.Agent.Frequency),
		zap.String("format", cfg.Agent.Format),
		zap.String("output_directory", cfg.Agent.OutputDirectory),
		zap.Any("targets", targets))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	go startAgent(ctx, logger, targets)

	select {
	case <-ctx.Done():
		logger.Info("Context done, shutting down")
	case <-interrupt:
		logger.Info("Received interrupt signal, shutting down")
		cancel()
	}

	time.Sleep(2 * time.Second) // Allow ongoing operations to finish
	logger.Info("Agent stopped gracefully")
	return nil
}

func startAgent(ctx context.Context, logger *zap.Logger, targets map[string][]int) {
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			MaxConnsPerHost:       2,
			DisableCompression:    true,
			MaxIdleConns:          2,
			MaxIdleConnsPerHost:   2,
			IdleConnTimeout:       5 * time.Second,
			TLSHandshakeTimeout:   2 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   2 * time.Second,
				KeepAlive: 5 * time.Second,
			}).DialContext,
		},
	}

	sessions := make(map[string]agent.FrameWriter)
	interval := time.Second / time.Duration(cfg.Agent.Frequency)
	cycleTicker := time.NewTicker(100 * time.Millisecond)
	scanTicker := time.NewTicker(10 * time.Millisecond)

OuterLoop:
	for {
		select {
		case <-ctx.Done():
			return
		case <-cycleTicker.C:
			cycleTicker.Reset(5 * time.Second)
		}

		logger.Debug("Scanning targets", zap.Any("targets", targets))
		for host, ports := range targets {
			logger := logger.With(zap.String("host", host))
			<-scanTicker.C

			for _, port := range ports {
				select {
				case <-ctx.Done():
					break OuterLoop
				default:
				}

				logger := logger.With(zap.Int("port", port))
				baseURL := fmt.Sprintf("http://%s:%d", host, port)

				if s, found := sessions[baseURL]; found {
					if !s.IsStopped() {
						logger.Debug("session still active, skipping")
						continue
					} else {
						delete(sessions, baseURL)
					}
				}

				meta, err := agent.GetSessionMeta(baseURL)
				if err != nil {
					switch err {
					case agent.ErrAPIAccessDisabled:
						logger.Warn("API access is disabled on the server")
					default:
						logger.Debug("Failed to get session metadata", zap.Error(err))
					}
					continue
				}
				if meta.SessionUUID == "" {
					continue
				}

				logger.Debug("Retrieved session metadata", zap.Any("meta", meta))

				var filename string
				var outputPath string

				writers := make([]agent.FrameWriter, 0)

				// Create the appropriate file writer based on format
				formats := strings.Split(cfg.Agent.Format, ",")

				for _, format := range formats {
					format = strings.TrimSpace(format)
					if format == "" || format == "none" {
						continue
					}

					switch format {
					case "replay":
						fallthrough
					default:
						filename = agent.EchoReplaySessionFilename(time.Now(), meta.SessionUUID)
						outputPath = filepath.Join(cfg.Agent.OutputDirectory, filename)
						replayWriter := agent.NewFrameDataLogSession(ctx, logger, outputPath, meta.SessionUUID)
						go replayWriter.ProcessFrames()
						writers = append(writers, replayWriter)
					}
				}

				logger = logger.With(zap.String("session_uuid", meta.SessionUUID))
				if filename != "" {
					logger = logger.With(zap.String("filename", filename))
				}

				// If events sending is enabled, add EventsAPI writer
				if cfg.Agent.EventsEnabled {
					eventsWriter := agent.NewEventsAPIWriter(logger, cfg.Agent.EventsURL, cfg.Agent.JWTToken)
					writers = append(writers, eventsWriter)
				}
				// If events streaming is enabled, add WebSocket writer
				if viper.GetBool("events-stream") {
					// Derive WebSocket URL from Events URL if not explicitly set
					wsURL := cfg.Agent.EventsURL
					if strings.HasPrefix(wsURL, "http") {
						wsURL = strings.Replace(wsURL, "http", "ws", 1)
					}
					wsURL = strings.TrimSuffix(wsURL, "/") + "/v3/stream"

					nodeID := viper.GetString("events-node-id")
					userID := viper.GetString("events-user-id")

					wsWriter := agent.NewWebSocketWriter(logger, wsURL, cfg.Agent.JWTToken, nodeID, userID)
					if err := wsWriter.Connect(); err != nil {
						logger.Error("Failed to connect WebSocket writer", zap.Error(err))
					} else {
						logger.Info("WebSocket writer connected successfully", zap.String("url", wsURL))
						writers = append(writers, wsWriter)
					}
				}

				if len(writers) == 0 {
					logger.Warn("No output format or destination specified, skipping session")
					continue
				}

				var session agent.FrameWriter
				if len(writers) == 1 {
					session = writers[0]
				} else {
					session = agent.NewMultiWriter(logger, writers...)
				}

				sessions[baseURL] = session
				go agent.NewHTTPFramePoller(session.Context(), logger, client, baseURL, interval, session)

				logger.Info("Added new frame client",
					zap.String("file_path", outputPath))
			}
		}

		select {
		case <-ctx.Done():
			break OuterLoop
		case <-time.After(3 * time.Second):
		}
	}

	logger.Info("Finished processing all targets, exiting")
	for _, session := range sessions {
		session.Close()
	}
	logger.Info("Closed sessions")
}

func parseHostPort(s string) (string, []int, error) {
	components := strings.Split(s, ":")
	if len(components) != 2 {
		return "", nil, errors.New("invalid format, expected host:port or host:startPort-endPort")
	}

	host := components[0]
	ports, err := parsePortRange(components[1])
	if err != nil {
		return "", nil, err
	}

	return host, ports, nil
}

func parsePortRange(port string) ([]int, error) {
	portRanges := strings.Split(port, ",")
	ports := make([]int, 0)

	for _, rangeStr := range portRanges {
		rangeStr = strings.TrimSpace(rangeStr)
		if rangeStr == "" {
			continue
		}
		parts := strings.SplitN(rangeStr, "-", 2)
		if len(parts) > 2 {
			return nil, fmt.Errorf("invalid port range %q", rangeStr)
		}

		if len(parts) == 1 {
			port, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid port %q: %v", rangeStr, err)
			}
			ports = append(ports, port)
		} else {
			startPort, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid port %q: %v", port, err)
			}
			endPort, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid port %q: %v", port, err)
			}
			if startPort > endPort {
				return nil, fmt.Errorf("invalid port range %q: startPort must be less than or equal to endPort", rangeStr)
			}

			for i := startPort; i <= endPort; i++ {
				ports = append(ports, i)
			}
		}

		for _, port := range ports {
			if port < 0 || port > 65535 {
				return nil, fmt.Errorf("invalid port %d: port must be between 0 and 65535", port)
			}
		}
	}
	return ports, nil
}
