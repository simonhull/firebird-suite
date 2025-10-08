package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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
		return !fi.IsDir() && filepath.Ext(fi.Name()) == ".go" && fi.Name()[0] != '.'
	}, parser.ParseComments)

	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		for filePath, astFile := range pkg.Files {
			files = append(files, &File{
				Path:    filePath,
				Package: pkg.Name,
				Doc:     astFile.Doc.Text(),
				AST:     astFile,
			})
		}
	}

	return files, nil
}

// ParsePackage extracts package information from parsed files
func (p *Parser) ParsePackage(files []*File) (*Package, error) {
	if len(files) == 0 {
		return nil, nil
	}

	pkg := &Package{
		Name:  files[0].Package,
		Files: files,
	}

	// TODO: Extract types, functions, variables, constants from AST
	// This will be implemented in the next phase

	return pkg, nil
}
