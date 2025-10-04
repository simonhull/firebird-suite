package middleware

import (
	"embed"
	"path/filepath"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates middleware
type Generator struct {
	renderer *generator.Renderer
	pkgPath  string
}

// New creates a new middleware generator
func New(pkgPath string) *Generator {
	return &Generator{
		renderer: generator.NewRenderer(),
		pkgPath:  pkgPath,
	}
}

// Generate generates middleware files
func (g *Generator) Generate() ([]generator.Operation, error) {
	var ops []generator.Operation

	// Generate logging middleware
	loggingContent, err := g.renderer.RenderFS(templatesFS, "templates/logging.go.tmpl", nil)
	if err != nil {
		return nil, err
	}

	loggingPath := filepath.Join(g.pkgPath, "internal", "middleware", "logging.go")
	ops = append(ops, &generator.WriteFileOp{
		Path:    loggingPath,
		Content: loggingContent,
		Mode:    0644,
	})

	// Generate recovery middleware
	recoveryContent, err := g.renderer.RenderFS(templatesFS, "templates/recovery.go.tmpl", nil)
	if err != nil {
		return nil, err
	}

	recoveryPath := filepath.Join(g.pkgPath, "internal", "middleware", "recovery.go")
	ops = append(ops, &generator.WriteFileOp{
		Path:    recoveryPath,
		Content: recoveryContent,
		Mode:    0644,
	})

	return ops, nil
}
