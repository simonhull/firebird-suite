package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWalk_BasicTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test structure
	dirs := []string{"dir1", "dir2", "dir1/subdir"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	files := []string{"file1.txt", "dir1/file2.txt", "dir1/subdir/file3.txt"}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	var visited []string
	err := Walk(tmpDir, WalkOptions{IgnoreDirs: []string{}}, func(path string, info os.FileInfo) error {
		rel, _ := filepath.Rel(tmpDir, path)
		if rel != "." {
			visited = append(visited, rel)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	if len(visited) < 6 { // 3 dirs + 3 files
		t.Errorf("Walk() visited %d paths, want at least 6", len(visited))
	}
}

func TestWalk_IgnoreDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directories that should be ignored
	for _, dir := range []string{"node_modules", "vendor", ".git"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, dir, "test.txt"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a file that should not be ignored
	if err := os.WriteFile(filepath.Join(tmpDir, "keep.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	var visited []string
	err := WalkWithDefaults(tmpDir, func(path string, info os.FileInfo) error {
		rel, _ := filepath.Rel(tmpDir, path)
		visited = append(visited, rel)
		return nil
	})

	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	// Should visit tmpDir (.) and keep.txt only
	for _, v := range visited {
		if strings.Contains(v, "node_modules") || strings.Contains(v, "vendor") || strings.Contains(v, ".git") {
			t.Errorf("Walk() visited ignored directory: %s", v)
		}
	}
}

func TestWalk_CustomIgnores(t *testing.T) {
	tmpDir := t.TempDir()

	// Create custom ignored directory
	if err := os.MkdirAll(filepath.Join(tmpDir, "custom_ignore"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "custom_ignore", "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	var visited []string
	err := Walk(tmpDir, WalkOptions{
		IgnoreDirs: []string{"custom_ignore"},
	}, func(path string, info os.FileInfo) error {
		rel, _ := filepath.Rel(tmpDir, path)
		visited = append(visited, rel)
		return nil
	})

	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	for _, v := range visited {
		if strings.Contains(v, "custom_ignore") {
			t.Errorf("Walk() visited custom ignored directory: %s", v)
		}
	}
}

func TestWalk_IgnorePatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with different extensions
	files := []string{"keep.txt", "ignore.tmp", "also_ignore.bak"}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	var visited []string
	err := Walk(tmpDir, WalkOptions{
		IgnoreDirs:     []string{},
		IgnorePatterns: []string{"*.tmp", "*.bak"},
	}, func(path string, info os.FileInfo) error {
		if !info.IsDir() {
			visited = append(visited, info.Name())
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	for _, v := range visited {
		if v == "ignore.tmp" || v == "also_ignore.bak" {
			t.Errorf("Walk() visited ignored pattern: %s", v)
		}
	}

	found := false
	for _, v := range visited {
		if v == "keep.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Walk() did not visit keep.txt")
	}
}

func TestWalk_HiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hidden file and directory
	if err := os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".hiddendir"), 0755); err != nil {
		t.Fatal(err)
	}

	// Default: should skip hidden
	var visitedDefault []string
	err := WalkWithDefaults(tmpDir, func(path string, info os.FileInfo) error {
		visitedDefault = append(visitedDefault, info.Name())
		return nil
	})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	for _, v := range visitedDefault {
		if strings.HasPrefix(v, ".") && v != filepath.Base(tmpDir) {
			t.Errorf("Walk() visited hidden file without IncludeHidden: %s", v)
		}
	}

	// With IncludeHidden: should visit hidden
	var visitedInclude []string
	err = Walk(tmpDir, WalkOptions{
		IncludeHidden: true,
		IgnoreDirs:    []string{},
	}, func(path string, info os.FileInfo) error {
		visitedInclude = append(visitedInclude, info.Name())
		return nil
	})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	foundHidden := false
	for _, v := range visitedInclude {
		if v == ".hidden" {
			foundHidden = true
			break
		}
	}
	if !foundHidden {
		t.Error("Walk() did not visit hidden file with IncludeHidden=true")
	}
}

func TestWalkWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	visited := false
	err := WalkWithDefaults(tmpDir, func(path string, info os.FileInfo) error {
		if info.Name() == "test.txt" {
			visited = true
		}
		return nil
	})

	if err != nil {
		t.Fatalf("WalkWithDefaults() error = %v", err)
	}

	if !visited {
		t.Error("WalkWithDefaults() did not visit test.txt")
	}
}

func TestWalk_SkipDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "skip/nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "skip/nested/file.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	var visited []string
	err := Walk(tmpDir, WalkOptions{IgnoreDirs: []string{}}, func(path string, info os.FileInfo) error {
		visited = append(visited, info.Name())
		if info.Name() == "skip" {
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	for _, v := range visited {
		if v == "nested" || v == "file.txt" {
			t.Errorf("Walk() visited path inside skipped directory: %s", v)
		}
	}
}
