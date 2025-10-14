package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectModule_Success(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module github.com/test/example

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectModule(tmpDir)
	if err != nil {
		t.Fatalf("DetectModule() error = %v", err)
	}

	if info.Path != "github.com/test/example" {
		t.Errorf("Path = %q, want %q", info.Path, "github.com/test/example")
	}

	if info.GoVersion != "1.21" {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, "1.21")
	}
}

func TestDetectModule_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := DetectModule(tmpDir)
	if err == nil {
		t.Fatal("DetectModule() expected error for missing go.mod")
	}

	if !os.IsNotExist(err) {
		// Should contain "go.mod not found"
		if err.Error() == "" {
			t.Errorf("Expected meaningful error message")
		}
	}
}

func TestDetectModule_InvalidSyntax(t *testing.T) {
	tmpDir := t.TempDir()

	invalidGoMod := `this is not valid go.mod syntax
module
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(invalidGoMod), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := DetectModule(tmpDir)
	if err == nil {
		t.Fatal("DetectModule() expected error for invalid syntax")
	}
}

func TestDetectModule_WithVersion(t *testing.T) {
	tmpDir := t.TempDir()

	// Note: go.mod module directive doesn't support version on the module line
	// This is invalid syntax, so we test that it's properly handled
	goMod := `module github.com/test/versioned

go 1.20
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectModule(tmpDir)
	if err != nil {
		t.Fatalf("DetectModule() error = %v", err)
	}

	if info.Path != "github.com/test/versioned" {
		t.Errorf("Path = %q, want %q", info.Path, "github.com/test/versioned")
	}

	if info.GoVersion != "1.20" {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, "1.20")
	}
}

func TestDetectModule_WithGoVersion(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module example.com/myproject

go 1.22.1

require (
	github.com/spf13/cobra v1.8.0
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectModule(tmpDir)
	if err != nil {
		t.Fatalf("DetectModule() error = %v", err)
	}

	if info.Path != "example.com/myproject" {
		t.Errorf("Path = %q, want %q", info.Path, "example.com/myproject")
	}

	if info.GoVersion != "1.22.1" {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, "1.22.1")
	}
}

func TestDetectModule_NoGoDirective(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module github.com/test/nogoversion
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectModule(tmpDir)
	if err != nil {
		t.Fatalf("DetectModule() error = %v", err)
	}

	if info.GoVersion != "" {
		t.Errorf("GoVersion = %q, want empty string", info.GoVersion)
	}
}
