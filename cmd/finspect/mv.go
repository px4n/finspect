package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var mvVerbose bool

// newMvCmd creates the mv command
func newMvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mv <source> <destination>",
		Short: "Move/rename files and directories",
		Long:  `Move or rename files and directories within the VFS.`,
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			source := resolvePath(args[0])
			dest := resolvePath(args[1])

			vfs := getVFS()

			// Check if source exists
			_, err := vfs.Stat(source)
			if err != nil {
				logger.Error("Failed to stat source",
					zap.String("source", source),
					zap.Error(err))
				fmt.Fprintf(os.Stderr, "mv: %s: %v\n", source, err)
				os.Exit(1)
			}

			// Check if destination exists
			destInfo, err := vfs.Stat(dest)
			destExists := err == nil
			destIsDir := destExists && destInfo.IsDir()

			// If destination is a directory, append source filename
			if destIsDir {
				dest = filepath.Join(dest, filepath.Base(source))

				// Check if the new destination already exists
				if _, err := vfs.Stat(dest); err == nil {
					fmt.Fprintf(os.Stderr, "mv: %s: destination already exists\n", dest)
					os.Exit(1)
				}
			} else if destExists {
				// Destination exists and is not a directory
				fmt.Fprintf(os.Stderr, "mv: %s: destination already exists\n", dest)
				os.Exit(1)
			}

			// Perform the rename
			if err := vfs.Rename(source, dest); err != nil {
				logger.Error("Failed to move",
					zap.String("source", source),
					zap.String("dest", dest),
					zap.Error(err))

				// Check if it's a cross-device error
				fmt.Fprintf(os.Stderr, "mv: %v\n", err)
				if err.Error() == "vfs: cross-device operation not permitted" {
					fmt.Fprintf(os.Stderr, "mv: %s and %s are on different mounts, use cp instead\n", source, dest)
				}
				os.Exit(1)
			}

			if mvVerbose {
				fmt.Printf("renamed '%s' -> '%s'\n", source, dest)
			}
		},
	}

	cmd.Flags().BoolVarP(&mvVerbose, "verbose", "v", false, "explain what is being done")

	return cmd
}

var mvCmd = newMvCmd()
