package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// client implements the Client interface
type client struct {
	httpClient *http.Client
	token      string
}

// NewClient creates a new GitHub client
func NewClient(token string) Client {
	return &client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token: token,
	}
}

// GetLatestRelease gets the latest release for a repository
func (c *client) GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get latest release: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &release, nil
}

// GetReleases gets all releases for a repository
func (c *client) GetReleases(ctx context.Context, owner, repo string) ([]*Release, error) {
	var allReleases []*Release
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?page=%d", owner, repo, page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		if c.token != "" {
			req.Header.Set("Authorization", "token "+c.token)
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to get releases: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get releases: %s", resp.Status)
		}

		var releases []*Release
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		allReleases = append(allReleases, releases...)

		// Check if there are more pages
		linkHeader := resp.Header.Get("Link")
		if !strings.Contains(linkHeader, `rel="next"`) {
			break
		}
		page++
	}

	return allReleases, nil
}

// DownloadAsset downloads a release asset to a specified path
func (c *client) DownloadAsset(ctx context.Context, asset *Asset, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", asset.DownloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download asset: %s", resp.Status)
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create destination file
	f, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Copy response body to file
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

type SearchResult struct {
	TotalCount int          `json:"total_count"`
	Items      []Repository `json:"items"`
}

type Repository struct {
	ID       int    `json:"id"`
	FullName string `json:"full_name"`
	Name     string `json:"name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	Description string `json:"description"`
	Stars       int    `json:"stargazers_count"`
	UpdatedAt   string `json:"updated_at"`
}

// SearchRepositories searches for repositories using the GitHub search API
func (c *client) SearchRepositories(ctx context.Context, query string) (*SearchResult, error) {
	var allResults SearchResult
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&per_page=100&page=%d", url.QueryEscape(query), page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		if c.token != "" {
			req.Header.Set("Authorization", "token "+c.token)
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to search repositories: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("failed to search repositories: %s - %s", resp.Status, string(body))
		}

		var result SearchResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		allResults.TotalCount = result.TotalCount
		allResults.Items = append(allResults.Items, result.Items...)

		// Check if there are more pages
		linkHeader := resp.Header.Get("Link")
		if !strings.Contains(linkHeader, `rel="next"`) {
			break
		}
		page++
	}

	return &allResults, nil
}

// SearchRepositoriesByName searches for repositories with a specific name
func (c *client) SearchRepositoriesByName(ctx context.Context, name string) (*SearchResult, error) {
	// Search for repositories with this name, sort by stars
	query := fmt.Sprintf("in:name %s sort:stars-desc", name)
	return c.SearchRepositories(ctx, query)
}

// SearchRepositoriesByUser searches for repositories owned by a specific user
func (c *client) SearchRepositoriesByUser(ctx context.Context, user string) (*SearchResult, error) {
	// Search for repositories owned by this user, sort by stars
	query := fmt.Sprintf("user:%s sort:stars-desc", user)
	return c.SearchRepositories(ctx, query)
}
