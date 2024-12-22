package storage

import (
	"context"
	"time"
)

// Package represents an installed package
type Package struct {
	ID          int64
	Owner       string
	Repo        string
	Version     string
	InstallPath string
	BinaryName  string
	Frozen      bool
	InstalledAt time.Time
	UpdatedAt   time.Time
}

// Storage defines the interface for database operations
type Storage interface {
	// Initialize creates the database schema
	Initialize(ctx context.Context) error

	// Package operations
	AddPackage(ctx context.Context, pkg *Package) error
	GetPackage(ctx context.Context, owner, repo string) (*Package, error)
	ListPackages(ctx context.Context) ([]*Package, error)
	UpdatePackage(ctx context.Context, pkg *Package) error
	DeletePackage(ctx context.Context, owner, repo string) error

	// Close closes the database connection
	Close() error
}
