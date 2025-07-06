package filesystem_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/px4n/finspect/adaptors/filesystem"
	"github.com/px4n/finspect/pkg/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemAdaptor(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "finspect-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test structure
	testDir := filepath.Join(tempDir, "testdir")
	err = os.Mkdir(testDir, 0o755)
	require.NoError(t, err)

	testFile := filepath.Join(tempDir, "testfile.txt")
	err = os.WriteFile(testFile, []byte("Hello, Finspect!"), 0o644)
	require.NoError(t, err)

	// Create adaptor
	adaptor := filesystem.New()

	t.Run("Connect", func(t *testing.T) {
		// Test connection
		config := map[string]interface{}{
			"root": tempDir,
		}
		err := adaptor.Connect(context.Background(), config)
		assert.NoError(t, err)
		assert.True(t, adaptor.IsConnected())

		// Test double connect
		err = adaptor.Connect(context.Background(), config)
		assert.Error(t, err)
	})

	t.Run("Capabilities", func(t *testing.T) {
		caps := adaptor.Capabilities()
		assert.True(t, caps.CanRead)
		assert.True(t, caps.CanWrite)
		assert.True(t, caps.CanDelete)
		assert.True(t, caps.CanMove)
		assert.True(t, caps.CanWatch)
		assert.True(t, caps.HasDirs)
	})

	t.Run("Open", func(t *testing.T) {
		file, err := adaptor.Open("/testfile.txt")
		require.NoError(t, err)
		defer file.Close()

		// Read content
		content, err := io.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Finspect!", string(content))

		// Test opening non-existent file
		_, err = adaptor.Open("/nonexistent.txt")
		assert.ErrorIs(t, err, vfs.ErrNotExist)
	})

	t.Run("Create", func(t *testing.T) {
		file, err := adaptor.Create("/newfile.txt")
		require.NoError(t, err)
		defer file.Close()

		// Write content
		_, err = file.Write([]byte("New content"))
		assert.NoError(t, err)

		// Verify file exists
		realPath := filepath.Join(tempDir, "newfile.txt")
		content, err := os.ReadFile(realPath)
		assert.NoError(t, err)
		assert.Equal(t, "New content", string(content))
	})

	t.Run("Stat", func(t *testing.T) {
		info, err := adaptor.Stat("/testfile.txt")
		require.NoError(t, err)
		assert.Equal(t, "testfile.txt", info.Name())
		assert.Equal(t, int64(16), info.Size())
		assert.False(t, info.IsDir())

		// Test directory
		info, err = adaptor.Stat("/testdir")
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Test non-existent
		_, err = adaptor.Stat("/nonexistent")
		assert.ErrorIs(t, err, vfs.ErrNotExist)
	})

	t.Run("ReadDir", func(t *testing.T) {
		entries, err := adaptor.ReadDir("/")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 2) // At least testdir and testfile.txt

		// Check entries
		names := make(map[string]bool)
		for _, entry := range entries {
			names[entry.Name()] = true
		}
		assert.True(t, names["testdir"])
		assert.True(t, names["testfile.txt"])
	})

	t.Run("Mkdir", func(t *testing.T) {
		err := adaptor.Mkdir("/newdir", 0o755)
		assert.NoError(t, err)

		// Verify directory exists
		info, err := adaptor.Stat("/newdir")
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Test creating existing directory
		err = adaptor.Mkdir("/newdir", 0o755)
		assert.Error(t, err)
	})

	t.Run("Remove", func(t *testing.T) {
		// Create a file to remove
		err := os.WriteFile(filepath.Join(tempDir, "removeme.txt"), []byte("remove"), 0o644)
		require.NoError(t, err)

		// Remove it
		err = adaptor.Remove("/removeme.txt")
		assert.NoError(t, err)

		// Verify it's gone
		_, err = adaptor.Stat("/removeme.txt")
		assert.ErrorIs(t, err, vfs.ErrNotExist)

		// Test removing non-existent
		err = adaptor.Remove("/nonexistent.txt")
		assert.ErrorIs(t, err, vfs.ErrNotExist)
	})

	t.Run("Rename", func(t *testing.T) {
		// Create a file to rename
		err := os.WriteFile(filepath.Join(tempDir, "oldname.txt"), []byte("rename me"), 0o644)
		require.NoError(t, err)

		// Rename it
		err = adaptor.Rename("/oldname.txt", "/newname.txt")
		assert.NoError(t, err)

		// Verify old name is gone
		_, err = adaptor.Stat("/oldname.txt")
		assert.ErrorIs(t, err, vfs.ErrNotExist)

		// Verify new name exists
		info, err := adaptor.Stat("/newname.txt")
		require.NoError(t, err)
		assert.Equal(t, "newname.txt", info.Name())
	})

	t.Run("Chmod", func(t *testing.T) {
		// Create a file
		testChmod := filepath.Join(tempDir, "chmod.txt")
		err := os.WriteFile(testChmod, []byte("chmod"), 0o644)
		require.NoError(t, err)

		// Change permissions
		err = adaptor.Chmod("/chmod.txt", 0o600)
		assert.NoError(t, err)

		// Verify permissions changed
		info, err := os.Stat(testChmod)
		require.NoError(t, err)
		// On Windows, file permissions work differently
		if runtime.GOOS != "windows" {
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		}
	})

	t.Run("Chtimes", func(t *testing.T) {
		// Create a file
		err := os.WriteFile(filepath.Join(tempDir, "times.txt"), []byte("times"), 0o644)
		require.NoError(t, err)

		// Set specific times
		atime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		mtime := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)

		err = adaptor.Chtimes("/times.txt", atime, mtime)
		assert.NoError(t, err)

		// Verify modification time
		info, err := adaptor.Stat("/times.txt")
		require.NoError(t, err)
		assert.Equal(t, mtime.Unix(), info.ModTime().Unix())
	})

	t.Run("Disconnect", func(t *testing.T) {
		err := adaptor.Disconnect()
		assert.NoError(t, err)
		assert.False(t, adaptor.IsConnected())

		// Operations should fail after disconnect
		_, err = adaptor.Open("/testfile.txt")
		assert.ErrorIs(t, err, vfs.ErrNotConnected)
	})
}

func TestFilesystemAdaptorWatch(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "finspect-watch-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create and connect adaptor
	adaptor := filesystem.New()
	config := map[string]interface{}{
		"root": tempDir,
	}
	err = adaptor.Connect(context.Background(), config)
	require.NoError(t, err)
	defer adaptor.Disconnect()

	t.Run("WatchFile", func(t *testing.T) {
		events := make(chan vfs.Event, 10)

		// Start watching
		err := adaptor.Watch("/", false, events)
		assert.NoError(t, err)
		defer adaptor.Unwatch("/")

		// Give watcher time to start
		time.Sleep(100 * time.Millisecond)

		// Create a file
		testFile := filepath.Join(tempDir, "watchtest.txt")
		err = os.WriteFile(testFile, []byte("watch me"), 0o644)
		require.NoError(t, err)

		// Wait for event(s) - os.WriteFile might trigger both CREATE and WRITE
		timeout := time.After(2 * time.Second)
		gotCreate := false
		for !gotCreate {
			select {
			case event := <-events:
				if event.Type == vfs.EventCreate && event.Path == "/watchtest.txt" {
					gotCreate = true
					assert.Equal(t, "filesystem", event.Source)
				}
				// Drain any additional events from the initial write
			case <-timeout:
				t.Fatal("Timeout waiting for create event")
			}
		}

		// Give filesystem time to settle
		time.Sleep(100 * time.Millisecond)

		// Clear any remaining events
		for len(events) > 0 {
			<-events
		}

		// Modify the file
		err = os.WriteFile(testFile, []byte("modified"), 0o644)
		require.NoError(t, err)

		// Wait for write event
		timeout = time.After(2 * time.Second)
		gotWrite := false
		for !gotWrite {
			select {
			case event := <-events:
				if event.Type == vfs.EventWrite && event.Path == "/watchtest.txt" {
					gotWrite = true
				}
			case <-timeout:
				t.Fatal("Timeout waiting for write event")
			}
		}

		// Give filesystem time to settle
		time.Sleep(100 * time.Millisecond)

		// Clear any remaining events
		for len(events) > 0 {
			<-events
		}

		// Remove the file
		err = os.Remove(testFile)
		require.NoError(t, err)

		// Wait for remove event
		select {
		case event := <-events:
			assert.Equal(t, vfs.EventRemove, event.Type)
			assert.Equal(t, "/watchtest.txt", event.Path)
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for remove event")
		}
	})

	t.Run("UnwatchStopsEvents", func(t *testing.T) {
		events := make(chan vfs.Event, 10)

		// Start watching
		err := adaptor.Watch("/", false, events)
		assert.NoError(t, err)

		// Stop watching
		err = adaptor.Unwatch("/")
		assert.NoError(t, err)

		// Give unwatch time to take effect
		time.Sleep(100 * time.Millisecond)

		// Create a file - should not generate event
		err = os.WriteFile(filepath.Join(tempDir, "nowatching.txt"), []byte("no event"), 0o644)
		require.NoError(t, err)

		// Should not receive any event
		select {
		case event := <-events:
			t.Fatalf("Unexpected event after unwatch: %v", event)
		case <-time.After(500 * time.Millisecond):
			// Expected: no event
		}
	})
}

func TestFilesystemAdaptorEdgeCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "finspect-edge-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	adaptor := filesystem.New()
	config := map[string]interface{}{
		"root": tempDir,
	}
	err = adaptor.Connect(context.Background(), config)
	require.NoError(t, err)
	defer adaptor.Disconnect()

	t.Run("SymlinkSupport", func(t *testing.T) {
		// Create a regular file
		target := filepath.Join(tempDir, "target.txt")
		err := os.WriteFile(target, []byte("target"), 0o644)
		require.NoError(t, err)

		// Try to create symlink
		err = adaptor.Symlink("/target.txt", "/link.txt")

		// Symlinks may not work on all systems (e.g., Windows without admin rights)
		if err != nil {
			t.Skip("Symlinks not supported on this system")
			return
		}

		// Try to read symlink
		linkTarget, err := adaptor.Readlink("/link.txt")
		require.NoError(t, err)

		// The returned path should always use forward slashes
		assert.Equal(t, "/target.txt", linkTarget)
	})

	t.Run("RemoveNonEmptyDirectory", func(t *testing.T) {
		// Create directory with file
		err := adaptor.Mkdir("/nonempty", 0o755)
		require.NoError(t, err)

		file, err := adaptor.Create("/nonempty/file.txt")
		require.NoError(t, err)
		file.Close()

		// Try to remove non-empty directory
		err = adaptor.Remove("/nonempty")
		assert.Error(t, err) // Should fail

		// RemoveAll should work
		err = adaptor.RemoveAll("/nonempty")
		assert.NoError(t, err)

		// Verify it's gone
		_, err = adaptor.Stat("/nonempty")
		assert.ErrorIs(t, err, vfs.ErrNotExist)
	})

	t.Run("OpenDirectory", func(t *testing.T) {
		// Try to open a directory as a file
		_, err := adaptor.Open("/")
		assert.Error(t, err)
	})
}
