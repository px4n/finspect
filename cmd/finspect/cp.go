package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/px4n/finspect/pkg/vfs"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	cpRecursive bool
	cpVerbose   bool
)

// newCpCmd creates the cp command
func newCpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cp <source> <destination>",
		Short: "Copy files and directories",
		Long:  `Copy files and directories within the VFS or between VFS and local filesystem.`,
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			source := resolvePath(args[0])
			dest := resolvePath(args[1])

			vfs := getVFS()

			// Get source info
			sourceInfo, err := vfs.Stat(source)
			if err != nil {
				logger.Error("Failed to stat source",
					zap.String("source", source),
					zap.Error(err))
				fmt.Fprintf(os.Stderr, "cp: %s: %v\n", source, err)
				os.Exit(1)
			}

			if sourceInfo.IsDir() && !cpRecursive {
				fmt.Fprintf(os.Stderr, "cp: %s is a directory (use -r to copy recursively)\n", source)
				os.Exit(1)
			}

			// Check if destination exists
			destInfo, err := vfs.Stat(dest)
			destExists := err == nil

			if sourceInfo.IsDir() {
				if destExists && !destInfo.IsDir() {
					fmt.Fprintf(os.Stderr, "cp: cannot overwrite non-directory %s with directory %s\n", dest, source)
					os.Exit(1)
				}
				err = copyDirectory(vfs, source, dest)
			} else {
				// If destination is a directory, append source filename
				if destExists && destInfo.IsDir() {
					dest = filepath.Join(dest, filepath.Base(source))
				}
				err = copyFile(vfs, source, dest)
			}

			if err != nil {
				logger.Error("Copy failed",
					zap.String("source", source),
					zap.String("dest", dest),
					zap.Error(err))
				fmt.Fprintf(os.Stderr, "cp: %v\n", err)
				os.Exit(1)
			}

			if cpVerbose {
				fmt.Printf("%s -> %s\n", source, dest)
			}
		},
	}

	cmd.Flags().BoolVarP(&cpRecursive, "recursive", "r", false, "copy directories recursively")
	cmd.Flags().BoolVarP(&cpVerbose, "verbose", "v", false, "verbose output")

	return cmd
}

var cpCmd = newCpCmd()

func copyFile(vfs vfs.VFS, source, dest string) error {
	// Open source file
	srcFile, err := vfs.Open(source)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	destFile, err := vfs.Create(dest)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer destFile.Close()

	// Copy data
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	// Sync to ensure data is written
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	// Copy permissions
	srcInfo, _ := srcFile.Stat()
	if srcInfo != nil {
		_ = vfs.Chmod(dest, srcInfo.Mode()) // Best effort to preserve permissions
	}

	return nil
}

func copyDirectory(vfs vfs.VFS, source, dest string) error {
	// Create destination directory
	srcInfo, _ := vfs.Stat(source)
	if err := vfs.MkdirAll(dest, srcInfo.Mode()); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Read source directory
	entries, err := vfs.ReadDir(source)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(source, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			if err := copyDirectory(vfs, srcPath, destPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(vfs, srcPath, destPath); err != nil {
				return err
			}
		}

		if cpVerbose {
			fmt.Printf("%s -> %s\n", srcPath, destPath)
		}
	}

	return nil
}
