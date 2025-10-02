package model

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/output"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates Go model structs from schemas
type Generator struct {
	resolver *generator.Resolver
	renderer *generator.Renderer
}

// NewGenerator creates a new model generator with the given conflict resolver
func NewGenerator(resolver *generator.Resolver) *Generator {
	return &Generator{
		resolver: resolver,
		renderer: generator.NewRenderer(),
	}
}

// Generate generates a Go model struct for the given resource name
func (g *Generator) Generate(name string) error {
	output.Verbose(fmt.Sprintf("Looking for schema file: %s", name))

	// 1. Find schema file
	schemaPath, err := FindSchemaFile(name)
	if err != nil {
		return err
	}
	output.Verbose(fmt.Sprintf("Found schema: %s", schemaPath))

	// 2. Parse schema
	def, err := schema.Parse(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
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
		return fmt.Errorf("failed to render template: %w", err)
	}
	output.Verbose("Template rendered successfully")

	// 6. Handle existing file (conflict resolution)
	if _, err := os.Stat(outputPath); err == nil {
		existing, err := os.ReadFile(outputPath)
		if err != nil {
			return fmt.Errorf("failed to read existing file: %w", err)
		}

		resolution, err := g.resolver.ResolveConflict(outputPath, existing, content)
		if err != nil {
			return err
		}

		switch resolution {
		case generator.Skip:
			output.Info(fmt.Sprintf("Skipped %s (already exists)", outputPath))
			return nil
		case generator.Cancel:
			return fmt.Errorf("generation cancelled by user")
		}
		// Overwrite or ShowDiff → continue to write
	}

	// 7. Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 8. Write file
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	output.Verbose(fmt.Sprintf("Wrote file: %s", outputPath))
	return nil
}
