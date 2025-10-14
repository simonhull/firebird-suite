// Package filesystem provides utilities for traversing and analyzing
// file systems with smart defaults for Go projects.
//
// # Overview
//
// This package helps CLI tools navigate directory structures while
// respecting common ignore patterns:
//   - Smart directory traversal (skip node_modules, .git, vendor)
//   - Go package discovery (find all packages in a tree)
//   - Pattern-based filtering (ignore *.tmp, test files, etc.)
//
// # Usage
//
// Walk a directory with default ignores:
//
//	err := filesystem.WalkWithDefaults(".", func(path string, info os.FileInfo) error {
//	    fmt.Println(path)
//	    return nil
//	})
//
// Discover all Go packages:
//
//	packages, err := filesystem.DiscoverGoPackages(".", filesystem.PackageDiscoveryOptions{
//	    IncludeTests: false,
//	})
//	for _, pkg := range packages {
//	    fmt.Println(pkg)
//	}
//
// Custom walk with ignore patterns:
//
//	err := filesystem.Walk(".", filesystem.WalkOptions{
//	    IgnoreDirs:     []string{".git", "tmp"},
//	    IgnorePatterns: []string{"*.tmp", "*.bak"},
//	}, func(path string, info os.FileInfo) error {
//	    // Process file
//	    return nil
//	})
package filesystem
