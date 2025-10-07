package astutil

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// validateAST checks if an AST is valid Go code
func validateAST(fset *token.FileSet, file *ast.File) error {
	// Check for nil nodes
	var errors []string

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		// Check for nil identifiers
		switch node := n.(type) {
		case *ast.Ident:
			if node.Name == "" {
				errors = append(errors, "found empty identifier")
			}
		case *ast.Field:
			if node.Type == nil {
				errors = append(errors, "found field with nil type")
			}
		case *ast.TypeSpec:
			if node.Name == nil {
				errors = append(errors, "found type spec with nil name")
			}
			if node.Type == nil {
				errors = append(errors, "found type spec with nil type")
			}
		case *ast.FuncDecl:
			if node.Name == nil {
				errors = append(errors, "found function with nil name")
			}
		case *ast.ImportSpec:
			if node.Path == nil {
				errors = append(errors, "found import with nil path")
			}
		}

		return true
	})

	if len(errors) > 0 {
		return fmt.Errorf("AST validation failed: %v", errors)
	}

	return nil
}

// ValidateSyntax parses bytes to ensure valid Go syntax
func ValidateSyntax(content []byte) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", content, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("syntax validation failed: %w", err)
	}
	return nil
}

// ValidateStructField checks if a struct field is valid
func ValidateStructField(field *ast.Field) error {
	if field == nil {
		return fmt.Errorf("field is nil")
	}

	if field.Type == nil {
		return fmt.Errorf("field type is nil")
	}

	if len(field.Names) == 0 {
		return fmt.Errorf("field has no names (embedded fields not supported)")
	}

	for _, name := range field.Names {
		if name == nil || name.Name == "" {
			return fmt.Errorf("field has nil or empty name")
		}
	}

	return nil
}

// ValidateTypeSpec checks if a type spec is valid
func ValidateTypeSpec(spec *ast.TypeSpec) error {
	if spec == nil {
		return fmt.Errorf("type spec is nil")
	}

	if spec.Name == nil || spec.Name.Name == "" {
		return fmt.Errorf("type spec has nil or empty name")
	}

	if spec.Type == nil {
		return fmt.Errorf("type spec has nil type")
	}

	// Validate struct type if applicable
	if structType, ok := spec.Type.(*ast.StructType); ok {
		if structType.Fields == nil {
			return fmt.Errorf("struct type has nil fields")
		}
		for i, field := range structType.Fields.List {
			if err := ValidateStructField(field); err != nil {
				return fmt.Errorf("field %d: %w", i, err)
			}
		}
	}

	return nil
}

// ValidateImport checks if an import spec is valid
func ValidateImport(spec *ast.ImportSpec) error {
	if spec == nil {
		return fmt.Errorf("import spec is nil")
	}

	if spec.Path == nil || spec.Path.Value == "" {
		return fmt.Errorf("import spec has nil or empty path")
	}

	// Check that path is quoted
	path := spec.Path.Value
	if len(path) < 2 || path[0] != '"' || path[len(path)-1] != '"' {
		return fmt.Errorf("import path must be quoted: %s", path)
	}

	return nil
}

// CheckCircularReferences looks for obvious circular type references
// (simplified implementation)
func CheckCircularReferences(file *ast.File) error {
	typeRefs := make(map[string][]string)

	// Build type reference graph
	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			typeName := typeSpec.Name.Name
			refs := []string{}

			// Find references in struct fields
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				for _, field := range structType.Fields.List {
					if ident, ok := field.Type.(*ast.Ident); ok {
						refs = append(refs, ident.Name)
					}
				}
			}

			typeRefs[typeName] = refs
		}
		return true
	})

	// Simple check for direct self-references
	for typeName, refs := range typeRefs {
		for _, ref := range refs {
			if ref == typeName {
				return fmt.Errorf("type %s directly references itself", typeName)
			}
		}
	}

	return nil
}
