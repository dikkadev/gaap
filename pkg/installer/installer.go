package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dikkadev/gaap/pkg/config"
	"github.com/dikkadev/gaap/pkg/github"
	"github.com/dikkadev/gaap/pkg/platform"
	"github.com/dikkadev/gaap/pkg/storage"
)

type Options struct {
	NonInteractive bool
	Freeze         bool
	DryRun         bool
}

func ensureDir(dir string) error {
	// Try to create directory normally first
	err := os.MkdirAll(dir, 0755)
	if err == nil {
		return nil
	}

	// If normal creation fails, try with sudo
	cmd := exec.Command("sudo", "mkdir", "-p", dir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create directory with sudo: %w", err)
	}

	// Set permissions
	cmd = exec.Command("sudo", "chown", "-R", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()), dir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set permissions with sudo: %w", err)
	}

	return nil
}

func Install(ctx context.Context, repoName string, cfg *config.Config, store storage.Storage, ghClient github.Client, opts Options) error {
	// Parse repository name
	parts := strings.Split(repoName, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository name: %s (expected format: owner/repo)", repoName)
	}
	owner, repo := parts[0], parts[1]

	// Check if package is already installed
	pkg, err := store.GetPackage(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to check if package is installed: %w", err)
	}
	if pkg != nil {
		return fmt.Errorf("package %s/%s is already installed", owner, repo)
	}

	// If in dry-run mode, just print what would be done
	if opts.DryRun {
		dirs := cfg.GetDirectories()
		fmt.Printf("Would install package: %s/%s\n", owner, repo)
		fmt.Printf("Target directory: %s\n", dirs.Bin)
		return nil
	}

	// Get latest release
	release, err := ghClient.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	// Select appropriate asset for current platform
	plat := platform.Current()
	asset, err := plat.SelectAsset(release.Assets)
	if err != nil {
		return fmt.Errorf("failed to select asset: %w", err)
	}

	// In non-interactive mode, proceed without confirmation
	if !opts.NonInteractive {
		fmt.Printf("Installing %s/%s version %s\n", owner, repo, release.TagName)
		fmt.Printf("Asset: %s\n", asset.Name)
		fmt.Print("Proceed? [Y/n]: ")

		var response string
		fmt.Scanln(&response)
		if response != "" && strings.ToLower(response) != "y" {
			return fmt.Errorf("installation cancelled by user")
		}
	}

	// Create unique binary name
	binaryName := fmt.Sprintf("%s-%s-%s", owner, repo, release.TagName)
	if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") {
		binaryName += ".exe"
	}

	// Get directories
	dirs := cfg.GetDirectories()

	// Ensure directories exist
	if err := ensureDir(dirs.BinActual); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}
	if err := ensureDir(dirs.Bin); err != nil {
		return fmt.Errorf("failed to create symlink directory: %w", err)
	}

	binaryPath := filepath.Join(dirs.BinActual, binaryName)
	symlinkPath := filepath.Join(dirs.Bin, repo)
	if platform.Current().OS == "windows" {
		symlinkPath += ".exe"
	}

	// Download asset
	if !opts.DryRun {
		if err := ghClient.DownloadAsset(ctx, asset, binaryPath); err != nil {
			return fmt.Errorf("failed to download asset: %w", err)
		}

		// Make binary executable
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}

		// Create symlink
		if err := os.Symlink(binaryPath, symlinkPath); err != nil {
			// Clean up on failure
			os.Remove(binaryPath)
			return fmt.Errorf("failed to create symlink: %w", err)
		}

		// Store package information with freeze status
		pkg = &storage.Package{
			Owner:    owner,
			Repo:     repo,
			Version:  release.TagName,
			Binary:   binaryName,
			Frozen:   opts.Freeze,
			Platform: plat.String(),
		}
		if err := store.AddPackage(ctx, pkg); err != nil {
			// Clean up on failure
			os.Remove(symlinkPath)
			os.Remove(binaryPath)
			return fmt.Errorf("failed to add package to database: %w", err)
		}
	}

	fmt.Printf("Successfully installed %s/%s@%s\n", owner, repo, release.TagName)
	if opts.Freeze {
		fmt.Printf("Package version is frozen at %s\n", release.TagName)
	}
	return nil
}

func Update(ctx context.Context, pkg *storage.Package, cfg *config.Config, store storage.Storage, ghClient github.Client, opts Options) error {
	// Skip frozen packages unless explicitly unfrozen
	if pkg.Frozen {
		fmt.Printf("Skipping frozen package %s/%s@%s\n", pkg.Owner, pkg.Repo, pkg.Version)
		return nil
	}

	// Get directories
	dirs := cfg.GetDirectories()

	// If in dry-run mode, just print what would be done
	if opts.DryRun {
		fmt.Printf("Would update package: %s/%s\n", pkg.Owner, pkg.Repo)
		return nil
	}

	// Get latest release
	release, err := ghClient.GetLatestRelease(ctx, pkg.Owner, pkg.Repo)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	// Skip if already at latest version
	if release.TagName == pkg.Version {
		fmt.Printf("Package %s/%s is already at latest version %s\n", pkg.Owner, pkg.Repo, pkg.Version)
		return nil
	}

	// Select appropriate asset for current platform
	plat := platform.Current()
	asset, err := plat.SelectAsset(release.Assets)
	if err != nil {
		return fmt.Errorf("failed to select asset: %w", err)
	}

	// In non-interactive mode, proceed without confirmation
	if !opts.NonInteractive {
		fmt.Printf("Updating %s/%s from %s to %s\n", pkg.Owner, pkg.Repo, pkg.Version, release.TagName)
		fmt.Printf("Asset: %s\n", asset.Name)
		fmt.Print("Proceed? [Y/n]: ")

		var response string
		fmt.Scanln(&response)
		if response != "" && strings.ToLower(response) != "y" {
			return fmt.Errorf("update cancelled by user")
		}
	}

	// Create unique binary name
	binaryName := fmt.Sprintf("%s-%s-%s", pkg.Owner, pkg.Repo, release.TagName)
	if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") {
		binaryName += ".exe"
	}

	// Ensure directories exist
	if err := ensureDir(dirs.BinActual); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}
	if err := ensureDir(dirs.Bin); err != nil {
		return fmt.Errorf("failed to create symlink directory: %w", err)
	}

	oldBinaryPath := filepath.Join(dirs.BinActual, pkg.Binary)
	newBinaryPath := filepath.Join(dirs.BinActual, binaryName)
	symlinkPath := filepath.Join(dirs.Bin, pkg.Repo)
	if plat.OS == "windows" {
		symlinkPath += ".exe"
	}

	if !opts.DryRun {
		// Download new version
		if err := ghClient.DownloadAsset(ctx, asset, newBinaryPath); err != nil {
			return fmt.Errorf("failed to download asset: %w", err)
		}

		// Make binary executable
		if err := os.Chmod(newBinaryPath, 0755); err != nil {
			os.Remove(newBinaryPath)
			return fmt.Errorf("failed to make binary executable: %w", err)
		}

		// Update symlink
		if err := os.Remove(symlinkPath); err != nil {
			os.Remove(newBinaryPath)
			return fmt.Errorf("failed to remove old symlink: %w", err)
		}
		if err := os.Symlink(newBinaryPath, symlinkPath); err != nil {
			os.Remove(newBinaryPath)
			return fmt.Errorf("failed to create new symlink: %w", err)
		}

		// Update database
		pkg.Version = release.TagName
		pkg.Binary = binaryName
		pkg.Platform = plat.String()
		if err := store.UpdatePackage(ctx, pkg); err != nil {
			os.Remove(symlinkPath)
			os.Remove(newBinaryPath)
			return fmt.Errorf("failed to update package in database: %w", err)
		}

		// Remove old binary
		os.Remove(oldBinaryPath)
	}

	fmt.Printf("Successfully updated %s/%s from %s to %s\n", pkg.Owner, pkg.Repo, pkg.Version, release.TagName)
	return nil
}

func Remove(ctx context.Context, pkg *storage.Package, cfg *config.Config, store storage.Storage, opts Options) error {
	// Get directories
	dirs := cfg.GetDirectories()

	// If in dry-run mode, just print what would be done
	if opts.DryRun {
		fmt.Printf("Would remove package: %s/%s\n", pkg.Owner, pkg.Repo)
		return nil
	}

	// Remove binary and symlink
	binaryPath := filepath.Join(dirs.BinActual, pkg.Binary)
	symlinkPath := filepath.Join(dirs.Bin, pkg.Repo)
	if platform.Current().OS == "windows" {
		symlinkPath += ".exe"
	}

	// Remove symlink first
	if err := os.Remove(symlinkPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	// Then remove binary
	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove binary: %w", err)
	}

	// Finally remove from database
	if err := store.DeletePackage(ctx, pkg.Owner, pkg.Repo); err != nil {
		return fmt.Errorf("failed to remove package from database: %w", err)
	}

	fmt.Printf("Successfully removed %s/%s\n", pkg.Owner, pkg.Repo)
	return nil
}
