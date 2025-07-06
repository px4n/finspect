// Package vfs provides a virtual filesystem abstraction for Finspect.
// It allows mounting different data sources (local filesystem, cloud storage, etc.)
// and provides a unified POSIX-like interface for accessing them.
package vfs

import (
	"context"
	"io"
	"io/fs"
	"time"
)

// VFS represents the virtual filesystem interface.
// All operations are performed through this interface, which routes
// requests to the appropriate mounted adaptors.
type VFS interface {
	// Mount mounts an adaptor at the specified path.
	// The path must be absolute and start with '/'.
	Mount(path string, adaptor Adaptor) error

	// Unmount removes the adaptor mounted at the specified path.
	Unmount(path string) error

	// ListMounts returns all currently mounted paths and their adaptors.
	ListMounts() map[string]Adaptor

	// Open opens the named file for reading.
	Open(path string) (File, error)

	// Create creates or truncates the named file.
	Create(path string) (File, error)

	// OpenFile opens a file with specified flags and permissions.
	OpenFile(path string, flag int, perm fs.FileMode) (File, error)

	// Remove removes the named file or empty directory.
	Remove(path string) error

	// RemoveAll removes the path and any children it contains.
	RemoveAll(path string) error

	// Rename renames (moves) a file or directory.
	Rename(oldpath, newpath string) error

	// Stat returns information about the named file.
	Stat(path string) (FileInfo, error)

	// Lstat returns information about the named file or link.
	Lstat(path string) (FileInfo, error)

	// ReadDir reads the directory and returns its contents.
	ReadDir(path string) ([]DirEntry, error)

	// Mkdir creates a new directory.
	Mkdir(path string, perm fs.FileMode) error

	// MkdirAll creates a directory and any necessary parents.
	MkdirAll(path string, perm fs.FileMode) error

	// Chmod changes the mode of the named file.
	Chmod(path string, mode fs.FileMode) error

	// Chown changes the owner and group of the named file.
	Chown(path string, uid, gid int) error

	// Chtimes changes the access and modification times.
	Chtimes(path string, atime, mtime time.Time) error

	// Readlink returns the destination of the named symbolic link.
	Readlink(path string) (string, error)

	// Symlink creates a symbolic link.
	Symlink(target, link string) error

	// Close closes the VFS and all mounted adaptors.
	Close() error
}

// File represents an open file in the VFS.
type File interface {
	io.Reader
	io.Writer
	io.Seeker
	io.Closer

	// Stat returns the FileInfo structure describing the file.
	Stat() (FileInfo, error)

	// Sync commits the current contents of the file to stable storage.
	Sync() error

	// Truncate changes the size of the file.
	Truncate(size int64) error

	// Name returns the name of the file.
	Name() string

	// Readdir reads the contents of the directory.
	// If n > 0, it returns at most n entries.
	Readdir(n int) ([]FileInfo, error)

	// Readdirnames reads and returns a slice of names from the directory.
	Readdirnames(n int) ([]string, error)
}

// FileInfo describes a file and is returned by Stat.
type FileInfo interface {
	fs.FileInfo

	// Sys returns the underlying data source (can be nil).
	Sys() interface{}
}

// DirEntry represents an entry read from a directory.
type DirEntry interface {
	// Name returns the name of the file (or subdirectory) described by the entry.
	Name() string

	// IsDir reports whether the entry describes a directory.
	IsDir() bool

	// Type returns the type bits for the entry.
	Type() fs.FileMode

	// Info returns the FileInfo for the file or subdirectory described by the entry.
	Info() (FileInfo, error)
}

// Adaptor represents a data source that can be mounted in the VFS.
// Each adaptor handles operations for a specific type of storage
// (e.g., local filesystem, Google Drive, S3, etc.).
type Adaptor interface {
	// Name returns the name of the adaptor (e.g., "filesystem", "gdrive").
	Name() string

	// Connect establishes a connection to the data source.
	Connect(ctx context.Context, config map[string]interface{}) error

	// Disconnect closes the connection to the data source.
	Disconnect() error

	// IsConnected returns true if the adaptor is connected.
	IsConnected() bool

	// Capabilities returns the capabilities of this adaptor.
	Capabilities() Capabilities

	// Open opens the named file for reading.
	Open(path string) (File, error)

	// Create creates or truncates the named file.
	Create(path string) (File, error)

	// OpenFile opens a file with specified flags and permissions.
	OpenFile(path string, flag int, perm fs.FileMode) (File, error)

	// Remove removes the named file or empty directory.
	Remove(path string) error

	// RemoveAll removes the path and any children it contains.
	RemoveAll(path string) error

	// Rename renames (moves) a file or directory.
	Rename(oldpath, newpath string) error

	// Stat returns information about the named file.
	Stat(path string) (FileInfo, error)

	// Lstat returns information about the named file or link.
	Lstat(path string) (FileInfo, error)

	// ReadDir reads the directory and returns its contents.
	ReadDir(path string) ([]DirEntry, error)

	// Mkdir creates a new directory.
	Mkdir(path string, perm fs.FileMode) error

	// MkdirAll creates a directory and any necessary parents.
	MkdirAll(path string, perm fs.FileMode) error

	// Chmod changes the mode of the named file.
	Chmod(path string, mode fs.FileMode) error

	// Chown changes the owner and group of the named file.
	Chown(path string, uid, gid int) error

	// Chtimes changes the access and modification times.
	Chtimes(path string, atime, mtime time.Time) error

	// Readlink returns the destination of the named symbolic link.
	Readlink(path string) (string, error)

	// Symlink creates a symbolic link.
	Symlink(target, link string) error

	// Watch watches for changes in the specified path.
	// If recursive is true, it watches all subdirectories.
	Watch(path string, recursive bool, events chan<- Event) error

	// Unwatch stops watching the specified path.
	Unwatch(path string) error
}

// Capabilities describes what operations an adaptor supports.
type Capabilities struct {
	CanRead   bool // Supports reading files
	CanWrite  bool // Supports writing files
	CanDelete bool // Supports deleting files
	CanMove   bool // Supports moving/renaming files
	CanWatch  bool // Supports file watching
	CanLink   bool // Supports symbolic links
	CanChmod  bool // Supports changing permissions
	CanChown  bool // Supports changing ownership
	HasDirs   bool // Has directory concept
}

// Event represents a filesystem event.
type Event struct {
	// Type is the type of event that occurred.
	Type EventType

	// Path is the path that triggered the event.
	Path string

	// OldPath is used for rename events.
	OldPath string

	// Time is when the event occurred.
	Time time.Time

	// Source is the adaptor that generated the event.
	Source string

	// Error is any error associated with the event.
	Error error
}

// EventType represents the type of filesystem event.
type EventType int

const (
	// EventUnknown indicates an unknown event type.
	EventUnknown EventType = iota

	// EventCreate indicates a file or directory was created.
	EventCreate

	// EventWrite indicates a file was written to.
	EventWrite

	// EventRemove indicates a file or directory was removed.
	EventRemove

	// EventRename indicates a file or directory was renamed.
	EventRename

	// EventChmod indicates permissions were changed.
	EventChmod

	// EventError indicates an error occurred while watching.
	EventError
)

// String returns the string representation of an EventType.
func (e EventType) String() string {
	switch e {
	case EventCreate:
		return "CREATE"
	case EventWrite:
		return "WRITE"
	case EventRemove:
		return "REMOVE"
	case EventRename:
		return "RENAME"
	case EventChmod:
		return "CHMOD"
	case EventError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
