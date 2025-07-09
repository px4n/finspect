package main

import (
	"context"
	"fmt"
	"os"

	"github.com/px4n/finspect/adaptors/filesystem"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var mountCmd = &cobra.Command{
	Use:   "mount <type> <path> [mount-point]",
	Short: "Mount a data source",
	Long: `Mount a data source into the VFS. Currently supported types:
  - local: Local filesystem

Examples:
  finspect mount local /Users/username /local
  finspect mount local ~/Documents /docs`,
	Args: cobra.MinimumNArgs(2),
	Run: func(_ *cobra.Command, args []string) {
		sourceType := args[0]
		sourcePath := args[1]

		// Default mount point is the source type
		mountPoint := "/" + sourceType
		if len(args) > 2 {
			mountPoint = args[2]
		}

		// Get or create VFS instance
		vfs := getVFS()

		// Create adaptor based on type
		var adaptor interface{}
		switch sourceType {
		case "local", "filesystem":
			fs := filesystem.New()

			// Connect to filesystem
			config := map[string]interface{}{
				"root": sourcePath,
			}

			if err := fs.Connect(context.Background(), config); err != nil {
				logger.Error("Failed to connect adaptor",
					zap.String("type", sourceType),
					zap.String("path", sourcePath),
					zap.Error(err))
				os.Exit(1)
			}

			adaptor = fs

		default:
			fmt.Fprintf(os.Stderr, "Unknown source type: %s\n", sourceType)
			os.Exit(1)
		}

		// Mount the adaptor
		if err := vfs.Mount(mountPoint, adaptor.(*filesystem.Adaptor)); err != nil {
			logger.Error("Failed to mount",
				zap.String("mount", mountPoint),
				zap.Error(err))
			os.Exit(1)
		}

		// Save mount configuration
		saveMountConfig(mountPoint, sourceType, sourcePath)

		fmt.Printf("Mounted %s at %s\n", sourcePath, mountPoint)
	},
}
