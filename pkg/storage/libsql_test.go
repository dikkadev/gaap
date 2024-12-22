package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLibSQL(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "gaap-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	storage, err := NewLibSQL(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	if err := storage.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Test adding a package
	pkg := &Package{
		Owner:       "owner",
		Repo:        "repo",
		Version:     "v1.0.0",
		InstallPath: "/path/to/binary",
		BinaryName:  "binary",
		Frozen:      false,
	}

	if err := storage.AddPackage(ctx, pkg); err != nil {
		t.Fatalf("Failed to add package: %v", err)
	}

	if pkg.ID == 0 {
		t.Error("Package ID was not set after adding")
	}

	// Test getting a package
	got, err := storage.GetPackage(ctx, pkg.Owner, pkg.Repo)
	if err != nil {
		t.Fatalf("Failed to get package: %v", err)
	}

	if got == nil {
		t.Fatal("GetPackage returned nil for existing package")
	}

	if got.Owner != pkg.Owner || got.Repo != pkg.Repo {
		t.Errorf("Got package %s/%s, want %s/%s", got.Owner, got.Repo, pkg.Owner, pkg.Repo)
	}

	// Test listing packages
	packages, err := storage.ListPackages(ctx)
	if err != nil {
		t.Fatalf("Failed to list packages: %v", err)
	}

	if len(packages) != 1 {
		t.Errorf("Got %d packages, want 1", len(packages))
	}

	// Test updating a package
	pkg.Version = "v2.0.0"
	pkg.Frozen = true

	if err := storage.UpdatePackage(ctx, pkg); err != nil {
		t.Fatalf("Failed to update package: %v", err)
	}

	got, err = storage.GetPackage(ctx, pkg.Owner, pkg.Repo)
	if err != nil {
		t.Fatalf("Failed to get updated package: %v", err)
	}

	if got.Version != pkg.Version || got.Frozen != pkg.Frozen {
		t.Errorf("Got version=%s frozen=%v, want version=%s frozen=%v",
			got.Version, got.Frozen, pkg.Version, pkg.Frozen)
	}

	// Test deleting a package
	if err := storage.DeletePackage(ctx, pkg.Owner, pkg.Repo); err != nil {
		t.Fatalf("Failed to delete package: %v", err)
	}

	got, err = storage.GetPackage(ctx, pkg.Owner, pkg.Repo)
	if err != nil {
		t.Fatalf("Failed to check deleted package: %v", err)
	}

	if got != nil {
		t.Error("Package still exists after deletion")
	}
}

func TestLibSQLErrors(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "gaap-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	storage, err := NewLibSQL(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	if err := storage.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Test duplicate package
	pkg := &Package{
		Owner:       "owner",
		Repo:        "repo",
		Version:     "v1.0.0",
		InstallPath: "/path/to/binary",
		BinaryName:  "binary",
	}

	if err := storage.AddPackage(ctx, pkg); err != nil {
		t.Fatalf("Failed to add first package: %v", err)
	}

	if err := storage.AddPackage(ctx, pkg); err == nil {
		t.Error("Expected error when adding duplicate package")
	}

	// Test updating non-existent package
	nonExistentPkg := &Package{
		Owner:       "nonexistent",
		Repo:        "nonexistent",
		Version:     "v1.0.0",
		InstallPath: "/path/to/binary",
		BinaryName:  "binary",
	}

	if err := storage.UpdatePackage(ctx, nonExistentPkg); err == nil {
		t.Error("Expected error when updating non-existent package")
	}

	// Test deleting non-existent package
	if err := storage.DeletePackage(ctx, "nonexistent", "nonexistent"); err == nil {
		t.Error("Expected error when deleting non-existent package")
	}
}
