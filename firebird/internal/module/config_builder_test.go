package module

import (
	"context"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test helper functions

// setupTestConfigFile creates a temporary config.go file for testing
func setupTestConfigFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.go")

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	return configPath
}

// readFile reads file content as string
func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	return string(content)
}

// Test 1: EnsureModulesField - First Time
func TestConfigBuilder_EnsureModulesField_FirstTime(t *testing.T) {
	// Given: A config file without Modules field
	initialConfig := `package config

type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
}

type DatabaseConfig struct {
	Host string
}

type ServerConfig struct {
	Port int
}
`

	configPath := setupTestConfigFile(t, initialConfig)

	// When: EnsureModulesField is called
	builder := NewConfigBuilder(configPath)
	err := builder.EnsureModulesField()
	if err != nil {
		t.Fatalf("EnsureModulesField failed: %v", err)
	}

	ops, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Execute operations
	for _, op := range ops {
		if err := op.Execute(context.Background()); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
	}

	// Then: File should have Modules field and ModulesConfig type
	result := readFile(t, configPath)

	if !strings.Contains(result, "Modules") {
		t.Error("Config struct missing Modules field")
	}

	if !strings.Contains(result, "ModulesConfig") {
		t.Error("ModulesConfig type not created")
	}

	// Verify it's valid Go
	if _, err := parser.ParseFile(token.NewFileSet(), configPath, result, 0); err != nil {
		t.Errorf("Result is not valid Go code: %v", err)
	}
}

// Test 2: EnsureModulesField - Idempotent
func TestConfigBuilder_EnsureModulesField_Idempotent(t *testing.T) {
	// Given: A config file that already has Modules field
	initialConfig := `package config

type Config struct {
	Database DatabaseConfig
	Modules  ModulesConfig
}

type ModulesConfig struct {
}
`

	configPath := setupTestConfigFile(t, initialConfig)

	// When: EnsureModulesField is called twice
	builder := NewConfigBuilder(configPath)

	err := builder.EnsureModulesField()
	if err != nil {
		t.Fatalf("First EnsureModulesField failed: %v", err)
	}

	err = builder.EnsureModulesField()
	if err != nil {
		t.Fatalf("Second EnsureModulesField failed: %v", err)
	}

	ops, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Then: No operations should be generated
	if len(ops) != 0 {
		t.Errorf("Expected no operations (idempotent), got %d", len(ops))
	}
}

// Test 3: AddModuleConfig - First Module
func TestConfigBuilder_AddModuleConfig_FirstModule(t *testing.T) {
	// Given: Config file with ModulesConfig but no module configs
	initialConfig := `package config

type Config struct {
	Database DatabaseConfig
	Modules  ModulesConfig
}

type ModulesConfig struct {
}
`

	configPath := setupTestConfigFile(t, initialConfig)

	// When: AddModuleConfig is called for Falcon
	builder := NewConfigBuilder(configPath)
	err := builder.AddModuleConfig("Falcon", []ConfigField{
		{Name: "JWTSecret", Type: "string", Tag: `yaml:"jwt_secret"`},
		{Name: "TokenExpiry", Type: "time.Duration", Tag: `yaml:"token_expiry"`},
		{Name: "BCryptCost", Type: "int", Tag: `yaml:"bcrypt_cost"`},
	})
	if err != nil {
		t.Fatalf("AddModuleConfig failed: %v", err)
	}

	ops, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	for _, op := range ops {
		if err := op.Execute(context.Background()); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
	}

	// Then: Should have Falcon field in ModulesConfig and FalconConfig type
	result := readFile(t, configPath)

	requiredStrings := []string{
		"Falcon",
		"FalconConfig",
		"JWTSecret",
		"string",
		"time.Duration",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(result, required) {
			t.Errorf("Result missing required string: %q", required)
		}
	}

	// Verify time import was added
	if !strings.Contains(result, `"time"`) && !strings.Contains(result, "import \"time\"") {
		t.Error("time import not added for time.Duration")
	}

	// Verify it's valid Go
	if _, err := parser.ParseFile(token.NewFileSet(), configPath, result, 0); err != nil {
		t.Errorf("Result is not valid Go code: %v", err)
	}
}

// Test 4: AddModuleConfig - Multiple Modules
func TestConfigBuilder_AddModuleConfig_MultipleModules(t *testing.T) {
	// Given: Config file with one module already configured
	initialConfig := `package config

import "time"

type Config struct {
	Database DatabaseConfig
	Modules  ModulesConfig
}

type ModulesConfig struct {
	Falcon FalconConfig
}

type FalconConfig struct {
	JWTSecret string
}
`

	configPath := setupTestConfigFile(t, initialConfig)

	// When: AddModuleConfig is called for Owl (second module)
	builder := NewConfigBuilder(configPath)
	err := builder.AddModuleConfig("Owl", []ConfigField{
		{Name: "SwaggerPath", Type: "string", Tag: `yaml:"swagger_path"`},
		{Name: "Theme", Type: "string", Tag: `yaml:"theme"`},
	})
	if err != nil {
		t.Fatalf("AddModuleConfig failed: %v", err)
	}

	ops, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	for _, op := range ops {
		if err := op.Execute(context.Background()); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
	}

	// Then: Should have both Falcon and Owl configs
	result := readFile(t, configPath)

	requiredStrings := []string{
		"Falcon",
		"FalconConfig",
		"Owl",
		"OwlConfig",
		"SwaggerPath",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(result, required) {
			t.Errorf("Result missing required string: %q", required)
		}
	}

	// Verify it's valid Go
	if _, err := parser.ParseFile(token.NewFileSet(), configPath, result, 0); err != nil {
		t.Errorf("Result is not valid Go code: %v", err)
	}
}

// Test 5: AddModuleConfig - Idempotent
func TestConfigBuilder_AddModuleConfig_Idempotent(t *testing.T) {
	// Given: Config file with Falcon already configured
	initialConfig := `package config

type Config struct {
	Modules ModulesConfig
}

type ModulesConfig struct {
	Falcon FalconConfig
}

type FalconConfig struct {
	JWTSecret string
}
`

	configPath := setupTestConfigFile(t, initialConfig)

	// When: AddModuleConfig is called for Falcon again
	builder := NewConfigBuilder(configPath)
	err := builder.AddModuleConfig("Falcon", []ConfigField{
		{Name: "JWTSecret", Type: "string", Tag: `yaml:"jwt_secret"`},
	})
	if err != nil {
		t.Fatalf("AddModuleConfig failed: %v", err)
	}

	ops, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Then: No operations should be generated (idempotent)
	if len(ops) != 0 {
		t.Errorf("Expected no operations (idempotent), got %d", len(ops))
	}
}

// Test 6: Full Workflow
func TestConfigBuilder_FullWorkflow(t *testing.T) {
	// Given: Fresh config file with just Config struct
	initialConfig := `package config

type Config struct {
	Database DatabaseConfig
}

type DatabaseConfig struct {
	Host string
}
`

	configPath := setupTestConfigFile(t, initialConfig)

	// When: Full workflow - add Modules field + two modules
	builder := NewConfigBuilder(configPath)

	// Step 1: Ensure Modules field
	if err := builder.EnsureModulesField(); err != nil {
		t.Fatalf("EnsureModulesField failed: %v", err)
	}

	// Step 2: Add Falcon
	if err := builder.AddModuleConfig("Falcon", []ConfigField{
		{Name: "JWTSecret", Type: "string", Tag: `yaml:"jwt_secret"`},
		{Name: "TokenExpiry", Type: "time.Duration", Tag: `yaml:"token_expiry"`},
	}); err != nil {
		t.Fatalf("AddModuleConfig(Falcon) failed: %v", err)
	}

	// Step 3: Add Owl
	if err := builder.AddModuleConfig("Owl", []ConfigField{
		{Name: "DocsPath", Type: "string", Tag: `yaml:"docs_path"`},
	}); err != nil {
		t.Fatalf("AddModuleConfig(Owl) failed: %v", err)
	}

	// Execute
	ops, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	for _, op := range ops {
		if err := op.Execute(context.Background()); err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
	}

	// Then: Verify complete structure
	result := readFile(t, configPath)

	// Check structure
	requiredStrings := []string{
		"Modules",
		"ModulesConfig",
		"Falcon",
		"FalconConfig",
		"Owl",
		"OwlConfig",
		"JWTSecret",
		"TokenExpiry",
		"DocsPath",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(result, required) {
			t.Errorf("Result missing required string: %q", required)
		}
	}

	// Verify it's valid Go
	if _, err := parser.ParseFile(token.NewFileSet(), configPath, result, 0); err != nil {
		t.Errorf("Result is not valid Go code: %v", err)
	}
}

// Test 7: toSnakeCase Helper Function
func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Falcon", "falcon"},
		{"FalconAuth", "falcon_auth"},
		{"APIKey", "a_p_i_key"},
		{"HTTPServer", "h_t_t_p_server"},
		{"Owl", "owl"},
		{"OwlDocs", "owl_docs"},
		{"JWT", "j_w_t"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
