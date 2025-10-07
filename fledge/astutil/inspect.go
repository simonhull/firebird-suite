package astutil

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

// HasField checks if a struct has a specific field
func HasField(filePath, structName, fieldName string) (bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("reading file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("parsing file: %w", err)
	}

	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == structName {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				for _, field := range structType.Fields.List {
					for _, name := range field.Names {
						if name.Name == fieldName {
							found = true
							return false
						}
					}
				}
			}
		}
		return true
	})

	return found, nil
}

// HasTypeDecl checks if a type declaration exists
func HasTypeDecl(filePath, typeName string) (bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("reading file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("parsing file: %w", err)
	}

	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if typeSpec.Name.Name == typeName {
				found = true
				return false
			}
		}
		return true
	})

	return found, nil
}

// HasImport checks if an import is present
func HasImport(filePath, importPath string) (bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("reading file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("parsing file: %w", err)
	}

	for _, imp := range file.Imports {
		// Remove quotes from import path
		path := imp.Path.Value[1 : len(imp.Path.Value)-1]
		if path == importPath {
			return true, nil
		}
	}

	return false, nil
}

// FindStructPosition finds the position to insert a new type after a specific type
func FindStructPosition(file *ast.File, afterType string) token.Pos {
	var pos token.Pos

	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if typeSpec.Name.Name == afterType {
				pos = typeSpec.End()
				return false
			}
		}
		return true
	})

	return pos
}

// GetStructType returns the ast.StructType for a given struct name
func GetStructType(file *ast.File, structName string) (*ast.StructType, error) {
	var structType *ast.StructType

	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == structName {
			if st, ok := typeSpec.Type.(*ast.StructType); ok {
				structType = st
				return false
			}
		}
		return true
	})

	if structType == nil {
		return nil, fmt.Errorf("struct %s not found", structName)
	}

	return structType, nil
}

// GetTypeSpec returns the ast.TypeSpec for a given type name
func GetTypeSpec(file *ast.File, typeName string) (*ast.TypeSpec, error) {
	var typeSpec *ast.TypeSpec

	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == typeName {
			typeSpec = ts
			return false
		}
		return true
	})

	if typeSpec == nil {
		return nil, fmt.Errorf("type %s not found", typeName)
	}

	return typeSpec, nil
}

// ListStructFields returns all field names in a struct
func ListStructFields(filePath, structName string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	var fields []string
	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == structName {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				for _, field := range structType.Fields.List {
					for _, name := range field.Names {
						fields = append(fields, name.Name)
					}
				}
			}
		}
		return true
	})

	return fields, nil
}

// ListTypes returns all type names declared in a file
func ListTypes(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	var types []string
	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			types = append(types, typeSpec.Name.Name)
		}
		return true
	})

	return types, nil
}
