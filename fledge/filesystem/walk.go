package filesystem

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultIgnoreDirs are common directories to skip during traversal
var DefaultIgnoreDirs = []string{
	"node_modules", "vendor", ".git", ".svn", ".hg",
	"dist", "build", "bin", "tmp", "temp",
	".idea", ".vscode", ".vs",
}

// WalkOptions configures directory traversal behavior
type WalkOptions struct {
	IgnoreDirs     []string // Directories to skip (default: DefaultIgnoreDirs)
	IgnorePatterns []string // File patterns to skip (e.g., "*.tmp")
	IncludeHidden  bool     // Include hidden files/dirs (default: false)
	FollowSymlinks bool     // Follow symbolic links (default: false)
}

// Walk traverses a directory tree with configurable ignore patterns.
// The visitor function is called for each file and directory.
// Return filepath.SkipDir from visitor to skip a directory.
func Walk(rootPath string, opts WalkOptions, visitor func(path string, info os.FileInfo) error) error {
	ignoreDirs := opts.IgnoreDirs
	if len(ignoreDirs) == 0 {
		ignoreDirs = DefaultIgnoreDirs
	}

	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files/directories unless explicitly included
		if !opts.IncludeHidden && strings.HasPrefix(info.Name(), ".") && path != rootPath {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check ignore directories
		if info.IsDir() {
			for _, ignore := range ignoreDirs {
				if info.Name() == ignore {
					return filepath.SkipDir
				}
			}
		}

		// Check ignore patterns
		if !info.IsDir() && len(opts.IgnorePatterns) > 0 {
			for _, pattern := range opts.IgnorePatterns {
				if matched, _ := filepath.Match(pattern, info.Name()); matched {
					return nil
				}
			}
		}

		return visitor(path, info)
	})
}

// WalkWithDefaults walks a directory tree with default ignore patterns.
// This is a convenience wrapper around Walk with sensible defaults.
func WalkWithDefaults(rootPath string, visitor func(path string, info os.FileInfo) error) error {
	return Walk(rootPath, WalkOptions{}, visitor)
}
