package model

import (
	"path/filepath"

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
	Name    string // PascalCase field name (e.g., "Email")
	Type    string // Go type (e.g., "string", "uuid.UUID")
	JSONTag string // JSON tag value (e.g., "email")
}

// PrepareModelData transforms a schema definition into template data
func PrepareModelData(def *schema.Definition, outputPath string) *ModelData {
	data := &ModelData{
		Package: inferPackage(outputPath),
		Name:    def.Name,
		Fields:  make([]FieldData, len(def.Spec.Fields)),
	}

	// Collect type names for import gathering
	var typeNames []string

	// Transform fields
	for i, field := range def.Spec.Fields {
		// Look up type in registry to get the Go type
		goType, _, err := types.GetGoType(field.Type)
		if err != nil {
			// Fallback: use field.Type as-is (custom types or unknown types)
			goType = field.Type
		} else {
			// Track type name for imports
			typeNames = append(typeNames, field.Type)
		}

		data.Fields[i] = FieldData{
			Name:    generator.PascalCase(field.Name),
			Type:    goType,
			JSONTag: generateJSONTag(field),
		}
	}

	// Collect imports from type registry
	data.Imports = types.CollectImports(typeNames)

	return data
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
