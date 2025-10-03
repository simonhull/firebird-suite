package model

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/output"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates Go model structs from schemas
type Generator struct {
	renderer *generator.Renderer
}

// GenerateOptions holds custom options for model generation
type GenerateOptions struct {
	Name       string // Resource name
	SchemaPath string // Custom schema file path (optional)
	OutputPath string // Custom output file path (optional)
	Package    string // Custom package name (optional)
}

// NewGenerator creates a new model generator
func NewGenerator() *Generator {
	return &Generator{
		renderer: generator.NewRenderer(),
	}
}

// Generate generates a Go model struct for the given resource name
// Returns a slice of Operations to be executed by the caller
func (g *Generator) Generate(name string) ([]generator.Operation, error) {
	output.Verbose(fmt.Sprintf("Looking for schema file: %s", name))

	// 1. Find schema file
	schemaPath, err := FindSchemaFile(name)
	if err != nil {
		return nil, err
	}
	output.Verbose(fmt.Sprintf("Found schema: %s", schemaPath))

	// 2. Parse schema
	def, err := schema.Parse(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}
	output.Verbose(fmt.Sprintf("Parsed schema for: %s", def.Name))

	return g.generateFromDefinition(name, def, "", "")
}

// GenerateFromSchema generates a Go model struct from an in-memory schema definition
// This is used by the scaffold generator with --generate flag
func (g *Generator) GenerateFromSchema(name string, def *schema.Definition) ([]generator.Operation, error) {
	output.Verbose(fmt.Sprintf("Generating model from in-memory schema: %s", name))
	return g.generateFromDefinition(name, def, "", "")
}

// GenerateWithOptions generates a Go model struct with custom options
func (g *Generator) GenerateWithOptions(opts GenerateOptions) ([]generator.Operation, error) {
	output.Verbose(fmt.Sprintf("Generating model with custom options: %s", opts.Name))

	// 1. Find schema file (use custom path if provided)
	schemaPath := opts.SchemaPath
	if schemaPath == "" {
		var err error
		schemaPath, err = FindSchemaFile(opts.Name)
		if err != nil {
			return nil, err
		}
	}
	output.Verbose(fmt.Sprintf("Using schema: %s", schemaPath))

	// 2. Parse schema
	def, err := schema.Parse(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}
	output.Verbose(fmt.Sprintf("Parsed schema for: %s", def.Name))

	// 3. Generate with custom output path and package
	return g.generateFromDefinition(opts.Name, def, opts.OutputPath, opts.Package)
}

// generateFromDefinition is the common implementation for both Generate methods
// customOutputPath and customPackage are optional - empty strings use defaults
func (g *Generator) generateFromDefinition(name string, def *schema.Definition, customOutputPath, customPackage string) ([]generator.Operation, error) {
	// 1. Determine output path
	outputPath := customOutputPath
	if outputPath == "" {
		outputPath = filepath.Join("internal", "models", generator.SnakeCase(name)+".go")
	}
	output.Verbose(fmt.Sprintf("Output path: %s", outputPath))

	// 2. Transform schema to template data
	data := PrepareModelData(def, outputPath)

	// Override package if custom package provided
	if customPackage != "" {
		data.Package = customPackage
		output.Verbose(fmt.Sprintf("Using custom package: %s", customPackage))
	}

	output.Verbose(fmt.Sprintf("Prepared data with %d fields", len(data.Fields)))

	// 3. Render template
	content, err := g.renderer.RenderFS(templatesFS, "templates/model.go.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}
	output.Verbose("Template rendered successfully")

	// 4. Build operation
	var ops []generator.Operation
	ops = append(ops, &generator.WriteFileOp{
		Path:    outputPath,
		Content: content,
		Mode:    0644,
	})

	return ops, nil
}
