package astutil

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
)

// FileModifier handles safe AST modifications with automatic backup/rollback
type FileModifier struct {
	path    string
	fset    *token.FileSet
	file    *ast.File
	backup  []byte
	changes []Modification
}

// Modification represents a single AST change
type Modification interface {
	Apply(fset *token.FileSet, file *ast.File) error
}

// Position specifies where to insert a new declaration
type Position int

const (
	PositionEnd Position = iota // Append to end of file
	PositionAfter               // After a specific type
	PositionBefore              // Before a specific type
)

// NewFileModifier creates a new file modifier with backup
func NewFileModifier(path string) (*FileModifier, error) {
	// Read original file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Parse file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	return &FileModifier{
		path:    path,
		fset:    fset,
		file:    file,
		backup:  content,
		changes: []Modification{},
	}, nil
}

// AddStructField adds a field to an existing struct
func (m *FileModifier) AddStructField(structName, fieldName, fieldType, tag string) error {
	mod := &addStructFieldMod{
		structName: structName,
		fieldName:  fieldName,
		fieldType:  fieldType,
		tag:        tag,
	}
	m.changes = append(m.changes, mod)
	return nil
}

// AddTypeDecl adds a new type declaration
func (m *FileModifier) AddTypeDecl(typeSpec *ast.TypeSpec, position Position, afterType string) error {
	mod := &addTypeDeclMod{
		typeSpec:  typeSpec,
		position:  position,
		afterType: afterType,
	}
	m.changes = append(m.changes, mod)
	return nil
}

// AddImport adds an import if not already present
func (m *FileModifier) AddImport(path, alias string) error {
	mod := &addImportMod{
		path:  path,
		alias: alias,
	}
	m.changes = append(m.changes, mod)
	return nil
}

// Apply executes all changes and validates
func (m *FileModifier) Apply() error {
	for i, change := range m.changes {
		if err := change.Apply(m.fset, m.file); err != nil {
			return fmt.Errorf("applying change %d: %w", i, err)
		}

		// Validate after each change
		if err := validateAST(m.fset, m.file); err != nil {
			return fmt.Errorf("validation failed after change %d: %w", i, err)
		}
	}
	return nil
}

// Write writes the modified file to disk
func (m *FileModifier) Write() error {
	// Format the AST
	var buf bytes.Buffer
	if err := format.Node(&buf, m.fset, m.file); err != nil {
		return fmt.Errorf("formatting AST: %w", err)
	}

	// Final syntax validation
	if err := ValidateSyntax(buf.Bytes()); err != nil {
		return fmt.Errorf("final validation failed: %w", err)
	}

	// Write to file
	if err := os.WriteFile(m.path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// Rollback restores the original file content
func (m *FileModifier) Rollback() error {
	if err := os.WriteFile(m.path, m.backup, 0644); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	return nil
}

// addStructFieldMod implements Modification for adding struct fields
type addStructFieldMod struct {
	structName string
	fieldName  string
	fieldType  string
	tag        string
}

func (mod *addStructFieldMod) Apply(fset *token.FileSet, file *ast.File) error {
	var found bool
	var targetStruct *ast.StructType

	// Find the struct
	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok && typeSpec.Name.Name == mod.structName {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				targetStruct = structType
				found = true
				return false
			}
		}
		return true
	})

	if !found {
		return fmt.Errorf("struct %s not found", mod.structName)
	}

	// Check if field already exists
	for _, field := range targetStruct.Fields.List {
		for _, name := range field.Names {
			if name.Name == mod.fieldName {
				// Field already exists, skip (idempotent)
				return nil
			}
		}
	}

	// Create new field
	field := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(mod.fieldName)},
		Type:  parseTypeExpr(mod.fieldType),
	}

	if mod.tag != "" {
		field.Tag = &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf("`%s`", mod.tag),
		}
	}

	// Add field to struct
	targetStruct.Fields.List = append(targetStruct.Fields.List, field)

	return nil
}

// addTypeDeclMod implements Modification for adding type declarations
type addTypeDeclMod struct {
	typeSpec  *ast.TypeSpec
	position  Position
	afterType string
}

func (mod *addTypeDeclMod) Apply(fset *token.FileSet, file *ast.File) error {
	// Check if type already exists
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == mod.typeSpec.Name.Name {
						// Type already exists, skip (idempotent)
						return nil
					}
				}
			}
		}
	}

	// Create new type declaration
	newDecl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			mod.typeSpec,
		},
	}

	// Insert at appropriate position
	switch mod.position {
	case PositionEnd:
		file.Decls = append(file.Decls, newDecl)

	case PositionAfter:
		insertPos := -1
		for i, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name.Name == mod.afterType {
							insertPos = i + 1
							break
						}
					}
				}
			}
			if insertPos != -1 {
				break
			}
		}

		if insertPos == -1 {
			// If afterType not found, append to end
			file.Decls = append(file.Decls, newDecl)
		} else {
			// Insert after found position
			file.Decls = append(file.Decls[:insertPos], append([]ast.Decl{newDecl}, file.Decls[insertPos:]...)...)
		}

	case PositionBefore:
		insertPos := -1
		for i, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name.Name == mod.afterType {
							insertPos = i
							break
						}
					}
				}
			}
			if insertPos != -1 {
				break
			}
		}

		if insertPos == -1 {
			// If beforeType not found, append to end
			file.Decls = append(file.Decls, newDecl)
		} else {
			// Insert before found position
			file.Decls = append(file.Decls[:insertPos], append([]ast.Decl{newDecl}, file.Decls[insertPos:]...)...)
		}
	}

	return nil
}

// addImportMod implements Modification for adding imports
type addImportMod struct {
	path  string
	alias string
}

func (mod *addImportMod) Apply(fset *token.FileSet, file *ast.File) error {
	// Check if import already exists
	for _, imp := range file.Imports {
		if imp.Path.Value == fmt.Sprintf(`"%s"`, mod.path) {
			// Import already exists, skip (idempotent)
			return nil
		}
	}

	// Create new import spec
	importSpec := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf(`"%s"`, mod.path),
		},
	}

	if mod.alias != "" {
		importSpec.Name = ast.NewIdent(mod.alias)
	}

	// Find or create import declaration
	var importDecl *ast.GenDecl
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			importDecl = genDecl
			break
		}
	}

	if importDecl == nil {
		// Create new import declaration at the beginning
		importDecl = &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: []ast.Spec{},
		}
		// Insert after package declaration
		if len(file.Decls) > 0 {
			file.Decls = append([]ast.Decl{importDecl}, file.Decls...)
		} else {
			file.Decls = []ast.Decl{importDecl}
		}
	}

	// Add import to declaration
	importDecl.Specs = append(importDecl.Specs, importSpec)
	file.Imports = append(file.Imports, importSpec)

	return nil
}
