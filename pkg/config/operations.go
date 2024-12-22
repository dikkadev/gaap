package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConfigureOptions represents options for configuration
type ConfigureOptions struct {
	NonInteractive bool
}

// ConfigureOperation represents a configuration operation
type ConfigureOperation struct {
	Name        string
	Description string
	Handler     func(*Config) error
}

// GetOperations returns available configuration operations
func GetOperations() []ConfigureOperation {
	return []ConfigureOperation{
		{
			Name:        "token",
			Description: "Set GitHub API token",
			Handler:     configureGitHubToken,
		},
		{
			Name:        "root",
			Description: "Change root directory",
			Handler:     configureRootDir,
		},
		{
			Name:        "show",
			Description: "Show current configuration",
			Handler:     showConfig,
		},
	}
}

func configureGitHubToken(cfg *Config) error {
	// Try to get token from gum input
	cmd := exec.Command("gum", "input", "--password", "--placeholder", "Enter GitHub token (press Enter to clear)")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil
		}
		return fmt.Errorf("failed to get input: %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		// Clear token if empty input
		cfg.GitHubToken = ""
		fmt.Println("GitHub token cleared")
	} else {
		cfg.GitHubToken = token
		fmt.Println("GitHub token updated")
	}

	return cfg.Save()
}

func configureRootDir(cfg *Config) error {
	// Get current directory as default
	currentDir := cfg.RootDir

	// Use gum input with current dir as placeholder
	cmd := exec.Command("gum", "input", "--placeholder", fmt.Sprintf("Enter new root directory (current: %s)", currentDir))
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil
		}
		return fmt.Errorf("failed to get input: %w", err)
	}

	newDir := strings.TrimSpace(string(output))
	if newDir == "" {
		fmt.Println("Root directory unchanged")
		return nil
	}

	// Expand ~ to home directory
	if newDir == "~" || newDir[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		if newDir == "~" {
			newDir = home
		} else {
			newDir = filepath.Join(home, newDir[2:])
		}
	}

	// Make path absolute
	absPath, err := filepath.Abs(newDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Confirm directory change
	cmd = exec.Command("gum", "confirm", fmt.Sprintf("Change root directory to %s?", absPath))
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil
		}
		fmt.Println("Operation cancelled")
		return nil
	}

	cfg.RootDir = absPath
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Root directory changed to %s\n", absPath)
	fmt.Println("Note: You'll need to move any existing packages to the new location manually")
	return nil
}

func showConfig(cfg *Config) error {
	fmt.Println("Current configuration:")
	fmt.Printf("Root directory: %s\n", cfg.RootDir)
	if cfg.GitHubToken != "" {
		fmt.Println("GitHub token: [set]")
	} else {
		fmt.Println("GitHub token: [not set]")
	}
	return nil
}
