package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/tursodatabase/go-libsql"
)

const schema = `
CREATE TABLE IF NOT EXISTS packages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner TEXT NOT NULL,
    repo TEXT NOT NULL,
    version TEXT NOT NULL,
    install_path TEXT NOT NULL,
    binary_name TEXT NOT NULL,
    frozen BOOLEAN NOT NULL DEFAULT 0,
    installed_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE(owner, repo)
);
`

type libsqlStorage struct {
	db *sql.DB
}

// NewLibSQL creates a new libsql storage instance
func NewLibSQL(dbPath string) (Storage, error) {
	db, err := sql.Open("libsql", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &libsqlStorage{db: db}, nil
}

func (s *libsqlStorage) Initialize(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

func (s *libsqlStorage) AddPackage(ctx context.Context, pkg *Package) error {
	query := `
		INSERT INTO packages (
			owner, repo, version, install_path, binary_name, frozen, installed_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, query,
		pkg.Owner,
		pkg.Repo,
		pkg.Version,
		pkg.InstallPath,
		pkg.BinaryName,
		pkg.Frozen,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert package: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	pkg.ID = id
	pkg.InstalledAt = now
	pkg.UpdatedAt = now
	return nil
}

func (s *libsqlStorage) GetPackage(ctx context.Context, owner, repo string) (*Package, error) {
	query := `
		SELECT id, owner, repo, version, install_path, binary_name, frozen, installed_at, updated_at
		FROM packages
		WHERE owner = ? AND repo = ?
	`
	pkg := &Package{}
	err := s.db.QueryRowContext(ctx, query, owner, repo).Scan(
		&pkg.ID,
		&pkg.Owner,
		&pkg.Repo,
		&pkg.Version,
		&pkg.InstallPath,
		&pkg.BinaryName,
		&pkg.Frozen,
		&pkg.InstalledAt,
		&pkg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get package: %w", err)
	}
	return pkg, nil
}

func (s *libsqlStorage) ListPackages(ctx context.Context) ([]*Package, error) {
	query := `
		SELECT id, owner, repo, version, install_path, binary_name, frozen, installed_at, updated_at
		FROM packages
		ORDER BY owner, repo
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}
	defer rows.Close()

	var packages []*Package
	for rows.Next() {
		pkg := &Package{}
		err := rows.Scan(
			&pkg.ID,
			&pkg.Owner,
			&pkg.Repo,
			&pkg.Version,
			&pkg.InstallPath,
			&pkg.BinaryName,
			&pkg.Frozen,
			&pkg.InstalledAt,
			&pkg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan package: %w", err)
		}
		packages = append(packages, pkg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating packages: %w", err)
	}
	return packages, nil
}

func (s *libsqlStorage) UpdatePackage(ctx context.Context, pkg *Package) error {
	query := `
		UPDATE packages
		SET version = ?, install_path = ?, binary_name = ?, frozen = ?, updated_at = ?
		WHERE owner = ? AND repo = ?
	`
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, query,
		pkg.Version,
		pkg.InstallPath,
		pkg.BinaryName,
		pkg.Frozen,
		now,
		pkg.Owner,
		pkg.Repo,
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

func (s *libsqlStorage) DeletePackage(ctx context.Context, owner, repo string) error {
	query := `DELETE FROM packages WHERE owner = ? AND repo = ?`
	result, err := s.db.ExecContext(ctx, query, owner, repo)
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

func (s *libsqlStorage) Close() error {
	return s.db.Close()
}
