package helpers

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates validation and handler helper files
type Generator struct {
	projectPath string
	modulePath  string
	renderer    *generator.Renderer
}

// New creates a new helpers generator
func New(projectPath, modulePath string) *Generator {
	return &Generator{
		projectPath: projectPath,
		modulePath:  modulePath,
		renderer:    generator.NewRenderer(),
	}
}

// Generate creates validator and handler helper files
func (g *Generator) Generate() ([]generator.Operation, error) {
	var ops []generator.Operation

	// Generate validator
	validatorOp, err := g.generateValidator()
	if err != nil {
		return nil, fmt.Errorf("generating validator: %w", err)
	}
	ops = append(ops, validatorOp)

	// Generate response helpers
	responseOp, err := g.generateResponseHelpers()
	if err != nil {
		return nil, fmt.Errorf("generating response helpers: %w", err)
	}
	ops = append(ops, responseOp)

	// Generate request helpers
	requestOp, err := g.generateRequestHelpers()
	if err != nil {
		return nil, fmt.Errorf("generating request helpers: %w", err)
	}
	ops = append(ops, requestOp)

	return ops, nil
}

func (g *Generator) generateValidator() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "validation", "validator.go")

	data := map[string]string{"ModulePath": g.modulePath}
	content, err := g.renderer.RenderFS(templatesFS, "templates/validator.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateResponseHelpers() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "handlers", "response.go")

	data := map[string]string{"ModulePath": g.modulePath}
	content, err := g.renderer.RenderFS(templatesFS, "templates/response.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateRequestHelpers() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "handlers", "request.go")

	data := map[string]string{"ModulePath": g.modulePath}
	content, err := g.renderer.RenderFS(templatesFS, "templates/request.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

// WriteFileIfNotExistsOp creates files only if they don't exist
