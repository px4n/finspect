package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var statCmd = &cobra.Command{
	Use:   "stat <path>...",
	Short: "Display file or directory status",
	Long:  `Display detailed information about files and directories.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		vfs := getVFS()

		for i, arg := range args {
			if i > 0 {
				fmt.Println()
			}

			path := resolvePath(arg)
			info, err := vfs.Stat(path)
			if err != nil {
				logger.Error("Failed to stat",
					zap.String("path", path),
					zap.Error(err))
				fmt.Fprintf(os.Stderr, "stat: %s: %v\n", path, err)
				continue
			}

			// Display file information
			fmt.Printf("  File: %s\n", path)
			fmt.Printf("  Size: %d", info.Size())
			if info.IsDir() {
				fmt.Printf("\tType: directory\n")
			} else {
				fmt.Printf("\tType: regular file\n")
			}
			fmt.Printf("  Mode: %s\n", info.Mode())
			fmt.Printf("Modify: %s\n", info.ModTime().Format("2006-01-02 15:04:05.000000000 -0700"))
		}
	},
}
