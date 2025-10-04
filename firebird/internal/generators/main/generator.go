package appgen

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates main.go
type Generator struct {
	renderer   *generator.Renderer
	pkgPath    string
	modulePath string
}

// New creates a new main generator
func New(pkgPath, modulePath string) *Generator {
	return &Generator{
		renderer:   generator.NewRenderer(),
		pkgPath:    pkgPath,
		modulePath: modulePath,
	}
}

// Generate generates the main.go file
func (g *Generator) Generate() ([]generator.Operation, error) {
	var ops []generator.Operation

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	// Generate main.go
	mainContent, err := g.renderer.RenderFS(templatesFS, "templates/main.go.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("failed to render main.go: %w", err)
	}

	mainPath := filepath.Join(g.pkgPath, "cmd", "server", "main.go")

	// Create cmd/server directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(mainPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create cmd/server directory: %w", err)
	}

	ops = append(ops, &generator.WriteFileOp{
		Path:    mainPath,
		Content: mainContent,
		Mode:    0644,
	})

	return ops, nil
}
