package schema

import (
	"fmt"
	"strings"
)

// ForeignKeyDetector identifies foreign key patterns and auto-adds FK metadata
type ForeignKeyDetector struct {
	interactive bool
}

func (v *ForeignKeyDetector) Name() string {
	return "ForeignKeyDetector"
}

func (v *ForeignKeyDetector) Validate(def *Definition, lineMap map[string]int) (ValidatorResult, error) {
	result := ValidatorResult{}

	for i, field := range def.Spec.Fields {
		// Skip if already has FK metadata
		if field.Tags != nil && field.Tags["fk"] != "" {
			continue
		}

		// Detect FK pattern: *_id field with int64/INTEGER type
		if isForeignKeyPattern(field) {
			referencedTable := extractTableName(field.Name)
			fieldPath := fmt.Sprintf("spec.fields[%d].name", i)

			// Non-interactive mode: auto-add FK constraint
			if !v.interactive {
				// Add FK metadata to field tags
				if def.Spec.Fields[i].Tags == nil {
					def.Spec.Fields[i].Tags = make(map[string]string)
				}
				def.Spec.Fields[i].Tags["fk"] = fmt.Sprintf("%s.id", referencedTable)
				def.Spec.Fields[i].Tags["fk_on_delete"] = "CASCADE"
				def.Spec.Fields[i].Tags["fk_on_update"] = "CASCADE"

				result.Infos = append(result.Infos, ValidationError{
					Field:   fieldPath,
					Message: fmt.Sprintf("Auto-detected FK: %s → %s.id (ON DELETE CASCADE)", field.Name, referencedTable),
					Line:    getLineNumber(lineMap, fmt.Sprintf("spec.fields.%d.name", i)),
				})
			} else {
				// Interactive mode: just warn (will be implemented later)
				result.Infos = append(result.Infos, ValidationError{
					Field:      fieldPath,
					Message:    fmt.Sprintf("Potential FK detected: %s → %s.id", field.Name, referencedTable),
					Suggestion: "Run in non-interactive mode to auto-add FK constraints",
					Line:       getLineNumber(lineMap, fmt.Sprintf("spec.fields.%d.name", i)),
				})
			}
		}
	}

	return result, nil
}

// isForeignKeyPattern checks if a field matches FK naming pattern
func isForeignKeyPattern(field Field) bool {
	name := strings.ToLower(field.Name)

	// Must end with _id
	if !strings.HasSuffix(name, "_id") {
		return false
	}

	// Must be int type (int, int64, or nullable versions)
	typeStr := strings.TrimPrefix(field.Type, "*")
	if typeStr != "int" && typeStr != "int64" {
		return false
	}

	return true
}

// extractTableName extracts the table name from a FK field name
// e.g., "post_id" -> "posts", "author_id" -> "authors"
func extractTableName(fieldName string) string {
	// Remove _id suffix
	name := strings.TrimSuffix(strings.ToLower(fieldName), "_id")

	// Simple pluralization
	if strings.HasSuffix(name, "s") {
		return name + "es"
	}
	if strings.HasSuffix(name, "y") {
		return strings.TrimSuffix(name, "y") + "ies"
	}
	return name + "s"
}

// FieldNameValidator checks for reserved words (minimal for demo)
type FieldNameValidator struct{}

func (v *FieldNameValidator) Name() string {
	return "FieldNameValidator"
}

func (v *FieldNameValidator) Validate(def *Definition, lineMap map[string]int) (ValidatorResult, error) {
	result := ValidatorResult{}

	// Reserved field names
	reservedNames := map[string]bool{
		"id": true, "created_at": true, "updated_at": true, "deleted_at": true,
	}

	for i, field := range def.Spec.Fields {
		fieldNameLower := strings.ToLower(field.Name)
		if reservedNames[fieldNameLower] {
			result.Warnings = append(result.Warnings, ValidationError{
				Field:      fmt.Sprintf("spec.fields[%d].name", i),
				Message:    fmt.Sprintf("field name '%s' may be auto-generated", field.Name),
				Suggestion: "check if this field should be explicit",
				Line:       getLineNumber(lineMap, fmt.Sprintf("spec.fields.%d.name", i)),
			})
		}
	}

	return result, nil
}

// TypeValidator validates type/db_type compatibility (minimal for demo)
type TypeValidator struct{}

func (v *TypeValidator) Name() string {
	return "TypeValidator"
}

func (v *TypeValidator) Validate(def *Definition, lineMap map[string]int) (ValidatorResult, error) {
	result := ValidatorResult{}

	// Basic type checking (expand as needed)
	typeCompatibility := map[string][]string{
		"string":    {"TEXT", "VARCHAR", "CHAR"},
		"*string":   {"TEXT", "VARCHAR", "CHAR"},
		"int":       {"INTEGER", "INT", "SMALLINT", "BIGINT"},
		"int64":     {"INTEGER", "INT", "BIGINT"},
		"*int":      {"INTEGER", "INT", "SMALLINT", "BIGINT"},
		"*int64":    {"INTEGER", "INT", "BIGINT"},
		"bool":      {"BOOLEAN", "BOOL"},
		"time.Time": {"TIMESTAMP", "DATETIME"},
	}

	for i, field := range def.Spec.Fields {
		compatibleTypes, found := typeCompatibility[field.Type]
		if !found {
			continue // Skip unknown types for demo
		}

		// Check DB type compatibility
		dbTypeUpper := strings.ToUpper(strings.Split(field.DBType, "(")[0])
		isCompatible := false
		for _, compatType := range compatibleTypes {
			if strings.HasPrefix(dbTypeUpper, compatType) {
				isCompatible = true
				break
			}
		}

		if !isCompatible {
			result.Errors = append(result.Errors, ValidationError{
				Field:      fmt.Sprintf("spec.fields[%d].db_type", i),
				Message:    fmt.Sprintf("db_type '%s' incompatible with Go type '%s'", field.DBType, field.Type),
				Suggestion: fmt.Sprintf("use one of: %s", strings.Join(compatibleTypes, ", ")),
				Line:       getLineNumber(lineMap, fmt.Sprintf("spec.fields.%d.db_type", i)),
			})
		}
	}

	return result, nil
}

// RelationshipValidator validates relationships (minimal for demo)
type RelationshipValidator struct{}

func (v *RelationshipValidator) Name() string {
	return "RelationshipValidator"
}

func (v *RelationshipValidator) Validate(def *Definition, lineMap map[string]int) (ValidatorResult, error) {
	result := ValidatorResult{}

	// For demo, just check that belongs_to relationships have FK fields
	for i, rel := range def.Spec.Relationships {
		if rel.Type == "belongs_to" && rel.ForeignKey != "" {
			fkField := findFieldByName(def.Spec.Fields, rel.ForeignKey)
			if fkField != nil && (fkField.Tags == nil || fkField.Tags["fk"] == "") {
				result.Infos = append(result.Infos, ValidationError{
					Field:   fmt.Sprintf("spec.relationships[%d].foreign_key", i),
					Message: fmt.Sprintf("FK field '%s' has no database constraint", rel.ForeignKey),
					Line:    getLineNumber(lineMap, fmt.Sprintf("spec.relationships.%d.foreign_key", i)),
				})
			}
		}
	}

	return result, nil
}

// findFieldByName finds a field by name
func findFieldByName(fields []Field, name string) *Field {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}
