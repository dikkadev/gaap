package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

type client struct {
	ghClient *github.Client
}

// NewClient creates a new GitHub client
func NewClient(token string) Client {
	var httpClient *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(context.Background(), ts)
	}

	return &client{
		ghClient: github.NewClient(httpClient),
	}
}

func (c *client) GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	release, _, err := c.ghClient.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	return convertGithubRelease(release), nil
}

func (c *client) GetReleases(ctx context.Context, owner, repo string) ([]*Release, error) {
	opt := &github.ListOptions{
		PerPage: 100,
	}
	var allReleases []*Release

	for {
		releases, resp, err := c.ghClient.Repositories.ListReleases(ctx, owner, repo, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list releases: %w", err)
		}

		for _, release := range releases {
			allReleases = append(allReleases, convertGithubRelease(release))
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allReleases, nil
}

func (c *client) DownloadAsset(ctx context.Context, assetURL, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", assetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download asset: status code %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func convertGithubRelease(release *github.RepositoryRelease) *Release {
	r := &Release{
		TagName: release.GetTagName(),
		Name:    release.GetName(),
		Body:    release.GetBody(),
	}

	if release.PublishedAt != nil {
		r.PublishedAt = release.PublishedAt.Format(time.RFC3339)
	}

	for _, asset := range release.Assets {
		r.Assets = append(r.Assets, Asset{
			Name:        asset.GetName(),
			Size:        int64(asset.GetSize()),
			DownloadURL: asset.GetBrowserDownloadURL(),
		})
	}

	return r
}
