# GAAP: GitHub as a Package Manager

## Overview

GAAP ("GitHub as a Package Manager") is a terminal-based package manager that manages GitHub repositories and releases. Initially, GAAP will support fetching and installing packages from GitHub releases. Later, it will expand to handle source builds, leveraging advanced features such as dependency discovery, dev container integration, and potentially generative AI for build automation. The initial focus is on implementing the release-based functionality.

GAAP is a stateless, one-shot application (no persistent daemon) and aims to provide modern, idiomatic Go code for ease of maintenance and contribution.

## Goals

- Implement a package manager using GitHub as the backend.
- Support fetching and installing packages via GitHub releases.
- Create an intuitive CLI interface with familiar subcommands.
- Prepare for future extensibility to handle building from source.
- Use `libsql` for local storage to ensure full open-source compliance.

## Command Structure

GAAP will mimic the command structure of traditional package managers. Examples:

```sh
# Install a package
gaap install <repo-name>

# Specify user explicitly
gaap install <package-name> --user <user>

# Non-interactive install
gaap install <repo-name> --non-interactive

# Update all packages
gaap update

# Remove a package
gaap remove <package-name>
```

### Subcommands

1. **install**: Fetch and install a package.

   - Flags:
     - `--user`: Specify the repository’s user if not using `user/repo` syntax.
     - `--non-interactive`: Automatically select defaults for any prompts.
     - `--freeze`: Instantly freeze the installed package version.
     - `--dry-run`: Simulate the install without making changes.

2. **update**: Update all installed packages.

   - Flags:
     - `--build`: Include source-built packages.
     - `--dry-run`: Simulate the update process without making changes.

3. **remove**: Uninstall a package by name.

   - Flags:
     - `--dry-run`: Simulate the removal without making changes.

4. **list**: List installed packages.

5. **reconfigure**: Reconfigure an installed package (e.g., change command name or refreeze version).

6. **integrity**:

   - **check**: Perform an integrity check between the app directory and the database.
   - **fix**: Restore coherence between the app directory and the database.
   - **dry-run**:
     - Available as a flag for `check` and `fix`.
     - Simulate the actions without making changes, logging potential fixes or issues.

7. **configure**: Interactive CLI to set or update configuration values.

## Implementation Details

### Local Storage

- **Backend**: `libsql` (fork of SQLite) for package metadata.
- All files and data will be stored in `~/gaap/` by default.
  - **Structure:**
    - `~/gaap/bin`: Contains symlinks to executable commands.
    - `~/gaap/bin/actual`: Stores the actual binaries with unique names (e.g., `user-repo-version`).
    - `~/gaap/config`: Stores configuration files managed via the `configure` subcommand.
    - `~/gaap/db`: SQLite database for metadata.
    - `~/gaap/logs`: Logs for updates and actions.
  - A README in the `bin` directory warns users not to manually modify contents.

### GitHub Interaction

- Use GitHub’s REST API for fetching releases.
- For unauthenticated users, ensure rate-limiting safety (up to 60 requests per hour).
- Future: Add support for OAuth tokens for higher request limits.

### Installation Logic

Installation logic will be built around modularity and testability. All external dependencies, such as file system operations, API calls, and database interactions, will be abstracted behind interfaces. This abstraction ensures that components can be tested in isolation. Fuzzy finding techniques will be applied to match repository names or versions accurately while maintaining flexibility for user input.

Testing for this functionality will simulate various input scenarios, including partial or ambiguous matches, ensuring consistency in results; this will be achieved by using fuzz testing as a technique.

### Update Logic

- Fetch the latest release information for all installed packages.
- Compare current and latest versions.
- Download and replace if newer versions are available. Frozen packages will remain at their pinned versions unless explicitly updated.
- Log actions and failures.
- Testing will cover multiple update scenarios, ensuring reliability under different conditions.

### Integrity Management

- Integrity management will ensure consistency between the database and file system. Testing will involve deliberate mismatches to verify error detection and reporting.
- A dry-run mode will log discrepancies without making changes. Fixing will restore coherence, with tests validating recovery paths.

### Testing Strategy

The GAAP codebase will prioritize testability as a core principle. This involves designing components with well-defined boundaries, enabling straightforward mocking and isolation during tests. Key aspects include:

- Dependency injection for critical components like API clients, file systems, and database operations. This approach ensures that tests are independent of actual external systems.
- Robust unit tests covering individual functions and modules.
- Higher-order integration tests to simulate real-world usage scenarios. These will be run in controlled environments and form a critical part of CI/CD pipelines.
- Interactive features like fuzzy finding will be tested extensively with simulated user inputs. This ensures that matching is both flexible and deterministic, avoiding unexpected behavior.
- Dry-run functionality will serve as a lightweight verification mechanism for command execution. Tests will validate the consistency and accuracy of dry-run outputs.
- Systematic integrity checks to identify and resolve discrepancies between stored metadata and the actual state of installed packages. Tests will cover edge cases, such as manual file system changes and how the tool behaves with them.
- Fuzz testing will be employed to test various functions and inputs comprehensively. Inputs will be fuzzed to ensure robustness against unexpected or malformed data.

### User Management

GAAP will create a dedicated system user named `gaap` to manage its application directory (`~/gaap/`) and its contents. This ensures a secure, isolated environment for package management operations. The `gaap` user will:

- Own the `~/gaap/` directory and all subdirectories.
- Restrict write permissions to the `gaap` user, ensuring other users cannot modify critical files or configurations.
- Allow execution permissions for binaries within `~/gaap/bin` to all users.
- Be automatically created during GAAP’s installation or initialization process. If the `gaap` user already exists, the application will validate and configure permissions accordingly.
- Handle file system operations with built-in robustness to detect and resolve permission mismatches.

This approach ensures a secure and consistent setup across all environments, minimizing risks of accidental modifications or security breaches.

## Key Considerations

- **Non-interactive Mode**: Ensure robustness for automation pipelines.
- **Extensibility**: Design APIs and data structures to easily support the later addition of source builds.
- **Error Handling**: Keep errors localized to ensure predictable behavior.
- **Backups**: Keep backups of the metadata database.
- **Permissions**: The `gaap` user will be responsible for managing the application directory securely.
- **Dry Run Smartness**: Ensure dry-run mode is available and logs detailed, actionable information for every subcommand.
- **Fuzzy Finding**: Apply fuzzy matching techniques for user-friendly and flexible command inputs, ensuring consistent and accurate results.
- **Fuzz Testing**: Incorporate fuzz testing to validate the robustness of individual components against unexpected or malformed inputs.

