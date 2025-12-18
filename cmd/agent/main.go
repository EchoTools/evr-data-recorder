package main

import (
	"fmt"
	"os"

	"github.com/echotools/evr-data-recorder/v3/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	version    = "dev"
	cfg        *config.Config
	logger     *zap.Logger
	configFile string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "agent",
		Short:   "NEVR Agent - Tools for recording and processing EchoVR telemetry",
		Version: version,
		Long: `NEVR Agent is a suite of tools for recording session and player bone 
data from the EchoVR game engine HTTP API, converting between formats, and 
serving recorded data.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = config.LoadConfig(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Override config with global flags
			if viper.IsSet("debug") {
				cfg.Debug = viper.GetBool("debug")
			}
			if viper.IsSet("log-level") {
				cfg.LogLevel = viper.GetString("log-level")
			}
			if viper.IsSet("log-file") {
				cfg.LogFile = viper.GetString("log-file")
			}

			logger, err = cfg.NewLogger()
			if err != nil {
				return fmt.Errorf("failed to create logger: %w", err)
			}

			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if logger != nil {
				_ = logger.Sync()
			}
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file (default is ./agent.yaml)")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "enable debug logging")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-file", "", "log file path")

	// Bind global flags to viper
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log-file", rootCmd.PersistentFlags().Lookup("log-file"))

	// Add subcommands
	rootCmd.AddCommand(newAgentCommand())
	rootCmd.AddCommand(newAPIServerCommand())
	rootCmd.AddCommand(newConverterCommand())
	rootCmd.AddCommand(newReplayerCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
