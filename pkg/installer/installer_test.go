package installer

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/dikkadev/gaap/pkg/config"
	"github.com/dikkadev/gaap/pkg/github"
	"github.com/dikkadev/gaap/pkg/storage"
)

// mockGitHubClient implements github.Client for testing
type mockGitHubClient struct {
	latestRelease       *github.Release
	releases            []*github.Release
	getLatestReleaseErr error
	getReleasesErr      error
	downloadAssetErr    error
	searchErr           error
}

func (m *mockGitHubClient) GetLatestRelease(ctx context.Context, owner, repo string) (*github.Release, error) {
	if m.getLatestReleaseErr != nil {
		return nil, m.getLatestReleaseErr
	}
	return m.latestRelease, nil
}

func (m *mockGitHubClient) GetReleases(ctx context.Context, owner, repo string) ([]*github.Release, error) {
	if m.getReleasesErr != nil {
		return nil, m.getReleasesErr
	}
	return m.releases, nil
}

func (m *mockGitHubClient) DownloadAsset(ctx context.Context, asset *github.Asset, destPath string) error {
	if m.downloadAssetErr != nil {
		return m.downloadAssetErr
	}
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	// Write a test binary
	return os.WriteFile(destPath, []byte("test binary"), 0755)
}

func (m *mockGitHubClient) SearchRepositories(ctx context.Context, query string) (*github.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return &github.SearchResult{}, nil
}

func (m *mockGitHubClient) SearchRepositoriesByName(ctx context.Context, name string) (*github.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return &github.SearchResult{}, nil
}

func (m *mockGitHubClient) SearchRepositoriesByUser(ctx context.Context, user string) (*github.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return &github.SearchResult{}, nil
}

// mockStorage implements storage.Storage for testing
type mockStorage struct {
	packages map[string]*storage.Package
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		packages: make(map[string]*storage.Package),
	}
}

func (m *mockStorage) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockStorage) AddPackage(ctx context.Context, pkg *storage.Package) error {
	key := pkg.Owner + "/" + pkg.Repo
	m.packages[key] = pkg
	return nil
}

func (m *mockStorage) GetPackage(ctx context.Context, owner, repo string) (*storage.Package, error) {
	key := owner + "/" + repo
	return m.packages[key], nil
}

func (m *mockStorage) ListPackages(ctx context.Context) ([]*storage.Package, error) {
	var packages []*storage.Package
	for _, pkg := range m.packages {
		packages = append(packages, pkg)
	}
	return packages, nil
}

func (m *mockStorage) UpdatePackage(ctx context.Context, pkg *storage.Package) error {
	key := pkg.Owner + "/" + pkg.Repo
	m.packages[key] = pkg
	return nil
}

func (m *mockStorage) DeletePackage(ctx context.Context, owner, repo string) error {
	key := owner + "/" + repo
	delete(m.packages, key)
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

func TestInstall(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "installer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binDir := filepath.Join(tmpDir, "bin")
	binActualDir := filepath.Join(tmpDir, "bin/actual")
	if err := os.MkdirAll(binActualDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directories: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		RootDir: tmpDir,
	}

	// Create mock GitHub client
	ghClient := &mockGitHubClient{
		latestRelease: &github.Release{
			TagName: "v1.0.0",
			Assets: []github.Asset{
				{
					Name:        "test-linux-amd64",
					DownloadURL: "https://example.com/test",
				},
			},
		},
	}

	// Create mock storage
	store := newMockStorage()

	// Test successful installation
	err = Install(context.Background(), "owner/repo", cfg, store, ghClient, Options{})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify binary was installed
	binaryPath := filepath.Join(binActualDir, "owner-repo-v1.0.0")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Error("Binary file was not created")
	}

	// Verify symlink was created
	symlinkPath := filepath.Join(binDir, "repo")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was not created")
	}

	// Verify package was added to storage
	pkg, err := store.GetPackage(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("Failed to get package: %v", err)
	}
	if pkg == nil {
		t.Error("Package was not added to storage")
	}
	if pkg.Version != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", pkg.Version)
	}
}

func TestUpdate(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "installer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binDir := filepath.Join(tmpDir, "bin")
	binActualDir := filepath.Join(tmpDir, "bin/actual")
	if err := os.MkdirAll(binActualDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directories: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		RootDir: tmpDir,
	}

	// Create mock GitHub client
	ghClient := &mockGitHubClient{
		latestRelease: &github.Release{
			TagName: "v2.0.0",
			Assets: []github.Asset{
				{
					Name:        "test-linux-amd64",
					DownloadURL: "https://example.com/test",
				},
			},
		},
	}

	// Create mock storage with existing package
	store := newMockStorage()
	existingPkg := &storage.Package{
		Owner:    "owner",
		Repo:     "repo",
		Version:  "v1.0.0",
		Binary:   "owner-repo-v1.0.0",
		Platform: "linux-amd64",
	}
	store.AddPackage(context.Background(), existingPkg)

	// Create old binary file
	oldBinaryPath := filepath.Join(binActualDir, existingPkg.Binary)
	if err := os.WriteFile(oldBinaryPath, []byte("old binary"), 0755); err != nil {
		t.Fatalf("Failed to create old binary: %v", err)
	}

	// Create symlink
	symlinkPath := filepath.Join(binDir, "repo")
	if err := os.Symlink(oldBinaryPath, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Test successful update
	err = Update(context.Background(), existingPkg, cfg, store, ghClient, Options{})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify new binary was installed
	newBinaryPath := filepath.Join(binActualDir, "owner-repo-v2.0.0")
	if _, err := os.Stat(newBinaryPath); os.IsNotExist(err) {
		t.Error("New binary file was not created")
	}

	// Verify old binary was removed
	if _, err := os.Stat(oldBinaryPath); !os.IsNotExist(err) {
		t.Error("Old binary file was not removed")
	}

	// Verify symlink was updated
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}
	if target != newBinaryPath {
		t.Errorf("Symlink points to %s, expected %s", target, newBinaryPath)
	}

	// Verify package was updated in storage
	pkg, err := store.GetPackage(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("Failed to get package: %v", err)
	}
	if pkg.Version != "v2.0.0" {
		t.Errorf("Expected version v2.0.0, got %s", pkg.Version)
	}
}

func TestRemove(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "installer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binDir := filepath.Join(tmpDir, "bin")
	binActualDir := filepath.Join(tmpDir, "bin/actual")
	if err := os.MkdirAll(binActualDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directories: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		RootDir: tmpDir,
	}

	// Create mock storage with existing package
	store := newMockStorage()
	pkg := &storage.Package{
		Owner:    "owner",
		Repo:     "repo",
		Version:  "v1.0.0",
		Binary:   "owner-repo-v1.0.0",
		Platform: "linux-amd64",
	}
	store.AddPackage(context.Background(), pkg)

	// Create binary file
	binaryPath := filepath.Join(binActualDir, pkg.Binary)
	if err := os.WriteFile(binaryPath, []byte("test binary"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	// Create symlink
	symlinkPath := filepath.Join(binDir, "repo")
	if err := os.Symlink(binaryPath, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Test successful removal
	err = Remove(context.Background(), pkg, cfg, store, Options{})
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify binary was removed
	if _, err := os.Stat(binaryPath); !os.IsNotExist(err) {
		t.Error("Binary file was not removed")
	}

	// Verify symlink was removed
	if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
		t.Error("Symlink was not removed")
	}

	// Verify package was removed from storage
	pkg, err = store.GetPackage(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("Failed to get package: %v", err)
	}
	if pkg != nil {
		t.Error("Package was not removed from storage")
	}
}

func TestInstallDryRun(t *testing.T) {
	cfg := &config.Config{RootDir: t.TempDir()}
	store := &mockStorage{packages: make(map[string]*storage.Package)}
	ghClient := &mockGitHubClient{
		latestRelease: &github.Release{
			TagName: "v1.0.0",
			Assets: []github.Asset{
				{
					Name:        "test-linux-amd64",
					DownloadURL: "https://example.com/test",
				},
			},
		},
	}

	// Test dry-run mode
	err := Install(context.Background(), "test/repo", cfg, store, ghClient, Options{DryRun: true})
	if err != nil {
		t.Errorf("Install with dry-run failed: %v", err)
	}

	// Verify no files were created
	dirs := cfg.GetDirectories()
	if _, err := os.Stat(dirs.BinActual); !os.IsNotExist(err) {
		t.Error("Dry run created files when it shouldn't")
	}
}

func TestInstallNonInteractive(t *testing.T) {
	cfg := &config.Config{RootDir: t.TempDir()}
	if err := cfg.EnsureDirectories(); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	store := &mockStorage{packages: make(map[string]*storage.Package)}
	ghClient := &mockGitHubClient{
		latestRelease: &github.Release{
			TagName: "v1.0.0",
			Assets: []github.Asset{
				{
					Name:        "test-linux-amd64",
					DownloadURL: "https://example.com/test",
				},
			},
		},
	}

	// Test non-interactive mode
	err := Install(context.Background(), "test/repo", cfg, store, ghClient, Options{NonInteractive: true})
	if err != nil {
		t.Errorf("Install with non-interactive failed: %v", err)
	}

	// Verify package was installed
	pkg, err := store.GetPackage(context.Background(), "test", "repo")
	if err != nil || pkg == nil {
		t.Error("Package was not installed in non-interactive mode")
	}
}

func TestInstallFreeze(t *testing.T) {
	cfg := &config.Config{RootDir: t.TempDir()}
	if err := cfg.EnsureDirectories(); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	store := &mockStorage{packages: make(map[string]*storage.Package)}
	ghClient := &mockGitHubClient{
		latestRelease: &github.Release{
			TagName: "v1.0.0",
			Assets: []github.Asset{
				{
					Name:        "test-linux-amd64",
					DownloadURL: "https://example.com/test",
				},
			},
		},
	}

	// Test freeze mode
	err := Install(context.Background(), "test/repo", cfg, store, ghClient, Options{Freeze: true})
	if err != nil {
		t.Errorf("Install with freeze failed: %v", err)
	}

	// Verify package was installed and frozen
	pkg, err := store.GetPackage(context.Background(), "test", "repo")
	if err != nil || pkg == nil {
		t.Error("Package was not installed in freeze mode")
	}
	if !pkg.Frozen {
		t.Error("Package was not frozen when freeze flag was set")
	}
}

func TestUpdateDryRun(t *testing.T) {
	cfg := &config.Config{RootDir: t.TempDir()}
	store := &mockStorage{packages: make(map[string]*storage.Package)}
	ghClient := &mockGitHubClient{
		latestRelease: &github.Release{
			TagName: "v2.0.0",
			Assets: []github.Asset{
				{
					Name:        "test-linux-amd64",
					DownloadURL: "https://example.com/test",
				},
			},
		},
	}

	// Add an existing package
	pkg := &storage.Package{
		Owner:    "test",
		Repo:     "repo",
		Version:  "v1.0.0",
		Binary:   "test-repo-v1.0.0",
		Platform: "linux-amd64",
	}
	store.AddPackage(context.Background(), pkg)

	// Test dry-run mode
	err := Update(context.Background(), pkg, cfg, store, ghClient, Options{DryRun: true})
	if err != nil {
		t.Errorf("Update with dry-run failed: %v", err)
	}

	// Verify package was not updated
	pkg, err = store.GetPackage(context.Background(), "test", "repo")
	if err != nil || pkg.Version != "v1.0.0" {
		t.Error("Package was updated when it shouldn't have been in dry-run mode")
	}
}

func TestUpdateFrozen(t *testing.T) {
	cfg := &config.Config{RootDir: t.TempDir()}
	store := &mockStorage{packages: make(map[string]*storage.Package)}
	ghClient := &mockGitHubClient{
		latestRelease: &github.Release{
			TagName: "v2.0.0",
			Assets: []github.Asset{
				{
					Name:        "test-linux-amd64",
					DownloadURL: "https://example.com/test",
				},
			},
		},
	}

	// Add a frozen package
	pkg := &storage.Package{
		Owner:    "test",
		Repo:     "repo",
		Version:  "v1.0.0",
		Binary:   "test-repo-v1.0.0",
		Platform: "linux-amd64",
		Frozen:   true,
	}
	store.AddPackage(context.Background(), pkg)

	// Test update of frozen package
	err := Update(context.Background(), pkg, cfg, store, ghClient, Options{})
	if err != nil {
		t.Errorf("Update of frozen package failed: %v", err)
	}

	// Verify package was not updated
	pkg, err = store.GetPackage(context.Background(), "test", "repo")
	if err != nil || pkg.Version != "v1.0.0" {
		t.Error("Frozen package was updated when it shouldn't have been")
	}
}

func TestRemoveDryRun(t *testing.T) {
	cfg := &config.Config{RootDir: t.TempDir()}
	if err := cfg.EnsureDirectories(); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	store := &mockStorage{packages: make(map[string]*storage.Package)}

	// Add a package
	pkg := &storage.Package{
		Owner:   "test",
		Repo:    "repo",
		Version: "v1.0.0",
		Binary:  "test-binary",
	}
	store.AddPackage(context.Background(), pkg)

	// Create dummy files
	dirs := cfg.GetDirectories()
	binaryPath := filepath.Join(dirs.BinActual, pkg.Binary)
	symlinkPath := filepath.Join(dirs.Bin, pkg.Repo)
	os.MkdirAll(filepath.Dir(binaryPath), 0755)
	os.WriteFile(binaryPath, []byte("test"), 0755)
	os.Symlink(binaryPath, symlinkPath)

	// Test dry-run mode
	err := Remove(context.Background(), pkg, cfg, store, Options{DryRun: true})
	if err != nil {
		t.Errorf("Remove with dry-run failed: %v", err)
	}

	// Verify files still exist
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Error("Binary was removed in dry-run mode")
	}
	if _, err := os.Stat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was removed in dry-run mode")
	}

	// Verify package still exists in database
	pkg, err = store.GetPackage(context.Background(), "test", "repo")
	if err != nil || pkg == nil {
		t.Error("Package was removed from database in dry-run mode")
	}
}

func TestInstallErrors(t *testing.T) {
	testCases := []struct {
		name        string
		repoName    string
		ghClient    *mockGitHubClient
		setupFunc   func(*config.Config) error
		wantErr     bool
		errContains string
	}{
		{
			name:        "Invalid repository name",
			repoName:    "invalid-repo",
			wantErr:     true,
			errContains: "invalid repository name",
		},
		{
			name:     "GitHub API error",
			repoName: "owner/repo",
			ghClient: &mockGitHubClient{
				getLatestReleaseErr: fmt.Errorf("API error"),
			},
			wantErr:     true,
			errContains: "failed to get latest release: API error",
		},
		{
			name:     "No suitable asset",
			repoName: "owner/repo",
			ghClient: &mockGitHubClient{
				latestRelease: &github.Release{
					TagName: "v1.0.0",
					Assets: []github.Asset{
						{
							Name:        "test-unsupported-platform",
							DownloadURL: "https://example.com/test",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "failed to select asset",
		},
		{
			name:     "Download error",
			repoName: "owner/repo",
			ghClient: &mockGitHubClient{
				latestRelease: &github.Release{
					TagName: "v1.0.0",
					Assets: []github.Asset{
						{
							Name:        "test-linux-amd64",
							DownloadURL: "https://example.com/test",
						},
					},
				},
				downloadAssetErr: fmt.Errorf("download error"),
			},
			wantErr:     true,
			errContains: "failed to download asset: download error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{RootDir: t.TempDir()}
			store := newMockStorage()
			ghClient := tc.ghClient
			if ghClient == nil {
				ghClient = &mockGitHubClient{}
			}

			if tc.setupFunc != nil {
				if err := tc.setupFunc(cfg); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			err := Install(context.Background(), tc.repoName, cfg, store, ghClient, Options{NonInteractive: true})
			if !tc.wantErr && err != nil {
				t.Errorf("Install() unexpected error: %v", err)
			}
			if tc.wantErr && err == nil {
				t.Error("Install() expected error but got none")
			}
			if tc.wantErr && !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("Install() error = %v, want error containing %v", err, tc.errContains)
			}
		})
	}
}

func TestUpdateErrors(t *testing.T) {
	testCases := []struct {
		name        string
		pkg         *storage.Package
		ghClient    *mockGitHubClient
		setupFunc   func(*config.Config) error
		wantErr     bool
		errContains string
	}{
		{
			name: "GitHub API error",
			pkg: &storage.Package{
				Owner: "owner",
				Repo:  "repo",
			},
			ghClient: &mockGitHubClient{
				getLatestReleaseErr: fmt.Errorf("API error"),
			},
			wantErr:     true,
			errContains: "failed to get latest release: API error",
		},
		{
			name: "No suitable asset",
			pkg: &storage.Package{
				Owner: "owner",
				Repo:  "repo",
			},
			ghClient: &mockGitHubClient{
				latestRelease: &github.Release{
					TagName: "v2.0.0",
					Assets: []github.Asset{
						{
							Name:        "test-unsupported-platform",
							DownloadURL: "https://example.com/test",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "failed to select asset",
		},
		{
			name: "Download error",
			pkg: &storage.Package{
				Owner:   "owner",
				Repo:    "repo",
				Version: "v1.0.0",
			},
			ghClient: &mockGitHubClient{
				latestRelease: &github.Release{
					TagName: "v2.0.0",
					Assets: []github.Asset{
						{
							Name:        "test-linux-amd64",
							DownloadURL: "https://example.com/test",
						},
					},
				},
				downloadAssetErr: fmt.Errorf("download error"),
			},
			wantErr:     true,
			errContains: "failed to download asset: download error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{RootDir: t.TempDir()}
			store := newMockStorage()
			ghClient := tc.ghClient
			if ghClient == nil {
				ghClient = &mockGitHubClient{}
			}

			if tc.setupFunc != nil {
				if err := tc.setupFunc(cfg); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			err := Update(context.Background(), tc.pkg, cfg, store, ghClient, Options{NonInteractive: true})
			if !tc.wantErr && err != nil {
				t.Errorf("Update() unexpected error: %v", err)
			}
			if tc.wantErr && err == nil {
				t.Error("Update() expected error but got none")
			}
			if tc.wantErr && !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("Update() error = %v, want error containing %v", err, tc.errContains)
			}
		})
	}
}

func TestRemoveErrors(t *testing.T) {
	// If running as root (in CI), drop privileges at the start
	if os.Geteuid() == 0 {
		uid := 1000
		if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
			if u, err := user.Lookup(sudoUser); err == nil {
				if parsedUID, err := strconv.Atoi(u.Uid); err == nil {
					uid = parsedUID
				}
			}
		}
		t.Logf("Dropping root privileges to uid %d", uid)
		if err := syscall.Setreuid(uid, uid); err != nil {
			t.Fatalf("Failed to drop privileges: %v", err)
		}
	}

	testCases := []struct {
		name        string
		pkg         *storage.Package
		setupFunc   func(*config.Config) error
		wantErr     bool
		errContains string
	}{
		{
			name: "Missing binary",
			pkg: &storage.Package{
				Owner:   "owner",
				Repo:    "repo",
				Version: "v1.0.0",
				Binary:  "nonexistent-binary",
			},
			wantErr: false, // Should not error if files don't exist
		},
		{
			name: "Missing symlink",
			pkg: &storage.Package{
				Owner:   "owner",
				Repo:    "repo",
				Version: "v1.0.0",
				Binary:  "test-binary",
			},
			wantErr: false, // Should not error if files don't exist
		},
		{
			name: "Permission error",
			pkg: &storage.Package{
				Owner:   "owner",
				Repo:    "repo",
				Version: "v1.0.0",
				Binary:  "test-binary",
			},
			setupFunc: func(cfg *config.Config) error {
				dirs := cfg.GetDirectories()

				// Create directory and binary with write permissions first
				if err := os.MkdirAll(dirs.BinActual, 0755); err != nil {
					return fmt.Errorf("failed to create BinActual dir: %v", err)
				}

				binPath := filepath.Join(dirs.BinActual, "test-binary")
				if err := os.WriteFile(binPath, []byte("test"), 0444); err != nil {
					return fmt.Errorf("failed to create test binary: %v", err)
				}

				// Make BinActual directory read-only
				if err := os.Chmod(dirs.BinActual, 0555); err != nil {
					return fmt.Errorf("failed to chmod BinActual: %v", err)
				}

				return nil
			},
			wantErr:     true,
			errContains: "failed to remove binary",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{RootDir: t.TempDir()}
			store := newMockStorage()

			if tc.setupFunc != nil {
				if err := tc.setupFunc(cfg); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			err := Remove(context.Background(), tc.pkg, cfg, store, Options{})
			t.Logf("Test case %q: err = %v", tc.name, err)
			if !tc.wantErr && err != nil {
				t.Errorf("Remove() unexpected error: %v", err)
			}
			if tc.wantErr && err == nil {
				t.Error("Remove() expected error but got none")
			}
			if tc.wantErr && err != nil && !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("Remove() error = %v, want error containing %v", err, tc.errContains)
			}

			// Cleanup: restore write permissions to allow cleanup
			if tc.setupFunc != nil {
				dirs := cfg.GetDirectories()
				os.Chmod(dirs.BinActual, 0755)
			}
		})
	}
}
