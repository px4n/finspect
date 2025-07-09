package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rmRecursive bool
	rmForce     bool
	rmVerbose   bool
)

// newRmCmd creates the rm command
func newRmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <path>...",
		Short: "Remove files and directories",
		Long:  `Remove files and directories from the VFS.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			vfs := getVFS()

			exitCode := 0
			for _, arg := range args {
				path := resolvePath(arg)

				// Check if path exists
				info, err := vfs.Stat(path)
				if err != nil {
					if !rmForce {
						handleCommandError("rm", "stat", path, err)
					}
					continue
				}

				// Handle directory removal
				if info.IsDir() && !rmRecursive {
					fmt.Fprintf(os.Stderr, "rm: %s: is a directory (use -r to remove)\n", path)
					exitCode = 1
					continue
				}

				// Remove the path
				if rmRecursive && info.IsDir() {
					err = vfs.RemoveAll(path)
				} else {
					err = vfs.Remove(path)
				}

				if err != nil {
					handleCommandError("rm", "remove", path, err)
				}

				if rmVerbose {
					fmt.Printf("removed '%s'\n", path)
				}
			}

			if exitCode != 0 {
				os.Exit(exitCode)
			}
		},
	}

	cmd.Flags().BoolVarP(&rmRecursive, "recursive", "r", false, "remove directories and their contents recursively")
	cmd.Flags().BoolVarP(&rmForce, "force", "f", false, "ignore nonexistent files, never prompt")
	cmd.Flags().BoolVarP(&rmVerbose, "verbose", "v", false, "explain what is being done")

	return cmd
}

var rmCmd = newRmCmd()
