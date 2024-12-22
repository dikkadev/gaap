package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestClient(server *httptest.Server) *client {
	serverURL, _ := url.Parse(server.URL + "/")
	httpClient := server.Client()
	httpClient.Transport = &testTransport{
		baseURL: serverURL,
		inner:   httpClient.Transport,
	}
	return &client{
		httpClient: httpClient,
		token:      "test-token",
	}
}

type testTransport struct {
	baseURL *url.URL
	inner   http.RoundTripper
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite request URL to use test server
	req.URL.Scheme = t.baseURL.Scheme
	req.URL.Host = t.baseURL.Host
	return t.inner.RoundTrip(req)
}

func TestGetLatestRelease(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			t.Errorf("Expected path /repos/owner/repo/releases/latest, got %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
			t.Errorf("Expected Accept header application/vnd.github.v3+json, got %s", r.Header.Get("Accept"))
		}

		// Return test response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tag_name":     "v1.0.0",
			"name":         "Release 1.0.0",
			"published_at": time.Now().Format(time.RFC3339),
			"body":         "Test release",
			"assets": []map[string]interface{}{
				{
					"name":                 "test-linux-amd64",
					"size":                 1024,
					"browser_download_url": "https://example.com/test",
				},
			},
		})
	}))
	defer server.Close()

	// Create client with test server URL
	c := newTestClient(server)

	// Test GetLatestRelease
	release, err := c.GetLatestRelease(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("GetLatestRelease failed: %v", err)
	}

	// Check response
	if release.TagName != "v1.0.0" {
		t.Errorf("Expected tag v1.0.0, got %s", release.TagName)
	}
	if len(release.Assets) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(release.Assets))
	}
}

func TestGetReleases(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/repos/owner/repo/releases" {
			t.Errorf("Expected path /repos/owner/repo/releases, got %s", r.URL.Path)
		}

		// Return test response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"tag_name":     "v1.0.0",
				"name":         "Release 1.0.0",
				"published_at": time.Now().Format(time.RFC3339),
				"body":         "Test release 1",
				"assets":       []map[string]interface{}{},
			},
			{
				"tag_name":     "v0.9.0",
				"name":         "Release 0.9.0",
				"published_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				"body":         "Test release 2",
				"assets":       []map[string]interface{}{},
			},
		})
	}))
	defer server.Close()

	// Create client with test server URL
	c := newTestClient(server)

	// Test GetReleases
	releases, err := c.GetReleases(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("GetReleases failed: %v", err)
	}

	// Check response
	if len(releases) != 2 {
		t.Errorf("Expected 2 releases, got %d", len(releases))
	}
	if releases[0].TagName != "v1.0.0" {
		t.Errorf("Expected first release v1.0.0, got %s", releases[0].TagName)
	}
}

func TestDownloadAsset(t *testing.T) {
	// Create temp dir for test file
	tmpDir, err := os.MkdirTemp("", "github-client-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.Header.Get("Accept") != "application/octet-stream" {
			t.Errorf("Expected Accept header application/octet-stream, got %s", r.Header.Get("Accept"))
		}

		// Return test file content
		fmt.Fprint(w, "test file content")
	}))
	defer server.Close()

	// Create client with test server URL
	c := newTestClient(server)

	// Test DownloadAsset
	asset := &Asset{
		Name:        "test-file",
		Size:        16,
		DownloadURL: server.URL + "/test-file",
	}
	destPath := filepath.Join(tmpDir, "test-file")

	err = c.DownloadAsset(context.Background(), asset, destPath)
	if err != nil {
		t.Fatalf("DownloadAsset failed: %v", err)
	}

	// Check downloaded file
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(content) != "test file content" {
		t.Errorf("Expected content 'test file content', got '%s'", string(content))
	}
}

func TestClientErrors(t *testing.T) {
	// Create test server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Not found")
	}))
	defer server.Close()

	// Create client with test server URL
	c := newTestClient(server)

	// Test GetLatestRelease error
	_, err := c.GetLatestRelease(context.Background(), "owner", "repo")
	if err == nil {
		t.Error("Expected GetLatestRelease to fail")
	}

	// Test GetReleases error
	_, err = c.GetReleases(context.Background(), "owner", "repo")
	if err == nil {
		t.Error("Expected GetReleases to fail")
	}

	// Test DownloadAsset error
	tmpDir, err := os.MkdirTemp("", "github-client-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	asset := &Asset{
		Name:        "test-file",
		Size:        16,
		DownloadURL: server.URL + "/test-file",
	}
	err = c.DownloadAsset(context.Background(), asset, filepath.Join(tmpDir, "test-file"))
	if err == nil {
		t.Error("Expected DownloadAsset to fail")
	}
}

func TestSearchRepositories(t *testing.T) {
	testCases := []struct {
		name           string
		query          string
		expectedPath   string
		mockResponse   *SearchResult
		mockStatusCode int
		wantError      bool
	}{
		{
			name:         "Basic search",
			query:        "test-repo",
			expectedPath: "/search/repositories",
			mockResponse: &SearchResult{
				TotalCount: 1,
				Items: []Repository{
					{
						FullName:    "owner/test-repo",
						Name:        "test-repo",
						Description: "A test repository",
						Stars:       100,
						Owner: struct {
							Login string "json:\"login\""
						}{
							Login: "owner",
						},
					},
				},
			},
			mockStatusCode: http.StatusOK,
		},
		{
			name:         "Exact repository match",
			query:        "repo:owner/repo",
			expectedPath: "/search/repositories",
			mockResponse: &SearchResult{
				TotalCount: 1,
				Items: []Repository{
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
			mockStatusCode: http.StatusOK,
		},
		{
			name:           "API error",
			query:          "test-repo",
			expectedPath:   "/search/repositories",
			mockStatusCode: http.StatusUnauthorized,
			wantError:      true,
		},
		{
			name:         "No results",
			query:        "nonexistent-repo",
			expectedPath: "/search/repositories",
			mockResponse: &SearchResult{
				TotalCount: 0,
				Items:      []Repository{},
			},
			mockStatusCode: http.StatusOK,
		},
		{
			name:         "Multiple results",
			query:        "popular-name",
			expectedPath: "/search/repositories",
			mockResponse: &SearchResult{
				TotalCount: 2,
				Items: []Repository{
					{
						FullName:    "popular/name",
						Name:        "name",
						Description: "Most popular",
						Stars:       1000,
					},
					{
						FullName:    "less/popular-name",
						Name:        "popular-name",
						Description: "Less popular",
						Stars:       100,
					},
				},
			},
			mockStatusCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check request
				if r.URL.Path != tc.expectedPath {
					t.Errorf("Expected path %s, got %s", tc.expectedPath, r.URL.Path)
				}
				if r.URL.Query().Get("q") != tc.query {
					t.Errorf("Expected query %s, got %s", tc.query, r.URL.Query().Get("q"))
				}
				if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
					t.Errorf("Expected Accept header application/vnd.github.v3+json, got %s", r.Header.Get("Accept"))
				}

				w.WriteHeader(tc.mockStatusCode)
				if tc.mockStatusCode == http.StatusOK {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(tc.mockResponse)
				} else {
					fmt.Fprintf(w, "Error: %d", tc.mockStatusCode)
				}
			}))
			defer server.Close()

			c := newTestClient(server)
			result, err := c.SearchRepositories(context.Background(), tc.query)

			if (err != nil) != tc.wantError {
				t.Errorf("SearchRepositories() error = %v, wantError %v", err, tc.wantError)
				return
			}

			if !tc.wantError {
				if result == nil {
					t.Fatal("SearchRepositories() returned nil result")
				}
				if result.TotalCount != tc.mockResponse.TotalCount {
					t.Errorf("SearchRepositories() returned %d results, want %d", result.TotalCount, tc.mockResponse.TotalCount)
				}
				if len(result.Items) != len(tc.mockResponse.Items) {
					t.Errorf("SearchRepositories() returned %d items, want %d", len(result.Items), len(tc.mockResponse.Items))
				}
				if len(result.Items) > 0 && result.Items[0].FullName != tc.mockResponse.Items[0].FullName {
					t.Errorf("SearchRepositories() first result = %v, want %v", result.Items[0].FullName, tc.mockResponse.Items[0].FullName)
				}
			}
		})
	}
}

func TestSearchRepositoriesByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/repositories" {
			t.Errorf("Expected path /search/repositories, got %s", r.URL.Path)
		}
		expectedQuery := "in:name test-repo sort:stars-desc"
		if r.URL.Query().Get("q") != expectedQuery {
			t.Errorf("Expected query %s, got %s", expectedQuery, r.URL.Query().Get("q"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&SearchResult{
			TotalCount: 1,
			Items: []Repository{
				{
					FullName: "owner/test-repo",
					Name:     "test-repo",
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	result, err := c.SearchRepositoriesByName(context.Background(), "test-repo")
	if err != nil {
		t.Fatalf("SearchRepositoriesByName failed: %v", err)
	}

	if result.TotalCount != 1 {
		t.Errorf("Expected 1 result, got %d", result.TotalCount)
	}
	if result.Items[0].Name != "test-repo" {
		t.Errorf("Expected repository name test-repo, got %s", result.Items[0].Name)
	}
}

func TestSearchRepositoriesByUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/repositories" {
			t.Errorf("Expected path /search/repositories, got %s", r.URL.Path)
		}
		expectedQuery := "user:testuser sort:stars-desc"
		if r.URL.Query().Get("q") != expectedQuery {
			t.Errorf("Expected query %s, got %s", expectedQuery, r.URL.Query().Get("q"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&SearchResult{
			TotalCount: 2,
			Items: []Repository{
				{
					FullName: "testuser/repo1",
					Name:     "repo1",
					Owner: struct {
						Login string "json:\"login\""
					}{
						Login: "testuser",
					},
				},
				{
					FullName: "testuser/repo2",
					Name:     "repo2",
					Owner: struct {
						Login string "json:\"login\""
					}{
						Login: "testuser",
					},
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server)
	result, err := c.SearchRepositoriesByUser(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("SearchRepositoriesByUser failed: %v", err)
	}

	if result.TotalCount != 2 {
		t.Errorf("Expected 2 results, got %d", result.TotalCount)
	}
	for _, repo := range result.Items {
		if repo.Owner.Login != "testuser" {
			t.Errorf("Expected owner testuser, got %s", repo.Owner.Login)
		}
	}
}

func TestSearchRepositoriesEdgeCases(t *testing.T) {
	testCases := []struct {
		name           string
		query          string
		mockResponse   interface{}
		mockHeaders    map[string]string
		mockStatusCode int
		wantError      bool
		errorContains  string
	}{
		{
			name:           "Rate limit exceeded",
			query:          "test-repo",
			mockStatusCode: http.StatusForbidden,
			mockHeaders: map[string]string{
				"X-RateLimit-Remaining": "0",
				"X-RateLimit-Reset":     fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()),
			},
			wantError:     true,
			errorContains: "failed to search repositories: 403",
		},
		{
			name:  "Malformed response",
			query: "test-repo",
			mockResponse: map[string]interface{}{
				"total_count": "not a number",
				"items":       "not an array",
			},
			mockStatusCode: http.StatusOK,
			wantError:      true,
			errorContains:  "failed to decode response",
		},
		{
			name:           "Empty query",
			query:          "",
			mockStatusCode: http.StatusUnprocessableEntity,
			wantError:      true,
			errorContains:  "failed to search repositories: 422",
		},
		{
			name:           "Very long query",
			query:          string(make([]byte, 1000, 1000)),
			mockStatusCode: http.StatusUnprocessableEntity,
			wantError:      true,
			errorContains:  "failed to search repositories: 422",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Set mock headers
				for k, v := range tc.mockHeaders {
					w.Header().Set(k, v)
				}

				w.WriteHeader(tc.mockStatusCode)
				if tc.mockStatusCode == http.StatusOK {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(tc.mockResponse)
				} else {
					fmt.Fprintf(w, "Error: %d", tc.mockStatusCode)
				}
			}))
			defer server.Close()

			c := newTestClient(server)
			_, err := c.SearchRepositories(context.Background(), tc.query)

			if !tc.wantError && err != nil {
				t.Errorf("SearchRepositories() unexpected error: %v", err)
				return
			}
			if tc.wantError && err == nil {
				t.Error("SearchRepositories() expected error but got none")
				return
			}
			if tc.wantError && !strings.Contains(err.Error(), tc.errorContains) {
				t.Errorf("SearchRepositories() error = %v, want error containing %v", err, tc.errorContains)
			}
		})
	}
}

func TestSearchRepositoriesContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow response
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&SearchResult{})
	}))
	defer server.Close()

	c := newTestClient(server)

	// Create a context that times out immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := c.SearchRepositories(ctx, "test-repo")
	if err == nil {
		t.Error("SearchRepositories() expected context timeout error but got none")
	}
	if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("SearchRepositories() error = %v, want context deadline exceeded", err)
	}
}

func TestPaginatedReleases(t *testing.T) {
	pageCount := 0
	totalReleases := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases" {
			t.Errorf("Expected path /repos/owner/repo/releases, got %s", r.URL.Path)
		}

		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}
		pageCount++

		w.Header().Set("Content-Type", "application/json")

		// Set pagination headers only for page 1
		if page == "1" {
			w.Header().Set("Link", `<https://api.github.com/repos/owner/repo/releases?page=2>; rel="next"`)
		}

		// Return different releases based on page
		var releases []map[string]interface{}
		switch page {
		case "1":
			releases = []map[string]interface{}{
				{
					"tag_name":     "v1.0.0",
					"name":         "Release 1.0.0",
					"published_at": time.Now().Format(time.RFC3339),
					"body":         "First release",
				},
				{
					"tag_name":     "v1.1.0",
					"name":         "Release 1.1.0",
					"published_at": time.Now().Format(time.RFC3339),
					"body":         "Second release",
				},
			}
		case "2":
			releases = []map[string]interface{}{
				{
					"tag_name":     "v1.2.0",
					"name":         "Release 1.2.0",
					"published_at": time.Now().Format(time.RFC3339),
					"body":         "Third release",
				},
			}
		default:
			releases = []map[string]interface{}{}
		}
		totalReleases += len(releases)
		json.NewEncoder(w).Encode(releases)
	}))
	defer server.Close()

	c := newTestClient(server)
	releases, err := c.GetReleases(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("GetReleases failed: %v", err)
	}

	// Verify we got all releases
	if len(releases) != totalReleases {
		t.Errorf("Expected %d releases, got %d", totalReleases, len(releases))
	}

	// Verify we made the correct number of requests
	if pageCount != 2 {
		t.Errorf("Expected 2 page requests, got %d", pageCount)
	}

	// Verify releases are in correct order
	expectedTags := []string{"v1.0.0", "v1.1.0", "v1.2.0"}
	for i, tag := range expectedTags {
		if releases[i].TagName != tag {
			t.Errorf("Expected release %d to have tag %s, got %s", i, tag, releases[i].TagName)
		}
	}
}

func TestPaginatedSearch(t *testing.T) {
	pageCount := 0
	totalResults := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/repositories" {
			t.Errorf("Expected path /search/repositories, got %s", r.URL.Path)
		}

		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}
		pageCount++

		w.Header().Set("Content-Type", "application/json")

		// Set pagination headers only for page 1
		if page == "1" {
			w.Header().Set("Link", `<https://api.github.com/search/repositories?q=test&page=2>; rel="next"`)
		}

		// Return different results based on page
		var result SearchResult
		switch page {
		case "1":
			result = SearchResult{
				TotalCount: 3,
				Items: []Repository{
					{
						FullName: "owner/repo1",
						Name:     "repo1",
						Stars:    100,
					},
					{
						FullName: "owner/repo2",
						Name:     "repo2",
						Stars:    90,
					},
				},
			}
		case "2":
			result = SearchResult{
				TotalCount: 3,
				Items: []Repository{
					{
						FullName: "owner/repo3",
						Name:     "repo3",
						Stars:    80,
					},
				},
			}
		default:
			result = SearchResult{
				TotalCount: 3,
				Items:      []Repository{},
			}
		}
		totalResults += len(result.Items)
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	c := newTestClient(server)
	result, err := c.SearchRepositories(context.Background(), "test")
	if err != nil {
		t.Fatalf("SearchRepositories failed: %v", err)
	}

	// Verify we got all results
	if len(result.Items) != totalResults {
		t.Errorf("Expected %d results, got %d", totalResults, len(result.Items))
	}

	// Verify we made the correct number of requests
	if pageCount != 2 {
		t.Errorf("Expected 2 page requests, got %d", pageCount)
	}

	// Verify results are in correct order (by stars)
	expectedRepos := []string{"repo1", "repo2", "repo3"}
	for i, repo := range expectedRepos {
		if result.Items[i].Name != repo {
			t.Errorf("Expected result %d to be %s, got %s", i, repo, result.Items[i].Name)
		}
	}
}
