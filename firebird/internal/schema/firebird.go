package schema

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Definition represents a parsed and validated .firebird.yml schema
type Definition struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Name       string `yaml:"name"`
	Spec       Spec   `yaml:"spec"`
}

// Spec contains the resource specification
type Spec struct {
	TableName string  `yaml:"table_name,omitempty"`
	Fields    []Field `yaml:"fields"`
}

// Field represents a single field in the resource
type Field struct {
	Name       string            `yaml:"name"`
	Type       string            `yaml:"type"`
	DBType     string            `yaml:"db_type"`
	PrimaryKey bool              `yaml:"primary_key,omitempty"`
	Unique     bool              `yaml:"unique,omitempty"`
	Index      bool              `yaml:"index,omitempty"`
	Nullable   bool              `yaml:"nullable,omitempty"`
	Required   bool              `yaml:"required,omitempty"`
	Default    any               `yaml:"default,omitempty"`
	Tags       map[string]string `yaml:"tags,omitempty"`
	Validation []string          `yaml:"validation,omitempty"`
	JSON       string            `yaml:"json,omitempty"`
	AutoNowAdd bool              `yaml:"auto_now_add,omitempty"`
	AutoNow    bool              `yaml:"auto_now,omitempty"`
}

// ValidationError represents a schema validation error with context
type ValidationError struct {
	Field      string // Field path (e.g., "spec.fields[0].name")
	Message    string // Error message
	Suggestion string // Helpful suggestion (optional)
	Line       int    // Line number in YAML (if available)
}

// Error returns a formatted error message
func (e *ValidationError) Error() string {
	var msg string
	if e.Line > 0 {
		msg = fmt.Sprintf("validation error at %s (line %d): %s", e.Field, e.Line, e.Message)
	} else {
		msg = fmt.Sprintf("validation error at %s: %s", e.Field, e.Message)
	}
	if e.Suggestion != "" {
		msg += fmt.Sprintf(". Suggestion: %s", e.Suggestion)
	}
	return msg
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error returns all validation errors formatted with clear separation
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "validation errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("found %d validation errors:\n", len(e)))
	for i, err := range e {
		buf.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return buf.String()
}

// Parse reads and validates a schema file
func Parse(path string) (*Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	return ParseBytes(data)
}

// ParseBytes reads and validates schema from bytes
func ParseBytes(data []byte) (*Definition, error) {
	// First pass: parse with node API to get line numbers
	var rootNode yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&rootNode); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Build line number map
	lineMap := make(map[string]int)
	extractLineNumbers(&rootNode, "", lineMap)

	// Second pass: strict parsing with KnownFields
	var def Definition
	decoder = yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true) // Enable strict mode to catch unknown fields

	if err := decoder.Decode(&def); err != nil {
		// Try to extract field name from error message
		return nil, fmt.Errorf("failed to parse schema (check for unknown/misspelled fields): %w", err)
	}

	// Validate the parsed definition
	if err := ValidateWithLineNumbers(&def, lineMap); err != nil {
		return nil, err
	}

	return &def, nil
}

// Validate validates a parsed schema
func Validate(def *Definition) error {
	return ValidateWithLineNumbers(def, nil)
}

// ValidateWithLineNumbers validates a parsed schema with optional line number information
func ValidateWithLineNumbers(def *Definition, lineMap map[string]int) error {
	var errors ValidationErrors

	// Validate top-level fields
	if def.APIVersion == "" {
		errors = append(errors, ValidationError{
			Field:   "apiVersion",
			Message: "apiVersion is required",
			Line:    getLineNumber(lineMap, "apiVersion"),
		})
	} else if def.APIVersion != "v1" {
		errors = append(errors, ValidationError{
			Field:      "apiVersion",
			Message:    fmt.Sprintf("invalid apiVersion '%s'", def.APIVersion),
			Suggestion: "use 'v1'",
			Line:       getLineNumber(lineMap, "apiVersion"),
		})
	}

	if def.Kind == "" {
		errors = append(errors, ValidationError{
			Field:   "kind",
			Message: "kind is required",
			Line:    getLineNumber(lineMap, "kind"),
		})
	} else if def.Kind != "Resource" {
		errors = append(errors, ValidationError{
			Field:      "kind",
			Message:    fmt.Sprintf("invalid kind '%s'", def.Kind),
			Suggestion: "use 'Resource'",
			Line:       getLineNumber(lineMap, "kind"),
		})
	}

	if def.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name is required",
			Line:    getLineNumber(lineMap, "name"),
		})
	} else if !isPascalCase(def.Name) {
		errors = append(errors, ValidationError{
			Field:      "name",
			Message:    fmt.Sprintf("name '%s' should be in PascalCase", def.Name),
			Suggestion: "use PascalCase like 'User' or 'BlogPost'",
			Line:       getLineNumber(lineMap, "name"),
		})
	}

	// Validate fields
	if len(def.Spec.Fields) == 0 {
		errors = append(errors, ValidationError{
			Field:      "spec.fields",
			Message:    "at least one field is required",
			Suggestion: "add fields to define your resource structure",
			Line:       getLineNumber(lineMap, "spec.fields"),
		})
	} else {
		// Check for primary key
		hasPrimaryKey := false
		for i, field := range def.Spec.Fields {
			fieldPath := fmt.Sprintf("spec.fields[%d]", i)

			// Validate field name
			if field.Name == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s.name", fieldPath),
					Message: "field name is required",
					Line:    getLineNumber(lineMap, fmt.Sprintf("spec.fields.%d.name", i)),
				})
			}

			// Validate field type
			if field.Type == "" {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("%s.type", fieldPath),
					Message: "field type is required",
					Line:    getLineNumber(lineMap, fmt.Sprintf("spec.fields.%d.type", i)),
				})
			} else if !IsValidGoType(field.Type) {
				errors = append(errors, ValidationError{
					Field:      fmt.Sprintf("%s.type", fieldPath),
					Message:    fmt.Sprintf("invalid Go type '%s'", field.Type),
					Suggestion: "use a valid Go type like 'string', 'int', 'bool', 'time.Time', etc.",
					Line:       getLineNumber(lineMap, fmt.Sprintf("spec.fields.%d.type", i)),
				})
			}

			// Validate db_type
			if field.DBType == "" {
				errors = append(errors, ValidationError{
					Field:      fmt.Sprintf("%s.db_type", fieldPath),
					Message:    "db_type is required",
					Suggestion: "specify the database column type (e.g., 'VARCHAR(255)', 'INTEGER', 'TIMESTAMP')",
					Line:       getLineNumber(lineMap, fmt.Sprintf("spec.fields.%d.db_type", i)),
				})
			}

			// Check primary key
			if field.PrimaryKey {
				hasPrimaryKey = true
			}

			// Validate auto_now fields
			if field.AutoNow || field.AutoNowAdd {
				if field.Type != "time.Time" && field.Type != "*time.Time" {
					errors = append(errors, ValidationError{
						Field:      fieldPath,
						Message:    fmt.Sprintf("auto_now/auto_now_add requires type to be 'time.Time' or '*time.Time', got '%s'", field.Type),
						Suggestion: "change type to 'time.Time' or remove auto_now/auto_now_add",
						Line:       getLineNumber(lineMap, fmt.Sprintf("spec.fields.%d.type", i)),
					})
				}
			}

			// Warn about nullable with non-pointer types
			if field.Nullable && !IsPointerType(field.Type) {
				errors = append(errors, ValidationError{
					Field:      fmt.Sprintf("%s.type", fieldPath),
					Message:    fmt.Sprintf("nullable field should use pointer type, got '%s'", field.Type),
					Suggestion: fmt.Sprintf("consider using '*%s' for nullable fields", field.Type),
					Line:       getLineNumber(lineMap, fmt.Sprintf("spec.fields.%d.type", i)),
				})
			}

			// Validate validation rules
			if len(field.Validation) > 0 {
				if err := ValidateValidationRules(field.Validation); err != nil {
					errors = append(errors, ValidationError{
						Field:   fmt.Sprintf("%s.validation", fieldPath),
						Message: err.Error(),
						Line:    getLineNumber(lineMap, fmt.Sprintf("spec.fields.%d.validation", i)),
					})
				}
			}
		}

		if !hasPrimaryKey {
			errors = append(errors, ValidationError{
				Field:      "spec.fields",
				Message:    "at least one field must have primary_key: true",
				Suggestion: "mark your ID field with 'primary_key: true'",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// extractLineNumbers walks the YAML node tree and builds a map of field paths to line numbers
func extractLineNumbers(node *yaml.Node, path string, lineMap map[string]int) {
	if node == nil {
		return
	}

	// Store line number for current path
	if path != "" {
		lineMap[path] = node.Line
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			extractLineNumbers(node.Content[0], path, lineMap)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 < len(node.Content) {
				key := node.Content[i].Value
				newPath := path
				if newPath == "" {
					newPath = key
				} else {
					newPath = fmt.Sprintf("%s.%s", path, key)
				}
				extractLineNumbers(node.Content[i+1], newPath, lineMap)
			}
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			newPath := fmt.Sprintf("%s.%d", path, i)
			extractLineNumbers(child, newPath, lineMap)
		}
	}
}

// getLineNumber retrieves the line number for a given path
func getLineNumber(lineMap map[string]int, path string) int {
	if lineMap == nil {
		return 0
	}
	return lineMap[path]
}

// isPascalCase checks if a string is in PascalCase
func isPascalCase(s string) bool {
	if s == "" {
		return false
	}
	// First character must be uppercase
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	// Rest must be alphanumeric (numbers allowed in PascalCase)
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}
