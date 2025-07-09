package vfs_test

import (
	"context"
	"io"
	"io/fs"
	"testing"
	"time"

	"github.com/px4n/finspect/pkg/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAdaptor implements a simple in-memory adaptor for testing
type mockAdaptor struct {
	name      string
	connected bool
	files     map[string]*mockFile
	caps      vfs.Capabilities
}

type mockFile struct {
	name    string
	content []byte
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func newMockAdaptor(name string) *mockAdaptor {
	return &mockAdaptor{
		name:      name,
		connected: true,
		files: map[string]*mockFile{
			"/": {name: "/", isDir: true, mode: 0o755},
		},
		caps: vfs.Capabilities{
			CanRead:   true,
			CanWrite:  true,
			CanDelete: true,
			CanMove:   true,
			CanWatch:  false,
			CanLink:   false,
			CanChmod:  true,
			CanChown:  false,
			HasDirs:   true,
		},
	}
}

func (m *mockAdaptor) Name() string                                              { return m.name }
func (m *mockAdaptor) IsConnected() bool                                         { return m.connected }
func (m *mockAdaptor) Capabilities() vfs.Capabilities                            { return m.caps }
func (m *mockAdaptor) Connect(_ context.Context, _ map[string]interface{}) error { return nil }
func (m *mockAdaptor) Disconnect() error {
	m.connected = false
	return nil
}

func (m *mockAdaptor) Open(path string) (vfs.File, error) {
	f, exists := m.files[path]
	if !exists {
		return nil, vfs.ErrNotExist
	}
	if f.isDir {
		return nil, vfs.ErrIsDirectory
	}
	return &mockFileHandle{
		mockFile: f,
		content:  f.content,
		pos:      0,
	}, nil
}

func (m *mockAdaptor) Create(path string) (vfs.File, error) {
	f := &mockFile{
		name:    path,
		content: []byte{},
		mode:    0o644,
		modTime: time.Now(),
		isDir:   false,
	}
	m.files[path] = f
	return &mockFileHandle{
		mockFile: f,
		adaptor:  m,
		content:  f.content,
		pos:      0,
	}, nil
}

func (m *mockAdaptor) Remove(path string) error {
	if _, exists := m.files[path]; !exists {
		return vfs.ErrNotExist
	}
	delete(m.files, path)
	return nil
}

func (m *mockAdaptor) Stat(path string) (vfs.FileInfo, error) {
	f, exists := m.files[path]
	if !exists {
		return nil, vfs.ErrNotExist
	}
	return &mockFileInfo{mockFile: f}, nil
}

func (m *mockAdaptor) ReadDir(path string) ([]vfs.DirEntry, error) {
	f, exists := m.files[path]
	if !exists {
		return nil, vfs.ErrNotExist
	}
	if !f.isDir {
		return nil, vfs.ErrNotDirectory
	}

	var entries []vfs.DirEntry
	// Simple implementation: just return some test entries
	if path == "/" {
		entries = append(entries, &mockDirEntry{name: "test.txt", isDir: false})
		entries = append(entries, &mockDirEntry{name: "subdir", isDir: true})
	}
	return entries, nil
}

// Implement other required methods with basic functionality
func (m *mockAdaptor) OpenFile(path string, _ int, _ fs.FileMode) (vfs.File, error) {
	return m.Open(path)
}
func (m *mockAdaptor) RemoveAll(path string) error { return m.Remove(path) }
func (m *mockAdaptor) Rename(oldpath, newpath string) error {
	if f, exists := m.files[oldpath]; exists {
		m.files[newpath] = f
		delete(m.files, oldpath)
		return nil
	}
	return vfs.ErrNotExist
}
func (m *mockAdaptor) Lstat(path string) (vfs.FileInfo, error) { return m.Stat(path) }
func (m *mockAdaptor) Mkdir(path string, perm fs.FileMode) error {
	m.files[path] = &mockFile{name: path, isDir: true, mode: perm}
	return nil
}
func (m *mockAdaptor) MkdirAll(path string, perm fs.FileMode) error { return m.Mkdir(path, perm) }
func (m *mockAdaptor) Chmod(_ string, _ fs.FileMode) error          { return nil }
func (m *mockAdaptor) Chown(_ string, _, _ int) error               { return nil }
func (m *mockAdaptor) Chtimes(_ string, _, _ time.Time) error       { return nil }
func (m *mockAdaptor) Readlink(_ string) (string, error)            { return "", vfs.ErrNotSupported }
func (m *mockAdaptor) Symlink(_, _ string) error                    { return vfs.ErrNotSupported }
func (m *mockAdaptor) Watch(_ string, _ bool, _ chan<- vfs.Event) error {
	return vfs.ErrNotSupported
}
func (m *mockAdaptor) Unwatch(_ string) error { return vfs.ErrNotSupported }

// Mock file handle implementation
type mockFileHandle struct {
	*mockFile
	adaptor *mockAdaptor
	content []byte
	pos     int
}

func (f *mockFileHandle) Read(p []byte) (n int, err error) {
	if f.pos >= len(f.content) {
		return 0, io.EOF
	}
	n = copy(p, f.content[f.pos:])
	f.pos += n
	return n, nil
}

func (f *mockFileHandle) Write(p []byte) (n int, err error) {
	f.content = append(f.content, p...)
	if f.adaptor != nil {
		f.mockFile.content = f.content
	}
	return len(p), nil
}

func (f *mockFileHandle) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0: // SEEK_SET
		f.pos = int(offset)
	case 1: // SEEK_CUR
		f.pos += int(offset)
	case 2: // SEEK_END
		f.pos = len(f.content) + int(offset)
	}
	return int64(f.pos), nil
}

func (f *mockFileHandle) Close() error                          { return nil }
func (f *mockFileHandle) Stat() (vfs.FileInfo, error)           { return &mockFileInfo{f.mockFile}, nil }
func (f *mockFileHandle) Sync() error                           { return nil }
func (f *mockFileHandle) Truncate(_ int64) error                { return nil }
func (f *mockFileHandle) Name() string                          { return f.name }
func (f *mockFileHandle) Readdir(_ int) ([]vfs.FileInfo, error) { return nil, nil }
func (f *mockFileHandle) Readdirnames(_ int) ([]string, error)  { return nil, nil }

// Mock file info
type mockFileInfo struct {
	*mockFile
}

func (i *mockFileInfo) Name() string       { return i.mockFile.name }
func (i *mockFileInfo) Size() int64        { return int64(len(i.mockFile.content)) }
func (i *mockFileInfo) Mode() fs.FileMode  { return i.mockFile.mode }
func (i *mockFileInfo) ModTime() time.Time { return i.mockFile.modTime }
func (i *mockFileInfo) IsDir() bool        { return i.mockFile.isDir }
func (i *mockFileInfo) Sys() interface{}   { return nil }

// Mock dir entry
type mockDirEntry struct {
	name  string
	isDir bool
}

func (e *mockDirEntry) Name() string { return e.name }
func (e *mockDirEntry) IsDir() bool  { return e.isDir }
func (e *mockDirEntry) Type() fs.FileMode {
	if e.isDir {
		return fs.ModeDir
	}
	return 0
}

func (e *mockDirEntry) Info() (vfs.FileInfo, error) {
	return &mockFileInfo{
		mockFile: &mockFile{
			name:  e.name,
			isDir: e.isDir,
			mode:  0o755,
		},
	}, nil
}

// Tests

func TestRouterMount(t *testing.T) {
	router := vfs.NewRouter()
	adaptor := newMockAdaptor("test")

	// Test mounting
	err := router.Mount("/test", adaptor)
	assert.NoError(t, err)

	// Test duplicate mount
	err = router.Mount("/test", adaptor)
	assert.ErrorIs(t, err, vfs.ErrAlreadyMounted)

	// Test invalid path
	err = router.Mount("relative/path", adaptor)
	assert.ErrorIs(t, err, vfs.ErrInvalidPath)

	// Test disconnected adaptor
	disconnected := newMockAdaptor("disconnected")
	disconnected.connected = false
	err = router.Mount("/disconnected", disconnected)
	assert.ErrorIs(t, err, vfs.ErrNotConnected)
}

func TestRouterUnmount(t *testing.T) {
	router := vfs.NewRouter()
	adaptor := newMockAdaptor("test")

	// Mount first
	err := router.Mount("/test", adaptor)
	require.NoError(t, err)

	// Test unmount
	err = router.Unmount("/test")
	assert.NoError(t, err)

	// Test unmount non-existent
	err = router.Unmount("/nonexistent")
	assert.ErrorIs(t, err, vfs.ErrNotMounted)
}

func TestRouterListMounts(t *testing.T) {
	router := vfs.NewRouter()
	adaptor1 := newMockAdaptor("test1")
	adaptor2 := newMockAdaptor("test2")

	_ = router.Mount("/mount1", adaptor1)
	_ = router.Mount("/mount2", adaptor2)

	mounts := router.ListMounts()
	assert.Len(t, mounts, 2)
	assert.Equal(t, adaptor1, mounts["/mount1"])
	assert.Equal(t, adaptor2, mounts["/mount2"])
}

func TestRouterFileOperations(t *testing.T) {
	router := vfs.NewRouter()
	adaptor := newMockAdaptor("test")
	_ = router.Mount("/test", adaptor)

	// Create a file
	file, err := router.Create("/test/newfile.txt")
	require.NoError(t, err)
	defer file.Close()

	// Write data
	data := []byte("Hello, VFS!")
	n, err := file.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	// Close and reopen
	file.Close()
	file, err = router.Open("/test/newfile.txt")
	require.NoError(t, err)
	defer file.Close()

	// Read data
	buf := make([]byte, len(data))
	n, err = file.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, buf)
}

func TestRouterStat(t *testing.T) {
	router := vfs.NewRouter()
	adaptor := newMockAdaptor("test")
	_ = router.Mount("/test", adaptor)

	// Create a file in the mock adaptor
	adaptor.files["/testfile.txt"] = &mockFile{
		name:    "testfile.txt",
		content: []byte("test content"),
		mode:    0o644,
		modTime: time.Now(),
		isDir:   false,
	}

	// Stat the file
	info, err := router.Stat("/test/testfile.txt")
	require.NoError(t, err)
	assert.Equal(t, "testfile.txt", info.Name())
	assert.Equal(t, int64(12), info.Size())
	assert.False(t, info.IsDir())
}

func TestRouterReadDir(t *testing.T) {
	router := vfs.NewRouter()
	adaptor := newMockAdaptor("test")
	_ = router.Mount("/test", adaptor)

	// Read directory
	entries, err := router.ReadDir("/test")
	require.NoError(t, err)
	assert.Len(t, entries, 2) // test.txt and subdir from mock

	// Test reading root directory
	entries, err = router.ReadDir("/")
	require.NoError(t, err)
	assert.Greater(t, len(entries), 0)
}

func TestRouterRename(t *testing.T) {
	router := vfs.NewRouter()
	adaptor := newMockAdaptor("test")
	_ = router.Mount("/test", adaptor)

	// Create a file
	adaptor.files["/oldname.txt"] = &mockFile{
		name:    "oldname.txt",
		content: []byte("content"),
		mode:    0o644,
		modTime: time.Now(),
		isDir:   false,
	}

	// Rename within same mount
	err := router.Rename("/test/oldname.txt", "/test/newname.txt")
	assert.NoError(t, err)

	// Verify old name is gone
	_, err = router.Stat("/test/oldname.txt")
	assert.Error(t, err)

	// Verify new name exists
	_, err = router.Stat("/test/newname.txt")
	assert.NoError(t, err)
}

func TestRouterCrossDeviceRename(t *testing.T) {
	router := vfs.NewRouter()
	adaptor1 := newMockAdaptor("test1")
	adaptor2 := newMockAdaptor("test2")

	_ = router.Mount("/mount1", adaptor1)
	_ = router.Mount("/mount2", adaptor2)

	// Create a file in mount1
	adaptor1.files["/file.txt"] = &mockFile{
		name:    "file.txt",
		content: []byte("content"),
		mode:    0o644,
		modTime: time.Now(),
		isDir:   false,
	}

	// Try to rename across mounts
	err := router.Rename("/mount1/file.txt", "/mount2/file.txt")
	assert.ErrorIs(t, err, vfs.ErrCrossDevice)
}

func TestRouterClose(t *testing.T) {
	router := vfs.NewRouter()
	adaptor := newMockAdaptor("test")
	_ = router.Mount("/test", adaptor)

	// Close router
	err := router.Close()
	assert.NoError(t, err)

	// Operations should fail after close
	_, err = router.Open("/test/file.txt")
	assert.ErrorIs(t, err, vfs.ErrClosed)
}
