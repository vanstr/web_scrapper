package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents application configuration
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Server   ServerConfig   `yaml:"server"`
	Files    FilesConfig    `yaml:"files"`
}

// DatabaseConfig contains database settings
type DatabaseConfig struct {
	Path string `yaml:"path"` // SQLite database file path
}

// ServerConfig contains server settings
type ServerConfig struct {
	Port int `yaml:"port"` // Web server port (default: 8080)
}

// FilesConfig contains file processing settings
type FilesConfig struct {
	FolderPath    string   `yaml:"folder_path"`    // Path to files folder (toprocess)
	FetchInterval int      `yaml:"fetch_interval"` // Fetch interval in minutes (default: 60)
	RemoteFeeds   []string `yaml:"remote_feeds"`   // URLs for remote feeds
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults if not provided
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Files.FetchInterval == 0 {
		config.Files.FetchInterval = 60
	}
	if config.Database.Path == "" {
		config.Database.Path = "./data.db"
	}
	if config.Files.FolderPath == "" {
		config.Files.FolderPath = "./toprocess"
	}

	return &config, nil
}
