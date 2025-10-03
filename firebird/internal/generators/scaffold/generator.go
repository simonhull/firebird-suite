package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/generators/migration"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/model"
	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"github.com/simonhull/firebird-suite/firebird/internal/types"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/output"
	"gopkg.in/yaml.v3"
)

// Options represents the configuration for scaffold generation
type Options struct {
	Name        string  // Model name (e.g., "Post")
	Fields      []Field // Parsed fields
	Timestamps  bool    // --timestamps flag
	SoftDeletes bool    // --soft-deletes flag
	Generate    bool    // --generate flag (orchestrate model + migration)
	IntID       bool    // Use int64 instead of UUID for primary key
}

// Field represents a single field specification from the command line
type Field struct {
	Name     string // "title"
	Type     string // "string"
	Modifier string // "index", "unique", or ""
}

// Generator generates schema files from command-line field specifications
type Generator struct{}

// NewGenerator creates a new scaffold generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate creates operations for scaffolding a new resource
func (g *Generator) Generate(opts Options) ([]generator.Operation, error) {
	var ops []generator.Operation

	output.Verbose(fmt.Sprintf("Scaffolding resource: %s", opts.Name))

	// 1. Read database driver from firebird.yml
	driver, err := readDatabaseDriver()
	if err != nil {
		return nil, fmt.Errorf("failed to read database config: %w", err)
	}
	output.Verbose(fmt.Sprintf("Database driver: %s", driver))

	// 2. Build schema content
	schemaContent, err := buildSchema(opts, driver)
	if err != nil {
		return nil, fmt.Errorf("failed to build schema: %w", err)
	}

	// 3. Create schema file operation
	schemaDir := "app/schemas"
	schemaFileName := strings.ToLower(opts.Name) + ".firebird.yml"
	schemaPath := filepath.Join(schemaDir, schemaFileName)

	output.Verbose(fmt.Sprintf("Schema path: %s", schemaPath))

	ops = append(ops, &generator.WriteFileOp{
		Path:    schemaPath,
		Content: schemaContent,
		Mode:    0644,
	})

	// 4. If --generate flag, orchestrate model + migration generators
	if opts.Generate {
		output.Verbose("Orchestrating model and migration generation")

		// Parse the schema we just built to pass to generators
		var schemaData Schema
		if err := yaml.Unmarshal(schemaContent, &schemaData); err != nil {
			return nil, fmt.Errorf("failed to parse generated schema: %w", err)
		}

		// Convert to schema.Definition for generators
		schemaDef := convertToDefinition(schemaData)

		// Generate model using in-memory schema
		modelGen := model.NewGenerator()
		modelOps, err := modelGen.GenerateFromSchema(opts.Name, schemaDef)
		if err != nil {
			return nil, fmt.Errorf("model generation failed: %w", err)
		}
		output.Verbose(fmt.Sprintf("Model operations: %d", len(modelOps)))
		ops = append(ops, modelOps...)

		// Generate migration using in-memory schema
		migrationGen := migration.NewGenerator()
		migrationOps, err := migrationGen.GenerateFromSchema(opts.Name, schemaDef)
		if err != nil {
			return nil, fmt.Errorf("migration generation failed: %w", err)
		}
		output.Verbose(fmt.Sprintf("Migration operations: %d", len(migrationOps)))
		ops = append(ops, migrationOps...)
	}

	return ops, nil
}

// convertToDefinition converts our Schema type to schema.Definition
func convertToDefinition(s Schema) *schema.Definition {
	fields := make([]schema.Field, len(s.Spec.Fields))
	for i, f := range s.Spec.Fields {
		fields[i] = schema.Field{
			Name:       f.Name,
			Type:       f.Type,
			DBType:     f.DBType,
			PrimaryKey: f.PrimaryKey,
			Nullable:   f.Nullable,
			Index:      f.Index,
			Unique:     f.Unique,
		}
	}

	return &schema.Definition{
		APIVersion: s.APIVersion,
		Kind:       s.Kind,
		Name:       s.Name,
		Spec: schema.Spec{
			TableName: s.Spec.TableName,
			Fields:    fields,
		},
	}
}

// readDatabaseDriver reads the database driver from firebird.yml
func readDatabaseDriver() (string, error) {
	data, err := os.ReadFile("firebird.yml")
	if err != nil {
		return "", fmt.Errorf("not in a Firebird project (no firebird.yml found): %w", err)
	}

	var config struct {
		Application struct {
			Database struct {
				Driver string `yaml:"driver"`
			} `yaml:"database"`
		} `yaml:"application"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("invalid firebird.yml: %w", err)
	}

	driver := config.Application.Database.Driver
	if driver == "" || driver == "none" {
		return "", fmt.Errorf("no database configured (driver: %s)", driver)
	}

	return driver, nil
}

// buildSchema generates the YAML schema content
func buildSchema(opts Options, driver string) ([]byte, error) {
	// Convert model name to table name (plural)
	tableName := generator.Pluralize(opts.Name)

	schema := Schema{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       opts.Name,
		Spec: Spec{
			TableName: tableName,
			Fields:    buildFields(opts.Fields, driver, opts.IntID),
		},
	}

	// Add timestamps metadata if requested
	if opts.Timestamps {
		schema.Spec.Timestamps = true
	}

	// Add soft deletes metadata if requested
	if opts.SoftDeletes {
		schema.Spec.SoftDeletes = true
	}

	return yaml.Marshal(schema)
}

// buildFields converts parsed fields to schema fields with db_type
func buildFields(fields []Field, driver string, useIntID bool) []SchemaField {
	// Determine primary key type
	idType := "UUID" // Default to UUID
	if useIntID {
		idType = "int64"
	}

	// Get primary key DB type using type registry
	pkDBType, err := types.GetPrimaryKeyType(idType, driver)
	if err != nil {
		// Fallback (should not happen with valid ID types)
		output.Verbose(fmt.Sprintf("Warning: %v, using BIGINT as fallback", err))
		pkDBType = "BIGINT"
	}

	result := []SchemaField{
		// Always include ID field
		{
			Name:       "id",
			Type:       idType,
			DBType:     pkDBType,
			PrimaryKey: true,
		},
	}

	for _, field := range fields {
		// Use type registry for DB type lookup
		dbType, err := types.GetDBType(field.Type, driver)
		if err != nil {
			// Fallback for unknown types
			output.Verbose(fmt.Sprintf("Warning: %v, using TEXT as fallback", err))
			dbType = "TEXT"
		}

		schemaField := SchemaField{
			Name:     field.Name,
			Type:     field.Type,
			DBType:   dbType,
			Nullable: isNullableByDefault(field.Type),
		}

		// Handle modifiers
		switch field.Modifier {
		case "index":
			schemaField.Index = true
		case "unique":
			schemaField.Unique = true
		}

		result = append(result, schemaField)
	}

	return result
}

// isNullableByDefault determines if a field type should be nullable
func isNullableByDefault(fieldType string) bool {
	// Timestamps are often nullable (optional)
	return fieldType == "timestamp" || fieldType == "date" || fieldType == "time"
}

// Schema types for YAML marshaling
type Schema struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Name       string `yaml:"name"`
	Spec       Spec   `yaml:"spec"`
}

type Spec struct {
	TableName   string        `yaml:"table_name,omitempty"`
	Fields      []SchemaField `yaml:"fields"`
	Timestamps  bool          `yaml:"timestamps,omitempty"`
	SoftDeletes bool          `yaml:"soft_deletes,omitempty"`
}

type SchemaField struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	DBType     string `yaml:"db_type"`
	PrimaryKey bool   `yaml:"primary_key,omitempty"`
	Nullable   bool   `yaml:"nullable,omitempty"`
	Index      bool   `yaml:"index,omitempty"`
	Unique     bool   `yaml:"unique,omitempty"`
}
