package module

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test 1: Install with config fields
func TestInstaller_Install_WithConfig(t *testing.T) {
	projectPath := setupTestProject(t)
	installer := NewInstaller(projectPath, "github.com/test/project")

	// Create internal/config directory and config.go
	configDir := filepath.Join(projectPath, "internal", "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `package config

type Config struct {
	AppName string ` + "`yaml:\"app_name\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(configDir, "config.go"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config.go: %v", err)
	}

	// Install module with config fields
	opts := InstallOptions{
		ModuleName:    "falcon",
		ModuleVersion: "1.0.0",
		ModuleConfig: map[string]interface{}{
			"jwt_secret": "secret123",
		},
		ConfigFields: []ConfigField{
			{
				Name: "JWTSecret",
				Type: "string",
				Tag:  `yaml:"jwt_secret"`,
				Doc:  "JWT secret key",
			},
		},
	}

	if err := installer.Install(context.Background(), opts); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify config.go was updated
	updatedContent, err := os.ReadFile(filepath.Join(configDir, "config.go"))
	if err != nil {
		t.Fatalf("failed to read config.go: %v", err)
	}

	if !strings.Contains(string(updatedContent), "type ModulesConfig struct") {
		t.Error("ModulesConfig not added to config.go")
	}

	if !strings.Contains(string(updatedContent), "type falconConfig struct") {
		t.Error("falconConfig not added to config.go")
	}

	// Verify wiring file was created
	wiringPath := filepath.Join(projectPath, "internal", "modules", "wiring_falcon.go")
	if _, err := os.Stat(wiringPath); os.IsNotExist(err) {
		t.Fatal("wiring_falcon.go was not created")
	}

	// Verify orchestrator was created
	orchestratorPath := filepath.Join(projectPath, "internal", "modules", "wiring_modules.go")
	if _, err := os.Stat(orchestratorPath); os.IsNotExist(err) {
		t.Fatal("wiring_modules.go was not created")
	}

	// Verify firebird.yml was updated
	cfg, err := LoadFirebirdConfig(filepath.Join(projectPath, "firebird.yml"))
	if err != nil {
		t.Fatalf("failed to load firebird.yml: %v", err)
	}

	if cfg.Modules["falcon"].Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", cfg.Modules["falcon"].Version)
	}
}

// Test 2: Install without config fields
func TestInstaller_Install_WithoutConfig(t *testing.T) {
	projectPath := setupTestProject(t)
	installer := NewInstaller(projectPath, "github.com/test/project")

	// Install module without config fields
	opts := InstallOptions{
		ModuleName:    "owl",
		ModuleVersion: "0.5.0",
		ConfigFields:  nil, // No config fields
	}

	if err := installer.Install(context.Background(), opts); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify wiring file was created
	wiringPath := filepath.Join(projectPath, "internal", "modules", "wiring_owl.go")
	if _, err := os.Stat(wiringPath); os.IsNotExist(err) {
		t.Fatal("wiring_owl.go was not created")
	}

	// Verify firebird.yml was updated
	cfg, err := LoadFirebirdConfig(filepath.Join(projectPath, "firebird.yml"))
	if err != nil {
		t.Fatalf("failed to load firebird.yml: %v", err)
	}

	if cfg.Modules["owl"].Version != "0.5.0" {
		t.Errorf("Expected version 0.5.0, got %s", cfg.Modules["owl"].Version)
	}
}

// Test 3: Install multiple modules
func TestInstaller_Install_Multiple(t *testing.T) {
	projectPath := setupTestProject(t)
	installer := NewInstaller(projectPath, "github.com/test/project")

	// Install first module
	opts1 := InstallOptions{
		ModuleName:    "falcon",
		ModuleVersion: "1.0.0",
	}
	if err := installer.Install(context.Background(), opts1); err != nil {
		t.Fatalf("Install falcon failed: %v", err)
	}

	// Install second module
	opts2 := InstallOptions{
		ModuleName:    "owl",
		ModuleVersion: "0.5.0",
	}
	if err := installer.Install(context.Background(), opts2); err != nil {
		t.Fatalf("Install owl failed: %v", err)
	}

	// Verify both modules in orchestrator
	orchestratorContent, err := os.ReadFile(filepath.Join(projectPath, "internal", "modules", "wiring_modules.go"))
	if err != nil {
		t.Fatalf("failed to read orchestrator: %v", err)
	}

	if !strings.Contains(string(orchestratorContent), "InitFalcon") {
		t.Error("Orchestrator missing InitFalcon")
		t.Logf("Orchestrator:\n%s", string(orchestratorContent))
	}
	if !strings.Contains(string(orchestratorContent), "InitOwl") {
		t.Error("Orchestrator missing InitOwl")
		t.Logf("Orchestrator:\n%s", string(orchestratorContent))
	}

	// Verify both in firebird.yml
	cfg, err := LoadFirebirdConfig(filepath.Join(projectPath, "firebird.yml"))
	if err != nil {
		t.Fatalf("failed to load firebird.yml: %v", err)
	}

	if _, exists := cfg.Modules["falcon"]; !exists {
		t.Error("falcon not in firebird.yml")
	}
	if _, exists := cfg.Modules["owl"]; !exists {
		t.Error("owl not in firebird.yml")
	}
}

// Test 4: Uninstall module
func TestInstaller_Uninstall(t *testing.T) {
	projectPath := setupTestProject(t)
	installer := NewInstaller(projectPath, "github.com/test/project")

	// Install module first
	opts := InstallOptions{
		ModuleName:    "falcon",
		ModuleVersion: "1.0.0",
	}
	if err := installer.Install(context.Background(), opts); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify wiring file exists
	wiringPath := filepath.Join(projectPath, "internal", "modules", "wiring_falcon.go")
	if _, err := os.Stat(wiringPath); os.IsNotExist(err) {
		t.Fatal("wiring_falcon.go was not created")
	}

	// Uninstall module
	if err := installer.Uninstall(context.Background(), "falcon"); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify wiring file was deleted
	if _, err := os.Stat(wiringPath); !os.IsNotExist(err) {
		t.Error("wiring_falcon.go was not deleted")
	}

	// Verify removed from firebird.yml
	cfg, err := LoadFirebirdConfig(filepath.Join(projectPath, "firebird.yml"))
	if err != nil {
		t.Fatalf("failed to load firebird.yml: %v", err)
	}

	if _, exists := cfg.Modules["falcon"]; exists {
		t.Error("falcon still in firebird.yml after uninstall")
	}

	// Verify orchestrator was regenerated without module
	orchestratorContent, err := os.ReadFile(filepath.Join(projectPath, "internal", "modules", "wiring_modules.go"))
	if err != nil {
		t.Fatalf("failed to read orchestrator: %v", err)
	}

	if strings.Contains(string(orchestratorContent), "InitFalcon") {
		t.Error("Orchestrator still contains InitFalcon after uninstall")
	}
}

// Test 5: Uninstall one of multiple modules
func TestInstaller_Uninstall_Multiple(t *testing.T) {
	projectPath := setupTestProject(t)
	installer := NewInstaller(projectPath, "github.com/test/project")

	// Install two modules
	installer.Install(context.Background(), InstallOptions{
		ModuleName:    "falcon",
		ModuleVersion: "1.0.0",
	})
	installer.Install(context.Background(), InstallOptions{
		ModuleName:    "owl",
		ModuleVersion: "0.5.0",
	})

	// Uninstall one
	if err := installer.Uninstall(context.Background(), "falcon"); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify falcon wiring deleted
	falconPath := filepath.Join(projectPath, "internal", "modules", "wiring_falcon.go")
	if _, err := os.Stat(falconPath); !os.IsNotExist(err) {
		t.Error("wiring_falcon.go was not deleted")
	}

	// Verify owl wiring still exists
	owlPath := filepath.Join(projectPath, "internal", "modules", "wiring_owl.go")
	if _, err := os.Stat(owlPath); os.IsNotExist(err) {
		t.Error("wiring_owl.go was deleted (should still exist)")
	}

	// Verify orchestrator only has owl
	orchestratorContent, err := os.ReadFile(filepath.Join(projectPath, "internal", "modules", "wiring_modules.go"))
	if err != nil {
		t.Fatalf("failed to read orchestrator: %v", err)
	}

	if strings.Contains(string(orchestratorContent), "InitFalcon") {
		t.Error("Orchestrator still contains InitFalcon")
	}
	if !strings.Contains(string(orchestratorContent), "InitOwl") {
		t.Error("Orchestrator missing InitOwl")
	}
}

// Test 6: Install validation - missing module name
func TestInstaller_Install_MissingName(t *testing.T) {
	projectPath := setupTestProject(t)
	installer := NewInstaller(projectPath, "github.com/test/project")

	opts := InstallOptions{
		ModuleName:    "", // Missing
		ModuleVersion: "1.0.0",
	}

	err := installer.Install(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error for missing module name")
	}

	if !strings.Contains(err.Error(), "module name is required") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// Test 7: Install validation - missing version
func TestInstaller_Install_MissingVersion(t *testing.T) {
	projectPath := setupTestProject(t)
	installer := NewInstaller(projectPath, "github.com/test/project")

	opts := InstallOptions{
		ModuleName:    "falcon",
		ModuleVersion: "", // Missing
	}

	err := installer.Install(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error for missing module version")
	}

	if !strings.Contains(err.Error(), "module version is required") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// Test 8: Uninstall validation - missing module name
func TestInstaller_Uninstall_MissingName(t *testing.T) {
	projectPath := setupTestProject(t)
	installer := NewInstaller(projectPath, "github.com/test/project")

	err := installer.Uninstall(context.Background(), "")
	if err == nil {
		t.Fatal("Expected error for missing module name")
	}

	if !strings.Contains(err.Error(), "module name is required") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// Test 9: Registry is created on first install
func TestInstaller_Registry_CreatedOnce(t *testing.T) {
	projectPath := setupTestProject(t)
	installer := NewInstaller(projectPath, "github.com/test/project")

	// Install first module
	installer.Install(context.Background(), InstallOptions{
		ModuleName:    "falcon",
		ModuleVersion: "1.0.0",
	})

	// Verify registry was created
	registryPath := filepath.Join(projectPath, "internal", "modules", "registry.go")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Fatal("registry.go was not created on first install")
	}

	// Modify registry to test WriteFileIfNotExistsOp
	modifiedContent := "// MODIFIED\n"
	if err := os.WriteFile(registryPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("failed to modify registry: %v", err)
	}

	// Install second module
	installer.Install(context.Background(), InstallOptions{
		ModuleName:    "owl",
		ModuleVersion: "0.5.0",
	})

	// Verify registry was NOT overwritten
	content, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("failed to read registry: %v", err)
	}

	if !strings.Contains(string(content), "// MODIFIED") {
		t.Error("registry.go was overwritten (should be preserved)")
	}
}
