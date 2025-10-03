package types

import (
	"fmt"
	"sort"
)

// TypeInfo contains metadata about a type
type TypeInfo struct {
	GoType      string            // "uuid.UUID", "decimal.Decimal", "string"
	ImportPath  string            // "github.com/google/uuid", "" for builtins
	DBTypes     map[string]string // Database-specific SQL types
	DefaultExpr string            // "uuid.New()", "decimal.Zero" (for SQLC)
	IsIDType    bool              // Can be used as primary key
}

// Registry contains all known types
var Registry = map[string]TypeInfo{
	// Built-in primitive types (no imports)
	"string": {
		GoType: "string",
		DBTypes: map[string]string{
			"postgres": "VARCHAR(255)",
			"mysql":    "VARCHAR(255)",
			"sqlite":   "TEXT",
		},
	},
	"text": {
		GoType: "string",
		DBTypes: map[string]string{
			"postgres": "TEXT",
			"mysql":    "TEXT",
			"sqlite":   "TEXT",
		},
	},
	"int": {
		GoType: "int",
		DBTypes: map[string]string{
			"postgres": "INTEGER",
			"mysql":    "INT",
			"sqlite":   "INTEGER",
		},
	},
	"int64": {
		GoType:   "int64",
		IsIDType: true, // Can be used as primary key
		DBTypes: map[string]string{
			"postgres": "BIGINT",
			"mysql":    "BIGINT",
			"sqlite":   "INTEGER",
		},
	},
	"float64": {
		GoType: "float64",
		DBTypes: map[string]string{
			"postgres": "DOUBLE PRECISION",
			"mysql":    "DOUBLE",
			"sqlite":   "REAL",
		},
	},
	"bool": {
		GoType: "bool",
		DBTypes: map[string]string{
			"postgres": "BOOLEAN",
			"mysql":    "TINYINT(1)",
			"sqlite":   "INTEGER",
		},
	},

	// Time types (standard library)
	"timestamp": {
		GoType:     "time.Time",
		ImportPath: "time",
		DBTypes: map[string]string{
			"postgres": "TIMESTAMP",
			"mysql":    "TIMESTAMP",
			"sqlite":   "DATETIME",
		},
	},
	"date": {
		GoType:     "time.Time",
		ImportPath: "time",
		DBTypes: map[string]string{
			"postgres": "DATE",
			"mysql":    "DATE",
			"sqlite":   "DATE",
		},
	},
	"time": {
		GoType:     "time.Time",
		ImportPath: "time",
		DBTypes: map[string]string{
			"postgres": "TIME",
			"mysql":    "TIME",
			"sqlite":   "TIME",
		},
	},

	// Third-party types
	"UUID": {
		GoType:      "uuid.UUID",
		ImportPath:  "github.com/google/uuid",
		DefaultExpr: "uuid.New()",
		IsIDType:    true, // Can be used as primary key
		DBTypes: map[string]string{
			"postgres": "UUID",     // PostgreSQL native UUID
			"mysql":    "CHAR(36)", // MySQL doesn't have native UUID
			"sqlite":   "TEXT",     // SQLite doesn't have native UUID
		},
	},
	"Decimal": {
		GoType:      "decimal.Decimal",
		ImportPath:  "github.com/shopspring/decimal",
		DefaultExpr: "decimal.Zero",
		DBTypes: map[string]string{
			"postgres": "NUMERIC(19,4)",
			"mysql":    "DECIMAL(19,4)",
			"sqlite":   "TEXT", // SQLite doesn't have DECIMAL
		},
	},
	"NullString": {
		GoType:     "sql.NullString",
		ImportPath: "database/sql",
		DBTypes: map[string]string{
			"postgres": "VARCHAR(255)",
			"mysql":    "VARCHAR(255)",
			"sqlite":   "TEXT",
		},
	},
}

// Lookup retrieves type info by name
func Lookup(typeName string) (TypeInfo, bool) {
	info, ok := Registry[typeName]
	return info, ok
}

// GetDBType returns the database-specific SQL type
func GetDBType(typeName, driver string) (string, error) {
	info, ok := Lookup(typeName)
	if !ok {
		return "", fmt.Errorf("unknown type: %s", typeName)
	}

	dbType, ok := info.DBTypes[driver]
	if !ok {
		return "", fmt.Errorf("type %s not supported for driver %s", typeName, driver)
	}

	return dbType, nil
}

// GetGoType returns the Go type and import path
func GetGoType(typeName string) (goType, importPath string, err error) {
	info, ok := Lookup(typeName)
	if !ok {
		return "", "", fmt.Errorf("unknown type: %s", typeName)
	}

	return info.GoType, info.ImportPath, nil
}

// GetPrimaryKeyType returns the DB type for primary keys with auto-increment
// This handles the special case where primary keys need different syntax
func GetPrimaryKeyType(typeName, driver string) (string, error) {
	info, ok := Lookup(typeName)
	if !ok {
		return "", fmt.Errorf("unknown type: %s", typeName)
	}

	if !info.IsIDType {
		return "", fmt.Errorf("type %s cannot be used as primary key", typeName)
	}

	// For int64, add auto-increment syntax
	if typeName == "int64" {
		switch driver {
		case "postgres":
			return "BIGSERIAL", nil
		case "mysql":
			return "BIGINT AUTO_INCREMENT", nil
		case "sqlite":
			return "INTEGER", nil
		default:
			return "BIGINT", nil
		}
	}

	// For UUID and other ID types, use standard DB type
	return GetDBType(typeName, driver)
}

// CollectImports gathers unique imports from a list of type names
func CollectImports(typeNames []string) []string {
	importSet := make(map[string]bool)

	for _, typeName := range typeNames {
		info, ok := Lookup(typeName)
		if !ok {
			continue
		}

		if info.ImportPath != "" {
			importSet[info.ImportPath] = true
		}
	}

	// Convert to sorted slice
	imports := make([]string, 0, len(importSet))
	for imp := range importSet {
		imports = append(imports, imp)
	}
	sort.Strings(imports)

	return imports
}
