package vfs

import (
	"errors"
	"fmt"
)

// Common VFS errors
var (
	// ErrNotMounted is returned when trying to access a path that has no mounted adaptor.
	ErrNotMounted = errors.New("vfs: path not mounted")

	// ErrAlreadyMounted is returned when trying to mount at a path that already has an adaptor.
	ErrAlreadyMounted = errors.New("vfs: path already mounted")

	// ErrInvalidPath is returned when a path is malformed or invalid.
	ErrInvalidPath = errors.New("vfs: invalid path")

	// ErrNotSupported is returned when an operation is not supported by the adaptor.
	ErrNotSupported = errors.New("vfs: operation not supported")

	// ErrCrossDevice is returned when trying to move files across different adaptors.
	ErrCrossDevice = errors.New("vfs: cross-device operation not permitted")

	// ErrNotConnected is returned when an adaptor is not connected.
	ErrNotConnected = errors.New("vfs: adaptor not connected")

	// ErrReadOnly is returned when trying to write to a read-only adaptor.
	ErrReadOnly = errors.New("vfs: read-only file system")

	// ErrNotEmpty is returned when trying to remove a non-empty directory.
	ErrNotEmpty = errors.New("vfs: directory not empty")

	// ErrIsDirectory is returned when a file operation is attempted on a directory.
	ErrIsDirectory = errors.New("vfs: is a directory")

	// ErrNotDirectory is returned when a directory operation is attempted on a file.
	ErrNotDirectory = errors.New("vfs: not a directory")

	// ErrClosed is returned when operating on a closed VFS or file.
	ErrClosed = errors.New("vfs: closed")
)

// PathError records an error and the operation and path that caused it.
type PathError struct {
	Op   string // Operation that caused the error
	Path string // Path that caused the error
	Err  error  // The underlying error
}

// Error returns the string representation of a PathError.
func (e *PathError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
}

// Unwrap returns the underlying error.
func (e *PathError) Unwrap() error {
	return e.Err
}

// LinkError records an error during a link operation.
type LinkError struct {
	Op  string // Operation that caused the error
	Old string // Old path
	New string // New path
	Err error  // The underlying error
}

// Error returns the string representation of a LinkError.
func (e *LinkError) Error() string {
	return fmt.Sprintf("%s %s %s: %v", e.Op, e.Old, e.New, e.Err)
}

// Unwrap returns the underlying error.
func (e *LinkError) Unwrap() error {
	return e.Err
}

// IsNotExist returns true if the error indicates that a file or directory does not exist.
func IsNotExist(err error) bool {
	if err == nil {
		return false
	}
	var pathErr *PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return errors.Is(err, ErrNotExist) || errors.Is(err, ErrNotMounted)
}

// IsExist returns true if the error indicates that a file or directory already exists.
func IsExist(err error) bool {
	if err == nil {
		return false
	}
	var pathErr *PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return errors.Is(err, ErrExist) || errors.Is(err, ErrAlreadyMounted)
}

// IsPermission returns true if the error indicates a permission problem.
func IsPermission(err error) bool {
	if err == nil {
		return false
	}
	var pathErr *PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return errors.Is(err, ErrPermission) || errors.Is(err, ErrReadOnly)
}

// Standard filesystem errors that adaptors should use
var (
	ErrNotExist   = errors.New("file does not exist")
	ErrExist      = errors.New("file already exists")
	ErrPermission = errors.New("permission denied")
)
