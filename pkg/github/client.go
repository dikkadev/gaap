package github

import (
	"context"
	"time"
)

// Release represents a GitHub release
type Release struct {
	TagName     string    `json:"tag_name"`     // Tag name (e.g., "v1.0.0")
	Name        string    `json:"name"`         // Release name
	Assets      []Asset   `json:"assets"`       // Release assets
	PublishedAt time.Time `json:"published_at"` // When the release was published
	Body        string    `json:"body"`         // Release description
}

// Asset represents a GitHub release asset
type Asset struct {
	Name        string `json:"name"`                 // Asset name
	Size        int64  `json:"size"`                 // Asset size in bytes
	DownloadURL string `json:"browser_download_url"` // URL to download the asset
}

// Client defines the interface for GitHub API operations
type Client interface {
	// GetLatestRelease gets the latest release for a repository
	GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error)

	// GetReleases gets all releases for a repository
	GetReleases(ctx context.Context, owner, repo string) ([]*Release, error)

	// DownloadAsset downloads a release asset to a specified path
	DownloadAsset(ctx context.Context, asset *Asset, destPath string) error

	// SearchRepositories searches for repositories using the GitHub search API
	SearchRepositories(ctx context.Context, query string) (*SearchResult, error)

	// SearchRepositoriesByName searches for repositories with a specific name
	SearchRepositoriesByName(ctx context.Context, name string) (*SearchResult, error)

	// SearchRepositoriesByUser searches for repositories owned by a specific user
	SearchRepositoriesByUser(ctx context.Context, user string) (*SearchResult, error)
}
