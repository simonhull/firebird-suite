package astutil

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"strings"
)

// ASTModifyOp is a generator operation for AST modifications
// It implements the generator.Operation interface
type ASTModifyOp struct {
	Path          string
	Modifications []ModificationSpec
	DryRunMode    bool
}

// ModificationSpec describes a single modification to perform
type ModificationSpec struct {
	Type   string                 // "add_struct_field", "add_type_decl", "add_import"
	Params map[string]interface{} // Type-specific parameters
}

// Validate checks if the operation can be performed
func (op *ASTModifyOp) Validate(ctx context.Context, force bool) error {
	// Check if file exists
	if _, err := os.Stat(op.Path); err != nil {
		return fmt.Errorf("file not found: %s: %w", op.Path, err)
	}

	// Create a modifier to validate the file can be parsed
	_, err := NewFileModifier(op.Path)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Validate each modification spec
	for i, mod := range op.Modifications {
		if err := validateModificationSpec(mod); err != nil {
			return fmt.Errorf("modification %d: %w", i, err)
		}
	}

	return nil
}

// Execute performs the AST modifications
func (op *ASTModifyOp) Execute(ctx context.Context) error {
	// Create file modifier
	modifier, err := NewFileModifier(op.Path)
	if err != nil {
		return fmt.Errorf("creating modifier: %w", err)
	}

	// Apply each modification
	for i, modSpec := range op.Modifications {
		if err := applyModificationSpec(modifier, modSpec); err != nil {
			// Rollback on error
			if rbErr := modifier.Rollback(); rbErr != nil {
				return fmt.Errorf("modification %d failed: %w (rollback also failed: %v)", i, err, rbErr)
			}
			return fmt.Errorf("modification %d failed (rolled back): %w", i, err)
		}
	}

	// Apply all changes to AST
	if err := modifier.Apply(); err != nil {
		if rbErr := modifier.Rollback(); rbErr != nil {
			return fmt.Errorf("applying changes: %w (rollback also failed: %v)", err, rbErr)
		}
		return fmt.Errorf("applying changes (rolled back): %w", err)
	}

	// Write modified file
	if err := modifier.Write(); err != nil {
		if rbErr := modifier.Rollback(); rbErr != nil {
			return fmt.Errorf("writing file: %w (rollback also failed: %v)", err, rbErr)
		}
		return fmt.Errorf("writing file (rolled back): %w", err)
	}

	return nil
}

// Description returns a human-readable description
func (op *ASTModifyOp) Description() string {
	return fmt.Sprintf("Modify %s (%d changes)", op.Path, len(op.Modifications))
}

// DryRun simulates the operation and returns a description
func (op *ASTModifyOp) DryRun(ctx context.Context) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("AST Modifications for %s:\n", op.Path))

	for i, mod := range op.Modifications {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, describeModification(mod)))
	}

	return sb.String(), nil
}

// validateModificationSpec checks if a modification spec is valid
func validateModificationSpec(spec ModificationSpec) error {
	switch spec.Type {
	case "add_struct_field":
		required := []string{"struct_name", "field_name", "field_type"}
		for _, key := range required {
			if _, ok := spec.Params[key]; !ok {
				return fmt.Errorf("missing required parameter: %s", key)
			}
		}

	case "add_type_decl":
		if _, ok := spec.Params["type_spec"]; !ok {
			return fmt.Errorf("missing required parameter: type_spec")
		}

	case "add_import":
		if _, ok := spec.Params["path"]; !ok {
			return fmt.Errorf("missing required parameter: path")
		}

	default:
		return fmt.Errorf("unknown modification type: %s", spec.Type)
	}

	return nil
}

// applyModificationSpec applies a single modification spec to a modifier
func applyModificationSpec(modifier *FileModifier, spec ModificationSpec) error {
	switch spec.Type {
	case "add_struct_field":
		structName := spec.Params["struct_name"].(string)
		fieldName := spec.Params["field_name"].(string)
		fieldType := spec.Params["field_type"].(string)
		tag := ""
		if t, ok := spec.Params["tag"]; ok {
			tag = t.(string)
		}
		return modifier.AddStructField(structName, fieldName, fieldType, tag)

	case "add_type_decl":
		typeSpec := spec.Params["type_spec"]
		position := PositionEnd
		if p, ok := spec.Params["position"]; ok {
			position = p.(Position)
		}
		afterType := ""
		if a, ok := spec.Params["after_type"]; ok {
			afterType = a.(string)
		}

		// Type assertion for typeSpec
		var ts interface{}
		ts = typeSpec
		return modifier.AddTypeDecl(ts.(*ast.TypeSpec), position, afterType)

	case "add_import":
		path := spec.Params["path"].(string)
		alias := ""
		if a, ok := spec.Params["alias"]; ok {
			alias = a.(string)
		}
		return modifier.AddImport(path, alias)

	default:
		return fmt.Errorf("unknown modification type: %s", spec.Type)
	}
}

// describeModification returns a human-readable description of a modification
func describeModification(spec ModificationSpec) string {
	switch spec.Type {
	case "add_struct_field":
		structName := spec.Params["struct_name"].(string)
		fieldName := spec.Params["field_name"].(string)
		fieldType := spec.Params["field_type"].(string)
		return fmt.Sprintf("Add field %s %s to struct %s", fieldName, fieldType, structName)

	case "add_type_decl":
		// typeSpec := spec.Params["type_spec"]
		return "Add new type declaration"

	case "add_import":
		path := spec.Params["path"].(string)
		return fmt.Sprintf("Add import %s", path)

	default:
		return "Unknown modification"
	}
}
