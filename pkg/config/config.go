package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure for finspect
type Config struct {
	// Global settings
	Global GlobalConfig `yaml:"global" json:"global"`

	// VFS mounts configuration
	Mounts []MountConfig `yaml:"mounts" json:"mounts"`

	// Metadata configuration
	Metadata MetadataConfig `yaml:"metadata" json:"metadata"`

	// Search configuration
	Search SearchConfig `yaml:"search" json:"search"`

	// Admin UI configuration
	AdminUI AdminUIConfig `yaml:"admin_ui" json:"admin_ui"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging" json:"logging"`
}

// GlobalConfig contains global application settings
type GlobalConfig struct {
	// Working directory
	WorkDir string `yaml:"work_dir" json:"work_dir"`

	// Temporary directory for operations
	TempDir string `yaml:"temp_dir" json:"temp_dir"`

	// Maximum concurrent operations
	MaxConcurrency int `yaml:"max_concurrency" json:"max_concurrency"`

	// Enable performance profiling
	EnableProfiling bool `yaml:"enable_profiling" json:"enable_profiling"`
}

// MountConfig represents a VFS mount configuration
type MountConfig struct {
	// Mount path in VFS
	Path string `yaml:"path" json:"path"`

	// Adaptor type (filesystem, s3, googledrive, etc.)
	Type string `yaml:"type" json:"type"`

	// Adaptor-specific configuration
	Config map[string]interface{} `yaml:"config" json:"config"`

	// Mount options
	Options MountOptions `yaml:"options" json:"options"`
}

// MountOptions contains mount-specific options
type MountOptions struct {
	// Read-only mount
	ReadOnly bool `yaml:"read_only" json:"read_only"`

	// Enable caching for this mount
	EnableCache bool `yaml:"enable_cache" json:"enable_cache"`

	// Cache TTL in seconds
	CacheTTL int `yaml:"cache_ttl" json:"cache_ttl"`

	// Enable metadata extraction
	EnableMetadata bool `yaml:"enable_metadata" json:"enable_metadata"`
}

// MetadataConfig contains metadata system configuration
type MetadataConfig struct {
	// Database path
	DatabasePath string `yaml:"database_path" json:"database_path"`

	// Enable full-text search with FTS5
	EnableFTS bool `yaml:"enable_fts" json:"enable_fts"`

	// Batch size for operations
	BatchSize int `yaml:"batch_size" json:"batch_size"`

	// Auto-index new files
	AutoIndex bool `yaml:"auto_index" json:"auto_index"`

	// Extractors configuration
	Extractors ExtractorsConfig `yaml:"extractors" json:"extractors"`
}

// ExtractorsConfig contains metadata extractor settings
type ExtractorsConfig struct {
	// Enable image metadata extraction
	EnableImage bool `yaml:"enable_image" json:"enable_image"`

	// Enable document metadata extraction
	EnableDocument bool `yaml:"enable_document" json:"enable_document"`

	// Enable audio/video metadata extraction
	EnableMedia bool `yaml:"enable_media" json:"enable_media"`

	// Custom extractors
	Custom []CustomExtractor `yaml:"custom" json:"custom"`
}

// CustomExtractor defines a custom metadata extractor
type CustomExtractor struct {
	// Name of the extractor
	Name string `yaml:"name" json:"name"`

	// File patterns to match (glob)
	Patterns []string `yaml:"patterns" json:"patterns"`

	// Command to execute
	Command string `yaml:"command" json:"command"`

	// Command arguments
	Args []string `yaml:"args" json:"args"`
}

// SearchConfig contains search system configuration
type SearchConfig struct {
	// Index path for Bleve
	IndexPath string `yaml:"index_path" json:"index_path"`

	// Enable search
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Index options
	IndexOptions IndexOptions `yaml:"index_options" json:"index_options"`

	// Default search limit
	DefaultLimit int `yaml:"default_limit" json:"default_limit"`
}

// IndexOptions contains search index options
type IndexOptions struct {
	// Include file content in index
	IndexContent bool `yaml:"index_content" json:"index_content"`

	// Maximum content size to index (bytes)
	MaxContentSize int64 `yaml:"max_content_size" json:"max_content_size"`

	// File types to index content for
	ContentTypes []string `yaml:"content_types" json:"content_types"`

	// Enable highlighting
	EnableHighlighting bool `yaml:"enable_highlighting" json:"enable_highlighting"`

	// Enable faceting
	EnableFacets bool `yaml:"enable_facets" json:"enable_facets"`
}

// AdminUIConfig contains admin UI configuration
type AdminUIConfig struct {
	// Enable admin UI
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Listen address
	Address string `yaml:"address" json:"address"`

	// Listen port
	Port int `yaml:"port" json:"port"`

	// Static files directory
	StaticDir string `yaml:"static_dir" json:"static_dir"`

	// Enable authentication
	EnableAuth bool `yaml:"enable_auth" json:"enable_auth"`

	// Authentication configuration
	Auth AuthConfig `yaml:"auth" json:"auth"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	// Authentication type (basic, oauth2, etc.)
	Type string `yaml:"type" json:"type"`

	// Type-specific configuration
	Config map[string]interface{} `yaml:"config" json:"config"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	// Log level (debug, info, warn, error)
	Level string `yaml:"level" json:"level"`

	// Log format (text, json)
	Format string `yaml:"format" json:"format"`

	// Log output (stdout, file)
	Output string `yaml:"output" json:"output"`

	// Log file path (if output is file)
	FilePath string `yaml:"file_path" json:"file_path"`

	// Enable log rotation
	EnableRotation bool `yaml:"enable_rotation" json:"enable_rotation"`

	// Maximum log file size (MB)
	MaxSize int `yaml:"max_size" json:"max_size"`

	// Maximum log files to keep
	MaxBackups int `yaml:"max_backups" json:"max_backups"`

	// Maximum days to keep logs
	MaxAge int `yaml:"max_age" json:"max_age"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			WorkDir:         ".",
			TempDir:         os.TempDir(),
			MaxConcurrency:  10,
			EnableProfiling: false,
		},
		Mounts: []MountConfig{
			{
				Path: "/",
				Type: "filesystem",
				Config: map[string]interface{}{
					"root": ".",
				},
				Options: MountOptions{
					ReadOnly:       false,
					EnableCache:    true,
					CacheTTL:       300,
					EnableMetadata: true,
				},
			},
		},
		Metadata: MetadataConfig{
			DatabasePath: "finspect.db",
			EnableFTS:    false,
			BatchSize:    100,
			AutoIndex:    false,
			Extractors: ExtractorsConfig{
				EnableImage:    true,
				EnableDocument: true,
				EnableMedia:    true,
			},
		},
		Search: SearchConfig{
			IndexPath:    "finspect.bleve",
			Enabled:      true,
			DefaultLimit: 20,
			IndexOptions: IndexOptions{
				IndexContent:       true,
				MaxContentSize:     1024 * 1024, // 1MB
				ContentTypes:       []string{"text/plain", "text/markdown", "text/html"},
				EnableHighlighting: true,
				EnableFacets:       true,
			},
		},
		AdminUI: AdminUIConfig{
			Enabled:    true,
			Address:    "127.0.0.1",
			Port:       8080,
			StaticDir:  "./admin-ui/dist",
			EnableAuth: false,
		},
		Logging: LoggingConfig{
			Level:          "info",
			Format:         "text",
			Output:         "stdout",
			EnableRotation: false,
			MaxSize:        100,
			MaxBackups:     3,
			MaxAge:         30,
		},
	}
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(path)
	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer file.Close()

	return LoadFrom(file, cleanPath)
}

// LoadFrom loads configuration from a reader
func LoadFrom(r io.Reader, path string) (*Config, error) {
	// Start with default config
	cfg := DefaultConfig()

	// Read all content
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Determine format by extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse JSON config: %w", err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, cfg); err != nil {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config (unknown format): %w", err)
			}
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	// Expand environment variables
	cfg.ExpandEnv()

	return cfg, nil
}

// Save saves configuration to a file
func (c *Config) Save(path string) error {
	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(path)
	file, err := os.Create(cleanPath)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer file.Close()

	return c.SaveTo(file, cleanPath)
}

// SaveTo saves configuration to a writer
func (c *Config) SaveTo(w io.Writer, path string) error {
	// Determine format by extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		encoder := yaml.NewEncoder(w)
		encoder.SetIndent(2)
		return encoder.Encode(c)
	case ".json":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(c)
	default:
		// Default to YAML
		encoder := yaml.NewEncoder(w)
		encoder.SetIndent(2)
		return encoder.Encode(c)
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate global settings
	if c.Global.MaxConcurrency < 1 {
		return fmt.Errorf("max_concurrency must be at least 1")
	}

	// Validate mounts
	mountPaths := make(map[string]bool)
	for i, mount := range c.Mounts {
		if mount.Path == "" {
			return fmt.Errorf("mount[%d]: path is required", i)
		}
		if mount.Type == "" {
			return fmt.Errorf("mount[%d]: type is required", i)
		}
		if mountPaths[mount.Path] {
			return fmt.Errorf("mount[%d]: duplicate mount path %s", i, mount.Path)
		}
		mountPaths[mount.Path] = true
	}

	// Validate metadata
	if c.Metadata.BatchSize < 1 {
		return fmt.Errorf("metadata.batch_size must be at least 1")
	}

	// Validate search
	if c.Search.DefaultLimit < 1 {
		return fmt.Errorf("search.default_limit must be at least 1")
	}
	if c.Search.IndexOptions.MaxContentSize < 0 {
		return fmt.Errorf("search.index_options.max_content_size cannot be negative")
	}

	// Validate admin UI
	if c.AdminUI.Enabled {
		if c.AdminUI.Port < 1 || c.AdminUI.Port > 65535 {
			return fmt.Errorf("admin_ui.port must be between 1 and 65535")
		}
	}

	// Validate logging
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error")
	}
	validFormats := map[string]bool{"text": true, "json": true}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("logging.format must be one of: text, json")
	}

	return nil
}

// ExpandEnv expands environment variables in the configuration
func (c *Config) ExpandEnv() {
	// Expand in global config
	c.Global.WorkDir = os.ExpandEnv(c.Global.WorkDir)
	c.Global.TempDir = os.ExpandEnv(c.Global.TempDir)

	// Expand in mounts
	for i := range c.Mounts {
		c.Mounts[i].Path = os.ExpandEnv(c.Mounts[i].Path)
		// Expand in mount config values
		for k, v := range c.Mounts[i].Config {
			if str, ok := v.(string); ok {
				c.Mounts[i].Config[k] = os.ExpandEnv(str)
			}
		}
	}

	// Expand in metadata
	c.Metadata.DatabasePath = os.ExpandEnv(c.Metadata.DatabasePath)

	// Expand in search
	c.Search.IndexPath = os.ExpandEnv(c.Search.IndexPath)

	// Expand in admin UI
	c.AdminUI.StaticDir = os.ExpandEnv(c.AdminUI.StaticDir)

	// Expand in logging
	c.Logging.FilePath = os.ExpandEnv(c.Logging.FilePath)
}

// GetMount returns the mount configuration for a given path
func (c *Config) GetMount(path string) (*MountConfig, error) {
	// Find the longest matching mount path
	var bestMatch *MountConfig
	bestLen := -1

	for i := range c.Mounts {
		mount := &c.Mounts[i]
		if strings.HasPrefix(path, mount.Path) {
			if len(mount.Path) > bestLen {
				bestMatch = mount
				bestLen = len(mount.Path)
			}
		}
	}

	if bestMatch == nil {
		return nil, fmt.Errorf("no mount found for path: %s", path)
	}

	return bestMatch, nil
}
