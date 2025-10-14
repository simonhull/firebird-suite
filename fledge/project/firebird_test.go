package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFirebirdProject_True(t *testing.T) {
	tmpDir := t.TempDir()

	firebirdYML := `application:
  database:
    driver: postgres
  router:
    type: chi
`
	if err := os.WriteFile(filepath.Join(tmpDir, "firebird.yml"), []byte(firebirdYML), 0644); err != nil {
		t.Fatal(err)
	}

	if !IsFirebirdProject(tmpDir) {
		t.Error("IsFirebirdProject() = false, want true")
	}
}

func TestIsFirebirdProject_False(t *testing.T) {
	tmpDir := t.TempDir()

	if IsFirebirdProject(tmpDir) {
		t.Error("IsFirebirdProject() = true, want false")
	}
}

func TestDetectFirebirdProject_Found(t *testing.T) {
	tmpDir := t.TempDir()

	firebirdYML := `application:
  database:
    driver: postgres
  router:
    type: stdlib
`
	if err := os.WriteFile(filepath.Join(tmpDir, "firebird.yml"), []byte(firebirdYML), 0644); err != nil {
		t.Fatal(err)
	}

	found, config, err := DetectFirebirdProject(tmpDir)
	if err != nil {
		t.Fatalf("DetectFirebirdProject() error = %v", err)
	}

	if !found {
		t.Error("found = false, want true")
	}

	if config == nil {
		t.Fatal("config = nil, want non-nil")
	}

	if config.Database != "postgres" {
		t.Errorf("Database = %q, want %q", config.Database, "postgres")
	}

	if config.Router != "stdlib" {
		t.Errorf("Router = %q, want %q", config.Router, "stdlib")
	}
}

func TestDetectFirebirdProject_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	found, config, err := DetectFirebirdProject(tmpDir)
	if err != nil {
		t.Fatalf("DetectFirebirdProject() error = %v", err)
	}

	if found {
		t.Error("found = true, want false")
	}

	if config != nil {
		t.Error("config should be nil when not found")
	}
}

func TestDetectFirebirdProject_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	invalidYML := `this is not: valid: yaml: content
	bad indentation
  wrong structure
`
	if err := os.WriteFile(filepath.Join(tmpDir, "firebird.yml"), []byte(invalidYML), 0644); err != nil {
		t.Fatal(err)
	}

	found, _, err := DetectFirebirdProject(tmpDir)
	if err == nil {
		t.Fatal("DetectFirebirdProject() expected error for invalid YAML")
	}

	if found {
		t.Error("found should be false for invalid YAML")
	}
}

func TestLoadFirebirdConfig_Success(t *testing.T) {
	tmpDir := t.TempDir()

	firebirdYML := `application:
  database:
    driver: mysql
  router:
    type: chi
`
	configPath := filepath.Join(tmpDir, "firebird.yml")
	if err := os.WriteFile(configPath, []byte(firebirdYML), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadFirebirdConfig(configPath)
	if err != nil {
		t.Fatalf("LoadFirebirdConfig() error = %v", err)
	}

	if config.Database != "mysql" {
		t.Errorf("Database = %q, want %q", config.Database, "mysql")
	}

	if config.Router != "chi" {
		t.Errorf("Router = %q, want %q", config.Router, "chi")
	}
}

func TestDetectFirebirdProject_PostgreSQL(t *testing.T) {
	tmpDir := t.TempDir()

	firebirdYML := `application:
  database:
    driver: postgres
  router:
    type: stdlib
`
	if err := os.WriteFile(filepath.Join(tmpDir, "firebird.yml"), []byte(firebirdYML), 0644); err != nil {
		t.Fatal(err)
	}

	_, config, err := DetectFirebirdProject(tmpDir)
	if err != nil {
		t.Fatalf("DetectFirebirdProject() error = %v", err)
	}

	if config.Database != "postgres" {
		t.Errorf("Database = %q, want %q", config.Database, "postgres")
	}
}

func TestDetectFirebirdProject_MySQL(t *testing.T) {
	tmpDir := t.TempDir()

	firebirdYML := `application:
  database:
    driver: mysql
  router:
    type: gin
`
	if err := os.WriteFile(filepath.Join(tmpDir, "firebird.yml"), []byte(firebirdYML), 0644); err != nil {
		t.Fatal(err)
	}

	_, config, err := DetectFirebirdProject(tmpDir)
	if err != nil {
		t.Fatalf("DetectFirebirdProject() error = %v", err)
	}

	if config.Database != "mysql" {
		t.Errorf("Database = %q, want %q", config.Database, "mysql")
	}

	if config.Router != "gin" {
		t.Errorf("Router = %q, want %q", config.Router, "gin")
	}
}

func TestDetectFirebirdProject_SQLite(t *testing.T) {
	tmpDir := t.TempDir()

	firebirdYML := `application:
  database:
    driver: sqlite
  router:
    type: echo
`
	if err := os.WriteFile(filepath.Join(tmpDir, "firebird.yml"), []byte(firebirdYML), 0644); err != nil {
		t.Fatal(err)
	}

	_, config, err := DetectFirebirdProject(tmpDir)
	if err != nil {
		t.Fatalf("DetectFirebirdProject() error = %v", err)
	}

	if config.Database != "sqlite" {
		t.Errorf("Database = %q, want %q", config.Database, "sqlite")
	}
}

func TestDetectFirebirdProject_NoDatabase(t *testing.T) {
	tmpDir := t.TempDir()

	firebirdYML := `application:
  database:
    driver: none
  router:
    type: stdlib
`
	if err := os.WriteFile(filepath.Join(tmpDir, "firebird.yml"), []byte(firebirdYML), 0644); err != nil {
		t.Fatal(err)
	}

	_, config, err := DetectFirebirdProject(tmpDir)
	if err != nil {
		t.Fatalf("DetectFirebirdProject() error = %v", err)
	}

	if config.Database != "none" {
		t.Errorf("Database = %q, want %q", config.Database, "none")
	}
}
