package logging

import (
	"embed"
	"path/filepath"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates logging utilities
type Generator struct {
	renderer *generator.Renderer
	pkgPath  string
}

// New creates a new logging generator
func New(pkgPath string) *Generator {
	return &Generator{
		renderer: generator.NewRenderer(),
		pkgPath:  pkgPath,
	}
}

// Generate generates the logging package
func (g *Generator) Generate() ([]generator.Operation, error) {
	var ops []generator.Operation

	// Generate console handler
	consoleContent, err := g.renderer.RenderFS(templatesFS, "templates/console.go.tmpl", nil)
	if err != nil {
		return nil, err
	}

	consolePath := filepath.Join(g.pkgPath, "internal", "logging", "console.go")
	ops = append(ops, &generator.WriteFileOp{
		Path:    consolePath,
		Content: consoleContent,
		Mode:    0644,
	})

	return ops, nil
}
