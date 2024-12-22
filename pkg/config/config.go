package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the GAAP configuration
type Config struct {
	// Directory where GAAP stores all its data
	RootDir string `json:"root_dir"`
	// GitHub token for API access (optional)
	GitHubToken string `json:"github_token,omitempty"`
}

// Directories represents the GAAP directory structure
type Directories struct {
	// Root directory for all GAAP data
	Root string
	// Directory containing symlinks to executables
	Bin string
	// Directory containing actual binaries
	BinActual string
	// Directory for configuration files
	Config string
	// Directory for database files
	DB string
	// Directory for log files
	Logs string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &Config{
		RootDir: filepath.Join(homeDir, "gaap"),
	}
}

// GetDirectories returns the GAAP directory structure based on the root directory
func (c *Config) GetDirectories() *Directories {
	return &Directories{
		Root:      c.RootDir,
		Bin:       filepath.Join(c.RootDir, "bin"),
		BinActual: filepath.Join(c.RootDir, "bin", "actual"),
		Config:    filepath.Join(c.RootDir, "config"),
		DB:        filepath.Join(c.RootDir, "db"),
		Logs:      filepath.Join(c.RootDir, "logs"),
	}
}

// Load loads the configuration from the default location
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "gaap")
	configFile := filepath.Join(configDir, "config.json")

	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Save saves the configuration to the default location
func (c *Config) Save() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "gaap")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// EnsureDirectories creates all necessary GAAP directories if they don't exist
func (c *Config) EnsureDirectories() error {
	dirs := c.GetDirectories()
	for _, dir := range []string{
		dirs.Root,
		dirs.Bin,
		dirs.BinActual,
		dirs.Config,
		dirs.DB,
		dirs.Logs,
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create a README in the bin directory
	readmePath := filepath.Join(dirs.Bin, "README.md")
	readmeContent := []byte("# GAAP Binaries Directory\n\nThis directory contains symlinks to installed binaries.\nDo not modify the contents of this directory manually.\nUse the `gaap` command-line tool to manage packages.\n")

	if err := os.WriteFile(readmePath, readmeContent, 0644); err != nil {
		return fmt.Errorf("failed to create bin README: %w", err)
	}

	return nil
}
