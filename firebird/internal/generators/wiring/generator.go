package wiring

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates the wiring.go file for route registration
type Generator struct {
	projectPath string
	modulePath  string
	renderer    *generator.Renderer
}

// New creates a new wiring generator
func New(projectPath, modulePath string) *Generator {
	return &Generator{
		projectPath: projectPath,
		modulePath:  modulePath,
		renderer:    generator.NewRenderer(),
	}
}

// ResourceData represents a discovered resource
type ResourceData struct {
	Name      string // e.g., "Todo"
	NameLower string // e.g., "todo"
}

// TemplateData holds data for the wiring template
type TemplateData struct {
	ModulePath string
	Resources  []ResourceData
}

// Generate creates or updates the wiring.go file
func (g *Generator) Generate() ([]generator.Operation, error) {
	// Discover resources
	resources, err := g.discoverResources()
	if err != nil {
		return nil, fmt.Errorf("discovering resources: %w", err)
	}

	// Prepare template data
	data := TemplateData{
		ModulePath: g.modulePath,
		Resources:  resources,
	}

	// Render template
	content, err := g.renderer.RenderFS(templatesFS, "templates/wiring.go.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("rendering template: %w", err)
	}

	// Create operation
	outputPath := filepath.Join(g.projectPath, "cmd", "server", "wiring.go")
	op := &generator.WriteFileOp{
		Path:    outputPath,
		Content: content,
		Mode:    0644,
	}

	return []generator.Operation{op}, nil
}

// discoverResources scans the repositories directory for generated resources
func (g *Generator) discoverResources() ([]ResourceData, error) {
	reposDir := filepath.Join(g.projectPath, "internal", "repositories")

	// Check if directory exists
	if _, err := os.Stat(reposDir); os.IsNotExist(err) {
		// No repositories yet, return empty list
		return []ResourceData{}, nil
	}

	entries, err := os.ReadDir(reposDir)
	if err != nil {
		return nil, fmt.Errorf("reading repositories directory: %w", err)
	}

	var resources []ResourceData

	for _, entry := range entries {
		// Skip directories and non-repository files
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_repository.go") {
			continue
		}

		// Extract resource name from filename
		// e.g., "todo_repository.go" -> "todo"
		filename := strings.TrimSuffix(entry.Name(), "_repository.go")

		// Convert to PascalCase (e.g., "todo" -> "Todo", "blog_post" -> "BlogPost")
		resourceName := generator.PascalCase(filename)

		resources = append(resources, ResourceData{
			Name:      resourceName,
			NameLower: strings.ToLower(string(resourceName[0])) + resourceName[1:],
		})
	}

	return resources, nil
}

