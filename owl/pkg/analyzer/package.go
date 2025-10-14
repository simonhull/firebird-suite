package analyzer

import "go/ast"

// Package represents an analyzed Go package
type Package struct {
	Name       string
	Path       string
	ImportPath string
	Doc        string
	Files      []*File
	Types      []*Type
	Functions  []*Function
	Variables  []*Variable
	Constants  []*Constant
	Imports    []string
}

// File represents a Go source file
type File struct {
	Path    string
	Package string
	Doc     string
	AST     *ast.File
	Imports map[string]string // alias/name -> import path
}

// Type represents a Go type (struct, interface, etc.)
type Type struct {
	Name          string
	Kind          string // "struct", "interface", "alias", "generic"
	Doc           string
	Fields        []*Field
	Methods       []*Function
	Package       string
	FilePath      string
	Line          int // Line number where type is defined
	GenericParams []GenericParam
	Convention    *Convention

	// Dependency tracking (from deep parse)
	UsedTypes []string // Type names referenced in fields
}

// Field represents a struct field
type Field struct {
	Name string
	Type string
	Tag  string
	Doc  string
}

// GenericParam represents a type parameter
type GenericParam struct {
	Name       string
	Constraint string // "any", "comparable", or custom constraint
}

// Function represents a function or method
type Function struct {
	Name       string
	Doc        string
	Signature  string
	Receiver   string
	Parameters []*Parameter
	Returns    []*Parameter
	Package    string
	FilePath   string
	Line       int // Line number where function is defined
	Convention *Convention

	// Deep parse results
	Calls       []string // Function/method names called
	UsesTypes   []string // Type names used in body
	UsesImports []string // Import packages used
}

// Parameter represents a function parameter or return value
type Parameter struct {
	Name string
	Type string
}

// Variable represents a package-level variable
type Variable struct {
	Name    string
	Type    string
	Doc     string
	Value   string
	Package string
}

// Constant represents a package-level constant
type Constant struct {
	Name    string
	Type    string
	Doc     string
	Value   string
	Package string
}

// Convention represents a detected pattern with confidence
type Convention struct {
	Name       string  // "Handler", "Service", etc.
	Category   string  // "handlers", "services" (for grouping)
	Layer      string  // "presentation", "business", "data" (if known)
	Confidence float64 // 0.0-1.0
	Reason     string  // Why we matched (for verbose mode)
	Tags       []string
}
