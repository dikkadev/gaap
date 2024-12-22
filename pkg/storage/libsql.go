package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

// LibSQL implements the Storage interface using libsql
type LibSQL struct {
	db *sql.DB
}

// NewLibSQL creates a new LibSQL storage
func NewLibSQL(url string) (*LibSQL, error) {
	db, err := sql.Open("libsql", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &LibSQL{db: db}, nil
}

// Initialize creates the database schema
func (s *LibSQL) Initialize(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS packages (
			owner TEXT NOT NULL,
			repo TEXT NOT NULL,
			version TEXT NOT NULL,
			binary TEXT NOT NULL,
			frozen BOOLEAN NOT NULL DEFAULT 0,
			platform TEXT NOT NULL,
			installed_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (owner, repo)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create packages table: %w", err)
	}

	return nil
}

// AddPackage adds a new package
func (s *LibSQL) AddPackage(ctx context.Context, pkg *Package) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO packages (
			owner, repo, version, binary, frozen, platform,
			installed_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		pkg.Owner, pkg.Repo, pkg.Version, pkg.Binary, pkg.Frozen, pkg.Platform,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert package: %w", err)
	}

	pkg.InstalledAt = now
	pkg.UpdatedAt = now
	return nil
}

// GetPackage gets a package by owner and repo
func (s *LibSQL) GetPackage(ctx context.Context, owner, repo string) (*Package, error) {
	pkg := &Package{}
	err := s.db.QueryRowContext(ctx, `
		SELECT owner, repo, version, binary, frozen, platform,
			   installed_at, updated_at
		FROM packages
		WHERE owner = ? AND repo = ?
	`, owner, repo).Scan(
		&pkg.Owner, &pkg.Repo, &pkg.Version, &pkg.Binary, &pkg.Frozen, &pkg.Platform,
		&pkg.InstalledAt, &pkg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get package: %w", err)
	}

	return pkg, nil
}

// ListPackages lists all installed packages
func (s *LibSQL) ListPackages(ctx context.Context) ([]*Package, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT owner, repo, version, binary, frozen, platform,
			   installed_at, updated_at
		FROM packages
		ORDER BY owner, repo
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}
	defer rows.Close()

	var packages []*Package
	for rows.Next() {
		pkg := &Package{}
		err := rows.Scan(
			&pkg.Owner, &pkg.Repo, &pkg.Version, &pkg.Binary, &pkg.Frozen, &pkg.Platform,
			&pkg.InstalledAt, &pkg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan package: %w", err)
		}
		packages = append(packages, pkg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate packages: %w", err)
	}

	return packages, nil
}

// UpdatePackage updates an existing package
func (s *LibSQL) UpdatePackage(ctx context.Context, pkg *Package) error {
	now := time.Now()
	result, err := s.db.ExecContext(ctx, `
		UPDATE packages
		SET version = ?, binary = ?, frozen = ?, platform = ?,
			updated_at = ?
		WHERE owner = ? AND repo = ?
	`,
		pkg.Version, pkg.Binary, pkg.Frozen, pkg.Platform,
		now,
		pkg.Owner, pkg.Repo,
	)
	if err != nil {
		return fmt.Errorf("failed to update package: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("package not found: %s/%s", pkg.Owner, pkg.Repo)
	}

	pkg.UpdatedAt = now
	return nil
}

// DeletePackage deletes a package
func (s *LibSQL) DeletePackage(ctx context.Context, owner, repo string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM packages
		WHERE owner = ? AND repo = ?
	`, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to delete package: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("package not found: %s/%s", owner, repo)
	}

	return nil
}

// Close closes the database connection
func (s *LibSQL) Close() error {
	return s.db.Close()
}
