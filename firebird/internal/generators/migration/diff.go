package migration

import (
	"fmt"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
)

// DiffSchemas compares two schema definitions and generates ALTER TABLE statements
// Returns UP and DOWN migration SQL, or error if no changes detected
func DiffSchemas(oldDef, newDef *schema.Definition, dialect DatabaseDialect) (upSQL, downSQL string, err error) {
	var upStatements []string
	var downStatements []string

	tableName := newDef.Spec.TableName
	if tableName == "" {
		tableName = strings.ToLower(pluralize(newDef.Name))
	}

	// 1. Check for field changes
	for _, newField := range newDef.Spec.Fields {
		// Skip auto-generated fields (ID, timestamps, soft deletes)
		if newField.Name == "id" || newField.Name == "created_at" ||
		   newField.Name == "updated_at" || newField.Name == "deleted_at" {
			continue
		}

		if !fieldExists(oldDef, newField.Name) {
			// Field added
			up, down := generateAddColumn(tableName, newField, dialect)
			upStatements = append(upStatements, up)
			downStatements = append(downStatements, down)
		} else if fieldModified(oldDef, newDef, newField.Name) {
			// Field modified
			oldField := findField(oldDef, newField.Name)
			up, down := generateModifyColumn(tableName, oldField, newField, dialect)
			upStatements = append(upStatements, up)
			downStatements = append(downStatements, down)
		}
	}

	// 2. Check for removed fields
	for _, oldField := range oldDef.Spec.Fields {
		// Skip auto-generated fields
		if oldField.Name == "id" || oldField.Name == "created_at" ||
		   oldField.Name == "updated_at" || oldField.Name == "deleted_at" {
			continue
		}

		if !fieldExists(newDef, oldField.Name) {
			// Field removed
			up, down := generateDropColumn(tableName, oldField, dialect)
			upStatements = append(upStatements, up)
			downStatements = append(downStatements, down)
		}
	}

	// 3. Check for timestamp changes
	if oldDef.Spec.Timestamps != newDef.Spec.Timestamps {
		if newDef.Spec.Timestamps {
			// Timestamps added
			up, down := generateAddTimestamps(tableName, dialect)
			upStatements = append(upStatements, up)
			downStatements = append(downStatements, down)
		} else {
			// Timestamps removed
			up, down := generateDropTimestamps(tableName, dialect)
			upStatements = append(upStatements, up)
			downStatements = append(downStatements, down)
		}
	}

	// 4. Check for soft delete changes
	if oldDef.Spec.SoftDeletes != newDef.Spec.SoftDeletes {
		if newDef.Spec.SoftDeletes {
			// Soft deletes added
			up, down := generateAddSoftDeletes(tableName, dialect)
			upStatements = append(upStatements, up)
			downStatements = append(downStatements, down)
		} else {
			// Soft deletes removed
			up, down := generateDropSoftDeletes(tableName, dialect)
			upStatements = append(upStatements, up)
			downStatements = append(downStatements, down)
		}
	}

	// 5. Check for index changes
	indexUp, indexDown := diffIndexes(oldDef, newDef, tableName, dialect)
	upStatements = append(upStatements, indexUp...)
	downStatements = append(downStatements, indexDown...)

	// Error if no changes detected
	if len(upStatements) == 0 {
		return "", "", fmt.Errorf("no schema changes detected - migration would be empty")
	}

	// Build final SQL
	upSQL = strings.Join(upStatements, "\n\n")
	downSQL = strings.Join(downStatements, "\n\n")

	return upSQL, downSQL, nil
}

// generateAddColumn creates ALTER TABLE statements to add a column
func generateAddColumn(tableName string, field schema.Field, dialect DatabaseDialect) (up, down string) {
	columnDef := fmt.Sprintf("%s %s", field.Name, field.DBType)
	if !field.Nullable {
		columnDef += " NOT NULL"
	}
	if field.Unique {
		columnDef += " UNIQUE"
	}
	if field.Default != nil {
		columnDef += fmt.Sprintf(" DEFAULT %v", field.Default)
	}

	up = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", tableName, columnDef)
	down = fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tableName, field.Name)
	return up, down
}

// generateDropColumn creates ALTER TABLE statements to drop a column
func generateDropColumn(tableName string, field schema.Field, dialect DatabaseDialect) (up, down string) {
	columnDef := fmt.Sprintf("%s %s", field.Name, field.DBType)
	if !field.Nullable {
		columnDef += " NOT NULL"
	}
	if field.Unique {
		columnDef += " UNIQUE"
	}
	if field.Default != nil {
		columnDef += fmt.Sprintf(" DEFAULT %v", field.Default)
	}

	up = fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tableName, field.Name)
	down = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", tableName, columnDef)
	return up, down
}

// generateModifyColumn creates ALTER TABLE statements to modify a column
func generateModifyColumn(tableName string, oldField, newField schema.Field, dialect DatabaseDialect) (up, down string) {
	var alterKeyword string
	switch dialect {
	case PostgreSQL:
		alterKeyword = "ALTER COLUMN"
	case MySQL:
		alterKeyword = "MODIFY COLUMN"
	case SQLite:
		// SQLite doesn't support ALTER COLUMN - would need table recreation
		// For now, provide a comment indicating manual intervention needed
		return fmt.Sprintf("-- SQLite does not support ALTER COLUMN. Manual migration required for %s.%s", tableName, newField.Name),
		       fmt.Sprintf("-- SQLite does not support ALTER COLUMN. Manual migration required for %s.%s", tableName, oldField.Name)
	}

	newColumnDef := fmt.Sprintf("%s %s", newField.Name, newField.DBType)
	if !newField.Nullable {
		newColumnDef += " NOT NULL"
	}

	oldColumnDef := fmt.Sprintf("%s %s", oldField.Name, oldField.DBType)
	if !oldField.Nullable {
		oldColumnDef += " NOT NULL"
	}

	up = fmt.Sprintf("ALTER TABLE %s %s %s;", tableName, alterKeyword, newColumnDef)
	down = fmt.Sprintf("ALTER TABLE %s %s %s;", tableName, alterKeyword, oldColumnDef)
	return up, down
}

// generateAddTimestamps creates ALTER TABLE statements to add timestamp columns
func generateAddTimestamps(tableName string, dialect DatabaseDialect) (up, down string) {
	var timestampType string
	switch dialect {
	case PostgreSQL:
		timestampType = "TIMESTAMP"
	case MySQL:
		timestampType = "TIMESTAMP"
	case SQLite:
		timestampType = "DATETIME"
	}

	up = fmt.Sprintf(`ALTER TABLE %s ADD COLUMN created_at %s NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE %s ADD COLUMN updated_at %s NOT NULL DEFAULT CURRENT_TIMESTAMP;`,
		tableName, timestampType, tableName, timestampType)

	down = fmt.Sprintf(`ALTER TABLE %s DROP COLUMN updated_at;
ALTER TABLE %s DROP COLUMN created_at;`, tableName, tableName)

	return up, down
}

// generateDropTimestamps creates ALTER TABLE statements to drop timestamp columns
func generateDropTimestamps(tableName string, dialect DatabaseDialect) (up, down string) {
	var timestampType string
	switch dialect {
	case PostgreSQL:
		timestampType = "TIMESTAMP"
	case MySQL:
		timestampType = "TIMESTAMP"
	case SQLite:
		timestampType = "DATETIME"
	}

	up = fmt.Sprintf(`ALTER TABLE %s DROP COLUMN updated_at;
ALTER TABLE %s DROP COLUMN created_at;`, tableName, tableName)

	down = fmt.Sprintf(`ALTER TABLE %s ADD COLUMN created_at %s NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE %s ADD COLUMN updated_at %s NOT NULL DEFAULT CURRENT_TIMESTAMP;`,
		tableName, timestampType, tableName, timestampType)

	return up, down
}

// generateAddSoftDeletes creates ALTER TABLE statements to add soft delete column
func generateAddSoftDeletes(tableName string, dialect DatabaseDialect) (up, down string) {
	var timestampType string
	switch dialect {
	case PostgreSQL:
		timestampType = "TIMESTAMP"
	case MySQL:
		timestampType = "TIMESTAMP"
	case SQLite:
		timestampType = "DATETIME"
	}

	up = fmt.Sprintf("ALTER TABLE %s ADD COLUMN deleted_at %s;", tableName, timestampType)
	down = fmt.Sprintf("ALTER TABLE %s DROP COLUMN deleted_at;", tableName)
	return up, down
}

// generateDropSoftDeletes creates ALTER TABLE statements to drop soft delete column
func generateDropSoftDeletes(tableName string, dialect DatabaseDialect) (up, down string) {
	var timestampType string
	switch dialect {
	case PostgreSQL:
		timestampType = "TIMESTAMP"
	case MySQL:
		timestampType = "TIMESTAMP"
	case SQLite:
		timestampType = "DATETIME"
	}

	up = fmt.Sprintf("ALTER TABLE %s DROP COLUMN deleted_at;", tableName)
	down = fmt.Sprintf("ALTER TABLE %s ADD COLUMN deleted_at %s;", tableName, timestampType)
	return up, down
}

// diffIndexes compares indexes between old and new schemas
func diffIndexes(oldDef, newDef *schema.Definition, tableName string, dialect DatabaseDialect) (upStatements, downStatements []string) {
	// Check for new indexes
	for _, newIdx := range newDef.Spec.Indexes {
		if !indexExists(oldDef, newIdx) {
			up, down := generateCreateIndex(tableName, newIdx, dialect)
			upStatements = append(upStatements, up)
			downStatements = append(downStatements, down)
		}
	}

	// Check for removed indexes
	for _, oldIdx := range oldDef.Spec.Indexes {
		if !indexExists(newDef, oldIdx) {
			up, down := generateDropIndex(tableName, oldIdx, dialect)
			upStatements = append(upStatements, up)
			downStatements = append(downStatements, down)
		}
	}

	return upStatements, downStatements
}

// generateCreateIndex creates CREATE INDEX statement
func generateCreateIndex(tableName string, index schema.Index, dialect DatabaseDialect) (up, down string) {
	indexName := index.Name
	if indexName == "" {
		// Generate index name from columns
		indexName = fmt.Sprintf("idx_%s_%s", tableName, strings.Join(index.Columns, "_"))
	}

	uniqueClause := ""
	if index.Unique {
		uniqueClause = "UNIQUE "
	}

	columns := strings.Join(index.Columns, ", ")

	up = fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", uniqueClause, indexName, tableName, columns)

	// Add type for PostgreSQL
	if dialect == PostgreSQL && index.Type != "" {
		up += fmt.Sprintf(" USING %s", index.Type)
	}

	// Add WHERE clause for partial indexes
	if index.Where != "" && (dialect == PostgreSQL || dialect == SQLite) {
		up += fmt.Sprintf(" WHERE %s", index.Where)
	}

	up += ";"

	down = fmt.Sprintf("DROP INDEX %s;", indexName)

	return up, down
}

// generateDropIndex creates DROP INDEX statement
func generateDropIndex(tableName string, index schema.Index, dialect DatabaseDialect) (up, down string) {
	indexName := index.Name
	if indexName == "" {
		indexName = fmt.Sprintf("idx_%s_%s", tableName, strings.Join(index.Columns, "_"))
	}

	up = fmt.Sprintf("DROP INDEX %s;", indexName)

	// Reconstruct CREATE INDEX for down migration
	uniqueClause := ""
	if index.Unique {
		uniqueClause = "UNIQUE "
	}

	columns := strings.Join(index.Columns, ", ")
	down = fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", uniqueClause, indexName, tableName, columns)

	if dialect == PostgreSQL && index.Type != "" {
		down += fmt.Sprintf(" USING %s", index.Type)
	}

	if index.Where != "" && (dialect == PostgreSQL || dialect == SQLite) {
		down += fmt.Sprintf(" WHERE %s", index.Where)
	}

	down += ";"

	return up, down
}

// fieldExists checks if a field exists in a schema
func fieldExists(def *schema.Definition, fieldName string) bool {
	for _, field := range def.Spec.Fields {
		if field.Name == fieldName {
			return true
		}
	}
	return false
}

// findField finds a field in a schema by name
func findField(def *schema.Definition, fieldName string) schema.Field {
	for _, field := range def.Spec.Fields {
		if field.Name == fieldName {
			return field
		}
	}
	return schema.Field{}
}

// fieldModified checks if a field has been modified between schemas
func fieldModified(oldDef, newDef *schema.Definition, fieldName string) bool {
	oldField := findField(oldDef, fieldName)
	newField := findField(newDef, fieldName)

	// Compare relevant properties
	if oldField.Type != newField.Type ||
	   oldField.DBType != newField.DBType ||
	   oldField.Nullable != newField.Nullable ||
	   oldField.Unique != newField.Unique {
		return true
	}

	return false
}

// indexExists checks if an index exists in a schema
func indexExists(def *schema.Definition, index schema.Index) bool {
	for _, idx := range def.Spec.Indexes {
		// Compare by columns (since name might be auto-generated)
		if len(idx.Columns) == len(index.Columns) {
			match := true
			for i, col := range idx.Columns {
				if col != index.Columns[i] {
					match = false
					break
				}
			}
			if match && idx.Unique == index.Unique {
				return true
			}
		}
	}
	return false
}
