package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Parser handles parsing Go source files
type Parser struct {
	fset *token.FileSet
}

// NewParser creates a new Parser
func NewParser() *Parser {
	return &Parser{
		fset: token.NewFileSet(),
	}
}

// ParseDirectory parses all Go files in a directory
func (p *Parser) ParseDirectory(path string) ([]*File, error) {
	var files []*File

	pkgs, err := parser.ParseDir(p.fset, path, func(fi os.FileInfo) bool {
		// Skip test files and hidden files
		name := fi.Name()
		return !fi.IsDir() &&
			filepath.Ext(name) == ".go" &&
			name[0] != '.' &&
			!strings.HasSuffix(name, "_test.go")
	}, parser.ParseComments)

	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		for filePath, astFile := range pkg.Files {
			file := &File{
				Path:    filePath,
				Package: pkg.Name,
				AST:     astFile,
				Imports: p.parseImports(astFile),
			}

			if astFile.Doc != nil {
				file.Doc = astFile.Doc.Text()
			}

			files = append(files, file)
		}
	}

	return files, nil
}

// parseImports extracts import statements from a file
func (p *Parser) parseImports(file *ast.File) map[string]string {
	imports := make(map[string]string)

	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)

		if imp.Name != nil {
			// Named import: import foo "github.com/bar"
			imports[imp.Name.Name] = path
		} else {
			// Default import: use last path component
			parts := strings.Split(path, "/")
			name := parts[len(parts)-1]
			imports[name] = path
		}
	}

	return imports
}

// ParsePackage extracts complete package information from parsed files
func (p *Parser) ParsePackage(files []*File) (*Package, error) {
	if len(files) == 0 {
		return nil, nil
	}

	pkg := &Package{
		Name:      files[0].Package,
		Files:     files,
		Types:     make([]*Type, 0),
		Functions: make([]*Function, 0),
		Variables: make([]*Variable, 0),
		Constants: make([]*Constant, 0),
	}

	// Extract declarations from all files
	for _, file := range files {
		p.extractDeclarations(file, pkg)
	}

	return pkg, nil
}

// extractDeclarations walks AST and extracts all declarations
func (p *Parser) extractDeclarations(file *File, pkg *Package) {
	for _, decl := range file.AST.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			p.parseGenDecl(d, file, pkg)
		case *ast.FuncDecl:
			fn := p.parseFuncDecl(d, file, pkg)
			if fn != nil {
				pkg.Functions = append(pkg.Functions, fn)
			}
		}
	}
}

// parseGenDecl handles type, const, var declarations
func (p *Parser) parseGenDecl(decl *ast.GenDecl, file *File, pkg *Package) {
	switch decl.Tok {
	case token.TYPE:
		for _, spec := range decl.Specs {
			if typeSpec, ok := spec.(*ast.TypeSpec); ok {
				typ := p.parseTypeSpec(typeSpec, decl.Doc, file, pkg)
				if typ != nil {
					pkg.Types = append(pkg.Types, typ)
				}
			}
		}

	case token.CONST:
		for _, spec := range decl.Specs {
			if valueSpec, ok := spec.(*ast.ValueSpec); ok {
				for i, name := range valueSpec.Names {
					c := &Constant{
						Name:    name.Name,
						Package: pkg.Name,
					}
					if valueSpec.Type != nil {
						c.Type = p.extractTypeName(valueSpec.Type)
					}
					if i < len(valueSpec.Values) {
						c.Value = p.exprToString(valueSpec.Values[i])
					}
					if valueSpec.Doc != nil {
						c.Doc = valueSpec.Doc.Text()
					} else if decl.Doc != nil {
						c.Doc = decl.Doc.Text()
					}
					pkg.Constants = append(pkg.Constants, c)
				}
			}
		}

	case token.VAR:
		for _, spec := range decl.Specs {
			if valueSpec, ok := spec.(*ast.ValueSpec); ok {
				for i, name := range valueSpec.Names {
					v := &Variable{
						Name:    name.Name,
						Package: pkg.Name,
					}
					if valueSpec.Type != nil {
						v.Type = p.extractTypeName(valueSpec.Type)
					}
					if i < len(valueSpec.Values) {
						v.Value = p.exprToString(valueSpec.Values[i])
					}
					if valueSpec.Doc != nil {
						v.Doc = valueSpec.Doc.Text()
					} else if decl.Doc != nil {
						v.Doc = decl.Doc.Text()
					}
					pkg.Variables = append(pkg.Variables, v)
				}
			}
		}
	}
}

// parseTypeSpec extracts complete type information
func (p *Parser) parseTypeSpec(spec *ast.TypeSpec, doc *ast.CommentGroup, file *File, pkg *Package) *Type {
	typ := &Type{
		Name:     spec.Name.Name,
		Package:  pkg.Name,
		FilePath: file.Path,
	}

	// Extract doc comment
	if spec.Doc != nil {
		typ.Doc = spec.Doc.Text()
	} else if doc != nil {
		typ.Doc = doc.Text()
	}

	// Handle generic type parameters (Go 1.18+)
	if spec.TypeParams != nil {
		typ.GenericParams = p.parseGenericParams(spec.TypeParams)
		typ.Kind = "generic"
	}

	// Determine type kind and extract details
	switch t := spec.Type.(type) {
	case *ast.StructType:
		if typ.Kind == "" {
			typ.Kind = "struct"
		}
		typ.Fields = p.parseFieldList(t.Fields)
		// Track types used in fields
		for _, field := range typ.Fields {
			typ.UsedTypes = append(typ.UsedTypes, field.Type)
		}

	case *ast.InterfaceType:
		if typ.Kind == "" {
			typ.Kind = "interface"
		}
		typ.Methods = p.parseInterfaceMethods(t.Methods)

	case *ast.Ident:
		if typ.Kind == "" {
			typ.Kind = "alias"
		}
		typ.UsedTypes = []string{t.Name}

	case *ast.SelectorExpr:
		if typ.Kind == "" {
			typ.Kind = "alias"
		}
		typ.UsedTypes = []string{p.extractTypeName(t)}

	case *ast.ArrayType, *ast.MapType, *ast.ChanType, *ast.FuncType:
		if typ.Kind == "" {
			typ.Kind = "alias"
		}
		typ.UsedTypes = []string{p.extractTypeName(t)}

	case *ast.IndexExpr, *ast.IndexListExpr:
		// Generic type instantiation
		if typ.Kind == "" {
			typ.Kind = "generic_instance"
		}
		typ.UsedTypes = []string{p.extractTypeName(t)}
	}

	return typ
}

// parseGenericParams extracts type parameters
func (p *Parser) parseGenericParams(fieldList *ast.FieldList) []GenericParam {
	var params []GenericParam

	if fieldList == nil {
		return params
	}

	for _, field := range fieldList.List {
		for _, name := range field.Names {
			param := GenericParam{
				Name: name.Name,
			}
			if field.Type != nil {
				param.Constraint = p.extractTypeName(field.Type)
			}
			params = append(params, param)
		}
	}

	return params
}

// parseFieldList extracts struct fields or function parameters
func (p *Parser) parseFieldList(fieldList *ast.FieldList) []*Field {
	var fields []*Field

	if fieldList == nil {
		return fields
	}

	for _, field := range fieldList.List {
		typeName := p.extractTypeName(field.Type)

		var tag string
		if field.Tag != nil {
			tag = field.Tag.Value
		}

		var doc string
		if field.Doc != nil {
			doc = field.Doc.Text()
		}

		if len(field.Names) == 0 {
			// Embedded field or unnamed parameter
			fields = append(fields, &Field{
				Name: "",
				Type: typeName,
				Tag:  tag,
				Doc:  doc,
			})
		} else {
			// Named fields/parameters
			for _, name := range field.Names {
				fields = append(fields, &Field{
					Name: name.Name,
					Type: typeName,
					Tag:  tag,
					Doc:  doc,
				})
			}
		}
	}

	return fields
}

// parseInterfaceMethods extracts methods from an interface
func (p *Parser) parseInterfaceMethods(fieldList *ast.FieldList) []*Function {
	var methods []*Function

	if fieldList == nil {
		return methods
	}

	for _, field := range fieldList.List {
		// Interface can have methods or embedded interfaces
		if len(field.Names) == 0 {
			// Embedded interface
			continue
		}

		for _, name := range field.Names {
			method := &Function{
				Name: name.Name,
			}

			if field.Doc != nil {
				method.Doc = field.Doc.Text()
			}

			if funcType, ok := field.Type.(*ast.FuncType); ok {
				method.Parameters = p.parseParameters(funcType.Params)
				method.Returns = p.parseParameters(funcType.Results)
				method.Signature = p.buildFuncSignature(name.Name, funcType, "")
			}

			methods = append(methods, method)
		}
	}

	return methods
}

// parseFuncDecl extracts function/method with body analysis
func (p *Parser) parseFuncDecl(decl *ast.FuncDecl, file *File, pkg *Package) *Function {
	fn := &Function{
		Name:        decl.Name.Name,
		Package:     pkg.Name,
		FilePath:    file.Path,
		Calls:       make([]string, 0),
		UsesTypes:   make([]string, 0),
		UsesImports: make([]string, 0),
	}

	// Extract doc comment
	if decl.Doc != nil {
		fn.Doc = decl.Doc.Text()
	}

	// Extract receiver (if method)
	var receiverType string
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		receiverType = p.extractTypeName(decl.Recv.List[0].Type)
		fn.Receiver = receiverType
	}

	// Parse parameters and returns
	fn.Parameters = p.parseParameters(decl.Type.Params)
	fn.Returns = p.parseParameters(decl.Type.Results)

	// Build signature
	fn.Signature = p.buildFuncSignature(decl.Name.Name, decl.Type, receiverType)

	// 🔥 DEEP PARSE: Analyze function body
	if decl.Body != nil {
		p.analyzeFunctionBody(decl.Body, fn, file)
	}

	// Deduplicate slices
	fn.Calls = uniqueStrings(fn.Calls)
	fn.UsesTypes = uniqueStrings(fn.UsesTypes)
	fn.UsesImports = uniqueStrings(fn.UsesImports)

	return fn
}

// analyzeFunctionBody performs deep analysis of function body
func (p *Parser) analyzeFunctionBody(body *ast.BlockStmt, fn *Function, file *File) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {

		case *ast.CallExpr:
			// Track function/method calls
			callName := p.extractCallName(node)
			if callName != "" {
				fn.Calls = append(fn.Calls, callName)

				// Check if it's a qualified call (pkg.Func)
				if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok {
						// Check if it's an import
						if importPath, exists := file.Imports[ident.Name]; exists {
							fn.UsesImports = append(fn.UsesImports, importPath)
						}
					}
				}
			}

		case *ast.SelectorExpr:
			// Track type usage through selectors (obj.Method, pkg.Type)
			if ident, ok := node.X.(*ast.Ident); ok {
				fn.UsesTypes = append(fn.UsesTypes, ident.Name)

				// Check imports
				if importPath, exists := file.Imports[ident.Name]; exists {
					fn.UsesImports = append(fn.UsesImports, importPath)
				}
			}

		case *ast.CompositeLit:
			// Track struct/slice/map initialization
			typeName := p.extractTypeName(node.Type)
			if typeName != "" {
				fn.UsesTypes = append(fn.UsesTypes, typeName)
			}

		case *ast.TypeAssertExpr:
			// Track type assertions: x.(Type)
			if node.Type != nil {
				typeName := p.extractTypeName(node.Type)
				fn.UsesTypes = append(fn.UsesTypes, typeName)
			}
		}

		return true
	})
}

// parseParameters extracts parameters or return values
func (p *Parser) parseParameters(fieldList *ast.FieldList) []*Parameter {
	var params []*Parameter

	if fieldList == nil {
		return params
	}

	for _, field := range fieldList.List {
		typeName := p.extractTypeName(field.Type)

		if len(field.Names) == 0 {
			// Unnamed parameter
			params = append(params, &Parameter{
				Name: "",
				Type: typeName,
			})
		} else {
			// Named parameters
			for _, name := range field.Names {
				params = append(params, &Parameter{
					Name: name.Name,
					Type: typeName,
				})
			}
		}
	}

	return params
}

// buildFuncSignature creates a readable function signature
func (p *Parser) buildFuncSignature(name string, funcType *ast.FuncType, receiver string) string {
	var sig strings.Builder

	sig.WriteString("func ")

	if receiver != "" {
		sig.WriteString("(")
		sig.WriteString(receiver)
		sig.WriteString(") ")
	}

	sig.WriteString(name)
	sig.WriteString("(")

	// Parameters
	if funcType.Params != nil {
		for i, field := range funcType.Params.List {
			if i > 0 {
				sig.WriteString(", ")
			}

			typeName := p.extractTypeName(field.Type)

			if len(field.Names) > 0 {
				for j, name := range field.Names {
					if j > 0 {
						sig.WriteString(", ")
					}
					sig.WriteString(name.Name)
				}
				sig.WriteString(" ")
			}
			sig.WriteString(typeName)
		}
	}

	sig.WriteString(")")

	// Returns
	if funcType.Results != nil && len(funcType.Results.List) > 0 {
		sig.WriteString(" ")

		if len(funcType.Results.List) > 1 {
			sig.WriteString("(")
		}

		for i, field := range funcType.Results.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			sig.WriteString(p.extractTypeName(field.Type))
		}

		if len(funcType.Results.List) > 1 {
			sig.WriteString(")")
		}
	}

	return sig.String()
}

// extractCallName extracts the name from a function call
func (p *Parser) extractCallName(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		return p.extractTypeName(fun)
	default:
		return ""
	}
}

// extractTypeName converts an ast.Expr to a type name string
func (p *Parser) extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name

	case *ast.SelectorExpr:
		pkg := p.extractTypeName(t.X)
		return pkg + "." + t.Sel.Name

	case *ast.StarExpr:
		return "*" + p.extractTypeName(t.X)

	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + p.extractTypeName(t.Elt)
		}
		return "[" + p.exprToString(t.Len) + "]" + p.extractTypeName(t.Elt)

	case *ast.MapType:
		return "map[" + p.extractTypeName(t.Key) + "]" + p.extractTypeName(t.Value)

	case *ast.ChanType:
		prefix := "chan "
		if t.Dir == ast.RECV {
			prefix = "<-chan "
		} else if t.Dir == ast.SEND {
			prefix = "chan<- "
		}
		return prefix + p.extractTypeName(t.Value)

	case *ast.FuncType:
		return "func" + p.funcTypeToString(t)

	case *ast.InterfaceType:
		return "interface{}"

	case *ast.StructType:
		return "struct{}"

	case *ast.IndexExpr:
		// Generic instantiation: Type[T]
		return p.extractTypeName(t.X) + "[" + p.extractTypeName(t.Index) + "]"

	case *ast.IndexListExpr:
		// Generic with multiple params: Type[T, U]
		var types []string
		for _, idx := range t.Indices {
			types = append(types, p.extractTypeName(idx))
		}
		return p.extractTypeName(t.X) + "[" + strings.Join(types, ", ") + "]"

	case *ast.Ellipsis:
		return "..." + p.extractTypeName(t.Elt)

	default:
		return ""
	}
}

// funcTypeToString converts function type to string
func (p *Parser) funcTypeToString(ft *ast.FuncType) string {
	var sig strings.Builder
	sig.WriteString("(")

	if ft.Params != nil {
		for i, field := range ft.Params.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			sig.WriteString(p.extractTypeName(field.Type))
		}
	}

	sig.WriteString(")")

	if ft.Results != nil && len(ft.Results.List) > 0 {
		sig.WriteString(" ")
		if len(ft.Results.List) > 1 {
			sig.WriteString("(")
		}
		for i, field := range ft.Results.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			sig.WriteString(p.extractTypeName(field.Type))
		}
		if len(ft.Results.List) > 1 {
			sig.WriteString(")")
		}
	}

	return sig.String()
}

// exprToString converts an expression to string (for const values, etc.)
func (p *Parser) exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	case *ast.BinaryExpr:
		return p.exprToString(e.X) + " " + e.Op.String() + " " + p.exprToString(e.Y)
	case *ast.UnaryExpr:
		return e.Op.String() + p.exprToString(e.X)
	default:
		return ""
	}
}

// uniqueStrings removes duplicates from a string slice
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(input))

	for _, item := range input {
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
