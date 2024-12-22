package selector

import (
	"context"
	"errors"
	"testing"

	"github.com/dikkadev/gaap/pkg/github"
)

var errNotFound = errors.New("repository not found")

// mockGitHubClient implements github.Client for testing
type mockGitHubClient struct {
	searchResults map[string]*github.SearchResult
	err           error
}

func (m *mockGitHubClient) GetLatestRelease(ctx context.Context, owner, repo string) (*github.Release, error) {
	return nil, nil
}

func (m *mockGitHubClient) GetReleases(ctx context.Context, owner, repo string) ([]*github.Release, error) {
	return nil, nil
}

func (m *mockGitHubClient) DownloadAsset(ctx context.Context, asset *github.Asset, destPath string) error {
	return nil
}

func (m *mockGitHubClient) SearchRepositories(ctx context.Context, query string) (*github.SearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if result, ok := m.searchResults[query]; ok {
		return result, nil
	}
	return &github.SearchResult{}, nil
}

func (m *mockGitHubClient) SearchRepositoriesByName(ctx context.Context, name string) (*github.SearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	query := "in:name " + name + " sort:stars-desc"
	return m.SearchRepositories(ctx, query)
}

func (m *mockGitHubClient) SearchRepositoriesByUser(ctx context.Context, user string) (*github.SearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	query := "user:" + user + " sort:stars-desc"
	return m.SearchRepositories(ctx, query)
}

func TestRepoItemMethods(t *testing.T) {
	repo := github.Repository{
		FullName:    "test/repo",
		Name:        "repo",
		Description: "Test repository",
		Stars:       100,
		Owner: struct {
			Login string "json:\"login\""
		}{
			Login: "test",
		},
	}

	item := RepoItem{repo: repo}

	t.Run("Title", func(t *testing.T) {
		if got := item.Title(); got != "test/repo" {
			t.Errorf("Title() = %v, want %v", got, "test/repo")
		}
	})

	t.Run("Description", func(t *testing.T) {
		expected := "⭐ 100 | Test repository"
		if got := item.Description(); got != expected {
			t.Errorf("Description() = %v, want %v", got, expected)
		}
	})

	t.Run("Description truncation", func(t *testing.T) {
		longDesc := github.Repository{
			Description: "This is a very long description that should be truncated because it exceeds the maximum length allowed for display in the repository selector interface",
			Stars:       100,
		}
		item := RepoItem{repo: longDesc}
		desc := item.Description()
		if len(desc) > 107 { // 100 chars + "⭐ 100 | " + "..."
			t.Errorf("Description() length = %v, want <= 107", len(desc))
		}
		if desc[len(desc)-3:] != "..." {
			t.Error("Long description should end with '...'")
		}
	})

	t.Run("FilterValue", func(t *testing.T) {
		if got := item.FilterValue(); got != "test/repo" {
			t.Errorf("FilterValue() = %v, want %v", got, "test/repo")
		}
	})
}

func TestSearchRepositories(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name      string
		input     string
		mockData  map[string]*github.SearchResult
		mockError error
		want      *github.Repository
		wantError bool
	}{
		{
			name:  "Exact match",
			input: "owner/repo",
			mockData: map[string]*github.SearchResult{
				"repo:owner/repo": {
					TotalCount: 1,
					Items: []github.Repository{
						{
							FullName: "owner/repo",
							Name:     "repo",
							Owner: struct {
								Login string "json:\"login\""
							}{
								Login: "owner",
							},
						},
					},
				},
			},
			want: &github.Repository{
				FullName: "owner/repo",
				Name:     "repo",
				Owner: struct {
					Login string "json:\"login\""
				}{
					Login: "owner",
				},
			},
		},
		{
			name:  "No results",
			input: "nonexistent/repo",
			mockData: map[string]*github.SearchResult{
				"repo:nonexistent/repo": {
					TotalCount: 0,
					Items:      []github.Repository{},
				},
			},
			wantError: true,
		},
		{
			name:      "GitHub API error",
			input:     "owner/repo",
			mockError: errNotFound,
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockGitHubClient{
				searchResults: tc.mockData,
				err:           tc.mockError,
			}

			result, err := searchRepositories(ctx, client, tc.input)
			if (err != nil) != tc.wantError {
				t.Errorf("searchRepositories() error = %v, wantError %v", err, tc.wantError)
				return
			}

			if !tc.wantError {
				if result == nil {
					t.Fatal("searchRepositories() returned nil, want result")
				}
				if result.TotalCount != 1 {
					t.Errorf("searchRepositories() returned %d results, want 1", result.TotalCount)
				}
				if result.Items[0].FullName != tc.want.FullName {
					t.Errorf("searchRepositories() = %v, want %v", result.Items[0].FullName, tc.want.FullName)
				}
			}
		})
	}
}

func TestSearchRepositoriesFuzzy(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name      string
		input     string
		mockData  map[string]*github.SearchResult
		want      *github.Repository
		wantError bool
	}{
		{
			name:  "Search by name",
			input: "cli",
			mockData: map[string]*github.SearchResult{
				"in:name cli sort:stars-desc": {
					TotalCount: 2,
					Items: []github.Repository{
						{
							FullName: "cli/cli",
							Name:     "cli",
							Stars:    1000,
						},
						{
							FullName: "other/cli",
							Name:     "cli",
							Stars:    500,
						},
					},
				},
			},
			want: &github.Repository{
				FullName: "cli/cli",
				Name:     "cli",
				Stars:    1000,
			},
		},
		{
			name:  "Search by user",
			input: "user",
			mockData: map[string]*github.SearchResult{
				"user:user sort:stars-desc": {
					TotalCount: 1,
					Items: []github.Repository{
						{
							FullName: "user/repo",
							Name:     "repo",
							Owner: struct {
								Login string "json:\"login\""
							}{
								Login: "user",
							},
						},
					},
				},
			},
			want: &github.Repository{
				FullName: "user/repo",
				Name:     "repo",
				Owner: struct {
					Login string "json:\"login\""
				}{
					Login: "user",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockGitHubClient{
				searchResults: tc.mockData,
			}

			result, err := searchRepositories(ctx, client, tc.input)
			if (err != nil) != tc.wantError {
				t.Errorf("searchRepositories() error = %v, wantError %v", err, tc.wantError)
				return
			}

			if !tc.wantError {
				if result == nil {
					t.Fatal("searchRepositories() returned nil, want result")
				}
				if len(result.Items) == 0 {
					t.Fatal("searchRepositories() returned no items")
				}
				if result.Items[0].FullName != tc.want.FullName {
					t.Errorf("searchRepositories() = %v, want %v", result.Items[0].FullName, tc.want.FullName)
				}
			}
		})
	}
}
