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

	// 3. Determine output path
	outputPath := filepath.Join("internal", "models", generator.SnakeCase(name)+".go")
	output.Verbose(fmt.Sprintf("Output path: %s", outputPath))

	// 4. Transform schema to template data
	data := PrepareModelData(def, outputPath)
	output.Verbose(fmt.Sprintf("Prepared data with %d fields", len(data.Fields)))

	// 5. Render template
	content, err := g.renderer.RenderFS(templatesFS, "templates/model.go.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}
	output.Verbose("Template rendered successfully")

	// 6. Build operation
	var ops []generator.Operation
	ops = append(ops, &generator.WriteFileOp{
		Path:    outputPath,
		Content: content,
		Mode:    0644,
	})

	return ops, nil
}
