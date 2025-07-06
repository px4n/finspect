//go:build integration
// +build integration

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIIntegration tests the CLI commands in an integration setting
func TestCLIIntegration(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "finspect-cli-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test structure
	testDir := filepath.Join(tempDir, "testdata")
	err = os.MkdirAll(filepath.Join(testDir, "subdir"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("Content 1"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(testDir, "file2.txt"), []byte("Content 2"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(testDir, "subdir", "file3.txt"), []byte("Content 3"), 0644)
	require.NoError(t, err)

	// Helper to capture command output
	captureOutput := func(f func()) string {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		f()

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		io.Copy(&buf, r)
		return buf.String()
	}

	// Test version command
	t.Run("Version", func(t *testing.T) {
		output := captureOutput(func() {
			rootCmd.SetArgs([]string{"version"})
			rootCmd.Execute()
		})

		assert.Contains(t, output, "finspect version")
		assert.Contains(t, output, "Built:")
	})

	// Test mount command
	t.Run("Mount", func(t *testing.T) {
		// Reset VFS for clean state
		vfsInstance = nil
		vfsOnce = sync.Once{}

		output := captureOutput(func() {
			rootCmd.SetArgs([]string{"mount", "local", testDir, "/test"})
			rootCmd.Execute()
		})

		assert.Contains(t, output, "Mounted")
		assert.Contains(t, output, "/test")
	})

	// Test ls command
	t.Run("List", func(t *testing.T) {
		output := captureOutput(func() {
			rootCmd.SetArgs([]string{"ls", "/test"})
			rootCmd.Execute()
		})

		assert.Contains(t, output, "file1.txt")
		assert.Contains(t, output, "file2.txt")
		assert.Contains(t, output, "subdir/")
	})

	// Test ls with long format
	t.Run("ListLong", func(t *testing.T) {
		output := captureOutput(func() {
			rootCmd.SetArgs([]string{"ls", "-l", "/test"})
			rootCmd.Execute()
		})

		lines := strings.Split(strings.TrimSpace(output), "\n")
		assert.GreaterOrEqual(t, len(lines), 3)

		// Check that it shows file permissions and sizes
		for _, line := range lines {
			if strings.Contains(line, "file1.txt") {
				assert.Contains(t, line, "-rw")
				assert.Contains(t, line, "9") // size of "Content 1"
			}
		}
	})

	// Test stat command
	t.Run("Stat", func(t *testing.T) {
		output := captureOutput(func() {
			rootCmd.SetArgs([]string{"stat", "/test/file1.txt"})
			rootCmd.Execute()
		})

		assert.Contains(t, output, "File: /test/file1.txt")
		assert.Contains(t, output, "Size: 9")
		assert.Contains(t, output, "Type: regular file")
	})

	// Test cp command
	t.Run("Copy", func(t *testing.T) {
		output := captureOutput(func() {
			rootCmd.SetArgs([]string{"cp", "/test/file1.txt", "/test/file1_copy.txt"})
			rootCmd.Execute()
		})

		// Verify copy exists
		copiedFile := filepath.Join(testDir, "file1_copy.txt")
		content, err := os.ReadFile(copiedFile)
		assert.NoError(t, err)
		assert.Equal(t, "Content 1", string(content))
	})

	// Test cp recursive
	t.Run("CopyRecursive", func(t *testing.T) {
		output := captureOutput(func() {
			rootCmd.SetArgs([]string{"cp", "-r", "/test/subdir", "/test/subdir_copy"})
			rootCmd.Execute()
		})

		// Verify recursive copy
		copiedFile := filepath.Join(testDir, "subdir_copy", "file3.txt")
		content, err := os.ReadFile(copiedFile)
		assert.NoError(t, err)
		assert.Equal(t, "Content 3", string(content))
	})

	// Test mv command
	t.Run("Move", func(t *testing.T) {
		// Create a file to move
		err := os.WriteFile(filepath.Join(testDir, "tomove.txt"), []byte("Move me"), 0644)
		require.NoError(t, err)

		output := captureOutput(func() {
			rootCmd.SetArgs([]string{"mv", "/test/tomove.txt", "/test/moved.txt"})
			rootCmd.Execute()
		})

		// Verify old file is gone
		_, err = os.Stat(filepath.Join(testDir, "tomove.txt"))
		assert.True(t, os.IsNotExist(err))

		// Verify new file exists
		content, err := os.ReadFile(filepath.Join(testDir, "moved.txt"))
		assert.NoError(t, err)
		assert.Equal(t, "Move me", string(content))
	})

	// Test rm command
	t.Run("Remove", func(t *testing.T) {
		// Create a file to remove
		err := os.WriteFile(filepath.Join(testDir, "toremove.txt"), []byte("Remove me"), 0644)
		require.NoError(t, err)

		output := captureOutput(func() {
			rootCmd.SetArgs([]string{"rm", "/test/toremove.txt"})
			rootCmd.Execute()
		})

		// Verify file is gone
		_, err = os.Stat(filepath.Join(testDir, "toremove.txt"))
		assert.True(t, os.IsNotExist(err))
	})

	// Test rm recursive
	t.Run("RemoveRecursive", func(t *testing.T) {
		// Create directory to remove
		err := os.MkdirAll(filepath.Join(testDir, "removedir", "subdir"), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(testDir, "removedir", "file.txt"), []byte("data"), 0644)
		require.NoError(t, err)

		output := captureOutput(func() {
			rootCmd.SetArgs([]string{"rm", "-r", "/test/removedir"})
			rootCmd.Execute()
		})

		// Verify directory is gone
		_, err = os.Stat(filepath.Join(testDir, "removedir"))
		assert.True(t, os.IsNotExist(err))
	})
}

// TestCLIErrorHandling tests error conditions
func TestCLIErrorHandling(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "finspect-cli-error-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Reset VFS
	vfsInstance = nil
	vfsOnce = sync.Once{}

	// Mount a test directory
	rootCmd.SetArgs([]string{"mount", "local", tempDir, "/test"})
	rootCmd.Execute()

	// Helper to capture stderr
	captureError := func(f func()) string {
		old := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		f()

		w.Close()
		os.Stderr = old

		var buf bytes.Buffer
		io.Copy(&buf, r)
		return buf.String()
	}

	t.Run("ListNonExistent", func(t *testing.T) {
		output := captureError(func() {
			rootCmd.SetArgs([]string{"ls", "/test/nonexistent"})
			rootCmd.Execute()
		})

		assert.Contains(t, output, "Failed to read directory")
	})

	t.Run("CopyNonExistent", func(t *testing.T) {
		output := captureError(func() {
			rootCmd.SetArgs([]string{"cp", "/test/nonexistent.txt", "/test/dest.txt"})
			rootCmd.Execute()
		})

		assert.Contains(t, output, "Failed to stat source")
	})

	t.Run("RemoveNonExistent", func(t *testing.T) {
		output := captureError(func() {
			rootCmd.SetArgs([]string{"rm", "/test/nonexistent.txt"})
			rootCmd.Execute()
		})

		assert.Contains(t, output, "Failed to stat path")
	})

	t.Run("MoveToExisting", func(t *testing.T) {
		// Create two files
		file1 := filepath.Join(tempDir, "file1.txt")
		file2 := filepath.Join(tempDir, "file2.txt")
		err := os.WriteFile(file1, []byte("file1"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(file2, []byte("file2"), 0644)
		require.NoError(t, err)

		output := captureError(func() {
			rootCmd.SetArgs([]string{"mv", "/test/file1.txt", "/test/file2.txt"})
			rootCmd.Execute()
		})

		assert.Contains(t, output, "destination already exists")
	})

	t.Run("RemoveDirectoryWithoutRecursive", func(t *testing.T) {
		// Create a directory
		err := os.Mkdir(filepath.Join(tempDir, "dir"), 0755)
		require.NoError(t, err)

		output := captureError(func() {
			rootCmd.SetArgs([]string{"rm", "/test/dir"})
			rootCmd.Execute()
		})

		assert.Contains(t, output, "is a directory")
	})
}
