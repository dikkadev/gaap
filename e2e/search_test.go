package e2e

import (
	"context"
	"strings"
	"testing"
)

func TestRepositorySearch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping repository search e2e test in short mode")
	}

	logger.Println("=== Starting Repository Search Test ===")
	ctx := context.Background()

	// Setup test container
	container, err := setupContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to setup container: %v", err)
	}
	defer func() {
		container.terminate(ctx)
	}()

	// Test cases for repository search
	testCases := []struct {
		name          string
		searchInput   string
		expectedRepo  string   // The repository we expect to be selected
		shouldContain []string // Strings that should be in the output
	}{
		{
			name:         "Exact repository match",
			searchInput:  "cli/cli",
			expectedRepo: "cli/cli",
			shouldContain: []string{
				"Successfully installed cli/cli@",
			},
		},
		{
			name:         "User repository search",
			searchInput:  "charmbracelet/glow",
			expectedRepo: "charmbracelet/glow",
			shouldContain: []string{
				"Successfully installed charmbracelet/glow@",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Run gaap install with the search input
			output, err := container.runGaap(ctx, "install", "--non-interactive", testCase.searchInput)

			// For exact matches, we expect the installation to proceed
			if testCase.expectedRepo != "" {
				if err != nil {
					t.Errorf("Expected successful installation for %s, got error: %v", testCase.searchInput, err)
				}

				// Verify the package was installed
				if !container.checkFileExists(ctx, "/home/testuser/gaap/bin/"+strings.Split(testCase.expectedRepo, "/")[1]) {
					t.Errorf("Expected binary to be installed for %s", testCase.expectedRepo)
				}
			}

			// Verify output contains expected strings
			for _, expected := range testCase.shouldContain {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput: %s", expected, output)
				}
			}
		})
	}

	// Test interactive search separately
	t.Run("Interactive search", func(t *testing.T) {
		// Skip this test for now as we need to figure out how to handle interactive input
		t.Skip("Interactive search test not implemented yet")

		output, err := container.runGaap(ctx, "install", "cli")
		if err != nil {
			t.Errorf("Interactive search failed: %v", err)
		}

		expectedStrings := []string{
			"Select a repository",
			"cli/cli",
			"GitHub's official command line tool",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(output, expected) {
				t.Errorf("Expected output to contain %q, but it didn't.\nOutput: %s", expected, output)
			}
		}
	})
}
