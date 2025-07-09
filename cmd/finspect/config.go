package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/px4n/finspect/pkg/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  "Manage finspect configuration files",
	}

	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigValidateCmd())
	cmd.AddCommand(newConfigPathCmd())

	return cmd
}

var configCmd = newConfigCmd()

func newConfigInitCmd() *cobra.Command {
	var (
		configPath string
		force      bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file",
		Long:  "Create a new configuration file with default values",
		RunE: func(_ *cobra.Command, _ []string) error {
			// Check if file exists
			if _, err := os.Stat(configPath); err == nil && !force {
				return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
			}

			// Create default config
			cfg := config.DefaultConfig()

			// Save to file
			if err := cfg.Save(configPath); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Printf("Configuration file created: %s\n", configPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "output", "o", "finspect.yaml", "output configuration file")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing file")

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	var (
		configPath string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  "Display the current configuration with all values expanded",
		RunE: func(_ *cobra.Command, _ []string) error {
			loader := config.NewLoader()

			var cfg *config.Config
			var err error

			if configPath != "" {
				// Load from specific path
				cfg, err = loader.LoadFromPath(configPath)
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
			} else {
				// Load from search paths
				var path string
				cfg, path, err = loader.Load()
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
				if path != "" {
					fmt.Printf("# Loaded from: %s\n", path)
				} else {
					fmt.Printf("# Using default configuration (no config file found)\n")
				}
			}

			// Merge environment variables
			config.MergeEnvironment(cfg)

			// Display config
			switch format {
			case "json":
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(cfg)
			case "yaml", "yml":
				encoder := yaml.NewEncoder(os.Stdout)
				encoder.SetIndent(2)
				return encoder.Encode(cfg)
			default:
				// Pretty print key values
				fmt.Println("\n# Global Settings")
				fmt.Printf("Work Directory: %s\n", cfg.Global.WorkDir)
				fmt.Printf("Temp Directory: %s\n", cfg.Global.TempDir)
				fmt.Printf("Max Concurrency: %d\n", cfg.Global.MaxConcurrency)
				fmt.Printf("Enable Profiling: %v\n", cfg.Global.EnableProfiling)

				fmt.Println("\n# Mounts")
				for i, mount := range cfg.Mounts {
					fmt.Printf("[%d] %s -> %s\n", i+1, mount.Path, mount.Type)
					if mount.Options.ReadOnly {
						fmt.Printf("    Read-only: true\n")
					}
					if mount.Options.EnableCache {
						fmt.Printf("    Cache TTL: %d seconds\n", mount.Options.CacheTTL)
					}
				}

				fmt.Println("\n# Metadata")
				fmt.Printf("Database: %s\n", cfg.Metadata.DatabasePath)
				fmt.Printf("Enable FTS: %v\n", cfg.Metadata.EnableFTS)
				fmt.Printf("Batch Size: %d\n", cfg.Metadata.BatchSize)
				fmt.Printf("Auto Index: %v\n", cfg.Metadata.AutoIndex)

				fmt.Println("\n# Search")
				fmt.Printf("Enabled: %v\n", cfg.Search.Enabled)
				fmt.Printf("Index Path: %s\n", cfg.Search.IndexPath)
				fmt.Printf("Default Limit: %d\n", cfg.Search.DefaultLimit)
				fmt.Printf("Index Content: %v\n", cfg.Search.IndexOptions.IndexContent)

				fmt.Println("\n# Admin UI")
				fmt.Printf("Enabled: %v\n", cfg.AdminUI.Enabled)
				if cfg.AdminUI.Enabled {
					fmt.Printf("Address: %s:%d\n", cfg.AdminUI.Address, cfg.AdminUI.Port)
					fmt.Printf("Static Directory: %s\n", cfg.AdminUI.StaticDir)
					fmt.Printf("Authentication: %v\n", cfg.AdminUI.EnableAuth)
				}

				fmt.Println("\n# Logging")
				fmt.Printf("Level: %s\n", cfg.Logging.Level)
				fmt.Printf("Format: %s\n", cfg.Logging.Format)
				fmt.Printf("Output: %s\n", cfg.Logging.Output)
				if cfg.Logging.Output == "file" {
					fmt.Printf("File Path: %s\n", cfg.Logging.FilePath)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "configuration file path")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "output format (text, json, yaml)")

	return cmd
}

func newConfigValidateCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long:  "Check if the configuration file is valid",
		RunE: func(_ *cobra.Command, _ []string) error {
			loader := config.NewLoader()

			var cfg *config.Config
			var err error
			var path string

			if configPath != "" {
				// Load from specific path
				cfg, err = loader.LoadFromPath(configPath)
				path = configPath
			} else {
				// Load from search paths
				cfg, path, err = loader.Load()
			}

			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if path == "" {
				fmt.Println("No configuration file found, using defaults")
			} else {
				fmt.Printf("Validating: %s\n", path)
			}

			// Validate is already called during load, but call again for clarity
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			fmt.Println("Configuration is valid")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "configuration file path")

	return cmd
}

func newConfigPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		Long:  "Display the path to the configuration file that would be loaded",
		RunE: func(_ *cobra.Command, _ []string) error {
			loader := config.NewLoader()

			path, err := loader.FindConfigFile()
			if err != nil {
				fmt.Println("No configuration file found in search paths")
				fmt.Println("\nSearch paths:")
				for _, p := range loader.GetSearchPaths() {
					fmt.Printf("  - %s\n", p)
				}
				fmt.Println("\nExpected filenames:")
				for _, f := range config.ConfigPaths {
					fmt.Printf("  - %s\n", f)
				}
				return nil
			}

			fmt.Println(path)
			return nil
		},
	}

	return cmd
}
