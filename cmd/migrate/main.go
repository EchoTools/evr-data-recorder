package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/echotools/evr-data-recorder/v4/internal/api"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Get MongoDB URI from environment or use default
	mongoURI := os.Getenv("EVR_APISERVER_MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	fmt.Printf("Connecting to MongoDB: %s\n", mongoURI)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, cancelling migration...")
		cancel()
	}()

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to MongoDB: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer disconnectCancel()
		client.Disconnect(disconnectCtx)
	}()

	// Ping MongoDB to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping MongoDB: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected to MongoDB successfully")

	// Create logger
	logger := &api.DefaultLogger{}

	// Run migration
	fmt.Println("Starting schema migration...")
	stats, err := api.MigrateSchema(ctx, client, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}

	// Print statistics
	fmt.Println("\n=== Migration Statistics ===")
	fmt.Printf("Total documents:    %d\n", stats.TotalDocuments)
	fmt.Printf("Migrated documents: %d\n", stats.MigratedDocuments)
	fmt.Printf("Skipped documents:  %d\n", stats.SkippedDocuments)
	fmt.Printf("Failed documents:   %d\n", stats.FailedDocuments)
	fmt.Printf("Duration:           %v\n", stats.EndTime.Sub(stats.StartTime))

	// Validate migration
	fmt.Println("\nValidating migration...")
	if err := api.ValidateMigration(ctx, client, logger); err != nil {
		fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nMigration completed successfully!")
}
