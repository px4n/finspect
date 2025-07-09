package config

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, ".", cfg.Global.WorkDir)
	assert.Equal(t, 10, cfg.Global.MaxConcurrency)
	assert.Len(t, cfg.Mounts, 1)
	assert.Equal(t, "/", cfg.Mounts[0].Path)
	assert.Equal(t, "filesystem", cfg.Mounts[0].Type)
	assert.Equal(t, "finspect.db", cfg.Metadata.DatabasePath)
	assert.True(t, cfg.Search.Enabled)
	assert.Equal(t, 8080, cfg.AdminUI.Port)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(*Config)
		wantError string
	}{
		{
			name: "valid config",
			modify: func(_ *Config) {
				// Default config is valid
			},
			wantError: "",
		},
		{
			name: "invalid max concurrency",
			modify: func(c *Config) {
				c.Global.MaxConcurrency = 0
			},
			wantError: "max_concurrency must be at least 1",
		},
		{
			name: "missing mount path",
			modify: func(c *Config) {
				c.Mounts[0].Path = ""
			},
			wantError: "mount[0]: path is required",
		},
		{
			name: "missing mount type",
			modify: func(c *Config) {
				c.Mounts[0].Type = ""
			},
			wantError: "mount[0]: type is required",
		},
		{
			name: "duplicate mount paths",
			modify: func(c *Config) {
				c.Mounts = append(c.Mounts, MountConfig{
					Path: "/",
					Type: "s3",
				})
			},
			wantError: "duplicate mount path /",
		},
		{
			name: "invalid batch size",
			modify: func(c *Config) {
				c.Metadata.BatchSize = 0
			},
			wantError: "metadata.batch_size must be at least 1",
		},
		{
			name: "invalid search limit",
			modify: func(c *Config) {
				c.Search.DefaultLimit = 0
			},
			wantError: "search.default_limit must be at least 1",
		},
		{
			name: "invalid port",
			modify: func(c *Config) {
				c.AdminUI.Port = 70000
			},
			wantError: "admin_ui.port must be between 1 and 65535",
		},
		{
			name: "invalid log level",
			modify: func(c *Config) {
				c.Logging.Level = "invalid"
			},
			wantError: "logging.level must be one of: debug, info, warn, error",
		},
		{
			name: "invalid log format",
			modify: func(c *Config) {
				c.Logging.Format = "invalid"
			},
			wantError: "logging.format must be one of: text, json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if tt.wantError == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			}
		})
	}
}

func TestLoadSaveYAML(t *testing.T) {
	// Create temp file
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Save config
	cfg := DefaultConfig()
	cfg.Global.WorkDir = "/test/work"
	cfg.Metadata.DatabasePath = "/test/db.sqlite"

	err = cfg.Save(tmpfile.Name())
	require.NoError(t, err)

	// Load config
	loaded, err := Load(tmpfile.Name())
	require.NoError(t, err)

	assert.Equal(t, cfg.Global.WorkDir, loaded.Global.WorkDir)
	assert.Equal(t, cfg.Metadata.DatabasePath, loaded.Metadata.DatabasePath)
}

func TestLoadSaveJSON(t *testing.T) {
	// Create temp file
	tmpfile, err := os.CreateTemp("", "config-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Save config
	cfg := DefaultConfig()
	cfg.Search.IndexPath = "/test/index"
	cfg.AdminUI.Port = 9090

	err = cfg.Save(tmpfile.Name())
	require.NoError(t, err)

	// Load config
	loaded, err := Load(tmpfile.Name())
	require.NoError(t, err)

	assert.Equal(t, cfg.Search.IndexPath, loaded.Search.IndexPath)
	assert.Equal(t, cfg.AdminUI.Port, loaded.AdminUI.Port)
}

func TestExpandEnv(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_WORK_DIR", "/expanded/work")
	os.Setenv("TEST_DB_PATH", "/expanded/db.sqlite")
	defer os.Unsetenv("TEST_WORK_DIR")
	defer os.Unsetenv("TEST_DB_PATH")

	cfg := &Config{
		Global: GlobalConfig{
			WorkDir: "${TEST_WORK_DIR}",
		},
		Metadata: MetadataConfig{
			DatabasePath: "${TEST_DB_PATH}",
		},
		Mounts: []MountConfig{
			{
				Path: "/data",
				Type: "filesystem",
				Config: map[string]interface{}{
					"root": "${TEST_WORK_DIR}/data",
				},
			},
		},
	}

	cfg.ExpandEnv()

	assert.Equal(t, "/expanded/work", cfg.Global.WorkDir)
	assert.Equal(t, "/expanded/db.sqlite", cfg.Metadata.DatabasePath)
	assert.Equal(t, "/expanded/work/data", cfg.Mounts[0].Config["root"])
}

func TestGetMount(t *testing.T) {
	cfg := &Config{
		Mounts: []MountConfig{
			{Path: "/", Type: "filesystem"},
			{Path: "/data", Type: "s3"},
			{Path: "/data/archive", Type: "googledrive"},
		},
	}

	tests := []struct {
		path      string
		wantPath  string
		wantType  string
		wantError bool
	}{
		{"/", "/", "filesystem", false},
		{"/file.txt", "/", "filesystem", false},
		{"/data", "/data", "s3", false},
		{"/data/file.txt", "/data", "s3", false},
		{"/data/archive", "/data/archive", "googledrive", false},
		{"/data/archive/old.zip", "/data/archive", "googledrive", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			mount, err := cfg.GetMount(tt.path)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantPath, mount.Path)
				assert.Equal(t, tt.wantType, mount.Type)
			}
		})
	}
}

func TestLoadFromReader(t *testing.T) {
	yamlConfig := `
global:
  work_dir: /custom/work
  max_concurrency: 20

mounts:
  - path: /
    type: filesystem
    config:
      root: /data
    options:
      read_only: false
      enable_cache: true

metadata:
  database_path: custom.db
  enable_fts: true

search:
  enabled: true
  index_path: custom.bleve
`

	buf := bytes.NewBufferString(yamlConfig)
	cfg, err := LoadFrom(buf, "config.yaml")
	require.NoError(t, err)

	assert.Equal(t, "/custom/work", cfg.Global.WorkDir)
	assert.Equal(t, 20, cfg.Global.MaxConcurrency)
	assert.Equal(t, "custom.db", cfg.Metadata.DatabasePath)
	assert.True(t, cfg.Metadata.EnableFTS)
	assert.Equal(t, "custom.bleve", cfg.Search.IndexPath)
}

func TestMergeEnvironment(t *testing.T) {
	// Set test environment variables
	os.Setenv("FINSPECT_GLOBAL_WORKDIR", "/env/work")
	os.Setenv("FINSPECT_GLOBAL_MAXCONCURRENCY", "50")
	os.Setenv("FINSPECT_METADATA_DATABASE", "/env/db.sqlite")
	os.Setenv("FINSPECT_METADATA_ENABLEFTS", "true")
	os.Setenv("FINSPECT_SEARCH_ENABLED", "false")
	os.Setenv("FINSPECT_ADMINUI_PORT", "9999")
	os.Setenv("FINSPECT_LOGGING_LEVEL", "debug")

	defer func() {
		os.Unsetenv("FINSPECT_GLOBAL_WORKDIR")
		os.Unsetenv("FINSPECT_GLOBAL_MAXCONCURRENCY")
		os.Unsetenv("FINSPECT_METADATA_DATABASE")
		os.Unsetenv("FINSPECT_METADATA_ENABLEFTS")
		os.Unsetenv("FINSPECT_SEARCH_ENABLED")
		os.Unsetenv("FINSPECT_ADMINUI_PORT")
		os.Unsetenv("FINSPECT_LOGGING_LEVEL")
	}()

	cfg := DefaultConfig()
	MergeEnvironment(cfg)

	assert.Equal(t, "/env/work", cfg.Global.WorkDir)
	assert.Equal(t, 50, cfg.Global.MaxConcurrency)
	assert.Equal(t, "/env/db.sqlite", cfg.Metadata.DatabasePath)
	assert.True(t, cfg.Metadata.EnableFTS)
	assert.False(t, cfg.Search.Enabled)
	assert.Equal(t, 9999, cfg.AdminUI.Port)
	assert.Equal(t, "debug", cfg.Logging.Level)
}
