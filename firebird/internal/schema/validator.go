package schema

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ValidatorResult holds validation results categorized by severity
type ValidatorResult struct {
	Errors   []ValidationError
	Warnings []ValidationError
	Infos    []ValidationError
}

// ExtendedValidationResult aggregates results from multiple validators
type ExtendedValidationResult struct {
	Errors   ValidationErrors
	Warnings []ValidationError
	Infos    []ValidationError
}

// HasErrors returns true if any errors are present
func (r *ExtendedValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// Error implements the error interface with formatted output
func (r *ExtendedValidationResult) Error() string {
	if len(r.Errors) == 0 && len(r.Warnings) == 0 && len(r.Infos) == 0 {
		return "validation completed with no issues"
	}

	var sb strings.Builder

	// Summary line
	if len(r.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("❌ Validation failed with %d error(s)", len(r.Errors)))
		if len(r.Warnings) > 0 {
			sb.WriteString(fmt.Sprintf(", %d warning(s)", len(r.Warnings)))
		}
	} else if len(r.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("⚠️  Validation completed with %d warning(s)", len(r.Warnings)))
	} else {
		sb.WriteString(fmt.Sprintf("ℹ️  Validation info (%d)", len(r.Infos)))
	}
	sb.WriteString("\n\n")

	// Errors
	for _, err := range r.Errors {
		sb.WriteString("✗ " + err.Error() + "\n")
	}

	// Warnings
	for _, warn := range r.Warnings {
		sb.WriteString("⚠ " + warn.Error() + "\n")
	}

	// Infos
	for _, info := range r.Infos {
		sb.WriteString("ℹ " + info.Error() + "\n")
	}

	return sb.String()
}

// Validator is the interface all validators must implement
type Validator interface {
	Name() string
	Validate(def *Definition, lineMap map[string]int) (ValidatorResult, error)
}

// ValidationPipeline orchestrates multiple validators
type ValidationPipeline struct {
	validators  []Validator
	interactive bool
}

// NewValidationPipeline creates a validation pipeline with default validators
func NewValidationPipeline(interactive bool) *ValidationPipeline {
	return &ValidationPipeline{
		validators: []Validator{
			&FieldNameValidator{},
			&TypeValidator{},
			&ForeignKeyDetector{interactive: interactive},
			&RelationshipValidator{},
		},
		interactive: interactive,
	}
}

// AddValidator adds a custom validator
func (p *ValidationPipeline) AddValidator(v Validator) {
	p.validators = append(p.validators, v)
}

// Validate runs all validators and aggregates results
func (p *ValidationPipeline) Validate(def *Definition, lineMap map[string]int) (*ExtendedValidationResult, error) {
	result := &ExtendedValidationResult{}

	for _, validator := range p.validators {
		vr, err := validator.Validate(def, lineMap)
		if err != nil {
			return nil, fmt.Errorf("validator %s failed: %w", validator.Name(), err)
		}
		result.Errors = append(result.Errors, vr.Errors...)
		result.Warnings = append(result.Warnings, vr.Warnings...)
		result.Infos = append(result.Infos, vr.Infos...)
	}

	return result, nil
}

// ValidateDefinition is a convenience function for full validation
func ValidateDefinition(def *Definition, lineMap map[string]int, interactive bool) error {
	pipeline := NewValidationPipeline(interactive)
	result, err := pipeline.Validate(def, lineMap)
	if err != nil {
		return err
	}

	// Print results
	if len(result.Errors)+len(result.Warnings)+len(result.Infos) > 0 {
		fmt.Println(result.Error())
	}

	// Return error if validation failed
	if result.HasErrors() {
		return errors.New("validation failed")
	}

	// Success message
	if len(result.Errors) == 0 && len(result.Warnings) == 0 && len(result.Infos) == 0 {
		fmt.Println("✓ Schema validation passed")
	}

	return nil
}

// PromptYesNo prompts for yes/no input
func PromptYesNo(prompt string, defaultYes bool) (bool, error) {
	reader := bufio.NewReader(os.Stdin)

	suffix := " [Y/n]: "
	if !defaultYes {
		suffix = " [y/N]: "
	}

	fmt.Print(prompt + suffix)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "" {
		return defaultYes, nil
	}

	if response == "y" || response == "yes" {
		return true, nil
	}
	if response == "n" || response == "no" {
		return false, nil
	}

	return PromptYesNo("Please enter 'y' or 'n'", defaultYes)
}
