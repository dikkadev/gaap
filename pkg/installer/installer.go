package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dikkadev/gaap/pkg/config"
	"github.com/dikkadev/gaap/pkg/github"
	"github.com/dikkadev/gaap/pkg/storage"
)

// Options represents installation options
type Options struct {
	NonInteractive bool
	Freeze         bool
	DryRun         bool
}

// Install installs a package from GitHub
func Install(ctx context.Context, repoPath string, cfg *config.Config, store storage.Storage, ghClient github.Client, opts Options) error {
	// Parse repository path (owner/repo)
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository path: %s (expected format: owner/repo)", repoPath)
	}
	owner, repo := parts[0], parts[1]

	// Check if package is already installed
	existing, err := store.GetPackage(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to check existing package: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("package %s/%s is already installed", owner, repo)
	}

	// Get latest release
	release, err := ghClient.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	if len(release.Assets) == 0 {
		return fmt.Errorf("no assets found in the latest release")
	}

	// TODO: Implement asset selection based on OS/arch
	asset := release.Assets[0]

	// Prepare installation paths
	dirs := cfg.GetDirectories()
	binaryName := strings.TrimSuffix(asset.Name, filepath.Ext(asset.Name))
	actualPath := filepath.Join(dirs.BinActual, fmt.Sprintf("%s-%s-%s", owner, repo, release.TagName))
	symlinkPath := filepath.Join(dirs.Bin, binaryName)

	if opts.DryRun {
		fmt.Printf("Would install %s/%s@%s:\n", owner, repo, release.TagName)
		fmt.Printf("  Asset: %s\n", asset.Name)
		fmt.Printf("  Binary: %s\n", actualPath)
		fmt.Printf("  Symlink: %s\n", symlinkPath)
		return nil
	}

	// Download the asset
	if err := ghClient.DownloadAsset(ctx, asset.DownloadURL, actualPath); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	// Make the binary executable
	if err := os.Chmod(actualPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Create symlink
	if err := os.Symlink(actualPath, symlinkPath); err != nil {
		os.Remove(actualPath) // Clean up on error
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Add to database
	pkg := &storage.Package{
		Owner:       owner,
		Repo:        repo,
		Version:     release.TagName,
		InstallPath: actualPath,
		BinaryName:  binaryName,
		Frozen:      opts.Freeze,
	}

	if err := store.AddPackage(ctx, pkg); err != nil {
		os.Remove(symlinkPath) // Clean up on error
		os.Remove(actualPath)
		return fmt.Errorf("failed to add package to database: %w", err)
	}

	fmt.Printf("Successfully installed %s/%s@%s\n", owner, repo, release.TagName)
	return nil
}

// Update updates an installed package
func Update(ctx context.Context, pkg *storage.Package, cfg *config.Config, store storage.Storage, ghClient github.Client, opts Options) error {
	// Get latest release
	release, err := ghClient.GetLatestRelease(ctx, pkg.Owner, pkg.Repo)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	// Check if update is needed
	if release.TagName == pkg.Version {
		fmt.Printf("%s/%s is already at the latest version (%s)\n", pkg.Owner, pkg.Repo, pkg.Version)
		return nil
	}

	if len(release.Assets) == 0 {
		return fmt.Errorf("no assets found in the latest release")
	}

	// TODO: Implement asset selection based on OS/arch
	asset := release.Assets[0]

	// Prepare installation paths
	dirs := cfg.GetDirectories()
	binaryName := pkg.BinaryName
	actualPath := filepath.Join(dirs.BinActual, fmt.Sprintf("%s-%s-%s", pkg.Owner, pkg.Repo, release.TagName))
	symlinkPath := filepath.Join(dirs.Bin, binaryName)

	if opts.DryRun {
		fmt.Printf("Would update %s/%s from %s to %s:\n", pkg.Owner, pkg.Repo, pkg.Version, release.TagName)
		fmt.Printf("  Asset: %s\n", asset.Name)
		fmt.Printf("  Binary: %s\n", actualPath)
		fmt.Printf("  Symlink: %s\n", symlinkPath)
		return nil
	}

	// Download the new version
	if err := ghClient.DownloadAsset(ctx, asset.DownloadURL, actualPath); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	// Make the binary executable
	if err := os.Chmod(actualPath, 0755); err != nil {
		os.Remove(actualPath) // Clean up on error
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Update symlink
	tmpSymlink := symlinkPath + ".tmp"
	if err := os.Symlink(actualPath, tmpSymlink); err != nil {
		os.Remove(actualPath) // Clean up on error
		return fmt.Errorf("failed to create temporary symlink: %w", err)
	}

	if err := os.Rename(tmpSymlink, symlinkPath); err != nil {
		os.Remove(tmpSymlink) // Clean up on error
		os.Remove(actualPath)
		return fmt.Errorf("failed to update symlink: %w", err)
	}

	// Remove old binary
	os.Remove(pkg.InstallPath) // Ignore error, as the file might be in use

	// Update database
	pkg.Version = release.TagName
	pkg.InstallPath = actualPath

	if err := store.UpdatePackage(ctx, pkg); err != nil {
		return fmt.Errorf("failed to update package in database: %w", err)
	}

	fmt.Printf("Successfully updated %s/%s to %s\n", pkg.Owner, pkg.Repo, release.TagName)
	return nil
}

// Remove removes an installed package
func Remove(ctx context.Context, pkg *storage.Package, cfg *config.Config, store storage.Storage, opts Options) error {
	dirs := cfg.GetDirectories()
	symlinkPath := filepath.Join(dirs.Bin, pkg.BinaryName)

	if opts.DryRun {
		fmt.Printf("Would remove %s/%s@%s:\n", pkg.Owner, pkg.Repo, pkg.Version)
		fmt.Printf("  Binary: %s\n", pkg.InstallPath)
		fmt.Printf("  Symlink: %s\n", symlinkPath)
		return nil
	}

	// Remove from database first
	if err := store.DeletePackage(ctx, pkg.Owner, pkg.Repo); err != nil {
		return fmt.Errorf("failed to remove package from database: %w", err)
	}

	// Remove symlink
	if err := os.Remove(symlinkPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	// Remove binary
	if err := os.Remove(pkg.InstallPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove binary: %w", err)
	}

	fmt.Printf("Successfully removed %s/%s\n", pkg.Owner, pkg.Repo)
	return nil
}
