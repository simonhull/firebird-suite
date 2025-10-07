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
	database    string // Database type: postgres, mysql, sqlite
	renderer    *generator.Renderer
}

// RelationshipQueryData holds data for generating relationship queries
type RelationshipQueryData struct {
	Name               string // Relationship name (e.g., "Author", "Comments", "Tags")
	Type               string // "belongs_to", "has_many", or "many_to_many"
	Model              string // Target model name (e.g., "User", "Comment", "Tag")
	ForeignKey         string // Snake_case FK field (e.g., "author_id", "post_id")
	RelatedKey         string // M2M related key (e.g., "tag_id")
	JunctionTable      string // M2M junction table (e.g., "post_tags")
	OrderBy            string // M2M order by clause (e.g., "name ASC")
	PrimaryKeyType     string // PostgreSQL array type (e.g., "uuid", "bigint")
	GetSingleQueryName string // Query name for single fetch (e.g., "GetPostAuthor", "GetPostTags")
	GetManyQueryName   string // Query name for batch fetch (e.g., "GetCommentsForPosts", "GetTagsForPosts")
	AddQueryName       string // M2M add query (e.g., "AddPostTag")
	RemoveQueryName    string // M2M remove query (e.g., "RemovePostTag")
	RemoveAllQueryName string // M2M remove all query (e.g., "RemoveAllPostTags")
	SourceTable        string // Source table name (e.g., "posts")
	TargetTable        string // Target table name (e.g., "users", "tags")
	TargetSoftDeletes  bool   // Does target model have soft deletes? (M2M only)
}

// New creates a new query generator.
func New(projectPath, schemaPath, database string) *Generator {
	return &Generator{
		projectPath: projectPath,
		schemaPath:  schemaPath,
		database:    database,
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
		insertParams = append(insertParams, g.getParamPlaceholder(paramIndex))
		paramIndex++
	}

	// Add timestamp columns if enabled
	if def.Spec.Timestamps {
		insertFields = append(insertFields, "created_at", "updated_at")
		timestampFunc := g.getTimestampFunction()
		insertParams = append(insertParams, timestampFunc, timestampFunc)
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
			fmt.Sprintf("%s = %s", generator.SnakeCase(field.Name), g.getParamPlaceholder(updateParamIndex)),
		)
		updateParamIndex++
	}

	// Add updated_at = NOW() at the end if timestamps enabled
	if def.Spec.Timestamps {
		updateFields = append(updateFields, fmt.Sprintf("updated_at = %s", g.getTimestampFunction()))
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

	// Prepare relationship data
	relationships := g.prepareRelationshipData(def)

	// Determine pagination support
	supportsCursor := false
	cursorField := "id" // default
	cursorFieldType := "bigint"

	if def.Spec.Pagination != nil {
		paginationType := def.Spec.Pagination.Type
		if paginationType == "" {
			paginationType = "offset" // default
		}

		supportsCursor = paginationType == "cursor" || paginationType == "both"

		if def.Spec.Pagination.CursorField != "" {
			cursorField = def.Spec.Pagination.CursorField
		}

		cursorFieldType = getCursorFieldType(def, cursorField)
	}

	return map[string]interface{}{
		"ModelName":                modelName,
		"TableName":                tableName,
		"InsertFields":             strings.Join(insertFields, ", "),
		"InsertParams":             strings.Join(insertParams, ", "),
		"SelectColumns":            strings.Join(selectColumns, ", "),
		"UpdateFields":             strings.Join(updateFields, ", "),
		"UpdateParamCount":         updateParamIndex, // This is the param for WHERE id = $N
		"SoftDeletes":              def.Spec.SoftDeletes,
		"SoftDeleteWhere":          softDeleteWhere,
		"HasTimestamps":            def.Spec.Timestamps,
		"QueryCount":               queryCount,
		"Relationships":            relationships,
		"SupportsCursorPagination": supportsCursor,
		"CursorField":              generator.SnakeCase(cursorField),
		"CursorFieldType":          cursorFieldType,
		"Database":                 g.database,
		"SupportsReturning":        g.supportsReturning(),
		"IDParam":                  g.getParamPlaceholder(1),        // For WHERE id = ?/$1
		"LimitParam":               g.getParamPlaceholder(1),        // For LIMIT ?/$1
		"OffsetParam":              g.getParamPlaceholder(2),        // For OFFSET ?/$2
		"WhereIDParam":             g.getParamPlaceholder(updateParamIndex), // For WHERE in UPDATE
		"TimestampFunc":            g.getTimestampFunction(),        // For NOW() or datetime('now')
	}
}

// prepareRelationshipData transforms schema relationships into template data
func (g *Generator) prepareRelationshipData(def *schema.Definition) []RelationshipQueryData {
	var result []RelationshipQueryData

	// Find primary key type for batch query generation
	pkType := getPrimaryKeyDBType(def)

	// Determine table name
	sourceTable := def.Spec.TableName
	if sourceTable == "" {
		sourceTable = generator.SnakeCase(generator.Pluralize(def.Name))
	}

	for _, rel := range def.Spec.Relationships {
		data := RelationshipQueryData{
			Name:           rel.Name,
			Type:           rel.Type,
			Model:          rel.Model,
			ForeignKey:     generator.SnakeCase(rel.ForeignKey),
			PrimaryKeyType: pkType,
			SourceTable:    sourceTable,
			TargetTable:    generator.SnakeCase(generator.Pluralize(rel.Model)),
		}

		// Generate query names based on relationship type
		if rel.Type == "belongs_to" {
			// GetPostAuthor
			data.GetSingleQueryName = fmt.Sprintf("Get%s%s", def.Name, rel.Name)
		} else if rel.Type == "has_many" {
			// GetPostComments
			data.GetSingleQueryName = fmt.Sprintf("Get%s%s", def.Name, rel.Name)
			// GetCommentsForPosts
			data.GetManyQueryName = fmt.Sprintf("Get%sFor%s", generator.Pluralize(rel.Model), generator.Pluralize(def.Name))
		} else if rel.Type == "many_to_many" {
			// M2M specific fields
			data.RelatedKey = rel.RelatedKey
			data.JunctionTable = rel.JunctionTable
			data.OrderBy = rel.OrderBy

			// Check if target model has soft deletes
			// This is critical for correct query generation - we need to filter
			// soft-deleted target entities, not source entities
			targetSoftDeletes := false
			if targetDef, err := g.loadTargetModelSchema(rel.Model); err == nil {
				targetSoftDeletes = targetDef.Spec.SoftDeletes
			}
			// If we can't load the target schema, conservatively assume false
			// This prevents generating a filter for a column that might not exist
			data.TargetSoftDeletes = targetSoftDeletes

			// GetPostTags
			data.GetSingleQueryName = fmt.Sprintf("Get%s%s", def.Name, rel.Name)
			// GetTagsForPosts
			data.GetManyQueryName = fmt.Sprintf("Get%sFor%s", rel.Name, generator.Pluralize(def.Name))
			// AddPostTag (use the Model name, which is singular like "Tag")
			data.AddQueryName = fmt.Sprintf("Add%s%s", def.Name, rel.Model)
			// RemovePostTag
			data.RemoveQueryName = fmt.Sprintf("Remove%s%s", def.Name, rel.Model)
			// RemoveAllPostTags
			data.RemoveAllQueryName = fmt.Sprintf("RemoveAll%s%s", def.Name, rel.Name)
		}

		result = append(result, data)
	}

	return result
}

// getPrimaryKeyDBType returns the DB type for the primary key field
func getPrimaryKeyDBType(def *schema.Definition) string {
	for _, field := range def.Spec.Fields {
		if field.PrimaryKey {
			// Map Go types to PostgreSQL array types
			switch field.DBType {
			case "UUID":
				return "uuid"
			case "BIGINT", "BIGSERIAL":
				return "bigint"
			case "INTEGER", "SERIAL":
				return "integer"
			default:
				return "bigint" // Safe default
			}
		}
	}
	return "bigint" // Fallback
}

// getCursorFieldType returns the DB type for the cursor field
func getCursorFieldType(def *schema.Definition, cursorFieldName string) string {
	for _, field := range def.Spec.Fields {
		if field.Name == cursorFieldName {
			switch field.DBType {
			case "UUID":
				return "uuid"
			case "BIGINT", "BIGSERIAL":
				return "bigint"
			case "INTEGER", "SERIAL":
				return "integer"
			case "TIMESTAMP", "TIMESTAMPTZ":
				return "timestamp"
			default:
				return "bigint"
			}
		}
	}
	return "bigint" // Fallback for id
}

// loadTargetModelSchema attempts to load the schema for a target model
// Returns the schema definition or nil if it cannot be loaded
func (g *Generator) loadTargetModelSchema(modelName string) (*schema.Definition, error) {
	// Convert model name to schema filename (e.g., "Tag" -> "tag.firebird.yml")
	schemaFileName := strings.ToLower(modelName) + ".firebird.yml"

	// Try common schema locations
	possiblePaths := []string{
		filepath.Join(g.projectPath, "app", "schemas", schemaFileName),
		filepath.Join(g.projectPath, "schemas", schemaFileName),
		filepath.Join(filepath.Dir(g.schemaPath), schemaFileName),
	}

	for _, schemaPath := range possiblePaths {
		def, err := schema.Parse(schemaPath)
		if err == nil {
			return def, nil
		}
	}

	// Could not load target schema - return nil (caller handles gracefully)
	return nil, fmt.Errorf("could not load schema for model %s", modelName)
}

// getParamPlaceholder returns the SQL parameter placeholder for the given database
// PostgreSQL: $1, $2, $3
// MySQL/SQLite: ?
func (g *Generator) getParamPlaceholder(index int) string {
	switch g.database {
	case "postgresql", "postgres":
		return fmt.Sprintf("$%d", index)
	case "mysql", "sqlite":
		return "?"
	default:
		// Default to PostgreSQL syntax
		return fmt.Sprintf("$%d", index)
	}
}

// getTimestampFunction returns the SQL function for current timestamp
// PostgreSQL: NOW()
// MySQL: NOW()
// SQLite: datetime('now')
func (g *Generator) getTimestampFunction() string {
	switch g.database {
	case "sqlite":
		return "datetime('now')"
	case "postgresql", "postgres", "mysql":
		return "NOW()"
	default:
		return "NOW()"
	}
}

// supportsReturning returns whether the database supports RETURNING clause
// PostgreSQL: Yes
// SQLite: Yes (3.35+)
// MySQL: No
func (g *Generator) supportsReturning() bool {
	switch g.database {
	case "postgresql", "postgres", "sqlite":
		return true
	case "mysql":
		return false
	default:
		return true
	}
}
