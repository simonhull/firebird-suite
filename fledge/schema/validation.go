package schema

import "fmt"

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

	result := fmt.Sprintf("found %d validation errors:\n", len(e))
	for i, err := range e {
		result += fmt.Sprintf("  %d. %s\n", i+1, err.Error())
	}
	return result
}

// ValidateBasicStructure validates the common fields all schemas must have
func ValidateBasicStructure(def *Definition) error {
	var errors ValidationErrors

	if def.APIVersion == "" {
		errors = append(errors, ValidationError{
			Field:   "apiVersion",
			Message: "apiVersion is required",
		})
	}

	if def.Kind == "" {
		errors = append(errors, ValidationError{
			Field:   "kind",
			Message: "kind is required",
		})
	}

	if def.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name is required",
		})
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}