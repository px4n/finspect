// Package filesystem provides a local filesystem adaptor for Finspect VFS.
package filesystem

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/px4n/finspect/pkg/vfs"
)

// Adaptor implements the VFS Adaptor interface for local filesystem access.
type Adaptor struct {
	root      string // Root directory for this adaptor
	connected bool
	watches   map[string]*watcher
}

// New creates a new filesystem adaptor.
func New() *Adaptor {
	return &Adaptor{
		watches: make(map[string]*watcher),
	}
}

// Name returns the name of the adaptor.
func (a *Adaptor) Name() string {
	return "filesystem"
}

// Connect establishes a connection to the filesystem.
func (a *Adaptor) Connect(_ context.Context, config map[string]interface{}) error {
	if a.connected {
		return fmt.Errorf("already connected")
	}

	// Get root directory from config
	root, ok := config["root"].(string)
	if !ok {
		root = "/"
	}

	// Verify the root exists and is a directory
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("connect: root is not a directory")
	}

	a.root = filepath.Clean(root)
	a.connected = true
	return nil
}

// Disconnect closes the connection.
func (a *Adaptor) Disconnect() error {
	if !a.connected {
		return nil
	}

	// Stop all watches
	for path, w := range a.watches {
		w.Stop()
		delete(a.watches, path)
	}

	a.connected = false
	return nil
}

// IsConnected returns true if the adaptor is connected.
func (a *Adaptor) IsConnected() bool {
	return a.connected
}

// Capabilities returns the capabilities of this adaptor.
func (a *Adaptor) Capabilities() vfs.Capabilities {
	return vfs.Capabilities{
		CanRead:   true,
		CanWrite:  true,
		CanDelete: true,
		CanMove:   true,
		CanWatch:  true,
		CanLink:   true,
		CanChmod:  true,
		CanChown:  true,
		HasDirs:   true,
	}
}

// resolvePath converts a VFS path to a real filesystem path.
func (a *Adaptor) resolvePath(vfsPath string) string {
	if vfsPath == "/" || vfsPath == "" {
		return a.root
	}
	// Remove leading slash if present and join with root
	if vfsPath[0] == '/' {
		vfsPath = vfsPath[1:]
	}
	return filepath.Join(a.root, vfsPath)
}

// Open opens the named file for reading.
func (a *Adaptor) Open(path string) (vfs.File, error) {
	if !a.connected {
		return nil, vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)

	// Check if it's a directory first
	info, err := os.Stat(realPath)
	if err != nil {
		return nil, mapError(err)
	}
	if info.IsDir() {
		return nil, vfs.ErrIsDirectory
	}

	file, err := os.Open(realPath) // #nosec G304 - path is sanitized by resolvePath
	if err != nil {
		return nil, mapError(err)
	}

	return &File{
		File:    file,
		vfsPath: path,
		adaptor: a,
	}, nil
}

// Create creates or truncates the named file.
func (a *Adaptor) Create(path string) (vfs.File, error) {
	if !a.connected {
		return nil, vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	file, err := os.Create(realPath) // #nosec G304 - path is sanitized by resolvePath
	if err != nil {
		return nil, mapError(err)
	}

	return &File{
		File:    file,
		vfsPath: path,
		adaptor: a,
	}, nil
}

// OpenFile opens a file with specified flags and permissions.
func (a *Adaptor) OpenFile(path string, flag int, perm fs.FileMode) (vfs.File, error) {
	if !a.connected {
		return nil, vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	file, err := os.OpenFile(realPath, flag, perm) // #nosec G304 - path is sanitized by resolvePath
	if err != nil {
		return nil, mapError(err)
	}

	return &File{
		File:    file,
		vfsPath: path,
		adaptor: a,
	}, nil
}

// Remove removes the named file or empty directory.
func (a *Adaptor) Remove(path string) error {
	if !a.connected {
		return vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	return mapError(os.Remove(realPath))
}

// RemoveAll removes the path and any children it contains.
func (a *Adaptor) RemoveAll(path string) error {
	if !a.connected {
		return vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	return mapError(os.RemoveAll(realPath))
}

// Rename renames (moves) a file or directory.
func (a *Adaptor) Rename(oldpath, newpath string) error {
	if !a.connected {
		return vfs.ErrNotConnected
	}

	oldReal := a.resolvePath(oldpath)
	newReal := a.resolvePath(newpath)
	return mapError(os.Rename(oldReal, newReal))
}

// Stat returns information about the named file.
func (a *Adaptor) Stat(path string) (vfs.FileInfo, error) {
	if !a.connected {
		return nil, vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	info, err := os.Stat(realPath)
	if err != nil {
		return nil, mapError(err)
	}

	return &FileInfo{FileInfo: info}, nil
}

// Lstat returns information about the named file or link.
func (a *Adaptor) Lstat(path string) (vfs.FileInfo, error) {
	if !a.connected {
		return nil, vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	info, err := os.Lstat(realPath)
	if err != nil {
		return nil, mapError(err)
	}

	return &FileInfo{FileInfo: info}, nil
}

// ReadDir reads the directory and returns its contents.
func (a *Adaptor) ReadDir(path string) ([]vfs.DirEntry, error) {
	if !a.connected {
		return nil, vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	entries, err := os.ReadDir(realPath)
	if err != nil {
		return nil, mapError(err)
	}

	// Convert to VFS DirEntry
	vfsEntries := make([]vfs.DirEntry, len(entries))
	for i, entry := range entries {
		vfsEntries[i] = &DirEntry{DirEntry: entry}
	}

	return vfsEntries, nil
}

// Mkdir creates a new directory.
func (a *Adaptor) Mkdir(path string, perm fs.FileMode) error {
	if !a.connected {
		return vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	return mapError(os.Mkdir(realPath, perm))
}

// MkdirAll creates a directory and any necessary parents.
func (a *Adaptor) MkdirAll(path string, perm fs.FileMode) error {
	if !a.connected {
		return vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	return mapError(os.MkdirAll(realPath, perm))
}

// Chmod changes the mode of the named file.
func (a *Adaptor) Chmod(path string, mode fs.FileMode) error {
	if !a.connected {
		return vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	return mapError(os.Chmod(realPath, mode))
}

// Chown changes the owner and group of the named file.
func (a *Adaptor) Chown(path string, uid, gid int) error {
	if !a.connected {
		return vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	return mapError(os.Chown(realPath, uid, gid))
}

// Chtimes changes the access and modification times.
func (a *Adaptor) Chtimes(path string, atime, mtime time.Time) error {
	if !a.connected {
		return vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	return mapError(os.Chtimes(realPath, atime, mtime))
}

// Readlink returns the destination of the named symbolic link.
func (a *Adaptor) Readlink(path string) (string, error) {
	if !a.connected {
		return "", vfs.ErrNotConnected
	}

	realPath := a.resolvePath(path)
	target, err := os.Readlink(realPath)
	if err != nil {
		return "", mapError(err)
	}

	// Convert OS path back to VFS path (always use forward slashes)
	if filepath.IsAbs(target) {
		// If target is absolute, try to make it relative to root
		if rel, err := filepath.Rel(a.root, target); err == nil && !strings.HasPrefix(rel, "..") {
			return "/" + filepath.ToSlash(rel), nil
		}
		// Otherwise, just convert slashes
		return filepath.ToSlash(target), nil
	}
	// For relative paths, just convert slashes
	return filepath.ToSlash(target), nil
}

// Symlink creates a symbolic link.
func (a *Adaptor) Symlink(target, link string) error {
	if !a.connected {
		return vfs.ErrNotConnected
	}

	linkReal := a.resolvePath(link)
	return mapError(os.Symlink(target, linkReal))
}

// File wraps os.File to implement vfs.File
type File struct {
	*os.File
	vfsPath string
	adaptor *Adaptor
}

// Stat returns the FileInfo structure describing the file.
func (f *File) Stat() (vfs.FileInfo, error) {
	info, err := f.File.Stat()
	if err != nil {
		return nil, err
	}
	return &FileInfo{FileInfo: info}, nil
}

// Readdir reads the contents of the directory.
func (f *File) Readdir(n int) ([]vfs.FileInfo, error) {
	infos, err := f.File.Readdir(n)
	if err != nil {
		return nil, err
	}

	vfsInfos := make([]vfs.FileInfo, len(infos))
	for i, info := range infos {
		vfsInfos[i] = &FileInfo{FileInfo: info}
	}

	return vfsInfos, nil
}

// FileInfo wraps os.FileInfo to implement vfs.FileInfo
type FileInfo struct {
	os.FileInfo
}

// DirEntry wraps fs.DirEntry to implement vfs.DirEntry
type DirEntry struct {
	fs.DirEntry
}

// Info returns the FileInfo for the entry.
func (d *DirEntry) Info() (vfs.FileInfo, error) {
	info, err := d.DirEntry.Info()
	if err != nil {
		return nil, err
	}
	return &FileInfo{FileInfo: info}, nil
}

// mapError maps OS errors to VFS errors
func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case os.IsNotExist(err):
		return vfs.ErrNotExist
	case os.IsExist(err):
		return vfs.ErrExist
	case os.IsPermission(err):
		return vfs.ErrPermission
	default:
		return err
	}
}
