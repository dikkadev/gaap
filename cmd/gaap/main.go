package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dikkadev/gaap/pkg/config"
	"github.com/dikkadev/gaap/pkg/github"
	"github.com/dikkadev/gaap/pkg/installer"
	"github.com/dikkadev/gaap/pkg/storage"
)

const usage = `GAAP: GitHub as a Package Manager

Usage:
  gaap [command] [flags] [arguments]

Commands:
  install     Install a package (format: owner/repo)
  update      Update installed packages
  remove      Remove a package (format: owner/repo)
  list        List installed packages
  configure   Configure GAAP settings

Common Flags:
  --dry-run   Show what would be done without making changes

Use "gaap [command] --help" for more information about a command.`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(usage)
		return nil
	}

	cmd := args[0]
	cmdArgs := args[1:]

	// Load configuration (but don't require database for help)
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Handle commands that don't need database
	switch cmd {
	case "help":
		fmt.Println(usage)
		return nil
	case "configure":
		return handleConfigure(cfg)
	}

	// Set up command flags
	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	installCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: gaap install [flags] owner/repo

Install a package from GitHub releases.

Flags:
  -non-interactive  Run in non-interactive mode
  -freeze          Freeze the package version
  -dry-run         Show what would be done without making changes
  -help            Show this help message
`)
	}
	installNonInteractive := installCmd.Bool("non-interactive", false, "Run in non-interactive mode")
	installFreeze := installCmd.Bool("freeze", false, "Freeze the package version")
	installDryRun := installCmd.Bool("dry-run", false, "Show what would be done without making changes")
	installHelp := installCmd.Bool("help", false, "Show help for install command")

	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	updateCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: gaap update [flags]

Update all installed packages to their latest versions.
Frozen packages will be skipped unless explicitly specified.

Flags:
  -dry-run         Show what would be done without making changes
  -help            Show this help message
`)
	}
	updateDryRun := updateCmd.Bool("dry-run", false, "Show what would be done without making changes")
	updateHelp := updateCmd.Bool("help", false, "Show help for update command")

	removeCmd := flag.NewFlagSet("remove", flag.ExitOnError)
	removeCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: gaap remove [flags] owner/repo

Remove an installed package.

Flags:
  -dry-run         Show what would be done without making changes
  -help            Show this help message
`)
	}
	removeDryRun := removeCmd.Bool("dry-run", false, "Show what would be done without making changes")
	removeHelp := removeCmd.Bool("help", false, "Show help for remove command")

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: gaap list [flags]

List all installed packages.

Flags:
  -help            Show this help message
`)
	}
	listHelp := listCmd.Bool("help", false, "Show help for list command")

	// Parse command specific flags and handle help
	switch cmd {
	case "install":
		installCmd.Parse(cmdArgs)
		if *installHelp {
			installCmd.Usage()
			return nil
		}
	case "update":
		updateCmd.Parse(cmdArgs)
		if *updateHelp {
			updateCmd.Usage()
			return nil
		}
	case "remove":
		removeCmd.Parse(cmdArgs)
		if *removeHelp {
			removeCmd.Usage()
			return nil
		}
	case "list":
		listCmd.Parse(cmdArgs)
		if *listHelp {
			listCmd.Usage()
			return nil
		}
	}

	// Ensure GAAP directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	// Initialize storage
	dirs := cfg.GetDirectories()
	dbPath := filepath.Join(dirs.DB, "gaap.db")
	store, err := storage.NewLibSQL("file:" + dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer store.Close()

	if err := store.Initialize(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create GitHub client
	ghClient := github.NewClient(cfg.GitHubToken)

	// Execute command
	ctx := context.Background()
	switch cmd {
	case "install":
		opts := installer.Options{
			NonInteractive: *installNonInteractive,
			Freeze:         *installFreeze,
			DryRun:         *installDryRun,
		}
		return handleInstall(ctx, installCmd.Args(), cfg, store, ghClient, opts)

	case "update":
		opts := installer.Options{
			DryRun: *updateDryRun,
		}
		return handleUpdate(ctx, updateCmd.Args(), cfg, store, ghClient, opts)

	case "remove":
		opts := installer.Options{
			DryRun: *removeDryRun,
		}
		return handleRemove(ctx, removeCmd.Args(), cfg, store, opts)

	case "list":
		return handleList(ctx, store)

	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func handleInstall(ctx context.Context, args []string, cfg *config.Config, store storage.Storage, ghClient github.Client, opts installer.Options) error {
	if len(args) < 1 {
		return fmt.Errorf("package name required")
	}

	return installer.Install(ctx, args[0], cfg, store, ghClient, opts)
}

func handleUpdate(ctx context.Context, args []string, cfg *config.Config, store storage.Storage, ghClient github.Client, opts installer.Options) error {
	packages, err := store.ListPackages(ctx)
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	if len(packages) == 0 {
		return fmt.Errorf("no packages installed")
	}

	for _, pkg := range packages {
		if pkg.Frozen {
			fmt.Printf("Skipping frozen package %s/%s\n", pkg.Owner, pkg.Repo)
			continue
		}

		if err := installer.Update(ctx, pkg, cfg, store, ghClient, opts); err != nil {
			fmt.Printf("Failed to update %s/%s: %v\n", pkg.Owner, pkg.Repo, err)
		}
	}

	return nil
}

func handleRemove(ctx context.Context, args []string, cfg *config.Config, store storage.Storage, opts installer.Options) error {
	if len(args) < 1 {
		return fmt.Errorf("package name required")
	}

	parts := strings.Split(args[0], "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid package name: %s (expected format: owner/repo)", args[0])
	}

	pkg, err := store.GetPackage(ctx, parts[0], parts[1])
	if err != nil {
		return fmt.Errorf("failed to get package: %w", err)
	}

	if pkg == nil {
		return fmt.Errorf("package not found: %s", args[0])
	}

	return installer.Remove(ctx, pkg, cfg, store, opts)
}

func handleList(ctx context.Context, store storage.Storage) error {
	packages, err := store.ListPackages(ctx)
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	if len(packages) == 0 {
		fmt.Println("No packages installed")
		return nil
	}

	fmt.Println("Installed packages:")
	for _, pkg := range packages {
		frozen := ""
		if pkg.Frozen {
			frozen = " (frozen)"
		}
		fmt.Printf("  %s/%s@%s%s\n", pkg.Owner, pkg.Repo, pkg.Version, frozen)
	}

	return nil
}

func handleConfigure(cfg *config.Config) error {
	// Get available operations
	ops := config.GetOperations()

	// Create choices for gum choose
	var choices []string
	for _, op := range ops {
		choices = append(choices, fmt.Sprintf("%s - %s", op.Name, op.Description))
	}

	// Use gum choose to select operation
	args := append([]string{"choose"}, choices...)
	cmd := exec.Command("gum", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			// User pressed Ctrl+C or similar
			return nil
		}
		return fmt.Errorf("failed to get choice: %w", err)
	}

	// Parse selected operation
	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return nil
	}

	// Find and execute selected operation
	selectedName := strings.Split(selected, " - ")[0]
	for _, op := range ops {
		if op.Name == selectedName {
			return op.Handler(cfg)
		}
	}

	return fmt.Errorf("unknown operation: %s", selectedName)
}
