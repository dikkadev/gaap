package platform

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/dikkadev/gaap/pkg/github"
)

// Platform represents a target platform
type Platform struct {
	OS   string
	Arch string
}

// Current returns the current platform
func Current() Platform {
	return Platform{
		OS:   runtime.GOOS,
		Arch: normalizeArch(runtime.GOARCH),
	}
}

// String returns a string representation of the platform
func (p Platform) String() string {
	return fmt.Sprintf("%s-%s", p.OS, p.Arch)
}

// SelectAsset selects the most appropriate asset for the platform
func (p Platform) SelectAsset(assets []github.Asset) (*github.Asset, error) {
	// Common naming patterns for different platforms
	patterns := []string{
		// Exact matches
		fmt.Sprintf("%s_%s", p.OS, p.Arch),
		fmt.Sprintf("%s-%s", p.OS, p.Arch),
		// OS-only matches (some assets only specify OS)
		p.OS,
		// Special cases
		fmt.Sprintf("%s%s", p.OS, p.Arch), // e.g., "darwin64"
	}

	// First try exact matches
	for _, pattern := range patterns {
		for _, asset := range assets {
			name := strings.ToLower(asset.Name)
			if strings.Contains(name, pattern) {
				// For ARM, prefer generic ARM over specific versions
				if p.Arch == "arm" && !strings.Contains(name, "armv") {
					return &asset, nil
				}
			}
		}
	}

	// If no exact match found for ARM, try again but accept specific versions
	if p.Arch == "arm" {
		for _, pattern := range patterns {
			for _, asset := range assets {
				name := strings.ToLower(asset.Name)
				if strings.Contains(name, pattern) {
					return &asset, nil
				}
			}
		}
	}

	// If no match found, try fuzzy matching
	bestMatch := -1
	bestScore := 0

	for i, asset := range assets {
		name := strings.ToLower(asset.Name)
		score := matchScore(name, p)
		if score > bestScore {
			bestScore = score
			bestMatch = i
		}
	}

	if bestMatch >= 0 && bestScore > 0 {
		return &assets[bestMatch], nil
	}

	return nil, fmt.Errorf("no suitable asset found for platform %s", p)
}

// matchScore returns a score indicating how well an asset name matches the platform
func matchScore(name string, p Platform) int {
	score := 0

	// Check OS matches
	switch p.OS {
	case "linux":
		if strings.Contains(name, "linux") {
			score += 10
		} else if strings.Contains(name, "gnu") {
			score += 5
		}
	case "darwin":
		if strings.Contains(name, "darwin") || strings.Contains(name, "macos") || strings.Contains(name, "osx") {
			score += 10
		}
	case "windows":
		if strings.Contains(name, "windows") || strings.Contains(name, "win") {
			score += 10
		} else if strings.HasSuffix(name, ".exe") {
			score += 5
		}
	}

	// Check arch matches
	switch p.Arch {
	case "amd64":
		if strings.Contains(name, "amd64") || strings.Contains(name, "x86_64") || strings.Contains(name, "64") {
			score += 5
		}
	case "386":
		if strings.Contains(name, "386") || strings.Contains(name, "x86") || strings.Contains(name, "32") {
			score += 5
		}
	case "arm64":
		if strings.Contains(name, "arm64") || strings.Contains(name, "aarch64") {
			score += 5
		}
	case "arm":
		if strings.Contains(name, "arm") && !strings.Contains(name, "arm64") {
			score += 5
			// Penalize specific ARM versions when we want generic ARM
			if strings.Contains(name, "armv") {
				score -= 2
			}
		}
	}

	// Penalize source code archives
	if strings.Contains(name, "src") || strings.Contains(name, "source") {
		score -= 10
	}

	return score
}

// normalizeArch normalizes architecture names
func normalizeArch(arch string) string {
	switch arch {
	case "x86_64":
		return "amd64"
	case "x86":
		return "386"
	case "aarch64":
		return "arm64"
	default:
		return arch
	}
}
