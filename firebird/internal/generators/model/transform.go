package model

import (
	"path/filepath"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
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

	// Transform fields
	for i, field := range def.Spec.Fields {
		data.Fields[i] = FieldData{
			Name:    generator.PascalCase(field.Name),
			Type:    field.Type,
			JSONTag: generateJSONTag(field),
		}
	}

	// Detect required imports
	data.Imports = detectImports(def.Spec.Fields)

	return data
}

// detectImports determines which packages need to be imported
func detectImports(fields []schema.Field) []string {
	imports := []string{}
	needsTime := false
	needsUUID := false

	for _, field := range fields {
		// Check for time.Time
		if strings.Contains(field.Type, "time.Time") {
			needsTime = true
		}

		// Check for uuid.UUID
		if strings.Contains(field.Type, "uuid.UUID") {
			needsUUID = true
		}
	}

	// Add imports in alphabetical order
	if needsTime {
		imports = append(imports, "time")
	}
	if needsUUID {
		imports = append(imports, "github.com/google/uuid")
	}

	return imports
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
