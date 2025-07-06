package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Version information (set by build flags)
	Version   = "dev"
	BuildTime = "unknown"

	// Global flags
	cfgFile string
	verbose bool

	// Logger
	logger *zap.Logger
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "finspect",
	Short: "Universal data management system",
	Long: `finspect is a universal data management platform that treats all your data sources
(local files, cloud storage, social media) as a unified POSIX-like filesystem with
rich metadata and workflow automation.`,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		// Initialize logger based on verbosity
		initLogger()
	},
}

// setupCommands configures all commands and flags
func setupCommands() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.finspect/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Add commands
	rootCmd.AddCommand(
		versionCmd,
		mountCmd,
		lsCmd,
		cpCmd,
		mvCmd,
		rmCmd,
		statCmd,
		searchCmd,
	)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Search config in home directory with name ".finspect"
		viper.AddConfigPath(home + "/.finspect")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvPrefix("FINSPECT")

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// initLogger initializes the zap logger
func initLogger() {
	var config zap.Config

	if verbose {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.OutputPaths = []string{} // No output unless verbose
		config.ErrorOutputPaths = []string{"stderr"}
	}

	var err error
	logger, err = config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
}

func main() {
	setupCommands()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
