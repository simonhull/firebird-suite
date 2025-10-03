package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simonhull/firebird-suite/firebird/internal/generators/scaffold"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"gopkg.in/yaml.v3"
)

func TestGenerate_BasicScaffold(t *testing.T) {
	// Setup: Create a temporary project with firebird.yml
	tmpDir := setupTestProject(t, "postgres")
	defer os.RemoveAll(tmpDir)

	opts := scaffold.Options{
		Name: "Post",
		Fields: []scaffold.Field{
			{Name: "title", Type: "string"},
			{Name: "body", Type: "text"},
		},
		Timestamps: true,
	}

	gen := scaffold.NewGenerator()
	ops, err := gen.Generate(opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should have 1 operation (schema only, no --generate)
	if len(ops) != 1 {
		t.Errorf("expected 1 operation (schema only), got %d", len(ops))
	}

	// Verify operation describes schema file
	desc := ops[0].Description()
	if !strings.Contains(desc, "post.firebird.yml") {
		t.Errorf("unexpected description: %s", desc)
	}
}

func TestGenerate_WithIndexes(t *testing.T) {
	tmpDir := setupTestProject(t, "postgres")
	defer os.RemoveAll(tmpDir)

	opts := scaffold.Options{
		Name: "User",
		Fields: []scaffold.Field{
			{Name: "email", Type: "string", Modifier: "unique"},
			{Name: "name", Type: "string", Modifier: "index"},
		},
		Timestamps: true,
	}

	gen := scaffold.NewGenerator()
	ops, err := gen.Generate(opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(ops) != 1 {
		t.Errorf("expected 1 operation, got %d", len(ops))
	}
}

func TestGenerate_PostgreSQLTypes(t *testing.T) {
	tmpDir := setupTestProject(t, "postgres")
	defer os.RemoveAll(tmpDir)

	opts := scaffold.Options{
		Name: "Product",
		Fields: []scaffold.Field{
			{Name: "name", Type: "string"},
			{Name: "price", Type: "float64"},
			{Name: "available", Type: "bool"},
		},
	}

	gen := scaffold.NewGenerator()
	ops, err := gen.Generate(opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(ops) != 1 {
		t.Errorf("expected 1 operation, got %d", len(ops))
	}

	// Parse the generated schema to verify db_type values
	writeOp, ok := ops[0].(*generator.WriteFileOp)
	if !ok {
		t.Fatalf("expected WriteFileOp, got %T", ops[0])
	}

	var schema scaffold.Schema
	if err := yaml.Unmarshal(writeOp.Content, &schema); err != nil {
		t.Fatalf("failed to parse generated schema: %v", err)
	}

	// Check PostgreSQL-specific types
	expectedTypes := map[string]string{
		"id":        "BIGSERIAL",
		"name":      "VARCHAR(255)",
		"price":     "DOUBLE PRECISION",
		"available": "BOOLEAN",
	}

	for _, field := range schema.Spec.Fields {
		expected, ok := expectedTypes[field.Name]
		if !ok {
			continue
		}
		if field.DBType != expected {
			t.Errorf("field %s: expected db_type %s, got %s", field.Name, expected, field.DBType)
		}
	}
}

func TestGenerate_MySQLTypes(t *testing.T) {
	tmpDir := setupTestProject(t, "mysql")
	defer os.RemoveAll(tmpDir)

	opts := scaffold.Options{
		Name: "Article",
		Fields: []scaffold.Field{
			{Name: "title", Type: "string"},
			{Name: "count", Type: "int"},
		},
	}

	gen := scaffold.NewGenerator()
	ops, err := gen.Generate(opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Parse the generated schema
	writeOp := ops[0].(*generator.WriteFileOp)
	var schema scaffold.Schema
	if err := yaml.Unmarshal(writeOp.Content, &schema); err != nil {
		t.Fatalf("failed to parse generated schema: %v", err)
	}

	// Check MySQL-specific types
	expectedTypes := map[string]string{
		"id":    "BIGINT AUTO_INCREMENT",
		"title": "VARCHAR(255)",
		"count": "INT",
	}

	for _, field := range schema.Spec.Fields {
		expected, ok := expectedTypes[field.Name]
		if !ok {
			continue
		}
		if field.DBType != expected {
			t.Errorf("field %s: expected db_type %s, got %s", field.Name, expected, field.DBType)
		}
	}
}

func TestGenerate_SQLiteTypes(t *testing.T) {
	tmpDir := setupTestProject(t, "sqlite")
	defer os.RemoveAll(tmpDir)

	opts := scaffold.Options{
		Name: "Comment",
		Fields: []scaffold.Field{
			{Name: "content", Type: "text"},
			{Name: "rating", Type: "float64"},
		},
	}

	gen := scaffold.NewGenerator()
	ops, err := gen.Generate(opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Parse the generated schema
	writeOp := ops[0].(*generator.WriteFileOp)
	var schema scaffold.Schema
	if err := yaml.Unmarshal(writeOp.Content, &schema); err != nil {
		t.Fatalf("failed to parse generated schema: %v", err)
	}

	// Check SQLite-specific types
	expectedTypes := map[string]string{
		"id":      "INTEGER",
		"content": "TEXT",
		"rating":  "REAL",
	}

	for _, field := range schema.Spec.Fields {
		expected, ok := expectedTypes[field.Name]
		if !ok {
			continue
		}
		if field.DBType != expected {
			t.Errorf("field %s: expected db_type %s, got %s", field.Name, expected, field.DBType)
		}
	}
}

func TestGenerate_NoDatabaseConfigured(t *testing.T) {
	tmpDir := setupTestProject(t, "none")
	defer os.RemoveAll(tmpDir)

	opts := scaffold.Options{
		Name: "Post",
		Fields: []scaffold.Field{
			{Name: "title", Type: "string"},
		},
	}

	gen := scaffold.NewGenerator()
	_, err := gen.Generate(opts)
	if err == nil {
		t.Error("expected error when database is 'none', got nil")
	}

	if !strings.Contains(err.Error(), "no database configured") {
		t.Errorf("expected 'no database configured' error, got: %v", err)
	}
}

func TestGenerate_SoftDeletes(t *testing.T) {
	tmpDir := setupTestProject(t, "postgres")
	defer os.RemoveAll(tmpDir)

	opts := scaffold.Options{
		Name:        "Document",
		Fields:      []scaffold.Field{{Name: "title", Type: "string"}},
		Timestamps:  true,
		SoftDeletes: true,
	}

	gen := scaffold.NewGenerator()
	ops, err := gen.Generate(opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Parse the generated schema
	writeOp := ops[0].(*generator.WriteFileOp)
	var schema scaffold.Schema
	if err := yaml.Unmarshal(writeOp.Content, &schema); err != nil {
		t.Fatalf("failed to parse generated schema: %v", err)
	}

	if !schema.Spec.Timestamps {
		t.Error("expected timestamps: true")
	}

	if !schema.Spec.SoftDeletes {
		t.Error("expected soft_deletes: true")
	}
}

func TestGenerate_EmptyFields(t *testing.T) {
	tmpDir := setupTestProject(t, "postgres")
	defer os.RemoveAll(tmpDir)

	opts := scaffold.Options{
		Name:       "Event",
		Fields:     []scaffold.Field{}, // Empty fields
		Timestamps: true,
	}

	gen := scaffold.NewGenerator()
	ops, err := gen.Generate(opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Parse the generated schema
	writeOp := ops[0].(*generator.WriteFileOp)
	var schema scaffold.Schema
	if err := yaml.Unmarshal(writeOp.Content, &schema); err != nil {
		t.Fatalf("failed to parse generated schema: %v", err)
	}

	// Should have at least the id field
	if len(schema.Spec.Fields) != 1 {
		t.Errorf("expected 1 field (id only), got %d", len(schema.Spec.Fields))
	}

	if schema.Spec.Fields[0].Name != "id" {
		t.Errorf("expected first field to be 'id', got %s", schema.Spec.Fields[0].Name)
	}
}

// Helper functions

// setupTestProject creates a temporary project directory with firebird.yml
func setupTestProject(t *testing.T, driver string) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "scaffold-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(origDir)
	})

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create firebird.yml with specified driver
	config := createFirebirdConfig(driver)
	configPath := filepath.Join(tmpDir, "firebird.yml")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to create firebird.yml: %v", err)
	}

	// Create app/schemas directory
	schemasDir := filepath.Join(tmpDir, "app", "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}

	return tmpDir
}

func createFirebirdConfig(driver string) string {
	if driver == "none" {
		return `application:
  server:
    host: localhost
    port: 8080
`
	}

	return `application:
  server:
    host: localhost
    port: 8080
  database:
    driver: ` + driver + `
`
}

