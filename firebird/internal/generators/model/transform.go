package model

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"github.com/simonhull/firebird-suite/firebird/internal/types"
	"github.com/simonhull/firebird-suite/fledge/generator"
)

// ModelData is the data passed to templates
type ModelData struct {
	Package string      // Package name (e.g., "models")
	Name    string      // Struct name (e.g., "User")
	Imports []string    // Required imports
	Fields  []FieldData // Struct fields
}

// FieldData represents a single struct field
type FieldData struct {
	Name string // PascalCase field name (e.g., "Email")
	Type string // Go type (e.g., "string", "uuid.UUID")
	Tags string // Complete tag string: `json:"email" db:"email"`
}

// PrepareModelData transforms a schema definition into template data
func PrepareModelData(def *schema.Definition, outputPath string) *ModelData {
	data := &ModelData{
		Package: inferPackage(outputPath),
		Name:    def.Name,
		Fields:  make([]FieldData, 0, len(def.Spec.Fields)+3), // +3 for potential timestamps and soft deletes
	}

	// Collect type names for import gathering
	var typeNames []string

	// Transform fields
	for _, field := range def.Spec.Fields {
		// Look up type in registry to get the Go type
		goType, _, err := types.GetGoType(field.Type)
		if err != nil {
			// Fallback: use field.Type as-is (custom types or unknown types)
			goType = field.Type
		} else {
			// Track type name for imports
			typeNames = append(typeNames, field.Type)
		}

		data.Fields = append(data.Fields, FieldData{
			Name: generator.PascalCase(field.Name),
			Type: goType,
			Tags: buildTagString(field),
		})
	}

	// Add timestamps if enabled
	if def.Spec.Timestamps {
		// Add timestamp to typeNames for import collection (registry key is "timestamp", not "time.Time")
		if !containsString(typeNames, "timestamp") {
			typeNames = append(typeNames, "timestamp")
		}
		data.Fields = append(data.Fields,
			FieldData{
				Name: "CreatedAt",
				Type: "time.Time",
				Tags: "`json:\"created_at\" db:\"created_at\"`",
			},
			FieldData{
				Name: "UpdatedAt",
				Type: "time.Time",
				Tags: "`json:\"updated_at\" db:\"updated_at\"`",
			},
		)
	}

	// Add soft deletes if enabled
	if def.Spec.SoftDeletes {
		// Add timestamp to typeNames for import collection (registry key is "timestamp", not "time.Time")
		if !containsString(typeNames, "timestamp") {
			typeNames = append(typeNames, "timestamp")
		}
		data.Fields = append(data.Fields, FieldData{
			Name: "DeletedAt",
			Type: "*time.Time",
			Tags: "`json:\"deleted_at,omitempty\" db:\"deleted_at\"`",
		})
	}

	// Collect imports from type registry
	data.Imports = types.CollectImports(typeNames)

	return data
}

// buildTagString creates the complete struct tag string for a field
func buildTagString(field schema.Field) string {
	if len(field.Tags) == 0 {
		// Default: just JSON tag
		jsonTag := field.JSON
		if jsonTag == "" {
			jsonTag = generator.SnakeCase(field.Name)
		}
		return fmt.Sprintf("`json:\"%s\"`", jsonTag)
	}

	// Build custom tags
	var parts []string

	// Always include JSON if not specified
	if _, ok := field.Tags["json"]; !ok {
		jsonTag := field.JSON
		if jsonTag == "" {
			jsonTag = generator.SnakeCase(field.Name)
		}
		parts = append(parts, fmt.Sprintf("json:\"%s\"", jsonTag))
	}

	// Add custom tags in consistent order
	tagOrder := []string{"json", "db", "validate", "xml", "yaml"}
	for _, key := range tagOrder {
		if val, ok := field.Tags[key]; ok {
			parts = append(parts, fmt.Sprintf("%s:\"%s\"", key, val))
		}
	}

	// Add any other tags not in the standard order
	for key, val := range field.Tags {
		if !contains(tagOrder, key) {
			parts = append(parts, fmt.Sprintf("%s:\"%s\"", key, val))
		}
	}

	return "`" + strings.Join(parts, " ") + "`"
}

// contains checks if a string slice contains a specific item
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// containsString checks if a string slice contains a specific item (alias for contains)
func containsString(slice []string, item string) bool {
	return contains(slice, item)
}

// generateJSONTag creates a JSON tag for a field
func generateJSONTag(field schema.Field) string {
	// If explicitly set, use it
	if field.JSON != "" {
		return field.JSON
	}

	// Auto-generate as snake_case
	return generator.SnakeCase(field.Name)
}

// inferPackage determines the package name from the output path
func inferPackage(outputPath string) string {
	// Get directory name
	dir := filepath.Dir(outputPath)

	// Extract last component
	// internal/models/user.go → models
	// internal/handlers/user.go → handlers
	pkg := filepath.Base(dir)

	// Default to "models" if can't infer
	if pkg == "." || pkg == "/" || pkg == "" {
		return "models"
	}

	return pkg
}
