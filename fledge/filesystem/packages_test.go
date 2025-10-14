package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverGoPackages_SinglePackage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create single package
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	packages, err := DiscoverGoPackages(tmpDir, PackageDiscoveryOptions{})
	if err != nil {
		t.Fatalf("DiscoverGoPackages() error = %v", err)
	}

	if len(packages) != 1 {
		t.Errorf("DiscoverGoPackages() found %d packages, want 1", len(packages))
	}

	if len(packages) > 0 && packages[0] != tmpDir {
		t.Errorf("DiscoverGoPackages() = %v, want %v", packages[0], tmpDir)
	}
}

func TestDiscoverGoPackages_MultiplePackages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple packages
	dirs := []string{"pkg1", "pkg2", "pkg1/subpkg"}
	for _, dir := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dirPath, "file.go"), []byte("package "+filepath.Base(dir)), 0644); err != nil {
			t.Fatal(err)
		}
	}

	packages, err := DiscoverGoPackages(tmpDir, PackageDiscoveryOptions{})
	if err != nil {
		t.Fatalf("DiscoverGoPackages() error = %v", err)
	}

	if len(packages) != 3 {
		t.Errorf("DiscoverGoPackages() found %d packages, want 3", len(packages))
	}
}

func TestDiscoverGoPackages_SkipTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package with only test files
	if err := os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	packages, err := DiscoverGoPackages(tmpDir, PackageDiscoveryOptions{
		IncludeTests: false,
	})
	if err != nil {
		t.Fatalf("DiscoverGoPackages() error = %v", err)
	}

	if len(packages) != 0 {
		t.Errorf("DiscoverGoPackages() found %d packages, want 0 (only test files)", len(packages))
	}
}

func TestDiscoverGoPackages_IncludeTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package with only test files
	if err := os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	packages, err := DiscoverGoPackages(tmpDir, PackageDiscoveryOptions{
		IncludeTests: true,
	})
	if err != nil {
		t.Fatalf("DiscoverGoPackages() error = %v", err)
	}

	if len(packages) != 1 {
		t.Errorf("DiscoverGoPackages() found %d packages, want 1", len(packages))
	}
}

func TestDiscoverGoPackages_SkipVendor(t *testing.T) {
	tmpDir := t.TempDir()

	// Create vendor directory with Go files
	vendorDir := filepath.Join(tmpDir, "vendor", "example.com", "pkg")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "vendor.go"), []byte("package pkg"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create non-vendor package
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	packages, err := DiscoverGoPackages(tmpDir, PackageDiscoveryOptions{
		IncludeVendor: false,
	})
	if err != nil {
		t.Fatalf("DiscoverGoPackages() error = %v", err)
	}

	// Should find only the non-vendor package
	if len(packages) != 1 {
		t.Errorf("DiscoverGoPackages() found %d packages, want 1", len(packages))
	}

	if len(packages) > 0 && packages[0] != tmpDir {
		t.Errorf("DiscoverGoPackages() = %v, want %v", packages[0], tmpDir)
	}
}

func TestDiscoverGoPackages_IncludeVendor(t *testing.T) {
	tmpDir := t.TempDir()

	// Create vendor directory with Go files
	// Note: vendor is still in DefaultIgnoreDirs, so it will be skipped
	// IncludeVendor removes "vendor" from the ignore list during discovery
	vendorDir := filepath.Join(tmpDir, "vendor", "example.com", "pkg")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "vendor.go"), []byte("package pkg"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create non-vendor package
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	packages, err := DiscoverGoPackages(tmpDir, PackageDiscoveryOptions{
		IncludeVendor: true,
	})
	if err != nil {
		t.Fatalf("DiscoverGoPackages() error = %v", err)
	}

	// Should find at least the main package and vendor package
	// Note: Implementation correctly filters vendor unless IncludeVendor is true
	// The vendor package should be included when IncludeVendor is true
	foundMain := false
	foundVendor := false
	for _, pkg := range packages {
		if pkg == tmpDir {
			foundMain = true
		}
		if filepath.Base(pkg) == "pkg" {
			foundVendor = true
		}
	}

	if !foundMain {
		t.Error("DiscoverGoPackages() did not find main package")
	}

	if !foundVendor {
		t.Error("DiscoverGoPackages() did not find vendor package with IncludeVendor=true")
	}
}

func TestDiscoverGoPackages_ExcludePaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create packages in different directories
	for _, dir := range []string{"keep", "exclude/subdir"} {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dirPath, "file.go"), []byte("package test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	packages, err := DiscoverGoPackages(tmpDir, PackageDiscoveryOptions{
		ExcludePaths: []string{"exclude"},
	})
	if err != nil {
		t.Fatalf("DiscoverGoPackages() error = %v", err)
	}

	// Should only find the "keep" package
	if len(packages) != 1 {
		t.Errorf("DiscoverGoPackages() found %d packages, want 1", len(packages))
	}

	for _, pkg := range packages {
		if filepath.Base(pkg) == "exclude" || filepath.Base(filepath.Dir(pkg)) == "exclude" {
			t.Errorf("DiscoverGoPackages() found excluded package: %s", pkg)
		}
	}
}

func TestDiscoverGoPackages_NoGoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory with no Go files
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	packages, err := DiscoverGoPackages(tmpDir, PackageDiscoveryOptions{})
	if err != nil {
		t.Fatalf("DiscoverGoPackages() error = %v", err)
	}

	if len(packages) != 0 {
		t.Errorf("DiscoverGoPackages() found %d packages, want 0", len(packages))
	}
}

func TestDiscoverGoPackages_Sorted(t *testing.T) {
	tmpDir := t.TempDir()

	// Create packages in random order (alphabetically)
	dirs := []string{"zebra", "alpha", "middle"}
	for _, dir := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dirPath, "file.go"), []byte("package "+dir), 0644); err != nil {
			t.Fatal(err)
		}
	}

	packages, err := DiscoverGoPackages(tmpDir, PackageDiscoveryOptions{})
	if err != nil {
		t.Fatalf("DiscoverGoPackages() error = %v", err)
	}

	if len(packages) != 3 {
		t.Errorf("DiscoverGoPackages() found %d packages, want 3", len(packages))
	}

	// Verify sorted order
	for i := 1; i < len(packages); i++ {
		if packages[i-1] > packages[i] {
			t.Errorf("DiscoverGoPackages() not sorted: %s > %s", packages[i-1], packages[i])
		}
	}
}
