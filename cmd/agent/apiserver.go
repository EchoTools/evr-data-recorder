package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/echotools/nevr-agent/v4/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// zapLoggerAdapter adapts zap.Logger to api.Logger interface
type zapLoggerAdapter struct {
	logger *zap.Logger
}

func (z *zapLoggerAdapter) Debug(msg string, fields ...any) {
	z.logger.Sugar().Debugw(msg, fields...)
}

func (z *zapLoggerAdapter) Info(msg string, fields ...any) {
	z.logger.Sugar().Infow(msg, fields...)
}

func (z *zapLoggerAdapter) Error(msg string, fields ...any) {
	z.logger.Sugar().Errorw(msg, fields...)
}

func (z *zapLoggerAdapter) Warn(msg string, fields ...any) {
	z.logger.Sugar().Warnw(msg, fields...)
}

func newAPIServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the session events API server",
		Long: `The serve command starts an HTTP server that provides endpoints 
for storing and retrieving session event data.`,
		Example: `  # Start API server on default port
	agent serve

  # Start with custom MongoDB URI
	agent serve --mongo-uri mongodb://localhost:27017

  # Use a config file
	agent serve -c config.yaml`,
		RunE: runAPIServer,
	}

	// APIServer-specific flags
	cmd.Flags().String("server-address", ":8081", "Server listen address")
	cmd.Flags().String("mongo-uri", "mongodb://localhost:27017", "MongoDB connection URI")
	cmd.Flags().String("jwt-secret", "", "JWT secret key for token validation")

	// Bind flags to viper
	viper.BindPFlags(cmd.Flags())

	return cmd
}

func runAPIServer(cmd *cobra.Command, args []string) error {
	// Override config with command flags
	cfg.APIServer.ServerAddress = viper.GetString("server-address")
	cfg.APIServer.MongoURI = viper.GetString("mongo-uri")
	cfg.APIServer.JWTSecret = viper.GetString("jwt-secret")

	// Validate configuration
	if err := cfg.ValidateAPIServerConfig(); err != nil {
		return err
	}

	logger.Info("Starting API server",
		zap.String("server_address", cfg.APIServer.ServerAddress),
		zap.String("mongo_uri", cfg.APIServer.MongoURI))

	// Create service configuration
	serviceConfig := api.DefaultConfig()
	serviceConfig.MongoURI = cfg.APIServer.MongoURI
	serviceConfig.ServerAddress = cfg.APIServer.ServerAddress
	serviceConfig.JWTSecret = cfg.APIServer.JWTSecret

	// Create service
	service, err := api.NewService(serviceConfig, &zapLoggerAdapter{logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Initialize service
	ctx := context.Background()
	if err := service.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutdown signal received, stopping service...")
		cancel()
	}()

	// Start service
	logger.Info("Starting session events service",
		zap.String("address", cfg.APIServer.ServerAddress))
	logger.Info("Available endpoints:",
		zap.String("POST", "/lobby-session-events - Store session event"),
		zap.String("GET", "/lobby-session-events/{match_id} - Get session events by match ID"),
		zap.String("WebSocket", "/v3/stream - WebSocket stream with JWT auth"),
		zap.String("GET", "/health - Health check"))

	if err := service.Start(ctx); err != nil {
		logger.Info("Service stopped", zap.Error(err))
	}

	// Stop service
	if err := service.Stop(context.Background()); err != nil {
		logger.Warn("Error stopping service", zap.Error(err))
	}

	logger.Info("API server stopped gracefully")
	return nil
}
