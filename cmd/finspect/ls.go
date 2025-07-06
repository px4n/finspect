package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/px4n/finspect/pkg/vfs"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	lsLong      bool
	lsHuman     bool
	lsAll       bool
	lsRecursive bool
)

// newLsCmd creates the ls command
func newLsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls [path]",
		Short: "List directory contents",
		Long:  `List information about files and directories in the VFS.`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			path := "/"
			if len(args) > 0 {
				path = resolvePath(args[0])
			}

			vfs := getVFS()

			// Read directory
			entries, err := vfs.ReadDir(path)
			if err != nil {
				logger.Error("Failed to read directory",
					zap.String("path", path),
					zap.Error(err))
				fmt.Fprintf(os.Stderr, "ls: %s: %v\n", path, err)
				os.Exit(1)
			}

			// Display entries
			if lsLong {
				displayLongFormat(entries)
			} else {
				displayShortFormat(entries)
			}
		},
	}

	cmd.Flags().BoolVarP(&lsLong, "long", "l", false, "use long listing format")
	cmd.Flags().BoolVarP(&lsHuman, "human-readable", "H", false, "print sizes in human readable format")
	cmd.Flags().BoolVarP(&lsAll, "all", "a", false, "show hidden files")
	cmd.Flags().BoolVarP(&lsRecursive, "recursive", "R", false, "list subdirectories recursively")

	return cmd
}

var lsCmd = newLsCmd()

func displayShortFormat(entries []vfs.DirEntry) {
	for _, entry := range entries {
		// Skip hidden files unless -a flag is set
		if !lsAll && len(entry.Name()) > 0 && entry.Name()[0] == '.' {
			continue
		}

		fmt.Print(entry.Name())
		if entry.IsDir() {
			fmt.Print("/")
		}
		fmt.Println()
	}
}

func displayLongFormat(entries []vfs.DirEntry) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	for _, entry := range entries {
		// Skip hidden files unless -a flag is set
		if !lsAll && len(entry.Name()) > 0 && entry.Name()[0] == '.' {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			logger.Warn("Failed to get file info",
				zap.String("name", entry.Name()),
				zap.Error(err))
			continue
		}

		// Format: mode size date name
		mode := info.Mode().String()
		size := formatSize(info.Size())
		modTime := formatTime(info.ModTime())
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", mode, size, modTime, name)
	}

	_ = w.Flush() // Best effort flush
}

func formatSize(size int64) string {
	if !lsHuman {
		return fmt.Sprintf("%d", size)
	}

	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%c", float64(size)/float64(div), "KMGTPE"[exp])
}

func formatTime(t time.Time) string {
	now := time.Now()
	if t.Year() == now.Year() {
		// Same year, show month day time
		return t.Format("Jan _2 15:04")
	}
	// Different year, show month day year
	return t.Format("Jan _2  2006")
}
