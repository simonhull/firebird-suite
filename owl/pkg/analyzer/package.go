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
}

// Type represents a Go type (struct, interface, etc.)
type Type struct {
	Name       string
	Kind       string // "struct", "interface", "alias", etc.
	Doc        string
	Fields     []*Field
	Methods    []*Function
	Package    string
	FilePath   string
	Convention *Convention // Detected convention (if any)
}

// Field represents a struct field
type Field struct {
	Name string
	Type string
	Tag  string
	Doc  string
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
	Convention *Convention // Detected convention (if any)
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

// Convention represents a detected architectural pattern
type Convention struct {
	Name     string
	Category string
	Layer    string
	Tags     []string
}
