package cloud

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/px4n/finspect/pkg/vfs"
)

// VFSCloudAdaptor wraps a CloudAdaptor to implement the VFS Adaptor interface
type VFSCloudAdaptor struct {
	cloud     Adaptor
	ctx       context.Context
	connected bool
}

// NewVFSCloudAdaptor creates a new VFS adaptor that wraps a cloud adaptor
func NewVFSCloudAdaptor(cloud Adaptor) vfs.Adaptor {
	return &VFSCloudAdaptor{
		cloud: cloud,
		ctx:   context.Background(),
	}
}

// Name returns the name of the adaptor
func (v *VFSCloudAdaptor) Name() string {
	return string(v.cloud.GetProvider())
}

// Connect connects to the cloud storage
func (v *VFSCloudAdaptor) Connect(ctx context.Context, config map[string]interface{}) error {
	// Convert map config to AdaptorConfig
	cloudConfig := AdaptorConfig{}

	if provider, ok := config["provider"].(string); ok {
		cloudConfig.Provider = Provider(provider)
	}

	if bucket, ok := config["bucket"].(string); ok {
		cloudConfig.Bucket = bucket
	}

	if region, ok := config["region"].(string); ok {
		cloudConfig.Region = region
	}

	if endpoint, ok := config["endpoint"].(string); ok {
		cloudConfig.Endpoint = endpoint
	}

	if creds, ok := config["credentials"].(map[string]interface{}); ok {
		cloudConfig.Credentials = creds
	}

	v.ctx = ctx
	err := v.cloud.Connect(ctx, cloudConfig)
	if err == nil {
		v.connected = true
	}
	return err
}

// Disconnect disconnects from the cloud storage
func (v *VFSCloudAdaptor) Disconnect() error {
	err := v.cloud.Disconnect(v.ctx)
	if err == nil {
		v.connected = false
	}
	return err
}

// IsConnected returns true if the adaptor is connected
func (v *VFSCloudAdaptor) IsConnected() bool {
	return v.connected
}

// Capabilities returns the capabilities of this adaptor
func (v *VFSCloudAdaptor) Capabilities() vfs.Capabilities {
	return vfs.Capabilities{
		CanRead:   true,
		CanWrite:  true,
		CanDelete: true,
		CanMove:   true,
		CanWatch:  false, // Most cloud storage doesn't support watching
		CanLink:   false, // Most cloud storage doesn't support symlinks
		CanChmod:  false, // Cloud storage doesn't support Unix permissions
		CanChown:  false, // Cloud storage doesn't support ownership
		HasDirs:   true,
	}
}

// Stat returns file information
func (v *VFSCloudAdaptor) Stat(path string) (vfs.FileInfo, error) {
	info, err := v.cloud.Stat(v.ctx, path)
	if err != nil {
		return nil, err
	}

	return &cloudFileInfo{
		cloud: info,
		name:  v.basename(path),
	}, nil
}

// ReadDir reads a directory
func (v *VFSCloudAdaptor) ReadDir(path string) ([]vfs.DirEntry, error) {
	list, err := v.cloud.List(v.ctx, path, ListOptions{
		Prefix:    path,
		Delimiter: "/",
		Recursive: false,
	})
	if err != nil {
		return nil, err
	}

	entries := make([]vfs.DirEntry, 0, len(list))
	for _, item := range list {
		entries = append(entries, &cloudDirEntry{
			info: &cloudFileInfo{
				cloud: item,
				name:  v.basename(item.Path),
			},
		})
	}

	return entries, nil
}

// Open opens a file for reading
func (v *VFSCloudAdaptor) Open(path string) (vfs.File, error) {
	// Check if it's a directory
	info, err := v.cloud.Stat(v.ctx, path)
	if err != nil {
		return nil, err
	}

	if info.IsDir {
		// Return a directory file handle
		return &cloudDirFile{
			adaptor: v,
			path:    path,
			info:    info,
		}, nil
	}

	// Open file for reading
	reader, err := v.cloud.Read(v.ctx, path, DownloadOptions{})
	if err != nil {
		return nil, err
	}

	return &cloudFile{
		reader: reader,
		info:   info,
		path:   path,
	}, nil
}

// Create creates a new file
func (v *VFSCloudAdaptor) Create(path string) (vfs.File, error) {
	return &cloudWriteFile{
		adaptor: v,
		path:    path,
		buffer:  make([]byte, 0),
	}, nil
}

// Remove removes a file
func (v *VFSCloudAdaptor) Remove(path string) error {
	return v.cloud.Delete(v.ctx, path)
}

// Rename renames a file
func (v *VFSCloudAdaptor) Rename(oldPath, newPath string) error {
	return v.cloud.Move(v.ctx, oldPath, newPath)
}

// MkdirAll creates a directory hierarchy
func (v *VFSCloudAdaptor) MkdirAll(path string, _ fs.FileMode) error {
	return v.cloud.CreateDirectory(v.ctx, path)
}

// RemoveAll removes a directory tree
func (v *VFSCloudAdaptor) RemoveAll(path string) error {
	// List all files recursively
	list, err := v.cloud.List(v.ctx, path, ListOptions{
		Prefix:    path,
		Recursive: true,
	})
	if err != nil {
		return err
	}

	// Delete all files
	for _, item := range list {
		if err := v.cloud.Delete(v.ctx, item.Path); err != nil {
			return err
		}
	}

	// Delete the directory itself
	return v.cloud.Delete(v.ctx, path)
}

// OpenFile opens a file with specified flags and permissions
func (v *VFSCloudAdaptor) OpenFile(path string, flag int, _ fs.FileMode) (vfs.File, error) {
	// For cloud storage, we simplify to either read or write
	if flag&os.O_WRONLY != 0 || flag&os.O_RDWR != 0 || flag&os.O_CREATE != 0 || flag&os.O_TRUNC != 0 {
		return v.Create(path)
	}
	return v.Open(path)
}

// Lstat returns information about the named file or link
func (v *VFSCloudAdaptor) Lstat(path string) (vfs.FileInfo, error) {
	// Cloud storage doesn't have symlinks, so Lstat is the same as Stat
	return v.Stat(path)
}

// Mkdir creates a new directory
func (v *VFSCloudAdaptor) Mkdir(path string, _ fs.FileMode) error {
	return v.cloud.CreateDirectory(v.ctx, path)
}

// Chmod changes file permissions (not supported)
func (v *VFSCloudAdaptor) Chmod(_ string, _ fs.FileMode) error {
	return fmt.Errorf("chmod not supported on cloud storage")
}

// Chown changes file ownership (not supported)
func (v *VFSCloudAdaptor) Chown(_ string, _, _ int) error {
	return fmt.Errorf("chown not supported on cloud storage")
}

// Chtimes changes file timestamps (not supported)
func (v *VFSCloudAdaptor) Chtimes(_ string, _, _ time.Time) error {
	return fmt.Errorf("chtimes not supported on cloud storage")
}

// Readlink returns the destination of a symbolic link (not supported)
func (v *VFSCloudAdaptor) Readlink(_ string) (string, error) {
	return "", fmt.Errorf("readlink not supported on cloud storage")
}

// Symlink creates a symbolic link (not supported)
func (v *VFSCloudAdaptor) Symlink(_, _ string) error {
	return fmt.Errorf("symlink not supported on cloud storage")
}

// Watch watches for changes (not supported)
func (v *VFSCloudAdaptor) Watch(_ string, _ bool, _ chan<- vfs.Event) error {
	return fmt.Errorf("watch not supported on cloud storage")
}

// Unwatch stops watching (not supported)
func (v *VFSCloudAdaptor) Unwatch(_ string) error {
	return fmt.Errorf("unwatch not supported on cloud storage")
}

// basename returns the last element of path
func (v *VFSCloudAdaptor) basename(p string) string {
	return path.Base(p)
}

// cloudFileInfo wraps FileInfo to implement vfs.FileInfo
type cloudFileInfo struct {
	cloud *FileInfo
	name  string
}

func (i *cloudFileInfo) Name() string { return i.name }
func (i *cloudFileInfo) Size() int64  { return i.cloud.Size }
func (i *cloudFileInfo) Mode() fs.FileMode {
	if i.cloud.IsDir {
		return fs.ModeDir | 0755
	}
	return 0644
}
func (i *cloudFileInfo) ModTime() time.Time { return i.cloud.ModTime }
func (i *cloudFileInfo) IsDir() bool        { return i.cloud.IsDir }
func (i *cloudFileInfo) Sys() interface{}   { return i.cloud }

// cloudDirEntry implements vfs.DirEntry
type cloudDirEntry struct {
	info vfs.FileInfo
}

func (e *cloudDirEntry) Name() string                { return e.info.Name() }
func (e *cloudDirEntry) IsDir() bool                 { return e.info.IsDir() }
func (e *cloudDirEntry) Type() fs.FileMode           { return e.info.Mode().Type() }
func (e *cloudDirEntry) Info() (vfs.FileInfo, error) { return e.info, nil }

// cloudFile implements vfs.File for cloud files
type cloudFile struct {
	reader io.ReadCloser
	info   *FileInfo
	path   string
	offset int64
}

func (f *cloudFile) Stat() (vfs.FileInfo, error) {
	return &cloudFileInfo{cloud: f.info, name: path.Base(f.path)}, nil
}

func (f *cloudFile) Read(p []byte) (n int, err error) {
	n, err = f.reader.Read(p)
	f.offset += int64(n)
	return
}

func (f *cloudFile) Write(_ []byte) (n int, err error) {
	return 0, fmt.Errorf("file opened for reading only")
}

func (f *cloudFile) Seek(_ int64, _ int) (int64, error) {
	return 0, fmt.Errorf("seek not supported on cloud files")
}

func (f *cloudFile) Close() error {
	return f.reader.Close()
}

func (f *cloudFile) Sync() error {
	return nil // No-op for read-only files
}

func (f *cloudFile) Truncate(_ int64) error {
	return fmt.Errorf("truncate not supported on read-only file")
}

func (f *cloudFile) Name() string {
	return f.path
}

func (f *cloudFile) Readdir(_ int) ([]vfs.FileInfo, error) {
	return nil, fmt.Errorf("readdir not supported on regular file")
}

func (f *cloudFile) Readdirnames(_ int) ([]string, error) {
	return nil, fmt.Errorf("readdirnames not supported on regular file")
}

// cloudDirFile implements vfs.File for cloud directories
type cloudDirFile struct {
	adaptor *VFSCloudAdaptor
	path    string
	info    *FileInfo
	entries []vfs.FileInfo
	offset  int
}

func (f *cloudDirFile) Stat() (vfs.FileInfo, error) {
	return &cloudFileInfo{cloud: f.info, name: path.Base(f.path)}, nil
}

func (f *cloudDirFile) Read(_ []byte) (n int, err error) {
	return 0, fmt.Errorf("cannot read directory")
}

func (f *cloudDirFile) Write(_ []byte) (n int, err error) {
	return 0, fmt.Errorf("cannot write to directory")
}

func (f *cloudDirFile) Seek(_ int64, _ int) (int64, error) {
	return 0, fmt.Errorf("seek not supported on directories")
}

func (f *cloudDirFile) Close() error {
	return nil
}

func (f *cloudDirFile) Sync() error {
	return nil
}

func (f *cloudDirFile) Truncate(_ int64) error {
	return fmt.Errorf("truncate not supported on directories")
}

func (f *cloudDirFile) Name() string {
	return f.path
}

func (f *cloudDirFile) Readdir(n int) ([]vfs.FileInfo, error) {
	if f.entries == nil {
		// Load entries
		dirEntries, err := f.adaptor.ReadDir(f.path)
		if err != nil {
			return nil, err
		}
		f.entries = make([]vfs.FileInfo, len(dirEntries))
		for i, entry := range dirEntries {
			info, _ := entry.Info()
			f.entries[i] = info
		}
		f.offset = 0
	}

	var result []vfs.FileInfo

	if n <= 0 {
		// Return all remaining entries
		result = f.entries[f.offset:]
		f.offset = len(f.entries)
	} else {
		// Return n entries
		end := f.offset + n
		if end > len(f.entries) {
			end = len(f.entries)
		}

		result = f.entries[f.offset:end]
		f.offset = end

		if f.offset >= len(f.entries) && len(result) < n {
			return result, io.EOF
		}
	}

	return result, nil
}

func (f *cloudDirFile) Readdirnames(n int) ([]string, error) {
	infos, err := f.Readdir(n)
	if err != nil && err != io.EOF {
		return nil, err
	}

	names := make([]string, len(infos))
	for i, info := range infos {
		names[i] = info.Name()
	}

	return names, err
}

// cloudWriteFile implements vfs.File for writing to cloud storage
type cloudWriteFile struct {
	adaptor *VFSCloudAdaptor
	path    string
	buffer  []byte
	closed  bool
}

func (f *cloudWriteFile) Write(p []byte) (n int, err error) {
	if f.closed {
		return 0, os.ErrClosed
	}
	f.buffer = append(f.buffer, p...)
	return len(p), nil
}

func (f *cloudWriteFile) Read(_ []byte) (n int, err error) {
	return 0, fmt.Errorf("file opened for writing only")
}

func (f *cloudWriteFile) Seek(_ int64, _ int) (int64, error) {
	return 0, fmt.Errorf("seek not supported on cloud write files")
}

func (f *cloudWriteFile) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true

	// Upload the buffer
	reader := &bytesReader{data: f.buffer}
	return f.adaptor.cloud.Write(f.adaptor.ctx, f.path, reader, UploadOptions{})
}

func (f *cloudWriteFile) Stat() (vfs.FileInfo, error) {
	return &cloudFileInfo{
		cloud: &FileInfo{
			Path:    f.path,
			Size:    int64(len(f.buffer)),
			ModTime: time.Now(),
			IsDir:   false,
		},
		name: path.Base(f.path),
	}, nil
}

func (f *cloudWriteFile) Sync() error {
	// No-op until Close is called
	return nil
}

func (f *cloudWriteFile) Truncate(size int64) error {
	if f.closed {
		return os.ErrClosed
	}
	if size < 0 {
		return fmt.Errorf("negative truncate size")
	}
	if size < int64(len(f.buffer)) {
		f.buffer = f.buffer[:size]
	} else {
		// Extend with zeros
		zeros := make([]byte, int(size)-len(f.buffer))
		f.buffer = append(f.buffer, zeros...)
	}
	return nil
}

func (f *cloudWriteFile) Name() string {
	return f.path
}

func (f *cloudWriteFile) Readdir(_ int) ([]vfs.FileInfo, error) {
	return nil, fmt.Errorf("readdir not supported on regular file")
}

func (f *cloudWriteFile) Readdirnames(_ int) ([]string, error) {
	return nil, fmt.Errorf("readdirnames not supported on regular file")
}

// bytesReader wraps a byte slice as io.Reader
type bytesReader struct {
	data   []byte
	offset int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}

	n = copy(p, r.data[r.offset:])
	r.offset += n

	if r.offset >= len(r.data) {
		err = io.EOF
	}

	return
}
