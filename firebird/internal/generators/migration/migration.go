package migration

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/simonhull/firebird-suite/firebird/internal/generators/model"
	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/output"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates SQL migrations from schemas
type Generator struct {
	resolver *generator.Resolver
	renderer *generator.Renderer
}

// NewGenerator creates a new migration generator with the given conflict resolver
func NewGenerator(resolver *generator.Resolver) *Generator {
	return &Generator{
		resolver: resolver,
		renderer: generator.NewRenderer(),
	}
}

// Generate generates a SQL migration for the given schema name
func (g *Generator) Generate(name string) error {
	output.Verbose(fmt.Sprintf("Looking for schema file: %s", name))

	// 1. Find schema file (reuse from model generator)
	schemaPath, err := model.FindSchemaFile(name)
	if err != nil {
		return err
	}
	output.Verbose(fmt.Sprintf("Found schema: %s", schemaPath))

	// 2. Parse schema
	def, err := schema.Parse(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}
	output.Verbose(fmt.Sprintf("Parsed schema for: %s", def.Name))

	// 3. Detect database dialect
	dialect, err := DetectDatabaseDialect()
	if err != nil {
		return err
	}
	output.Verbose(fmt.Sprintf("Detected database dialect: %s", dialect))

	// 4. Check if migration already exists
	migrationsDir := "migrations"
	migrationName := "create_" + generator.SnakeCase(generator.Pluralize(name))
	exists, err := MigrationExists(migrationsDir, migrationName)
	if err != nil {
		return fmt.Errorf("failed to check existing migrations: %w", err)
	}
	if exists {
		return fmt.Errorf("migration for '%s' already exists. Migrations are immutable - create a new migration to modify the table", name)
	}

	// 5. Generate migration number
	number, err := GenerateMigrationNumber(TimestampNumbering, migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to generate migration number: %w", err)
	}
	output.Verbose(fmt.Sprintf("Generated migration number: %s", number))

	// 6. Get migration filenames
	upFile, downFile := GetMigrationFilenames(number, migrationName)
	upPath := filepath.Join(migrationsDir, upFile)
	downPath := filepath.Join(migrationsDir, downFile)

	// 7. Transform schema to migration data
	data := PrepareMigrationData(def, dialect)
	output.Verbose(fmt.Sprintf("Prepared migration for table: %s (%d columns)", data.TableName, len(data.Columns)))

	// Log any explicit db_type overrides
	for i, field := range def.Spec.Fields {
		if field.DBType != "" {
			output.Verbose(fmt.Sprintf("  Column '%s': using explicit db_type '%s' (Go type: %s)",
				data.Columns[i].Name, field.DBType, field.Type))
		}
	}

	// 8. Render up migration
	upTemplate := fmt.Sprintf("templates/%s.up.sql.tmpl", dialect)
	upContent, err := g.renderer.RenderFS(templatesFS, upTemplate, data)
	if err != nil {
		return fmt.Errorf("failed to render up migration: %w", err)
	}

	// 9. Render down migration
	downTemplate := fmt.Sprintf("templates/%s.down.sql.tmpl", dialect)
	downContent, err := g.renderer.RenderFS(templatesFS, downTemplate, data)
	if err != nil {
		return fmt.Errorf("failed to render down migration: %w", err)
	}

	// 10. Ensure migrations directory exists
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// 11. Write up migration
	if err := os.WriteFile(upPath, upContent, 0644); err != nil {
		return fmt.Errorf("failed to write up migration: %w", err)
	}
	output.Verbose(fmt.Sprintf("Wrote up migration: %s", upPath))

	// 12. Write down migration
	if err := os.WriteFile(downPath, downContent, 0644); err != nil {
		return fmt.Errorf("failed to write down migration: %w", err)
	}
	output.Verbose(fmt.Sprintf("Wrote down migration: %s", downPath))

	return nil
}
