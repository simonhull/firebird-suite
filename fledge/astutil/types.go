package astutil

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// StructFieldBuilder creates struct fields with a fluent API
type StructFieldBuilder struct {
	name string
	typ  ast.Expr
	tag  string
	doc  string
}

// NewField creates a new field builder
func NewField(name string) *StructFieldBuilder {
	return &StructFieldBuilder{
		name: name,
	}
}

// Type sets the field type
func (b *StructFieldBuilder) Type(typeName string) *StructFieldBuilder {
	b.typ = parseTypeExpr(typeName)
	return b
}

// Tag sets the struct tag
func (b *StructFieldBuilder) Tag(tag string) *StructFieldBuilder {
	b.tag = tag
	return b
}

// Doc sets the documentation comment
func (b *StructFieldBuilder) Doc(doc string) *StructFieldBuilder {
	b.doc = doc
	return b
}

// Build constructs the ast.Field
func (b *StructFieldBuilder) Build() *ast.Field {
	field := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(b.name)},
		Type:  b.typ,
	}

	if b.tag != "" {
		field.Tag = &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf("`%s`", b.tag),
		}
	}

	if b.doc != "" {
		field.Doc = &ast.CommentGroup{
			List: []*ast.Comment{
				{
					Text: fmt.Sprintf("// %s", b.doc),
				},
			},
		}
	}

	return field
}

// TypeSpecBuilder creates type declarations with a fluent API
type TypeSpecBuilder struct {
	name   string
	fields []*ast.Field
	doc    string
}

// NewStruct creates a new struct type builder
func NewStruct(name string) *TypeSpecBuilder {
	return &TypeSpecBuilder{
		name:   name,
		fields: []*ast.Field{},
	}
}

// AddField adds a field to the struct
func (b *TypeSpecBuilder) AddField(field *ast.Field) *TypeSpecBuilder {
	b.fields = append(b.fields, field)
	return b
}

// AddFieldSimple adds a field with just name and type
func (b *TypeSpecBuilder) AddFieldSimple(name, typeName, tag string) *TypeSpecBuilder {
	field := NewField(name).Type(typeName).Tag(tag).Build()
	b.fields = append(b.fields, field)
	return b
}

// Doc sets the documentation comment
func (b *TypeSpecBuilder) Doc(doc string) *TypeSpecBuilder {
	b.doc = doc
	return b
}

// Build constructs the ast.TypeSpec
func (b *TypeSpecBuilder) Build() *ast.TypeSpec {
	typeSpec := &ast.TypeSpec{
		Name: ast.NewIdent(b.name),
		Type: &ast.StructType{
			Fields: &ast.FieldList{
				List: b.fields,
			},
		},
	}

	return typeSpec
}

// BuildWithDoc constructs the ast.TypeSpec with documentation
func (b *TypeSpecBuilder) BuildWithDoc() (*ast.TypeSpec, *ast.CommentGroup) {
	typeSpec := b.Build()

	var doc *ast.CommentGroup
	if b.doc != "" {
		doc = &ast.CommentGroup{
			List: []*ast.Comment{
				{
					Text: fmt.Sprintf("// %s", b.doc),
				},
			},
		}
	}

	return typeSpec, doc
}

// parseTypeExpr converts a type string to an ast.Expr
// Supports: string, int, bool, time.Time, time.Duration, pkg.Type, *Type, []Type, map[K]V
func parseTypeExpr(typeName string) ast.Expr {
	typeName = strings.TrimSpace(typeName)

	// Handle pointer types
	if strings.HasPrefix(typeName, "*") {
		return &ast.StarExpr{
			X: parseTypeExpr(typeName[1:]),
		}
	}

	// Handle slice types
	if strings.HasPrefix(typeName, "[]") {
		return &ast.ArrayType{
			Elt: parseTypeExpr(typeName[2:]),
		}
	}

	// Handle map types (simplified)
	if strings.HasPrefix(typeName, "map[") {
		// Find the closing bracket for the key type
		closeBracket := strings.Index(typeName, "]")
		if closeBracket != -1 {
			keyType := typeName[4:closeBracket]
			valueType := typeName[closeBracket+1:]
			return &ast.MapType{
				Key:   parseTypeExpr(keyType),
				Value: parseTypeExpr(valueType),
			}
		}
	}

	// Handle qualified types (e.g., "pkg.Type")
	if dotIdx := strings.Index(typeName, "."); dotIdx != -1 {
		pkg := typeName[:dotIdx]
		name := typeName[dotIdx+1:]
		return &ast.SelectorExpr{
			X:   ast.NewIdent(pkg),
			Sel: ast.NewIdent(name),
		}
	}

	// Simple type
	return ast.NewIdent(typeName)
}

// BuildEmptyStruct creates an empty struct type
func BuildEmptyStruct(name string) *ast.TypeSpec {
	return &ast.TypeSpec{
		Name: ast.NewIdent(name),
		Type: &ast.StructType{
			Fields: &ast.FieldList{
				List: []*ast.Field{},
			},
		},
	}
}

// BuildInterface creates an interface type (simplified)
func BuildInterface(name string, methods []string) *ast.TypeSpec {
	methodList := &ast.FieldList{
		List: []*ast.Field{},
	}

	// This is a simplified implementation
	// Full implementation would need method signatures
	for _, methodName := range methods {
		methodList.List = append(methodList.List, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(methodName)},
			Type: &ast.FuncType{
				Params:  &ast.FieldList{},
				Results: &ast.FieldList{},
			},
		})
	}

	return &ast.TypeSpec{
		Name: ast.NewIdent(name),
		Type: &ast.InterfaceType{
			Methods: methodList,
		},
	}
}

// TypeAlias creates a type alias
func TypeAlias(name, target string) *ast.TypeSpec {
	return &ast.TypeSpec{
		Name:   ast.NewIdent(name),
		Type:   parseTypeExpr(target),
		Assign: token.NoPos, // Not an alias (=), just a type definition
	}
}
