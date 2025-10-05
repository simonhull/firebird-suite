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
	TableName     string         `yaml:"table_name,omitempty"`
	Fields        []Field        `yaml:"fields"`
	Indexes       []Index        `yaml:"indexes,omitempty"`
	Relationships []Relationship `yaml:"relationships,omitempty"`
	Timestamps    bool           `yaml:"timestamps,omitempty"`
	SoftDeletes   bool           `yaml:"soft_deletes,omitempty"`
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

// Index represents a database index definition
type Index struct {
	Name    string   `yaml:"name,omitempty"`    // Index name (optional, will be generated if empty)
	Columns []string `yaml:"columns"`           // Column names to index
	Unique  bool     `yaml:"unique,omitempty"`  // Unique constraint
	Where   string   `yaml:"where,omitempty"`   // Partial index condition (PostgreSQL/SQLite only)
	Type    string   `yaml:"type,omitempty"`    // Index type: btree, hash, gin, gist (PostgreSQL only)
}

// Relationship represents a relationship between resources
type Relationship struct {
	Name        string `yaml:"name"`         // Relationship name (e.g., "Author", "Comments")
	Type        string `yaml:"type"`         // Relationship type: "belongs_to" or "has_many"
	Model       string `yaml:"model"`        // Target model name (e.g., "User", "Comment")
	ForeignKey  string `yaml:"foreign_key"`  // Foreign key field name (e.g., "author_id", "post_id")
	APILoadable bool   `yaml:"api_loadable"` // Allow loading via API includes (default: false, secure by default)
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

	// Validate indexes
	for i, index := range def.Spec.Indexes {
		indexPath := fmt.Sprintf("spec.indexes[%d]", i)

		// Validate columns
		if len(index.Columns) == 0 {
			errors = append(errors, ValidationError{
				Field:      fmt.Sprintf("%s.columns", indexPath),
				Message:    "at least one column is required for index",
				Suggestion: "specify column names to index",
				Line:       getLineNumber(lineMap, fmt.Sprintf("spec.indexes.%d.columns", i)),
			})
		}

		// Validate column references
		for _, colName := range index.Columns {
			found := false
			for _, field := range def.Spec.Fields {
				if field.Name == colName {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, ValidationError{
					Field:      fmt.Sprintf("%s.columns", indexPath),
					Message:    fmt.Sprintf("column '%s' not found in fields", colName),
					Suggestion: fmt.Sprintf("ensure '%s' is defined in spec.fields", colName),
					Line:       getLineNumber(lineMap, fmt.Sprintf("spec.indexes.%d.columns", i)),
				})
			}
		}
	}

	// Validate relationships
	for i, rel := range def.Spec.Relationships {
		relPath := fmt.Sprintf("spec.relationships[%d]", i)

		// 1. Validate name
		if rel.Name == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.name", relPath),
				Message: "relationship name is required",
				Line:    getLineNumber(lineMap, fmt.Sprintf("spec.relationships.%d.name", i)),
			})
		} else if !isPascalCase(rel.Name) {
			errors = append(errors, ValidationError{
				Field:      fmt.Sprintf("%s.name", relPath),
				Message:    fmt.Sprintf("relationship name '%s' should be in PascalCase", rel.Name),
				Suggestion: "use PascalCase like 'Author' or 'Comments'",
				Line:       getLineNumber(lineMap, fmt.Sprintf("spec.relationships.%d.name", i)),
			})
		}

		// 2. Validate type
		if rel.Type == "" {
			errors = append(errors, ValidationError{
				Field:      fmt.Sprintf("%s.type", relPath),
				Message:    "relationship type is required",
				Suggestion: "use 'belongs_to' or 'has_many'",
				Line:       getLineNumber(lineMap, fmt.Sprintf("spec.relationships.%d.type", i)),
			})
		} else if !ValidateRelationshipType(rel.Type) {
			errors = append(errors, ValidationError{
				Field:      fmt.Sprintf("%s.type", relPath),
				Message:    fmt.Sprintf("invalid relationship type '%s'", rel.Type),
				Suggestion: "use 'belongs_to' or 'has_many'",
				Line:       getLineNumber(lineMap, fmt.Sprintf("spec.relationships.%d.type", i)),
			})
		}

		// 3. Validate model
		if rel.Model == "" {
			errors = append(errors, ValidationError{
				Field:      fmt.Sprintf("%s.model", relPath),
				Message:    "relationship model is required",
				Suggestion: "specify the target model name (e.g., 'User', 'Comment')",
				Line:       getLineNumber(lineMap, fmt.Sprintf("spec.relationships.%d.model", i)),
			})
		} else if !isPascalCase(rel.Model) {
			errors = append(errors, ValidationError{
				Field:      fmt.Sprintf("%s.model", relPath),
				Message:    fmt.Sprintf("model name '%s' should be in PascalCase", rel.Model),
				Suggestion: "use PascalCase like 'User' or 'Comment'",
				Line:       getLineNumber(lineMap, fmt.Sprintf("spec.relationships.%d.model", i)),
			})
		}

		// 4. Validate foreign_key
		if rel.ForeignKey == "" {
			errors = append(errors, ValidationError{
				Field:      fmt.Sprintf("%s.foreign_key", relPath),
				Message:    "foreign_key is required",
				Suggestion: "specify the foreign key field name (e.g., 'author_id', 'post_id')",
				Line:       getLineNumber(lineMap, fmt.Sprintf("spec.relationships.%d.foreign_key", i)),
			})
		}

		// 5. For belongs_to: validate FK field exists in this model
		if rel.Type == "belongs_to" && rel.ForeignKey != "" {
			fkExists := false
			for _, field := range def.Spec.Fields {
				if field.Name == rel.ForeignKey {
					fkExists = true
					break
				}
			}

			if !fkExists {
				errors = append(errors, ValidationError{
					Field:      fmt.Sprintf("%s.foreign_key", relPath),
					Message:    fmt.Sprintf("foreign key field '%s' not found in fields", rel.ForeignKey),
					Suggestion: fmt.Sprintf("add a field named '%s' to spec.fields before defining this relationship", rel.ForeignKey),
					Line:       getLineNumber(lineMap, fmt.Sprintf("spec.relationships.%d.foreign_key", i)),
				})
			}
			// TODO(relationships-phase3): Validate FK type matches target model's primary key type
			// This requires loading the target schema file and comparing types
		}

		// 6. For has_many: FK field lives in the related model
		// We can't validate this without loading the related schema
		// Document this as a limitation for Phase 1
		// TODO(relationships-phase3): Optionally validate related model's schema exists and has FK field
		// For has_many, the FK field lives in the related model, which we can't validate without loading that schema
	}

	// Check for duplicate relationship names
	relationshipNames := make(map[string]int)
	for i, rel := range def.Spec.Relationships {
		if rel.Name == "" {
			continue // Already validated above
		}

		if firstIndex, exists := relationshipNames[rel.Name]; exists {
			errors = append(errors, ValidationError{
				Field:      fmt.Sprintf("spec.relationships[%d].name", i),
				Message:    fmt.Sprintf("duplicate relationship name '%s' (first defined at relationships[%d])", rel.Name, firstIndex),
				Suggestion: "each relationship must have a unique name",
				Line:       getLineNumber(lineMap, fmt.Sprintf("spec.relationships.%d.name", i)),
			})
		} else {
			relationshipNames[rel.Name] = i
		}
	}

	// Check for relationship name conflicts with field names
	for i, rel := range def.Spec.Relationships {
		if rel.Name == "" {
			continue // Already validated above
		}

		for _, field := range def.Spec.Fields {
			if field.Name == rel.Name {
				errors = append(errors, ValidationError{
					Field:      fmt.Sprintf("spec.relationships[%d].name", i),
					Message:    fmt.Sprintf("relationship name '%s' conflicts with field name", rel.Name),
					Suggestion: "choose a different relationship name or rename the field",
					Line:       getLineNumber(lineMap, fmt.Sprintf("spec.relationships.%d.name", i)),
				})
				break
			}
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
