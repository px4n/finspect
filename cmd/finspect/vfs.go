package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/px4n/finspect/adaptors/filesystem"
	"github.com/px4n/finspect/pkg/vfs"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	vfsInstance vfs.VFS
	vfsOnce     sync.Once
)

// getVFS returns the global VFS instance, creating it if necessary
func getVFS() vfs.VFS {
	vfsOnce.Do(func() {
		vfsInstance = vfs.NewRouter()

		// Load saved mounts
		loadMounts()
	})
	return vfsInstance
}

// loadMounts loads previously configured mounts from the config file
func loadMounts() {
	mounts := viper.GetStringMap("mounts")
	if len(mounts) == 0 {
		return
	}

	for mountPoint, config := range mounts {
		mountConfig, ok := config.(map[string]interface{})
		if !ok {
			logger.Warn("Invalid mount config", zap.String("mount", mountPoint))
			continue
		}

		sourceType, _ := mountConfig["type"].(string)
		sourcePath, _ := mountConfig["path"].(string)

		if sourceType == "" || sourcePath == "" {
			logger.Warn("Missing mount config",
				zap.String("mount", mountPoint),
				zap.String("type", sourceType),
				zap.String("path", sourcePath))
			continue
		}

		// Create and connect adaptor
		switch sourceType {
		case "local", "filesystem":
			fs := filesystem.New()
			config := map[string]interface{}{
				"root": sourcePath,
			}

			if err := fs.Connect(context.Background(), config); err != nil {
				logger.Warn("Failed to connect adaptor",
					zap.String("mount", mountPoint),
					zap.String("type", sourceType),
					zap.Error(err))
				continue
			}

			if err := vfsInstance.Mount(mountPoint, fs); err != nil {
				logger.Warn("Failed to mount",
					zap.String("mount", mountPoint),
					zap.Error(err))
				continue
			}

			logger.Info("Loaded mount",
				zap.String("mount", mountPoint),
				zap.String("type", sourceType),
				zap.String("path", sourcePath))
		}
	}
}

// saveMountConfig saves a mount configuration
func saveMountConfig(mountPoint, sourceType, sourcePath string) {
	// Ensure config directory exists
	configDir := filepath.Join(os.Getenv("HOME"), ".finspect")
	if err := os.MkdirAll(configDir, 0o750); err != nil { // #nosec G301 - config dir needs read access
		logger.Warn("Failed to create config directory", zap.Error(err))
		return
	}

	// Set mount config
	mountKey := fmt.Sprintf("mounts.%s", mountPoint)
	viper.Set(mountKey+".type", sourceType)
	viper.Set(mountKey+".path", sourcePath)

	// Save config
	configFile := filepath.Join(configDir, "config.yaml")
	if err := viper.WriteConfigAs(configFile); err != nil {
		logger.Warn("Failed to save config", zap.Error(err))
	}
}

// resolvePath resolves a potentially relative path to an absolute VFS path
func resolvePath(path string) string {
	if !filepath.IsAbs(path) {
		// For now, assume relative to root
		path = "/" + path
	}
	return filepath.Clean(path)
}
