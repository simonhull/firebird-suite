package migration

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/generators/model"
	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/output"
	"gopkg.in/yaml.v3"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates SQL migrations from schemas
type Generator struct {
	renderer *generator.Renderer
}

// NewGenerator creates a new migration generator
func NewGenerator() *Generator {
	return &Generator{
		renderer: generator.NewRenderer(),
	}
}

// Generate generates a SQL migration for the given schema name
// Returns a slice of Operations to be executed by the caller
func (g *Generator) Generate(name string) ([]generator.Operation, error) {
	output.Verbose(fmt.Sprintf("Looking for schema file: %s", name))

	// 1. Find schema file (reuse from model generator)
	schemaPath, err := model.FindSchemaFile(name)
	if err != nil {
		return nil, err
	}
	output.Verbose(fmt.Sprintf("Found schema: %s", schemaPath))

	// 2. Parse schema
	def, err := schema.Parse(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}
	output.Verbose(fmt.Sprintf("Parsed schema for: %s", def.Name))

	return g.generateFromDefinition(name, def)
}

// GenerateFromSchema generates a SQL migration from an in-memory schema definition
// This is used by the scaffold generator with --generate flag
func (g *Generator) GenerateFromSchema(name string, def *schema.Definition) ([]generator.Operation, error) {
	output.Verbose(fmt.Sprintf("Generating migration from in-memory schema: %s", name))
	return g.generateFromDefinition(name, def)
}

// generateFromDefinition is the common implementation for both Generate methods
func (g *Generator) generateFromDefinition(name string, def *schema.Definition) ([]generator.Operation, error) {
	// 1. Detect database dialect
	dialect, err := DetectDatabaseDialect()
	if err != nil {
		return nil, err
	}
	output.Verbose(fmt.Sprintf("Detected database dialect: %s", dialect))

	// 2. Check if migration already exists and detect ALTER TABLE scenario
	migrationsDir := filepath.Join("db", "migrations")

	// Try to extract previous schema
	oldDef, err := extractLastSnapshot(migrationsDir, name)
	if err != nil {
		// If error is about missing snapshot, treat as first migration
		if !strings.Contains(err.Error(), "no schema snapshot found") {
			return nil, fmt.Errorf("failed to extract previous schema: %w", err)
		}
		output.Verbose(fmt.Sprintf("Previous migration exists but has no snapshot (older Firebird version)"))
		oldDef = nil
	}

	// If previous schema exists, generate ALTER TABLE migration
	if oldDef != nil {
		output.Verbose(fmt.Sprintf("Previous schema found - generating ALTER TABLE migration"))
		return g.generateAlterTable(name, oldDef, def, dialect, migrationsDir)
	}

	// Otherwise, generate CREATE TABLE migration
	output.Verbose(fmt.Sprintf("No previous schema found - generating CREATE TABLE migration"))

	// Double-check that CREATE TABLE migration doesn't already exist
	createMigrationName := "create_" + generator.SnakeCase(generator.Pluralize(name))
	exists, err := MigrationExists(migrationsDir, createMigrationName)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing migrations: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("migration for '%s' already exists. Migrations are immutable - create a new migration to modify the table", name)
	}

	// 3. Generate migration number
	number, err := GenerateMigrationNumber(TimestampNumbering, migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate migration number: %w", err)
	}
	output.Verbose(fmt.Sprintf("Generated migration number: %s", number))

	// 4. Get migration filenames
	upFile, downFile := GetMigrationFilenames(number, createMigrationName)
	upPath := filepath.Join(migrationsDir, upFile)
	downPath := filepath.Join(migrationsDir, downFile)

	// 5. Transform schema to migration data
	data := PrepareMigrationData(def, dialect)
	output.Verbose(fmt.Sprintf("Prepared migration for table: %s (%d columns)", data.TableName, len(data.Columns)))

	// Log any explicit db_type overrides
	for i, field := range def.Spec.Fields {
		if field.DBType != "" {
			output.Verbose(fmt.Sprintf("  Column '%s': using explicit db_type '%s' (Go type: %s)",
				data.Columns[i].Name, field.DBType, field.Type))
		}
	}

	// 6. Render up migration
	upTemplate := fmt.Sprintf("templates/%s.up.sql.tmpl", dialect)
	upContent, err := g.renderer.RenderFS(templatesFS, upTemplate, data)
	if err != nil {
		return nil, fmt.Errorf("failed to render up migration: %w", err)
	}

	// 6.1 Embed schema snapshot in UP migration
	snapshot, err := embedSchemaSnapshot(def)
	if err != nil {
		return nil, fmt.Errorf("failed to embed schema snapshot: %w", err)
	}
	upContent = append([]byte(snapshot), upContent...)

	// 7. Render down migration
	downTemplate := fmt.Sprintf("templates/%s.down.sql.tmpl", dialect)
	downContent, err := g.renderer.RenderFS(templatesFS, downTemplate, data)
	if err != nil {
		return nil, fmt.Errorf("failed to render down migration: %w", err)
	}

	// 8. Build operations
	var ops []generator.Operation
	ops = append(ops, &generator.WriteFileOp{
		Path:    upPath,
		Content: upContent,
		Mode:    0644,
	})
	ops = append(ops, &generator.WriteFileOp{
		Path:    downPath,
		Content: downContent,
		Mode:    0644,
	})

	output.Verbose(fmt.Sprintf("Prepared operations: %s, %s", upPath, downPath))

	return ops, nil
}

// embedSchemaSnapshot converts a schema definition to YAML and embeds it in SQL comments
// This allows migration diffing by extracting and comparing schemas from previous migrations
func embedSchemaSnapshot(def *schema.Definition) (string, error) {
	// Marshal schema to YAML
	yamlBytes, err := yaml.Marshal(def)
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema to YAML: %w", err)
	}

	// Build SQL comment block with markers
	var result string
	result += "-- FIREBIRD_SCHEMA_SNAPSHOT_BEGIN\n"

	// Wrap each line of YAML in SQL comments
	lines := splitLines(string(yamlBytes))
	for _, line := range lines {
		result += "-- " + line + "\n"
	}

	result += "-- FIREBIRD_SCHEMA_SNAPSHOT_END\n\n"

	return result, nil
}

// splitLines splits a string into lines, preserving empty lines
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}

	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}

	// Add last line if not ending with newline
	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}

// generateAlterTable generates an ALTER TABLE migration when schema changes are detected
func (g *Generator) generateAlterTable(name string, oldDef, newDef *schema.Definition, dialect DatabaseDialect, migrationsDir string) ([]generator.Operation, error) {
	// Diff the schemas to generate ALTER statements
	upSQL, downSQL, err := DiffSchemas(oldDef, newDef, dialect)
	if err != nil {
		return nil, err // Returns error if no changes detected
	}

	// Generate migration number
	number, err := GenerateMigrationNumber(TimestampNumbering, migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate migration number: %w", err)
	}
	output.Verbose(fmt.Sprintf("Generated migration number: %s", number))

	// Get migration filenames with "alter" prefix
	migrationName := "alter_" + generator.SnakeCase(generator.Pluralize(name))
	upFile, downFile := GetMigrationFilenames(number, migrationName)
	upPath := filepath.Join(migrationsDir, upFile)
	downPath := filepath.Join(migrationsDir, downFile)

	// Embed schema snapshot in UP migration
	snapshot, err := embedSchemaSnapshot(newDef)
	if err != nil {
		return nil, fmt.Errorf("failed to embed schema snapshot: %w", err)
	}
	upContent := []byte(snapshot + upSQL)
	downContent := []byte(downSQL)

	// Build operations
	var ops []generator.Operation
	ops = append(ops, &generator.WriteFileOp{
		Path:    upPath,
		Content: upContent,
		Mode:    0644,
	})
	ops = append(ops, &generator.WriteFileOp{
		Path:    downPath,
		Content: downContent,
		Mode:    0644,
	})

	output.Verbose(fmt.Sprintf("Prepared ALTER operations: %s, %s", upPath, downPath))

	return ops, nil
}
