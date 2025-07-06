//go:build !windows
// +build !windows

package filesystem

import (
	"os"
	"syscall"
)

// SystemInfo contains system-specific file information
type SystemInfo struct {
	UID int
	GID int
}

// getSystemInfo extracts system-specific information from FileInfo
func getSystemInfo(info os.FileInfo) *SystemInfo {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return &SystemInfo{
			UID: int(stat.Uid),
			GID: int(stat.Gid),
		}
	}
	return nil
}
