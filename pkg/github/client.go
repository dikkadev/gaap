package github

import (
	"context"
)

// Release represents a GitHub release
type Release struct {
	TagName     string
	Name        string
	Assets      []Asset
	PublishedAt string
	Body        string
}

// Asset represents a GitHub release asset
type Asset struct {
	Name        string
	Size        int64
	DownloadURL string
}

// Client defines the interface for GitHub operations
type Client interface {
	// GetLatestRelease fetches the latest release for a repository
	GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error)

	// GetReleases fetches all releases for a repository
	GetReleases(ctx context.Context, owner, repo string) ([]*Release, error)

	// DownloadAsset downloads a release asset to the specified path
	DownloadAsset(ctx context.Context, assetURL, destPath string) error
}
