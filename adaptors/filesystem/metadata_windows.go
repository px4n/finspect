//go:build windows
// +build windows

package filesystem

import "os"

// SystemInfo contains system-specific file information
type SystemInfo struct {
	UID int
	GID int
}

// getSystemInfo extracts system-specific information from FileInfo
func getSystemInfo(info os.FileInfo) *SystemInfo {
	// Windows doesn't have UID/GID in the same way as Unix
	return &SystemInfo{
		UID: -1,
		GID: -1,
	}
}
