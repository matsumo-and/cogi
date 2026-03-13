package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Database    DatabaseConfig    `mapstructure:"database"`
	Embedding   EmbeddingConfig   `mapstructure:"embedding"`
	Indexing    IndexingConfig    `mapstructure:"indexing"`
	Performance PerformanceConfig `mapstructure:"performance"`
}

// DatabaseConfig contains SQLite database settings
type DatabaseConfig struct {
	Path        string `mapstructure:"path"`
	WALMode     bool   `mapstructure:"wal_mode"`
	CacheSizeMB int    `mapstructure:"cache_size_mb"`
}

// EmbeddingConfig contains embedding model settings
type EmbeddingConfig struct {
	Provider  string `mapstructure:"provider"`
	Model     string `mapstructure:"model"`
	Endpoint  string `mapstructure:"endpoint"`
	Dimension int    `mapstructure:"dimension"`
	BatchSize int    `mapstructure:"batch_size"`
}

// IndexingConfig contains indexing settings
type IndexingConfig struct {
	MaxFileSizeMB   int      `mapstructure:"max_file_size_mb"`
	ExcludePatterns []string `mapstructure:"exclude_patterns"`
}

// PerformanceConfig contains performance tuning settings
type PerformanceConfig struct {
	MaxWorkers int `mapstructure:"max_workers"`
}

var globalConfig *Config

// Load loads the configuration from file and returns it
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Determine config file path
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir := filepath.Join(home, ".cogi")
		v.AddConfigPath(configDir)
		v.SetConfigType("yaml")
		v.SetConfigName("config")
	}

	// Read config file if it exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; use defaults
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Expand paths
	if err := expandPaths(&cfg); err != nil {
		return nil, err
	}

	globalConfig = &cfg
	return &cfg, nil
}

// Get returns the global configuration
func Get() *Config {
	if globalConfig == nil {
		// Load with defaults if not initialized
		cfg, err := Load("")
		if err != nil {
			panic(fmt.Sprintf("failed to load config: %v", err))
		}
		return cfg
	}
	return globalConfig
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	home, _ := os.UserHomeDir()
	cogiDir := filepath.Join(home, ".cogi")

	// Database defaults
	v.SetDefault("database.path", filepath.Join(cogiDir, "data.db"))
	v.SetDefault("database.wal_mode", true)
	v.SetDefault("database.cache_size_mb", 256)

	// Embedding defaults
	v.SetDefault("embedding.provider", "ollama")
	v.SetDefault("embedding.model", "mxbai-embed-large")
	v.SetDefault("embedding.endpoint", "http://localhost:11434")
	v.SetDefault("embedding.dimension", 1024)
	v.SetDefault("embedding.batch_size", 32)

	// Indexing defaults
	v.SetDefault("indexing.max_file_size_mb", 10)
	v.SetDefault("indexing.exclude_patterns", []string{
		"*/node_modules/*",
		"*/vendor/*",
		"*/.git/*",
		"*/dist/*",
		"*/build/*",
		"*/.next/*",
		"*/.venv/*",
		"*/venv/*",
		"*/__pycache__/*",
		"*.pyc",
		"*.min.js",
		"*.min.css",
	})

	// Performance defaults
	v.SetDefault("performance.max_workers", 8)
}

// expandPaths expands ~ and environment variables in paths
func expandPaths(cfg *Config) error {
	var err error

	cfg.Database.Path, err = expandPath(cfg.Database.Path)
	if err != nil {
		return err
	}

	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) (string, error) {
	if path == "" {
		return path, nil
	}

	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}

	return path, nil
}

// EnsureConfigDir ensures the config directory exists
func EnsureConfigDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".cogi")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return nil
}
