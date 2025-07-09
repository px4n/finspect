package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigPaths contains standard configuration file locations
var ConfigPaths = []string{
	"finspect.yaml",
	"finspect.yml",
	"finspect.json",
	".finspect.yaml",
	".finspect.yml",
	".finspect.json",
}

// SystemConfigDirs contains system-wide configuration directories
var SystemConfigDirs = []string{
	"/etc/finspect",
	"/usr/local/etc/finspect",
}

// Loader provides configuration loading functionality
type Loader struct {
	searchPaths []string
}

// GetSearchPaths returns the current search paths
func (l *Loader) GetSearchPaths() []string {
	return l.searchPaths
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{
		searchPaths: getDefaultSearchPaths(),
	}
}

// getDefaultSearchPaths returns the default configuration search paths
func getDefaultSearchPaths() []string {
	var paths []string

	// Current directory
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, cwd)
	}

	// User config directory
	if configDir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(configDir, "finspect"))
	}

	// Home directory
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, home)
		paths = append(paths, filepath.Join(home, ".config", "finspect"))
	}

	// System directories
	paths = append(paths, SystemConfigDirs...)

	return paths
}

// AddSearchPath adds a search path to the loader
func (l *Loader) AddSearchPath(path string) {
	l.searchPaths = append(l.searchPaths, path)
}

// SetSearchPaths sets the search paths for the loader
func (l *Loader) SetSearchPaths(paths []string) {
	l.searchPaths = paths
}

// Load loads configuration from the first found config file
func (l *Loader) Load() (*Config, string, error) {
	// Check each search path
	for _, dir := range l.searchPaths {
		for _, filename := range ConfigPaths {
			path := filepath.Join(dir, filename)
			if _, err := os.Stat(path); err == nil {
				cfg, err := Load(path)
				if err != nil {
					return nil, path, fmt.Errorf("load config from %s: %w", path, err)
				}
				return cfg, path, nil
			}
		}
	}

	// No config file found, return default
	return DefaultConfig(), "", nil
}

// LoadFromPath loads configuration from a specific path
func (l *Loader) LoadFromPath(path string) (*Config, error) {
	// Check if path exists
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	return Load(path)
}

// LoadOrCreate loads configuration or creates default if not found
func (l *Loader) LoadOrCreate(path string) (*Config, error) {
	// Try to load existing config
	if _, err := os.Stat(path); err == nil {
		return Load(path)
	}

	// Create default config
	cfg := DefaultConfig()

	// Save to file
	if err := cfg.Save(path); err != nil {
		return nil, fmt.Errorf("save default config: %w", err)
	}

	return cfg, nil
}

// FindConfigFile finds the configuration file in search paths
func (l *Loader) FindConfigFile() (string, error) {
	for _, dir := range l.searchPaths {
		for _, filename := range ConfigPaths {
			path := filepath.Join(dir, filename)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("no configuration file found in search paths")
}

// MergeEnvironment merges environment variables into configuration
func MergeEnvironment(cfg *Config) {
	// Override with environment variables
	// Format: FINSPECT_SECTION_KEY (e.g., FINSPECT_GLOBAL_WORKDIR)

	mergeGlobalEnv(&cfg.Global)
	mergeMetadataEnv(&cfg.Metadata)
	mergeSearchEnv(&cfg.Search)
	mergeAdminUIEnv(&cfg.AdminUI)
	mergeLoggingEnv(&cfg.Logging)
}

func mergeGlobalEnv(global *GlobalConfig) {
	if v := os.Getenv("FINSPECT_GLOBAL_WORKDIR"); v != "" {
		global.WorkDir = v
	}
	if v := os.Getenv("FINSPECT_GLOBAL_TEMPDIR"); v != "" {
		global.TempDir = v
	}
	if v := os.Getenv("FINSPECT_GLOBAL_MAXCONCURRENCY"); v != "" {
		if n, err := parseInt(v); err == nil {
			global.MaxConcurrency = n
		}
	}
}

func mergeMetadataEnv(metadata *MetadataConfig) {
	if v := os.Getenv("FINSPECT_METADATA_DATABASE"); v != "" {
		metadata.DatabasePath = v
	}
	if v := os.Getenv("FINSPECT_METADATA_ENABLEFTS"); v != "" {
		metadata.EnableFTS = parseBool(v)
	}
}

func mergeSearchEnv(search *SearchConfig) {
	if v := os.Getenv("FINSPECT_SEARCH_INDEX"); v != "" {
		search.IndexPath = v
	}
	if v := os.Getenv("FINSPECT_SEARCH_ENABLED"); v != "" {
		search.Enabled = parseBool(v)
	}
}

func mergeAdminUIEnv(adminUI *AdminUIConfig) {
	if v := os.Getenv("FINSPECT_ADMINUI_ENABLED"); v != "" {
		adminUI.Enabled = parseBool(v)
	}
	if v := os.Getenv("FINSPECT_ADMINUI_ADDRESS"); v != "" {
		adminUI.Address = v
	}
	if v := os.Getenv("FINSPECT_ADMINUI_PORT"); v != "" {
		if n, err := parseInt(v); err == nil {
			adminUI.Port = n
		}
	}
}

func mergeLoggingEnv(logging *LoggingConfig) {
	if v := os.Getenv("FINSPECT_LOGGING_LEVEL"); v != "" {
		logging.Level = v
	}
	if v := os.Getenv("FINSPECT_LOGGING_FORMAT"); v != "" {
		logging.Format = v
	}
	if v := os.Getenv("FINSPECT_LOGGING_OUTPUT"); v != "" {
		logging.Output = v
	}
}

// Helper functions
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func parseBool(s string) bool {
	switch s {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}
