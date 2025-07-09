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
			paths := resolvePaths(args[0], args[1])
			source, dest := paths[0], paths[1]

			vfs := getVFS()

			// Check if source exists
			_, err := vfs.Stat(source)
			if err != nil {
				handleCommandError("mv", "stat source", source, err)
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
				// Check if it's a cross-device error
				if err.Error() == "vfs: cross-device operation not permitted" {
					fmt.Fprintf(os.Stderr, "mv: %s and %s are on different mounts, use cp instead\n", source, dest)
				}
				handleCommandErrorWithFields("mv", "move", err,
					zap.String("source", source),
					zap.String("dest", dest))
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
