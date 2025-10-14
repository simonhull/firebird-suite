package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PackageDiscoveryOptions configures Go package discovery
type PackageDiscoveryOptions struct {
	IncludeTests  bool     // Include test files in detection (default: false)
	IncludeVendor bool     // Include vendor directory (default: false)
	ExcludePaths  []string // Additional paths to exclude
}

// DiscoverGoPackages finds all Go packages in a directory tree.
// Returns a sorted list of package directory paths.
func DiscoverGoPackages(rootPath string, opts PackageDiscoveryOptions) ([]string, error) {
	pkgDirs := make(map[string]bool)

	// Build ignore list
	ignoreDirs := make([]string, 0, len(DefaultIgnoreDirs))
	for _, dir := range DefaultIgnoreDirs {
		// Skip vendor from default ignores if IncludeVendor is true
		if dir == "vendor" && opts.IncludeVendor {
			continue
		}
		ignoreDirs = append(ignoreDirs, dir)
	}

	walkOpts := WalkOptions{
		IgnoreDirs: ignoreDirs,
	}

	err := Walk(rootPath, walkOpts, func(path string, info os.FileInfo) error {
		// Only process .go files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		// Skip test files unless explicitly included
		if !opts.IncludeTests && strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		// Check exclude paths
		pkgDir := filepath.Dir(path)
		for _, exclude := range opts.ExcludePaths {
			if strings.Contains(pkgDir, exclude) {
				return nil
			}
		}

		pkgDirs[pkgDir] = true
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover packages: %w", err)
	}

	// Convert map to sorted slice
	result := make([]string, 0, len(pkgDirs))
	for dir := range pkgDirs {
		result = append(result, dir)
	}
	sort.Strings(result)

	return result, nil
}
