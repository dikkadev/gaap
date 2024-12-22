package e2e

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

var logger = log.New(os.Stdout, "E2E_TEST| ", log.LstdFlags|log.Lmicroseconds)

type testContainer struct {
	container testcontainers.Container
}

func buildTestImage(t *testing.T) error {
	logger.Println("Starting to build test image...")
	dir, err := os.Getwd()
	if err != nil {
		logger.Printf("ERROR: Failed to get working directory: %v\n", err)
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	logger.Printf("Working directory: %s\n", dir)

	buildScript := filepath.Join(dir, "build.sh")
	logger.Printf("Running build script: %s\n", buildScript)

	cmd := exec.Command("/bin/bash", buildScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Printf("ERROR: Build script failed: %v\n", err)
		return fmt.Errorf("failed to build test image: %w", err)
	}
	logger.Println("Test image built successfully")
	return nil
}

func setupContainer(ctx context.Context, t *testing.T) (*testContainer, error) {
	logger.Println("Setting up test container...")

	if err := buildTestImage(t); err != nil {
		logger.Printf("ERROR: Failed to build test image: %v\n", err)
		return nil, err
	}

	logger.Println("Creating container request...")
	req := testcontainers.ContainerRequest{
		Image:        "gaap-e2e-test:latest",
		ExposedPorts: []string{},
		Cmd:          []string{"tail", "-f", "/dev/null"},
	}

	logger.Println("Starting container...")
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		logger.Printf("ERROR: Failed to start container: %v\n", err)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	logger.Println("Container started successfully")
	return &testContainer{container: container}, nil
}

func (tc *testContainer) runGaap(ctx context.Context, args ...string) (string, error) {
	cmdStr := fmt.Sprintf("gaap %s", strings.Join(args, " "))
	logger.Printf("Executing command: %s\n", cmdStr)

	// Check gaap binary permissions
	exitCode, output, err := tc.container.Exec(ctx, []string{"ls", "-l", "/usr/local/bin/gaap"})
	if err == nil && exitCode == 0 {
		outputBytes, _ := io.ReadAll(output)
		logger.Printf("gaap binary permissions: %s\n", string(outputBytes))
	}

	// Check current working directory
	exitCode, output, err = tc.container.Exec(ctx, []string{"pwd"})
	if err == nil && exitCode == 0 {
		outputBytes, _ := io.ReadAll(output)
		logger.Printf("Current working directory: %s\n", string(outputBytes))
	}

	// Check user and groups
	exitCode, output, err = tc.container.Exec(ctx, []string{"id"})
	if err == nil && exitCode == 0 {
		outputBytes, _ := io.ReadAll(output)
		logger.Printf("User and group info: %s\n", string(outputBytes))
	}

	// Run gaap command
	exitCode, output, err = tc.container.Exec(ctx, append([]string{"gaap"}, args...))
	if err != nil {
		logger.Printf("ERROR: Failed to execute command: %v\n", err)
		return "", fmt.Errorf("failed to execute gaap: %w", err)
	}

	if output == nil {
		logger.Println("Command produced no output")
		return "", nil
	}

	outputBytes, err := io.ReadAll(output)
	if err != nil {
		logger.Printf("ERROR: Failed to read command output: %v\n", err)
		return "", fmt.Errorf("failed to read command output: %w", err)
	}

	outputStr := string(outputBytes)
	logger.Printf("Command output:\n%s\n", outputStr)

	if exitCode != 0 {
		logger.Printf("Command failed with exit code %d\n", exitCode)
		return outputStr, fmt.Errorf("gaap exited with code %d", exitCode)
	}

	return outputStr, nil
}

func (tc *testContainer) checkFileExists(ctx context.Context, path string) bool {
	logger.Printf("Checking if file exists: %s\n", path)

	// First list the directory contents
	exitCode, output, err := tc.container.Exec(ctx, []string{"ls", "-la", filepath.Dir(path)})
	if err == nil && exitCode == 0 {
		outputBytes, _ := io.ReadAll(output)
		logger.Printf("Directory contents of %s:\n%s\n", filepath.Dir(path), string(outputBytes))
	} else {
		logger.Printf("Failed to list directory %s: %v\n", filepath.Dir(path), err)
	}

	exitCode, _, err = tc.container.Exec(ctx, []string{"test", "-f", path})
	exists := err == nil && exitCode == 0
	logger.Printf("File %s exists: %v\n", path, exists)
	return exists
}

func (tc *testContainer) checkSymlinkExists(ctx context.Context, path string) bool {
	logger.Printf("Checking if symlink exists: %s\n", path)

	// First list the directory contents
	exitCode, output, err := tc.container.Exec(ctx, []string{"ls", "-la", filepath.Dir(path)})
	if err == nil && exitCode == 0 {
		outputBytes, _ := io.ReadAll(output)
		logger.Printf("Directory contents of %s:\n%s\n", filepath.Dir(path), string(outputBytes))
	} else {
		logger.Printf("Failed to list directory %s: %v\n", filepath.Dir(path), err)
	}

	exitCode, _, err = tc.container.Exec(ctx, []string{"test", "-L", path})
	exists := err == nil && exitCode == 0
	logger.Printf("Symlink %s exists: %v\n", path, exists)
	return exists
}

func (tc *testContainer) logDirectoryTree(ctx context.Context, path string, description string) {
	logger.Printf("=== Directory Tree for %s ===\n", description)

	// Run tree command with full details
	exitCode, output, err := tc.container.Exec(ctx, []string{"tree", "-a", "-p", "-u", "-g", "-L", "4", path})
	if err == nil && exitCode == 0 {
		outputBytes, _ := io.ReadAll(output)
		logger.Printf("Tree structure:\n%s\n", string(outputBytes))
	} else {
		logger.Printf("Failed to get tree for %s: %v\n", path, err)
	}

	// Also show detailed find output
	exitCode, output, err = tc.container.Exec(ctx, []string{"find", path, "-ls"})
	if err == nil && exitCode == 0 {
		outputBytes, _ := io.ReadAll(output)
		logger.Printf("Detailed file listing:\n%s\n", string(outputBytes))
	}

	logger.Printf("=== End Directory Tree for %s ===\n", description)
}

func (c *testContainer) exec(ctx context.Context, command string) (string, error) {
	code, output, err := c.container.Exec(ctx, []string{"bash", "-c", command})
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}
	if code != 0 {
		outputBytes, err := io.ReadAll(output)
		if err != nil {
			return "", fmt.Errorf("failed to read output: %w", err)
		}
		return string(outputBytes), fmt.Errorf("command exited with code %d", code)
	}
	outputBytes, err := io.ReadAll(output)
	if err != nil {
		return "", fmt.Errorf("failed to read output: %w", err)
	}
	return string(outputBytes), nil
}

func (c *testContainer) terminate(ctx context.Context) {
	logger.Println("🐳 Stopping container:", c.container.GetContainerID())
	if err := c.container.Stop(ctx, nil); err != nil {
		logger.Printf("Failed to stop container: %v\n", err)
	}
	logger.Println("✅ Container stopped:", c.container.GetContainerID())

	logger.Println("🐳 Terminating container:", c.container.GetContainerID())
	if err := c.container.Terminate(ctx); err != nil {
		logger.Printf("Failed to terminate container: %v\n", err)
	}
	logger.Println("🚫 Container terminated:", c.container.GetContainerID())
}

func TestEndToEnd(t *testing.T) {
	logger.Println("=== Starting End-to-End Test ===")
	ctx := context.Background()

	// Declare variables used throughout the function
	var (
		exitCode    int
		dirOutput   io.Reader
		err         error
		cmdOutput   string
		outputBytes []byte
	)

	logger.Println("Setting up test container...")
	tc, err := setupContainer(ctx, t)
	if err != nil {
		logger.Printf("FATAL: Failed to setup container: %v\n", err)
		t.Fatalf("Failed to setup container: %v", err)
	}
	defer func() {
		logger.Println("Terminating container...")
		tc.terminate(ctx)
	}()

	// Initial directory structure check
	tc.logDirectoryTree(ctx, "/home/testuser", "Initial Home Directory")
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "Initial GAAP Directory")

	// Verify initial directory structure
	logger.Println("Verifying initial directory structure...")
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"ls", "-la", "/home/testuser/gaap"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("Initial gaap directory contents:\n%s\n", string(outputBytes))
	}

	// Also check the bin directory
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"ls", "-la", "/home/testuser/gaap/bin"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("Initial bin directory contents:\n%s\n", string(outputBytes))
	}

	repo := "cli/cli"
	logger.Printf("Testing package installation for: %s\n", repo)

	// Check initial permissions of .gaap directory
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"ls", "-la", "/home/testuser/gaap"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("Initial .gaap directory permissions:\n%s\n", string(outputBytes))
	}

	// Check if we can write to the directories
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"touch", "/home/testuser/gaap/bin/actual/test"})
	if err != nil || exitCode != 0 {
		logger.Printf("WARNING: Cannot write to actual directory: %v (exit code: %d)\n", err, exitCode)
	} else {
		logger.Println("Successfully wrote test file to actual directory")
		tc.container.Exec(ctx, []string{"rm", "/home/testuser/gaap/bin/actual/test"})
	}

	cmdOutput, err = tc.runGaap(ctx, "install", repo)
	if err != nil {
		logger.Printf("FATAL: Failed to install package: %v\nOutput: %s\n", err, cmdOutput)
		t.Fatalf("Failed to install package: %v\nOutput: %s", err, cmdOutput)
	}

	// Check directory structure immediately after installation command
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Installation Command")

	// Check the entire home directory
	tc.logDirectoryTree(ctx, "/home/testuser", "Complete Home Directory After Installation")

	// Check for any new files in home directory
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"find", "/home/testuser", "-type", "f", "-newer", "/home/testuser/.bashrc"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("New files in home directory:\n%s\n", string(outputBytes))
	}

	// Check for any gh files anywhere in home
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"find", "/home/testuser", "-name", "gh", "-o", "-name", "*.gh"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("Any 'gh' files in home:\n%s\n", string(outputBytes))
	}

	// Check download directory if it exists
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"ls", "-la", "/home/testuser/gaap/downloads"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("Downloads directory contents:\n%s\n", string(outputBytes))
	}

	// Check if there are any temporary directories
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"find", "/home/testuser/gaap", "-type", "d", "-name", "tmp*"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("Temporary directories:\n%s\n", string(outputBytes))
	}

	// Check database state
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"find", "/home/testuser/gaap", "-name", "*.db"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("Database files:\n%s\n", string(outputBytes))

		// If we found a database, try to read its contents
		dbFiles := strings.Split(strings.TrimSpace(string(outputBytes)), "\n")
		for _, dbFile := range dbFiles {
			exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"sqlite3", dbFile, ".tables"})
			if err == nil && exitCode == 0 {
				outputBytes, _ = io.ReadAll(dirOutput)
				logger.Printf("Tables in %s:\n%s\n", dbFile, string(outputBytes))

				// Try to read the packages table
				exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"sqlite3", dbFile, "SELECT * FROM packages;"})
				if err == nil && exitCode == 0 {
					outputBytes, _ = io.ReadAll(dirOutput)
					logger.Printf("Packages in database:\n%s\n", string(outputBytes))
				}
			}
		}
	}

	// Check directory permissions immediately after installation
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"ls", "-la", "/home/testuser/gaap/bin/actual"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("Actual directory permissions after install:\n%s\n", string(outputBytes))
	}

	// Check processes that might be using the directory
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"lsof", "/home/testuser/gaap/bin/actual"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("Processes using actual directory:\n%s\n", string(outputBytes))
	}

	// Increase wait time and add progress logging
	waitTime := 10 // seconds
	logger.Printf("Waiting %d seconds for installation to complete...\n", waitTime)
	for i := 0; i < waitTime; i++ {
		time.Sleep(1 * time.Second)
		logger.Printf("Waiting... %d seconds remaining\n", waitTime-i-1)

		// Check directory contents every second
		exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"ls", "-la", "/home/testuser/gaap/bin/actual"})
		if err == nil && exitCode == 0 {
			outputBytes, _ = io.ReadAll(dirOutput)
			logger.Printf("Current actual directory contents:\n%s\n", string(outputBytes))
		}
	}

	// Check directory structure after waiting
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Wait Period")

	// Check the entire .gaap directory structure after installation
	logger.Println("Checking .gaap directory structure after installation...")
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"find", "/home/testuser/gaap", "-ls"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("Complete .gaap directory structure:\n%s\n", string(outputBytes))
	}

	// Check file permissions and ownership
	logger.Println("Checking file permissions and ownership...")
	exitCode, dirOutput, err = tc.container.Exec(ctx, []string{"ls", "-la", "/home/testuser/gaap/bin"})
	if err == nil && exitCode == 0 {
		outputBytes, _ = io.ReadAll(dirOutput)
		logger.Printf("Bin directory permissions:\n%s\n", string(outputBytes))
	}

	// Update the paths to match where the binary is actually installed
	binaryPath := "/home/testuser/gaap/bin/actual/cli-cli-v2.64.0"
	symlinkPath := "/home/testuser/gaap/bin/cli"

	logger.Println("Verifying binary installation...")
	if !tc.checkFileExists(ctx, binaryPath) {
		// Check if the binary exists in a different location
		logger.Println("Binary not found at expected location, searching in other directories...")
		exitCode, findOutput, err := tc.container.Exec(ctx, []string{"find", "/home/testuser", "-type", "f", "-name", "cli-cli-*"})
		if err == nil && exitCode == 0 {
			outputBytes, _ = io.ReadAll(findOutput)
			logger.Printf("Found cli binary files:\n%s\n", string(outputBytes))
		}

		// Check if the binary exists anywhere in the container
		logger.Println("Searching for cli binary in entire container...")
		exitCode, findOutput, err = tc.container.Exec(ctx, []string{"find", "/", "-type", "f", "-name", "cli-cli-*", "2>/dev/null"})
		if err == nil && exitCode == 0 {
			outputBytes, _ = io.ReadAll(findOutput)
			logger.Printf("Found cli binary files in container:\n%s\n", string(outputBytes))
		}

		logger.Printf("FATAL: Binary file not found at %s\n", binaryPath)
		t.Fatal("Binary file not found")
	}

	logger.Println("Verifying symlink creation...")
	if !tc.checkSymlinkExists(ctx, symlinkPath) {
		// Check all symlinks in the gaap directory
		logger.Println("Symlink not found at expected location, searching for all symlinks...")
		exitCode, findOutput, err := tc.container.Exec(ctx, []string{"find", "/home/testuser/gaap", "-type", "l", "-ls"})
		if err == nil && exitCode == 0 {
			outputBytes, _ = io.ReadAll(findOutput)
			logger.Printf("Found symlinks in gaap directory:\n%s\n", string(outputBytes))
		}

		logger.Printf("FATAL: Symlink not found at %s\n", symlinkPath)
		t.Fatal("Symlink not found")
	}

	logger.Println("Testing package update...")
	cmdOutput, err = tc.runGaap(ctx, "update", repo)
	if err != nil {
		logger.Printf("FATAL: Failed to update package: %v\nOutput: %s\n", err, cmdOutput)
		t.Fatalf("Failed to update package: %v\nOutput: %s", err, cmdOutput)
	}

	// Check directory structure after update
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Update")

	logger.Printf("Waiting %d seconds for update to complete...\n", waitTime)
	time.Sleep(time.Duration(waitTime) * time.Second)

	// Check directory structure after update wait
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Update Wait")

	logger.Println("Testing package removal...")
	cmdOutput, err = tc.runGaap(ctx, "remove", repo)
	if err != nil {
		logger.Printf("FATAL: Failed to remove package: %v\nOutput: %s\n", err, cmdOutput)
		t.Fatalf("Failed to remove package: %v\nOutput: %s", err, cmdOutput)
	}

	// Check directory structure after removal
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Removal")

	logger.Println("Verifying binary removal...")
	if tc.checkFileExists(ctx, binaryPath) {
		logger.Printf("FATAL: Binary file still exists at %s\n", binaryPath)
		t.Fatal("Binary file still exists after removal")
	}

	logger.Println("Verifying symlink removal...")
	if tc.checkSymlinkExists(ctx, symlinkPath) {
		logger.Printf("FATAL: Symlink still exists at %s\n", symlinkPath)
		t.Fatal("Symlink still exists after removal")
	}

	logger.Println("=== End-to-End Test Completed Successfully ===")
}

func TestListPackages(t *testing.T) {
	logger.Println("=== Starting List Packages Test ===")
	ctx := context.Background()

	logger.Println("Setting up test container...")
	tc, err := setupContainer(ctx, t)
	if err != nil {
		logger.Printf("FATAL: Failed to setup container: %v\n", err)
		t.Fatalf("Failed to setup container: %v", err)
	}
	defer func() {
		logger.Println("Terminating container...")
		tc.terminate(ctx)
	}()

	// Initial directory check
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "Initial GAAP Directory")

	repo := "cli/cli"
	logger.Printf("Installing test package: %s\n", repo)
	_, err = tc.runGaap(ctx, "install", repo)
	if err != nil {
		logger.Printf("FATAL: Failed to install package: %v\n", err)
		t.Fatalf("Failed to install package: %v", err)
	}

	// Check directory after installation
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Installation")

	logger.Printf("Waiting %d seconds for installation to complete...\n", 5)
	time.Sleep(5 * time.Second)

	// Check directory after wait
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Wait")

	logger.Println("Testing list command...")
	output, err := tc.runGaap(ctx, "list")
	if err != nil {
		logger.Printf("FATAL: Failed to list packages: %v\nOutput: %s\n", err, output)
		t.Fatalf("Failed to list packages: %v\nOutput: %s", err, output)
	}

	logger.Printf("List command output:\n%s\n", output)
	if !strings.Contains(output, repo) {
		logger.Printf("FATAL: List output does not contain installed package %s\n", repo)
		t.Fatalf("List output does not contain installed package. Output: %s", output)
	}

	logger.Println("=== List Packages Test Completed Successfully ===")
}

func TestInstallErrors(t *testing.T) {
	logger.Println("=== Starting Install Errors Test ===")
	ctx := context.Background()

	logger.Println("Setting up test container...")
	tc, err := setupContainer(ctx, t)
	if err != nil {
		logger.Printf("FATAL: Failed to setup container: %v\n", err)
		t.Fatalf("Failed to setup container: %v", err)
	}
	defer func() {
		logger.Println("Terminating container...")
		tc.terminate(ctx)
	}()

	// Initial directory check
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "Initial GAAP Directory")

	logger.Println("Testing installation of non-existent repository...")
	_, err = tc.runGaap(ctx, "install", "nonexistent/repo")
	if err == nil {
		logger.Println("FATAL: Expected error when installing non-existent repository, but got none")
		t.Fatal("Expected error when installing non-existent repository")
	}
	logger.Printf("Got expected error: %v\n", err)

	// Check directory after failed installation
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Failed Installation")

	logger.Println("Testing installation with invalid repository format...")
	_, err = tc.runGaap(ctx, "install", "invalid-format")
	if err == nil {
		logger.Println("FATAL: Expected error when installing invalid repository format, but got none")
		t.Fatal("Expected error when installing invalid repository format")
	}
	logger.Printf("Got expected error: %v\n", err)

	// Check directory after invalid format installation
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Invalid Format Installation")

	repo := "cli/cli"
	logger.Printf("Installing test package: %s\n", repo)
	_, err = tc.runGaap(ctx, "install", repo)
	if err != nil {
		logger.Printf("FATAL: Failed to install package: %v\n", err)
		t.Fatalf("Failed to install package: %v", err)
	}

	// Check directory after successful installation
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Successful Installation")

	logger.Printf("Waiting %d seconds for installation to complete...\n", 5)
	time.Sleep(5 * time.Second)

	// Check directory after wait
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Wait")

	logger.Println("Testing installation of already installed package...")
	_, err = tc.runGaap(ctx, "install", repo)
	if err == nil {
		logger.Println("FATAL: Expected error when installing already installed package, but got none")
		t.Fatal("Expected error when installing already installed package")
	}
	logger.Printf("Got expected error: %v\n", err)

	// Check directory after duplicate installation attempt
	tc.logDirectoryTree(ctx, "/home/testuser/gaap", "GAAP Directory After Duplicate Installation Attempt")

	logger.Println("=== Install Errors Test Completed Successfully ===")
}

func TestFlagBehavior(t *testing.T) {
	logger.Println("=== Starting Flag Behavior Test ===")

	// Set up test container
	logger.Println("Setting up test container...")
	ctx := context.Background()
	container, err := setupContainer(ctx, t)
	if err != nil {
		t.Fatalf("Failed to set up container: %v", err)
	}
	defer func() {
		logger.Println("Terminating container...")
		container.terminate(ctx)
	}()

	logger.Println("Container started successfully")

	// Test dry-run install
	logger.Println("Testing dry-run install...")
	output, err := container.exec(ctx, "gaap install --dry-run cli/cli")
	if err != nil {
		t.Fatalf("Dry-run install failed: %v", err)
	}
	if !strings.Contains(output, "Would install package: cli/cli") {
		t.Error("Dry-run install output doesn't contain expected message")
	}

	// Test non-interactive install
	logger.Println("Testing non-interactive install...")
	output, err = container.exec(ctx, "gaap install --non-interactive cli/cli")
	if err != nil {
		t.Fatalf("Non-interactive install failed: %v", err)
	}
	if !strings.Contains(output, "Successfully installed cli/cli") {
		t.Error("Non-interactive install output doesn't contain success message")
	}

	// Test frozen install
	logger.Println("Testing frozen install...")
	output, err = container.exec(ctx, "gaap install --freeze sharkdp/bat")
	if err != nil {
		t.Fatalf("Frozen install failed: %v", err)
	}
	if !strings.Contains(output, "Package version is frozen") {
		t.Error("Frozen install output doesn't contain freeze message")
	}

	// Test dry-run update
	logger.Println("Testing dry-run update...")
	output, err = container.exec(ctx, "gaap update --dry-run")
	if err != nil {
		t.Fatalf("Dry-run update failed: %v", err)
	}
	if !strings.Contains(output, "Would update package") {
		t.Error("Dry-run update output doesn't contain expected message")
	}

	// Test build flag warning
	logger.Println("Testing build flag warning...")
	output, err = container.exec(ctx, "gaap update --build")
	if err != nil {
		t.Fatalf("Update with build flag failed: %v", err)
	}
	if !strings.Contains(output, "Warning: The --build flag is not implemented yet") {
		t.Error("Build flag warning message not found")
	}

	// Test dry-run remove
	logger.Println("Testing dry-run remove...")
	output, err = container.exec(ctx, "gaap remove --dry-run cli/cli")
	if err != nil {
		t.Fatalf("Dry-run remove failed: %v", err)
	}
	if !strings.Contains(output, "Would remove package") {
		t.Error("Dry-run remove output doesn't contain expected message")
	}

	// Test update behavior with frozen package
	logger.Println("Testing update with frozen package...")
	output, err = container.exec(ctx, "gaap update")
	if err != nil {
		t.Fatalf("Update with frozen package failed: %v", err)
	}
	if !strings.Contains(output, "Skipping frozen package") {
		t.Error("Update output doesn't show skipping frozen package")
	}

	logger.Println("=== Flag Behavior Test Completed Successfully ===")
}
