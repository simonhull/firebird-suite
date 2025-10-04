package query

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates SQLC query files from schemas.
type Generator struct {
	projectPath string
	schemaPath  string
	renderer    *generator.Renderer
}

// New creates a new query generator.
func New(projectPath, schemaPath string) *Generator {
	return &Generator{
		projectPath: projectPath,
		schemaPath:  schemaPath,
		renderer:    generator.NewRenderer(),
	}
}

// Generate creates SQLC query file for the schema.
func (g *Generator) Generate() ([]generator.Operation, error) {
	// Parse schema
	spec, err := schema.Parse(g.schemaPath)
	if err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}

	// Prepare template data
	data := g.templateData(spec)

	// Generate queries file
	modelName := strings.ToLower(spec.Name)
	queriesPath := filepath.Join(
		g.projectPath,
		"internal",
		"db",
		"queries",
		modelName+".sql",
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/queries.sql.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("rendering queries template: %w", err)
	}

	return []generator.Operation{
		&generator.WriteFileOp{
			Path:    queriesPath,
			Content: content,
			Mode:    0644,
		},
	}, nil
}

// templateData prepares data for the queries template.
func (g *Generator) templateData(def *schema.Definition) map[string]interface{} {
	// Build list of insertable fields (exclude id, timestamps if auto-managed)
	var insertFields []string
	var insertParams []string
	paramIndex := 1

	for _, field := range def.Spec.Fields {
		// Skip ID (auto-generated)
		if field.PrimaryKey {
			continue
		}

		insertFields = append(insertFields, generator.SnakeCase(field.Name))
		insertParams = append(insertParams, fmt.Sprintf("$%d", paramIndex))
		paramIndex++
	}

	// Add timestamp columns if enabled
	if def.Spec.Timestamps {
		insertFields = append(insertFields, "created_at", "updated_at")
		insertParams = append(insertParams, "NOW()", "NOW()")
	}

	// Build list of updatable fields (exclude id, created_at, deleted_at)
	var updateFields []string
	updateParamIndex := 1

	for _, field := range def.Spec.Fields {
		// Skip ID and created_at
		if field.PrimaryKey {
			continue
		}

		updateFields = append(updateFields,
			fmt.Sprintf("%s = $%d", generator.SnakeCase(field.Name), updateParamIndex),
		)
		updateParamIndex++
	}

	// Add updated_at = NOW() at the end if timestamps enabled
	if def.Spec.Timestamps {
		updateFields = append(updateFields, "updated_at = NOW()")
	}

	// Build SELECT column list
	var selectColumns []string
	for _, field := range def.Spec.Fields {
		selectColumns = append(selectColumns, generator.SnakeCase(field.Name))
	}
	if def.Spec.Timestamps {
		selectColumns = append(selectColumns, "created_at", "updated_at")
	}
	if def.Spec.SoftDeletes {
		selectColumns = append(selectColumns, "deleted_at")
	}

	// Determine WHERE clause for soft deletes
	softDeleteWhere := ""
	if def.Spec.SoftDeletes {
		softDeleteWhere = " AND deleted_at IS NULL"
	}

	// Table name (use explicit or derive from model name)
	tableName := def.Spec.TableName
	if tableName == "" {
		tableName = generator.SnakeCase(generator.Pluralize(def.Name))
	}

	// Model name (capitalized)
	modelName := def.Name

	// Count queries
	queryCount := 7 // Base queries
	if def.Spec.SoftDeletes {
		queryCount = 8 // +1 for Restore
	}

	return map[string]interface{}{
		"ModelName":         modelName,
		"TableName":         tableName,
		"InsertFields":      strings.Join(insertFields, ", "),
		"InsertParams":      strings.Join(insertParams, ", "),
		"SelectColumns":     strings.Join(selectColumns, ", "),
		"UpdateFields":      strings.Join(updateFields, ", "),
		"UpdateParamCount":  updateParamIndex, // This is the param for WHERE id = $N
		"SoftDeletes":       def.Spec.SoftDeletes,
		"SoftDeleteWhere":   softDeleteWhere,
		"HasTimestamps":     def.Spec.Timestamps,
		"QueryCount":        queryCount,
	}
}
