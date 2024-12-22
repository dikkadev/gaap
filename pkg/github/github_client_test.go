package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
)

func TestGetLatestRelease(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			t.Errorf("Expected request to '/repos/owner/repo/releases/latest', got: %s", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		release := &github.RepositoryRelease{
			TagName: github.String("v1.0.0"),
			Name:    github.String("Release 1.0.0"),
			Body:    github.String("Release notes"),
			PublishedAt: &github.Timestamp{
				Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			Assets: []*github.ReleaseAsset{
				{
					Name:               github.String("binary"),
					Size:               github.Int(1000),
					BrowserDownloadURL: github.String("https://example.com/binary"),
				},
			},
		}

		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	// Create a custom client with the test server URL
	serverURL, _ := url.Parse(server.URL + "/")
	testClient := github.NewClient(&http.Client{})
	testClient.BaseURL = serverURL

	client := &client{
		ghClient: testClient,
	}

	release, err := client.GetLatestRelease(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("GetLatestRelease returned error: %v", err)
	}

	if release.TagName != "v1.0.0" {
		t.Errorf("Expected tag name 'v1.0.0', got %s", release.TagName)
	}

	if len(release.Assets) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(release.Assets))
	}
}

func TestDownloadAsset(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "gaap-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock server that serves a test file
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	client := NewClient("")
	destPath := filepath.Join(tmpDir, "test-file")

	err = client.DownloadAsset(context.Background(), server.URL, destPath)
	if err != nil {
		t.Fatalf("DownloadAsset returned error: %v", err)
	}

	// Verify the file was downloaded correctly
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("Expected content 'test content', got %s", string(content))
	}
}
