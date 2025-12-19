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
	"github.com/echotools/nevr-agent/v4/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func newAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent [host:port[-endPort]] [host:port[-endPort]...]",
		Short: "Record session and player bone data from EchoVR game servers",
		Long: `The agent command regularly scans specified ports and starts polling 
the HTTP API at the configured frequency, storing output to files.`,
		Example: `  # Record from ports 6721-6730 on localhost at 30Hz
	  agent agent --frequency 30 --output ./output 127.0.0.1:6721-6730

  # Record with streaming enabled
	  agent agent --stream --stream-username myuser 127.0.0.1:6721-6730

  # Use a config file
	  agent agent -c config.yaml 127.0.0.1:6721`,
		RunE: runAgent,
	}

	// Agent-specific flags
	cmd.Flags().IntP("frequency", "f", 10, "Polling frequency in Hz")
	cmd.Flags().String("format", "nevrcap", "Output format (nevrcap, replay, stream, or comma-separated)")
	cmd.Flags().StringP("output", "o", "output", "Output directory")

	// JWT token for API authentication
	cmd.Flags().String("token", "", "JWT token for API authentication (stream and events)")

	// Stream options
	cmd.Flags().Bool("stream", false, "Enable streaming to Nakama server")
	cmd.Flags().String("stream-http", "https://g.echovrce.com:7350", "Stream HTTP URL")
	cmd.Flags().String("stream-socket", "wss://g.echovrce.com:7350/ws", "Stream WebSocket URL")
	cmd.Flags().String("stream-server-key", "", "Stream server key")

	// Events API options
	cmd.Flags().Bool("events", false, "Enable sending frames to events API")
	cmd.Flags().String("events-url", "http://localhost:8081", "Base URL of the events API")

	// Bind flags to viper - this must happen before PersistentPreRunE
	viper.BindPFlag("agent.frequency", cmd.Flags().Lookup("frequency"))
	viper.BindPFlag("agent.format", cmd.Flags().Lookup("format"))
	viper.BindPFlag("agent.output_directory", cmd.Flags().Lookup("output"))
	viper.BindPFlag("agent.jwt_token", cmd.Flags().Lookup("token"))
	viper.BindPFlag("agent.stream_enabled", cmd.Flags().Lookup("stream"))
	viper.BindPFlag("agent.stream_http_url", cmd.Flags().Lookup("stream-http"))
	viper.BindPFlag("agent.stream_socket_url", cmd.Flags().Lookup("stream-socket"))
	viper.BindPFlag("agent.stream_server_key", cmd.Flags().Lookup("stream-server-key"))
	viper.BindPFlag("agent.events_enabled", cmd.Flags().Lookup("events"))
	viper.BindPFlag("agent.events_url", cmd.Flags().Lookup("events-url"))

	return cmd
}

func runAgent(cmd *cobra.Command, args []string) error {
	// Override config with command flags (only if explicitly set)
	// Viper has flags bound, so we can check if they were set and override config
	if cmd.Flags().Changed("frequency") {
		cfg.Agent.Frequency = viper.GetInt("agent.frequency")
	}
	if cmd.Flags().Changed("format") {
		cfg.Agent.Format = viper.GetString("agent.format")
	}
	if cmd.Flags().Changed("output") {
		cfg.Agent.OutputDirectory = viper.GetString("agent.output_directory")
	}
	if cmd.Flags().Changed("token") {
		cfg.Agent.JWTToken = viper.GetString("agent.jwt_token")
	}
	if cmd.Flags().Changed("stream") {
		cfg.Agent.StreamEnabled = viper.GetBool("agent.stream_enabled")
	}
	if cmd.Flags().Changed("stream-http") {
		cfg.Agent.StreamHTTPURL = viper.GetString("agent.stream_http_url")
	}
	if cmd.Flags().Changed("stream-socket") {
		cfg.Agent.StreamSocketURL = viper.GetString("agent.stream_socket_url")
	}
	if cmd.Flags().Changed("stream-server-key") {
		cfg.Agent.StreamServerKey = viper.GetString("agent.stream_server_key")
	}
	if cmd.Flags().Changed("events") {
		cfg.Agent.EventsEnabled = viper.GetBool("agent.events_enabled")
	}
	if cmd.Flags().Changed("events-url") {
		cfg.Agent.EventsURL = viper.GetString("agent.events_url")
	}

	// Test connectivity to external services at startup
	if err := testExternalServices(logger, cfg.Agent); err != nil {
		return fmt.Errorf("external service health check failed: %w", err)
	}

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

	logger.Info("NEVR Agent started",
		zap.String("version", version),
		zap.Int("frequency", cfg.Agent.Frequency),
		zap.String("format", cfg.Agent.Format),
		zap.String("output_directory", cfg.Agent.OutputDirectory),
		zap.Bool("events_enabled", cfg.Agent.EventsEnabled),
		zap.String("events_url", cfg.Agent.EventsURL),
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
	// Create custom transport with User-Agent header
	userAgent := fmt.Sprintf("NEVR-Agent/%s", version)
	transport := &http.Transport{
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
	}

	client := &http.Client{
		Timeout:   3 * time.Second,
		Transport: &userAgentTransport{Transport: transport, UserAgent: userAgent},
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

				logger := logger.With(zap.String("host_addresss", fmt.Sprintf("%s:%d", host, port)))
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
				var fileWriter agent.FrameWriter

				// Create the appropriate file writer based on format
				formats := strings.Split(cfg.Agent.Format, ",")
				hasStreamFormat := false
				for _, format := range formats {
					if strings.TrimSpace(format) == "stream" {
						hasStreamFormat = true
						break
					}
				}

				if len(formats) > 1 {
					// Create multi-writer
					writers := make([]agent.FrameWriter, 0, len(formats))
					for _, format := range formats {
						format = strings.TrimSpace(format)
						var fw agent.FrameWriter
						switch format {
						case "stream":
							rtapiWriter := agent.NewStreamWriter(logger, cfg.Agent.StreamHTTPURL, cfg.Agent.StreamSocketURL,
								cfg.Agent.JWTToken, cfg.Agent.StreamServerKey)
							if err := rtapiWriter.Connect(); err != nil {
								logger.Error("Failed to connect stream writer", zap.Error(err))
								continue
							}
							logger.Info("Stream writer connected successfully")
							fw = rtapiWriter
						case "nevrcap":
							filename = agent.NevrCapSessionFilename(time.Now(), meta.SessionUUID)
							outputPath = filepath.Join(cfg.Agent.OutputDirectory, filename)
							nevrCapWriter := agent.NewNevrCapLogSession(ctx, logger, outputPath, meta.SessionUUID)
							go nevrCapWriter.ProcessFrames()
							fw = nevrCapWriter
						case "replay":
							fallthrough
						default:
							filename = agent.EchoReplaySessionFilename(time.Now(), meta.SessionUUID)
							outputPath = filepath.Join(cfg.Agent.OutputDirectory, filename)
							replayWriter := agent.NewFrameDataLogSession(ctx, logger, outputPath, meta.SessionUUID)
							go replayWriter.ProcessFrames()
							fw = replayWriter
						}
						writers = append(writers, fw)
					}
					fileWriter = agent.NewMultiWriter(logger, writers...)
				} else {
					switch formats[0] {
					case "stream":
						rtapiWriter := agent.NewStreamWriter(logger, cfg.Agent.StreamHTTPURL, cfg.Agent.StreamSocketURL,
							cfg.Agent.JWTToken, cfg.Agent.StreamServerKey)
						if err := rtapiWriter.Connect(); err != nil {
							logger.Error("Failed to connect stream writer", zap.Error(err))
							continue
						}
						logger.Info("Stream writer connected successfully")
						fileWriter = rtapiWriter
					case "nevrcap":
						filename = agent.NevrCapSessionFilename(time.Now(), meta.SessionUUID)
						outputPath = filepath.Join(cfg.Agent.OutputDirectory, filename)
						nevrCapWriter := agent.NewNevrCapLogSession(ctx, logger, outputPath, meta.SessionUUID)
						go nevrCapWriter.ProcessFrames()
						fileWriter = nevrCapWriter
					case "replay":
						fallthrough
					default:
						filename = agent.EchoReplaySessionFilename(time.Now(), meta.SessionUUID)
						outputPath = filepath.Join(cfg.Agent.OutputDirectory, filename)
						replayWriter := agent.NewFrameDataLogSession(ctx, logger, outputPath, meta.SessionUUID)
						go replayWriter.ProcessFrames()
						fileWriter = replayWriter
					}
				}

				logger = logger.With(zap.String("session_uuid", meta.SessionUUID))

				var session agent.FrameWriter = fileWriter

				// If streaming is enabled via flag (and not already in format list), add stream writer
				if cfg.Agent.StreamEnabled && !hasStreamFormat {
					streamWriter := agent.NewStreamWriter(logger, cfg.Agent.StreamHTTPURL, cfg.Agent.StreamSocketURL,
						cfg.Agent.JWTToken, cfg.Agent.StreamServerKey)
					if err := streamWriter.Connect(); err != nil {
						logger.Error("Failed to connect stream writer", zap.Error(err))
					} else {
						logger.Info("Stream writer connected successfully")
						session = agent.NewMultiWriter(logger, fileWriter, streamWriter)
					}
				}

				// If events sending is enabled, add EventsAPI writer
				if cfg.Agent.EventsEnabled {
					eventsWriter := agent.NewEventsAPIWriter(logger, cfg.Agent.EventsURL, cfg.Agent.JWTToken)
					session = agent.NewMultiWriter(logger, session, eventsWriter)
				}

				sessions[baseURL] = session
				go agent.NewHTTPFramePoller(session.Context(), logger, client, baseURL, interval, session)

				logger.Info("Added new frame client",
					zap.String("file_path", outputPath),
					zap.Bool("streaming_enabled", cfg.Agent.StreamEnabled))
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

// testExternalServices performs health checks on configured external services at startup
// This ensures failures are caught immediately rather than waiting for the first event
func testExternalServices(logger *zap.Logger, agentCfg config.AgentConfig) error {
	var errors []string

	// Test Events API connectivity if enabled
	if agentCfg.EventsEnabled {
		logger.Info("Testing Events API connectivity", zap.String("url", agentCfg.EventsURL))
		if err := testEventsAPI(agentCfg.EventsURL); err != nil {
			errMsg := fmt.Sprintf("Events API health check failed: %v", err)
			logger.Error(errMsg)
			errors = append(errors, errMsg)
		} else {
			logger.Info("Events API health check passed")
		}
	}

	// Test Stream connectivity if enabled
	if agentCfg.StreamEnabled {
		logger.Info("Testing Stream server connectivity",
			zap.String("http_url", agentCfg.StreamHTTPURL),
			zap.String("socket_url", agentCfg.StreamSocketURL))
		if err := testStreamConnectivity(agentCfg); err != nil {
			errMsg := fmt.Sprintf("Stream server health check failed: %v", err)
			logger.Error(errMsg)
			errors = append(errors, errMsg)
		} else {
			logger.Info("Stream server health check passed")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed health checks: %v", strings.Join(errors, "; "))
	}

	return nil
}

// testEventsAPI checks if the Events API server is reachable and responding
func testEventsAPI(eventsURL string) error {
	client := &http.Client{Timeout: 5 * time.Second}

	// Try a simple HEAD or GET request to the base URL to test connectivity
	resp, err := client.Head(eventsURL)
	if err != nil {
		return fmt.Errorf("failed to connect to events API at %s: %w", eventsURL, err)
	}
	defer resp.Body.Close()

	// Accept any response - just checking if the server is reachable
	// 404 is fine, 500+ errors indicate server issues
	if resp.StatusCode >= 500 {
		return fmt.Errorf("events API returned error status %d", resp.StatusCode)
	}

	return nil
}

// testStreamConnectivity tests connection to the Nakama stream server
func testStreamConnectivity(cfg config.AgentConfig) error {
	// Create a stream client and attempt to connect
	logger := logger.With(
		zap.String("component", "stream_health_check"),
	)

	streamClient := agent.NewStreamClient(
		logger,
		cfg.StreamHTTPURL,
		cfg.StreamSocketURL,
		cfg.JWTToken,
		cfg.StreamServerKey,
	)

	// Attempt to connect - this includes both HTTP auth and WebSocket connection
	if err := streamClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to stream server: %w", err)
	}

	// Close the test connection
	streamClient.Close()

	return nil
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

// userAgentTransport is a custom RoundTripper that adds User-Agent header to all requests
type userAgentTransport struct {
	Transport *http.Transport
	UserAgent string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.UserAgent)
	return t.Transport.RoundTrip(req)
}
