package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/echotools/evr-data-recorder/v4/internal/api"
)

func main() {
	// Create configuration
	config := api.DefaultConfig()

	// Configuration is now read from environment variables with EVR_APISERVER_ prefix:
	// EVR_APISERVER_MONGO_URI - MongoDB connection URI
	// EVR_APISERVER_SERVER_ADDRESS - Server bind address
	// EVR_APISERVER_AMQP_URI - AMQP connection URI
	// EVR_APISERVER_AMQP_ENABLED - Enable AMQP publishing
	// EVR_APISERVER_CORS_ORIGINS - Allowed CORS origins (comma-separated)

	// Create service
	service, err := api.NewService(config, nil)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Initialize service
	ctx := context.Background()
	if err := service.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutdown signal received, stopping service...")
		cancel()
	}()

	// Start service
	fmt.Printf("Starting session events service on %s\n", config.ServerAddress)
	fmt.Println("Available endpoints:")
	fmt.Println("  POST /lobby-session-events - Store session event")
	fmt.Println("  GET  /lobby-session-events/{match_id} - Get session events by match ID")
	fmt.Println("  GET  /health - Health check")

	if err := service.Start(ctx); err != nil {
		log.Printf("Service stopped: %v", err)
	}

	// Stop service
	if err := service.Stop(context.Background()); err != nil {
		log.Printf("Error stopping service: %v", err)
	}

	fmt.Println("Service stopped gracefully")
}
