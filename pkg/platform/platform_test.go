package platform

import (
	"strings"
	"testing"

	"github.com/dikkadev/gaap/pkg/github"
)

func TestPlatformString(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		want     string
	}{
		{
			name:     "linux-amd64",
			platform: Platform{OS: "linux", Arch: "amd64"},
			want:     "linux-amd64",
		},
		{
			name:     "darwin-arm64",
			platform: Platform{OS: "darwin", Arch: "arm64"},
			want:     "darwin-arm64",
		},
		{
			name:     "windows-386",
			platform: Platform{OS: "windows", Arch: "386"},
			want:     "windows-386",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.platform.String(); got != tt.want {
				t.Errorf("Platform.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectAsset(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		assets   []github.Asset
		want     string
		wantErr  bool
	}{
		{
			name:     "exact match linux-amd64",
			platform: Platform{OS: "linux", Arch: "amd64"},
			assets: []github.Asset{
				{Name: "app-windows-amd64.exe"},
				{Name: "app-linux-amd64"},
				{Name: "app-darwin-amd64"},
			},
			want:    "app-linux-amd64",
			wantErr: false,
		},
		{
			name:     "fuzzy match darwin (macos)",
			platform: Platform{OS: "darwin", Arch: "arm64"},
			assets: []github.Asset{
				{Name: "app-windows.exe"},
				{Name: "app-macos-arm64"},
				{Name: "app-linux"},
			},
			want:    "app-macos-arm64",
			wantErr: false,
		},
		{
			name:     "os only match",
			platform: Platform{OS: "linux", Arch: "amd64"},
			assets: []github.Asset{
				{Name: "app-windows.exe"},
				{Name: "app-linux"},
				{Name: "app-darwin"},
			},
			want:    "app-linux",
			wantErr: false,
		},
		{
			name:     "no suitable asset",
			platform: Platform{OS: "linux", Arch: "arm64"},
			assets: []github.Asset{
				{Name: "app-windows-amd64.exe"},
				{Name: "app-darwin-amd64"},
			},
			want:    "",
			wantErr: true,
		},
		{
			name:     "prefer binary over source",
			platform: Platform{OS: "linux", Arch: "amd64"},
			assets: []github.Asset{
				{Name: "app-linux-amd64"},
				{Name: "app-src.tar.gz"},
				{Name: "app-source.zip"},
			},
			want:    "app-linux-amd64",
			wantErr: false,
		},
		{
			name:     "special case darwin64",
			platform: Platform{OS: "darwin", Arch: "amd64"},
			assets: []github.Asset{
				{Name: "app-win64.exe"},
				{Name: "app-darwin64"},
				{Name: "app-linux64"},
			},
			want:    "app-darwin64",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.platform.SelectAsset(tt.assets)
			if (err != nil) != tt.wantErr {
				t.Errorf("Platform.SelectAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Name != tt.want {
				t.Errorf("Platform.SelectAsset() = %v, want %v", got.Name, tt.want)
			}
		})
	}
}

func TestSelectAssetEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		assets   []github.Asset
		want     string
		wantErr  bool
	}{
		{
			name:     "multiple matching assets",
			platform: Platform{OS: "linux", Arch: "amd64"},
			assets: []github.Asset{
				{Name: "app-linux-amd64-v1.0.0"},
				{Name: "app-linux-amd64-latest"},
				{Name: "app-linux-amd64.tar.gz"},
			},
			want:    "app-linux-amd64-v1.0.0", // Should prefer non-archive version
			wantErr: false,
		},
		{
			name:     "versioned assets",
			platform: Platform{OS: "linux", Arch: "amd64"},
			assets: []github.Asset{
				{Name: "app-v1.0.0-linux-amd64"},
				{Name: "app-v1.0.0-windows-amd64.exe"},
				{Name: "app-v1.0.0-darwin-amd64"},
			},
			want:    "app-v1.0.0-linux-amd64",
			wantErr: false,
		},
		{
			name:     "different compression formats",
			platform: Platform{OS: "linux", Arch: "amd64"},
			assets: []github.Asset{
				{Name: "app-linux-amd64.tar.gz"},
				{Name: "app-linux-amd64.zip"},
				{Name: "app-linux-amd64.tar.xz"},
			},
			want:    "app-linux-amd64.tar.gz", // Should pick first matching archive
			wantErr: false,
		},
		{
			name:     "universal macOS binary",
			platform: Platform{OS: "darwin", Arch: "arm64"},
			assets: []github.Asset{
				{Name: "app-macos-universal"},
				{Name: "app-macos-x86_64"},
				{Name: "app-macos-arm64"},
			},
			want:    "app-macos-arm64", // Should prefer exact match over universal
			wantErr: false,
		},
		{
			name:     "ARM variants",
			platform: Platform{OS: "linux", Arch: "arm"},
			assets: []github.Asset{
				{Name: "app-linux-armv7"},
				{Name: "app-linux-armv6"},
				{Name: "app-linux-arm"},
			},
			want:    "app-linux-arm", // Should prefer generic ARM over specific version
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.platform.SelectAsset(tt.assets)
			if (err != nil) != tt.wantErr {
				t.Errorf("Platform.SelectAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Name != tt.want {
				t.Errorf("Platform.SelectAsset() = %v, want %v", got.Name, tt.want)
			}
		})
	}
}

func TestMatchScore(t *testing.T) {
	tests := []struct {
		name      string
		assetName string
		platform  Platform
		want      int
	}{
		{
			name:      "exact OS and arch match",
			assetName: "app-linux-amd64",
			platform:  Platform{OS: "linux", Arch: "amd64"},
			want:      15, // 10 for OS + 5 for arch
		},
		{
			name:      "OS match with alternative name",
			assetName: "app-macos-amd64",
			platform:  Platform{OS: "darwin", Arch: "amd64"},
			want:      15, // 10 for OS + 5 for arch
		},
		{
			name:      "Windows with exe extension",
			assetName: "app.exe",
			platform:  Platform{OS: "windows", Arch: "amd64"},
			want:      5, // 5 for .exe extension
		},
		{
			name:      "source code penalty",
			assetName: "app-linux-amd64-src.tar.gz",
			platform:  Platform{OS: "linux", Arch: "amd64"},
			want:      5, // 15 for matches - 10 for source
		},
		{
			name:      "GNU/Linux variant",
			assetName: "app-gnu-amd64",
			platform:  Platform{OS: "linux", Arch: "amd64"},
			want:      10, // 5 for GNU + 5 for arch
		},
		{
			name:      "ARM without version",
			assetName: "app-linux-arm",
			platform:  Platform{OS: "linux", Arch: "arm"},
			want:      15, // 10 for OS + 5 for arch
		},
		{
			name:      "ARM64 specific",
			assetName: "app-linux-arm64",
			platform:  Platform{OS: "linux", Arch: "arm64"},
			want:      15, // 10 for OS + 5 for arch
		},
		{
			name:      "no matches",
			assetName: "something-completely-different",
			platform:  Platform{OS: "linux", Arch: "amd64"},
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchScore(strings.ToLower(tt.assetName), tt.platform); got != tt.want {
				t.Errorf("matchScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeArch(t *testing.T) {
	tests := []struct {
		name string
		arch string
		want string
	}{
		{"amd64 as is", "amd64", "amd64"},
		{"x86_64 to amd64", "x86_64", "amd64"},
		{"386 as is", "386", "386"},
		{"x86 to 386", "x86", "386"},
		{"arm64 as is", "arm64", "arm64"},
		{"aarch64 to arm64", "aarch64", "arm64"},
		{"arm as is", "arm", "arm"},
		{"unknown as is", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeArch(tt.arch); got != tt.want {
				t.Errorf("normalizeArch() = %v, want %v", got, tt.want)
			}
		})
	}
}
