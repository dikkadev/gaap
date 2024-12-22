package storage

import (
	"context"
	"time"
)

// Package represents an installed package
type Package struct {
	Owner       string    // GitHub repository owner
	Repo        string    // GitHub repository name
	Version     string    // Installed version (tag name)
	Binary      string    // Binary name in bin/actual
	Frozen      bool      // Whether the package is frozen at its current version
	Platform    string    // Platform the package was installed for (e.g., linux-amd64)
	InstalledAt time.Time // When the package was installed
	UpdatedAt   time.Time // When the package was last updated
}

// Storage defines the interface for package storage
type Storage interface {
	// Initialize initializes the storage (e.g., creates tables)
	Initialize(ctx context.Context) error

	// AddPackage adds a new package
	AddPackage(ctx context.Context, pkg *Package) error

	// GetPackage gets a package by owner and repo
	GetPackage(ctx context.Context, owner, repo string) (*Package, error)

	// ListPackages lists all installed packages
	ListPackages(ctx context.Context) ([]*Package, error)

	// UpdatePackage updates an existing package
	UpdatePackage(ctx context.Context, pkg *Package) error

	// DeletePackage deletes a package
	DeletePackage(ctx context.Context, owner, repo string) error

	// Close closes the storage
	Close() error
}
