package vfs

import (
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

// Router implements the VFS interface and routes operations to mounted adaptors.
type Router struct {
	mu      sync.RWMutex
	mounts  map[string]Adaptor // Map of mount points to adaptors
	paths   []string           // Sorted list of mount paths for longest-prefix matching
	closed  bool
	watches map[string]chan<- Event // Active watches
}

// NewRouter creates a new VFS router.
func NewRouter() *Router {
	return &Router{
		mounts:  make(map[string]Adaptor),
		paths:   make([]string, 0),
		watches: make(map[string]chan<- Event),
	}
}

// Mount mounts an adaptor at the specified path.
func (r *Router) Mount(mountPath string, adaptor Adaptor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrClosed
	}

	// Validate and normalize the path
	mountPath = path.Clean(mountPath)
	if !path.IsAbs(mountPath) {
		return &PathError{Op: "mount", Path: mountPath, Err: ErrInvalidPath}
	}

	// Check if already mounted
	if _, exists := r.mounts[mountPath]; exists {
		return &PathError{Op: "mount", Path: mountPath, Err: ErrAlreadyMounted}
	}

	// Check if adaptor is connected
	if !adaptor.IsConnected() {
		return &PathError{Op: "mount", Path: mountPath, Err: ErrNotConnected}
	}

	// Mount the adaptor
	r.mounts[mountPath] = adaptor
	r.paths = append(r.paths, mountPath)

	// Sort paths by length (longest first) for proper prefix matching
	sort.Slice(r.paths, func(i, j int) bool {
		return len(r.paths[i]) > len(r.paths[j])
	})

	return nil
}

// Unmount removes the adaptor mounted at the specified path.
func (r *Router) Unmount(mountPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrClosed
	}

	mountPath = path.Clean(mountPath)

	// Check if mounted
	if _, exists := r.mounts[mountPath]; !exists {
		return &PathError{Op: "unmount", Path: mountPath, Err: ErrNotMounted}
	}

	// Remove from mounts
	delete(r.mounts, mountPath)

	// Remove from sorted paths
	for i, p := range r.paths {
		if p == mountPath {
			r.paths = append(r.paths[:i], r.paths[i+1:]...)
			break
		}
	}

	return nil
}

// ListMounts returns all currently mounted paths and their adaptors.
func (r *Router) ListMounts() map[string]Adaptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mounts := make(map[string]Adaptor)
	for path, adaptor := range r.mounts {
		mounts[path] = adaptor
	}
	return mounts
}

// resolve finds the adaptor and relative path for a given absolute path.
func (r *Router) resolve(absPath string) (Adaptor, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return nil, "", ErrClosed
	}

	absPath = path.Clean(absPath)
	if !path.IsAbs(absPath) {
		return nil, "", &PathError{Op: "resolve", Path: absPath, Err: ErrInvalidPath}
	}

	// Find the longest matching mount point
	for _, mountPath := range r.paths {
		if absPath == mountPath ||
			strings.HasPrefix(absPath, mountPath+"/") ||
			(mountPath == "/" && strings.HasPrefix(absPath, "/")) {
			adaptor := r.mounts[mountPath]
			relPath := strings.TrimPrefix(absPath, mountPath)
			if relPath == "" {
				relPath = "/"
			}
			return adaptor, relPath, nil
		}
	}

	return nil, "", &PathError{Op: "resolve", Path: absPath, Err: ErrNotMounted}
}

// Open opens the named file for reading.
func (r *Router) Open(name string) (File, error) {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return nil, err
	}

	file, err := adaptor.Open(relPath)
	if err != nil {
		return nil, &PathError{Op: "open", Path: name, Err: err}
	}

	return &routedFile{
		File:    file,
		absPath: name,
		router:  r,
	}, nil
}

// Create creates or truncates the named file.
func (r *Router) Create(name string) (File, error) {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return nil, err
	}

	if !adaptor.Capabilities().CanWrite {
		return nil, &PathError{Op: "create", Path: name, Err: ErrReadOnly}
	}

	file, err := adaptor.Create(relPath)
	if err != nil {
		return nil, &PathError{Op: "create", Path: name, Err: err}
	}

	return &routedFile{
		File:    file,
		absPath: name,
		router:  r,
	}, nil
}

// OpenFile opens a file with specified flags and permissions.
func (r *Router) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return nil, err
	}

	// Check capabilities based on flags
	if flag&(OWronly|ORdwr|OCreate|OTrunc) != 0 && !adaptor.Capabilities().CanWrite {
		return nil, &PathError{Op: "open", Path: name, Err: ErrReadOnly}
	}

	file, err := adaptor.OpenFile(relPath, flag, perm)
	if err != nil {
		return nil, &PathError{Op: "open", Path: name, Err: err}
	}

	return &routedFile{
		File:    file,
		absPath: name,
		router:  r,
	}, nil
}

// Remove removes the named file or empty directory.
func (r *Router) Remove(name string) error {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return err
	}

	if !adaptor.Capabilities().CanDelete {
		return &PathError{Op: "remove", Path: name, Err: ErrNotSupported}
	}

	if err := adaptor.Remove(relPath); err != nil {
		return &PathError{Op: "remove", Path: name, Err: err}
	}

	return nil
}

// RemoveAll removes the path and any children it contains.
func (r *Router) RemoveAll(name string) error {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return err
	}

	if !adaptor.Capabilities().CanDelete {
		return &PathError{Op: "remove", Path: name, Err: ErrNotSupported}
	}

	if err := adaptor.RemoveAll(relPath); err != nil {
		return &PathError{Op: "remove", Path: name, Err: err}
	}

	return nil
}

// Rename renames (moves) a file or directory.
func (r *Router) Rename(oldpath, newpath string) error {
	oldAdaptor, oldRelPath, err := r.resolve(oldpath)
	if err != nil {
		return err
	}

	newAdaptor, newRelPath, err := r.resolve(newpath)
	if err != nil {
		return err
	}

	// Check if both paths are on the same adaptor
	if oldAdaptor != newAdaptor {
		return &LinkError{Op: "rename", Old: oldpath, New: newpath, Err: ErrCrossDevice}
	}

	if !oldAdaptor.Capabilities().CanMove {
		return &LinkError{Op: "rename", Old: oldpath, New: newpath, Err: ErrNotSupported}
	}

	if err := oldAdaptor.Rename(oldRelPath, newRelPath); err != nil {
		return &LinkError{Op: "rename", Old: oldpath, New: newpath, Err: err}
	}

	return nil
}

// Stat returns information about the named file.
func (r *Router) Stat(name string) (FileInfo, error) {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return nil, err
	}

	info, err := adaptor.Stat(relPath)
	if err != nil {
		return nil, &PathError{Op: "stat", Path: name, Err: err}
	}

	return info, nil
}

// Lstat returns information about the named file or link.
func (r *Router) Lstat(name string) (FileInfo, error) {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return nil, err
	}

	info, err := adaptor.Lstat(relPath)
	if err != nil {
		return nil, &PathError{Op: "lstat", Path: name, Err: err}
	}

	return info, nil
}

// ReadDir reads the directory and returns its contents.
func (r *Router) ReadDir(name string) ([]DirEntry, error) {
	// Special case: if we have a root mount and asking for root dir,
	// delegate to the root mount instead of listing mount points
	r.mu.RLock()
	_, hasRootMount := r.mounts["/"]
	r.mu.RUnlock()

	if name == "/" && !hasRootMount {
		// No root mount, list mount points
		return r.readRootDir()
	}

	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return nil, err
	}

	entries, err := adaptor.ReadDir(relPath)
	if err != nil {
		return nil, &PathError{Op: "readdir", Path: name, Err: err}
	}

	return entries, nil
}

// readRootDir returns the mount points as directory entries.
func (r *Router) readRootDir() ([]DirEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var entries []DirEntry
	for mountPath := range r.mounts {
		// Skip the root mount itself
		if mountPath == "/" {
			continue
		}

		// Extract the first component of the mount path
		parts := strings.Split(strings.TrimPrefix(mountPath, "/"), "/")
		if len(parts) > 0 && parts[0] != "" {
			entries = append(entries, &mountDirEntry{
				name: parts[0],
				mode: fs.ModeDir | 0o755,
			})
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := entries[:0]
	for _, entry := range entries {
		if !seen[entry.Name()] {
			seen[entry.Name()] = true
			unique = append(unique, entry)
		}
	}

	return unique, nil
}

// Mkdir creates a new directory.
func (r *Router) Mkdir(name string, perm fs.FileMode) error {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return err
	}

	if !adaptor.Capabilities().CanWrite {
		return &PathError{Op: "mkdir", Path: name, Err: ErrReadOnly}
	}

	if err := adaptor.Mkdir(relPath, perm); err != nil {
		return &PathError{Op: "mkdir", Path: name, Err: err}
	}

	return nil
}

// MkdirAll creates a directory and any necessary parents.
func (r *Router) MkdirAll(name string, perm fs.FileMode) error {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return err
	}

	if !adaptor.Capabilities().CanWrite {
		return &PathError{Op: "mkdir", Path: name, Err: ErrReadOnly}
	}

	if err := adaptor.MkdirAll(relPath, perm); err != nil {
		return &PathError{Op: "mkdir", Path: name, Err: err}
	}

	return nil
}

// Chmod changes the mode of the named file.
func (r *Router) Chmod(name string, mode fs.FileMode) error {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return err
	}

	if !adaptor.Capabilities().CanChmod {
		return &PathError{Op: "chmod", Path: name, Err: ErrNotSupported}
	}

	if err := adaptor.Chmod(relPath, mode); err != nil {
		return &PathError{Op: "chmod", Path: name, Err: err}
	}

	return nil
}

// Chown changes the owner and group of the named file.
func (r *Router) Chown(name string, uid, gid int) error {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return err
	}

	if !adaptor.Capabilities().CanChown {
		return &PathError{Op: "chown", Path: name, Err: ErrNotSupported}
	}

	if err := adaptor.Chown(relPath, uid, gid); err != nil {
		return &PathError{Op: "chown", Path: name, Err: err}
	}

	return nil
}

// Chtimes changes the access and modification times.
func (r *Router) Chtimes(name string, atime, mtime time.Time) error {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return err
	}

	if err := adaptor.Chtimes(relPath, atime, mtime); err != nil {
		return &PathError{Op: "chtimes", Path: name, Err: err}
	}

	return nil
}

// Readlink returns the destination of the named symbolic link.
func (r *Router) Readlink(name string) (string, error) {
	adaptor, relPath, err := r.resolve(name)
	if err != nil {
		return "", err
	}

	if !adaptor.Capabilities().CanLink {
		return "", &PathError{Op: "readlink", Path: name, Err: ErrNotSupported}
	}

	target, err := adaptor.Readlink(relPath)
	if err != nil {
		return "", &PathError{Op: "readlink", Path: name, Err: err}
	}

	return target, nil
}

// Symlink creates a symbolic link.
func (r *Router) Symlink(target, link string) error {
	adaptor, relPath, err := r.resolve(link)
	if err != nil {
		return err
	}

	if !adaptor.Capabilities().CanLink {
		return &LinkError{Op: "symlink", Old: target, New: link, Err: ErrNotSupported}
	}

	if err := adaptor.Symlink(target, relPath); err != nil {
		return &LinkError{Op: "symlink", Old: target, New: link, Err: err}
	}

	return nil
}

// Close closes the VFS and all mounted adaptors.
func (r *Router) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrClosed
	}

	r.closed = true

	// Close all watches
	for _, ch := range r.watches {
		close(ch)
	}
	r.watches = nil

	// Note: We don't disconnect adaptors here as they might be
	// used by other VFS instances. The caller is responsible
	// for managing adaptor lifecycle.

	return nil
}

// File open flags (subset of os package constants)
const (
	ORdonly = 0x0
	OWronly = 0x1
	ORdwr   = 0x2
	OCreate = 0x40
	OExcl   = 0x80
	OTrunc  = 0x200
	OAppend = 0x400
)

// routedFile wraps a File with VFS-specific behavior.
type routedFile struct {
	File
	absPath string
	router  *Router
}

// Name returns the absolute VFS path of the file.
func (f *routedFile) Name() string {
	return f.absPath
}

// mountDirEntry represents a mount point as a directory entry.
type mountDirEntry struct {
	name string
	mode fs.FileMode
}

func (e *mountDirEntry) Name() string            { return e.name }
func (e *mountDirEntry) IsDir() bool             { return true }
func (e *mountDirEntry) Type() fs.FileMode       { return fs.ModeDir }
func (e *mountDirEntry) Info() (FileInfo, error) { return &mountFileInfo{e}, nil }

// mountFileInfo represents a mount point as file info.
type mountFileInfo struct {
	entry *mountDirEntry
}

func (i *mountFileInfo) Name() string       { return i.entry.name }
func (i *mountFileInfo) Size() int64        { return 0 }
func (i *mountFileInfo) Mode() fs.FileMode  { return i.entry.mode }
func (i *mountFileInfo) ModTime() time.Time { return time.Now() }
func (i *mountFileInfo) IsDir() bool        { return true }
func (i *mountFileInfo) Sys() interface{}   { return nil }
