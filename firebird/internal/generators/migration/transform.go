package migration

import (
	"fmt"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"github.com/simonhull/firebird-suite/fledge/generator"
)

// MigrationData is the data passed to templates
type MigrationData struct {
	TableName string          // Snake_case table name (e.g., "users")
	Columns   []ColumnData    // Column definitions
	Indexes   []IndexData     // Index definitions
	Dialect   DatabaseDialect // Database dialect
}

// ColumnData represents a single column definition
type ColumnData struct {
	Name       string // Snake_case column name
	Type       string // SQL type (e.g., "VARCHAR(255)")
	Nullable   bool   // Allow NULL?
	PrimaryKey bool   // Is primary key?
	Unique     bool   // Unique constraint?
	Default    string // Default value (empty if none)
}

// IndexData represents an index definition
type IndexData struct {
	Name    string   // Index name (e.g., "idx_users_email")
	Columns []string // Column names
	Unique  bool     // Unique index?
	Where   string   // Partial index condition (PostgreSQL/SQLite only)
	Type    string   // Index type (btree, hash, gin, gist, etc.)
}

// PrepareMigrationData transforms a schema definition into migration data
func PrepareMigrationData(def *schema.Definition, dialect DatabaseDialect) *MigrationData {
	tableName := generator.SnakeCase(generator.Pluralize(def.Name))

	data := &MigrationData{
		TableName: tableName,
		Columns:   make([]ColumnData, 0, len(def.Spec.Fields)),
		Indexes:   make([]IndexData, 0),
		Dialect:   dialect,
	}

	// Transform fields to columns
	for _, field := range def.Spec.Fields {
		column := transformField(field, dialect)
		data.Columns = append(data.Columns, column)

		// Note: We don't automatically generate indexes for UNIQUE constraints
		// because the inline UNIQUE constraint already creates an index in most databases.
	}

	// Add timestamps if enabled
	if def.Spec.Timestamps {
		data.Columns = append(data.Columns,
			ColumnData{
				Name:     "created_at",
				Type:     getTimestampType(dialect),
				Nullable: false,
				Default:  getTimestampDefault(dialect, "created_at"),
			},
			ColumnData{
				Name:     "updated_at",
				Type:     getTimestampType(dialect),
				Nullable: false,
				Default:  getTimestampDefault(dialect, "updated_at"),
			},
		)
	}

	// Add soft deletes if enabled
	if def.Spec.SoftDeletes {
		data.Columns = append(data.Columns, ColumnData{
			Name:     "deleted_at",
			Type:     getTimestampType(dialect),
			Nullable: true, // Nullable - NULL means not deleted
			Default:  "",   // No default
		})
	}

	// Transform indexes
	data.Indexes = transformIndexes(def.Spec.Indexes, tableName)

	return data
}

// transformField converts a schema field to a SQL column definition
func transformField(field schema.Field, dialect DatabaseDialect) ColumnData {
	// Determine if column should be NOT NULL
	// Priority: Required > PrimaryKey > Nullable flag > pointer type
	nullable := field.Nullable
	if field.Required || field.PrimaryKey {
		nullable = false
	} else if strings.HasPrefix(field.Type, "*") {
		nullable = true
	}

	column := ColumnData{
		Name:       generator.SnakeCase(field.Name),
		Nullable:   nullable,
		PrimaryKey: field.PrimaryKey,
		Unique:     field.Unique,
	}

	// Determine SQL type based on Go type and dialect
	column.Type = mapGoTypeToSQL(field.Type, field.DBType, dialect)

	// Determine default value
	column.Default = generateDefault(field, dialect)

	return column
}

// mapGoTypeToSQL converts a Go type to SQL type for the given dialect
func mapGoTypeToSQL(goType, dbType string, dialect DatabaseDialect) string {
	// If db_type is explicitly specified, use it
	if dbType != "" {
		return dbType
	}

	// Strip pointer
	goType = strings.TrimPrefix(goType, "*")

	// Map Go types to SQL types
	switch dialect {
	case PostgreSQL:
		return mapGoTypeToPostgreSQL(goType)
	case MySQL:
		return mapGoTypeToMySQL(goType)
	case SQLite:
		return mapGoTypeToSQLite(goType)
	default:
		return "TEXT" // Safe fallback
	}
}

// mapGoTypeToPostgreSQL maps Go types to PostgreSQL types
func mapGoTypeToPostgreSQL(goType string) string {
	switch goType {
	case "string":
		return "TEXT"
	case "int", "int32":
		return "INTEGER"
	case "int64":
		return "BIGINT"
	case "bool":
		return "BOOLEAN"
	case "float64":
		return "DOUBLE PRECISION"
	case "time.Time":
		return "TIMESTAMPTZ"
	case "uuid.UUID":
		return "UUID"
	case "[]byte":
		return "BYTEA"
	default:
		return "TEXT"
	}
}

// mapGoTypeToMySQL maps Go types to MySQL types
func mapGoTypeToMySQL(goType string) string {
	switch goType {
	case "string":
		return "TEXT"
	case "int", "int32":
		return "INT"
	case "int64":
		return "BIGINT"
	case "bool":
		return "BOOLEAN"
	case "float64":
		return "DOUBLE"
	case "time.Time":
		return "TIMESTAMP"
	case "uuid.UUID":
		return "CHAR(36)"
	case "[]byte":
		return "BLOB"
	default:
		return "TEXT"
	}
}

// mapGoTypeToSQLite maps Go types to SQLite types
func mapGoTypeToSQLite(goType string) string {
	switch goType {
	case "string":
		return "TEXT"
	case "int", "int32", "int64":
		return "INTEGER"
	case "bool":
		return "INTEGER" // SQLite uses 0/1 for boolean
	case "float64":
		return "REAL"
	case "time.Time":
		return "TEXT" // SQLite stores timestamps as text
	case "uuid.UUID":
		return "TEXT"
	case "[]byte":
		return "BLOB"
	default:
		return "TEXT"
	}
}

// generateDefault creates a default value clause for a field
func generateDefault(field schema.Field, dialect DatabaseDialect) string {
	// Check for explicit default value
	if field.Default != nil {
		return formatDefaultValue(field.Default, field.Type)
	}

	// Check for auto_now_add (created_at)
	if field.AutoNowAdd {
		switch dialect {
		case PostgreSQL:
			return "NOW()"
		case MySQL:
			return "CURRENT_TIMESTAMP"
		case SQLite:
			return "CURRENT_TIMESTAMP"
		}
	}

	// Check for auto_now (updated_at)
	if field.AutoNow {
		switch dialect {
		case PostgreSQL:
			return "NOW()"
		case MySQL:
			return "CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
		case SQLite:
			return "CURRENT_TIMESTAMP"
		}
	}

	// UUID primary key default
	if field.PrimaryKey && strings.Contains(field.Type, "uuid.UUID") {
		if dialect == PostgreSQL {
			return "gen_random_uuid()"
		}
		// MySQL and SQLite: app generates UUID
	}

	return ""
}

// formatDefaultValue formats a default value for SQL
func formatDefaultValue(defaultVal any, fieldType string) string {
	if defaultVal == nil {
		return ""
	}

	// Convert to string
	strVal := ""
	switch v := defaultVal.(type) {
	case string:
		strVal = v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		strVal = fmt.Sprintf("%v", v)
	}

	// Quote strings and text types
	if fieldType == "string" || fieldType == "text" || strings.Contains(fieldType, "String") {
		return fmt.Sprintf("'%s'", strVal)
	}

	return strVal
}

// transformIndexes converts schema indexes to migration index data
func transformIndexes(indexes []schema.Index, tableName string) []IndexData {
	var result []IndexData

	for _, idx := range indexes {
		indexData := IndexData{
			Name:    idx.Name,
			Columns: make([]string, len(idx.Columns)),
			Unique:  idx.Unique,
			Where:   idx.Where,
			Type:    idx.Type,
		}

		// Convert column names to snake_case
		for i, col := range idx.Columns {
			indexData.Columns[i] = generator.SnakeCase(col)
		}

		// Generate name if not provided
		if indexData.Name == "" {
			indexData.Name = generateIndexName(tableName, indexData.Columns, indexData.Unique)
		}

		result = append(result, indexData)
	}

	return result
}

// generateIndexName creates an index name from table name, columns, and uniqueness
func generateIndexName(tableName string, columns []string, unique bool) string {
	prefix := "idx"
	if unique {
		prefix = "uniq"
	}
	return fmt.Sprintf("%s_%s_%s", prefix, tableName, strings.Join(columns, "_"))
}

// getTimestampType returns the appropriate timestamp type for the dialect
func getTimestampType(dialect DatabaseDialect) string {
	switch dialect {
	case PostgreSQL:
		return "TIMESTAMPTZ"
	case MySQL:
		return "TIMESTAMP"
	case SQLite:
		return "TEXT"
	default:
		return "TIMESTAMP"
	}
}

// getTimestampDefault returns the appropriate default value for timestamp columns
func getTimestampDefault(dialect DatabaseDialect, columnName string) string {
	switch dialect {
	case PostgreSQL:
		return "NOW()"
	case MySQL:
		if columnName == "updated_at" {
			return "CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
		}
		return "CURRENT_TIMESTAMP"
	case SQLite:
		return "CURRENT_TIMESTAMP"
	default:
		return "CURRENT_TIMESTAMP"
	}
}
