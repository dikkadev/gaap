package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	expectedRootDir := filepath.Join(homeDir, "gaap")
	if config.RootDir != expectedRootDir {
		t.Errorf("Expected root dir %s, got %s", expectedRootDir, config.RootDir)
	}
}

func TestGetDirectories(t *testing.T) {
	config := &Config{
		RootDir: "/test/root",
	}

	dirs := config.GetDirectories()

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"Root", dirs.Root, "/test/root"},
		{"Bin", dirs.Bin, "/test/root/bin"},
		{"BinActual", dirs.BinActual, "/test/root/bin/actual"},
		{"Config", dirs.Config, "/test/root/config"},
		{"DB", dirs.DB, "/test/root/db"},
		{"Logs", dirs.Logs, "/test/root/logs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.got)
			}
		})
	}
}

func TestConfigSaveLoad(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "gaap-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up a test environment
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	// Create a test config
	testConfig := &Config{
		RootDir:     filepath.Join(tmpDir, "gaap"),
		GitHubToken: "test-token",
	}

	// Save the config
	if err := testConfig.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load the config
	loadedConfig, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Compare the configs
	if loadedConfig.RootDir != testConfig.RootDir {
		t.Errorf("Expected root dir %s, got %s", testConfig.RootDir, loadedConfig.RootDir)
	}
	if loadedConfig.GitHubToken != testConfig.GitHubToken {
		t.Errorf("Expected GitHub token %s, got %s", testConfig.GitHubToken, loadedConfig.GitHubToken)
	}
}

func TestEnsureDirectories(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "gaap-dirs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		RootDir: tmpDir,
	}

	if err := config.EnsureDirectories(); err != nil {
		t.Fatalf("Failed to ensure directories: %v", err)
	}

	// Check if all directories were created
	dirs := config.GetDirectories()
	for _, dir := range []string{
		dirs.Root,
		dirs.Bin,
		dirs.BinActual,
		dirs.Config,
		dirs.DB,
		dirs.Logs,
	} {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}

	// Check if README was created in bin directory
	readmePath := filepath.Join(dirs.Bin, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Error("README.md was not created in bin directory")
	}
}
