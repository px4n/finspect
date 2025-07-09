// Package pathutil provides utilities for parsing and manipulating VFS paths.
package pathutil

import (
	"path"
	"strings"
)

// ParsePath splits a VFS path into its mount point and relative path components.
// For example, "/gdrive/Photos/vacation.jpg" returns "/gdrive" and "/Photos/vacation.jpg".
func ParsePath(vfsPath string) (mountPoint, relativePath string) {
	// Clean and ensure absolute path
	vfsPath = path.Clean(vfsPath)
	if !path.IsAbs(vfsPath) {
		return "", ""
	}

	// Root path special case
	if vfsPath == "/" {
		return "/", "/"
	}

	// Split the path
	parts := strings.Split(vfsPath[1:], "/") // Skip leading slash
	if len(parts) == 0 {
		return "/", "/"
	}

	// First part is the mount point
	mountPoint = "/" + parts[0]

	// Rest is the relative path
	if len(parts) > 1 {
		relativePath = "/" + strings.Join(parts[1:], "/")
	} else {
		relativePath = "/"
	}

	return mountPoint, relativePath
}

// Join joins VFS path elements, handling mount points correctly.
func Join(elem ...string) string {
	if len(elem) == 0 {
		return ""
	}
	return path.Join(elem...)
}

// Split splits a path into directory and file components.
func Split(vfsPath string) (dir, file string) {
	return path.Split(vfsPath)
}

// Base returns the last element of a path.
func Base(vfsPath string) string {
	return path.Base(vfsPath)
}

// Dir returns all but the last element of a path.
func Dir(vfsPath string) string {
	return path.Dir(vfsPath)
}

// IsAbs reports whether the path is absolute.
func IsAbs(vfsPath string) bool {
	return path.IsAbs(vfsPath)
}

// Clean returns the canonical version of the path.
func Clean(vfsPath string) string {
	return path.Clean(vfsPath)
}

// Ext returns the file extension.
func Ext(vfsPath string) string {
	base := path.Base(vfsPath)
	// For hidden files that start with a dot and have no additional extension,
	// return empty string
	if strings.HasPrefix(base, ".") && strings.Count(base, ".") == 1 {
		return ""
	}
	return path.Ext(vfsPath)
}

// Match reports whether name matches the shell pattern.
func Match(pattern, name string) (bool, error) {
	return path.Match(pattern, name)
}

// IsMount checks if a path is a mount point (has no relative component).
func IsMount(vfsPath string) bool {
	mount, rel := ParsePath(vfsPath)
	return mount != "" && rel == "/"
}

// NormalizePath ensures a path is clean and absolute.
func NormalizePath(vfsPath string) string {
	if vfsPath == "" {
		return "/"
	}
	vfsPath = Clean(vfsPath)
	if !IsAbs(vfsPath) {
		vfsPath = "/" + vfsPath
	}
	return vfsPath
}

// RelativeTo returns a relative path from base to target.
// Both paths must be absolute.
func RelativeTo(base, target string) (string, error) {
	base = NormalizePath(base)
	target = NormalizePath(target)

	// Check if target is under base
	if !strings.HasPrefix(target, base) {
		return "", nil
	}

	rel := strings.TrimPrefix(target, base)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		rel = "."
	}

	return rel, nil
}

// SplitList splits a path list into individual paths.
// The path list separator is OS-specific.
func SplitList(pathList string) []string {
	if pathList == "" {
		return nil
	}
	return strings.Split(pathList, ":")
}

// HasPrefix checks if a path has a given prefix.
func HasPrefix(vfsPath, prefix string) bool {
	vfsPath = NormalizePath(vfsPath)
	prefix = NormalizePath(prefix)

	// If prefix doesn't end with /, add it to ensure we match path boundaries
	// unless the prefix is exactly the path
	if vfsPath == prefix {
		return true
	}

	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return strings.HasPrefix(vfsPath+"/", prefix)
}

// TrimPrefix removes a prefix from a path if present.
func TrimPrefix(vfsPath, prefix string) string {
	vfsPath = NormalizePath(vfsPath)
	prefix = NormalizePath(prefix)

	if !strings.HasPrefix(vfsPath, prefix) {
		return vfsPath
	}

	result := strings.TrimPrefix(vfsPath, prefix)
	if result == "" {
		return "/"
	}
	if !strings.HasPrefix(result, "/") {
		result = "/" + result
	}

	return result
}
