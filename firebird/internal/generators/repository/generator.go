package repository

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates repository files from schemas.
type Generator struct {
	projectPath string
	schemaPath  string
	modulePath  string
	renderer    *generator.Renderer
}

// New creates a new repository generator.
func New(projectPath, schemaPath, modulePath string) *Generator {
	return &Generator{
		projectPath: projectPath,
		schemaPath:  schemaPath,
		modulePath:  modulePath,
		renderer:    generator.NewRenderer(),
	}
}

// Generate creates repository files (base + interface + user + tests).
func (g *Generator) Generate() ([]generator.Operation, error) {
	// Parse schema
	spec, err := schema.Parse(g.schemaPath)
	if err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}

	// Prepare template data
	data := g.templateData(spec)

	var ops []generator.Operation

	// Generate base repository (always regenerated)
	baseOp, err := g.generateBase(data)
	if err != nil {
		return nil, fmt.Errorf("generating base repository: %w", err)
	}
	ops = append(ops, baseOp)

	// Generate interface (always regenerated)
	interfaceOp, err := g.generateInterface(data)
	if err != nil {
		return nil, fmt.Errorf("generating repository interface: %w", err)
	}
	ops = append(ops, interfaceOp)

	// Generate user repository (created once, never touched)
	userOp, err := g.generateUser(data)
	if err != nil {
		return nil, fmt.Errorf("generating user repository: %w", err)
	}
	ops = append(ops, userOp)

	// Generate tests (always regenerated)
	testOp, err := g.generateTests(data)
	if err != nil {
		return nil, fmt.Errorf("generating repository tests: %w", err)
	}
	ops = append(ops, testOp)

	return ops, nil
}

func (g *Generator) generateBase(data map[string]interface{}) (generator.Operation, error) {
	modelName := data["ModelName"].(string)
	basePath := filepath.Join(
		g.projectPath,
		"internal",
		"repositories",
		"generated",
		strings.ToLower(modelName)+"_repository_base.go",
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/repository_base.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    basePath,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateInterface(data map[string]interface{}) (generator.Operation, error) {
	modelName := data["ModelName"].(string)
	interfacePath := filepath.Join(
		g.projectPath,
		"internal",
		"repositories",
		strings.ToLower(modelName)+"_repository_interface.go",
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/repository_interface.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    interfacePath,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateUser(data map[string]interface{}) (generator.Operation, error) {
	modelName := data["ModelName"].(string)
	userPath := filepath.Join(
		g.projectPath,
		"internal",
		"repositories",
		strings.ToLower(modelName)+"_repository.go",
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/repository.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &WriteFileIfNotExistsOp{
		Path:    userPath,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateTests(data map[string]interface{}) (generator.Operation, error) {
	modelName := data["ModelName"].(string)
	testPath := filepath.Join(
		g.projectPath,
		"internal",
		"repositories",
		strings.ToLower(modelName)+"_repository_test.go",
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/repository_test.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    testPath,
		Content: content,
		Mode:    0644,
	}, nil
}

// templateData prepares data for repository templates.
func (g *Generator) templateData(def *schema.Definition) map[string]interface{} {
	modelName := def.Name
	tableName := def.Spec.TableName
	if tableName == "" {
		tableName = generator.SnakeCase(generator.Pluralize(def.Name))
	}

	// Detect primary key type from schema
	pkType := "int64" // default fallback
	for _, field := range def.Spec.Fields {
		if field.PrimaryKey {
			pkType = field.Type
			// Strip pointer prefix if present
			pkType = strings.TrimPrefix(pkType, "*")
			break
		}
	}

	return map[string]interface{}{
		"ModelName":      modelName,
		"TableName":      tableName,
		"ModulePath":     g.modulePath,
		"SoftDeletes":    def.Spec.SoftDeletes,
		"PrimaryKeyType": pkType,
	}
}

// WriteFileIfNotExistsOp creates a file only if it doesn't already exist.
// This allows user customizations to be preserved across regenerations.
type WriteFileIfNotExistsOp struct {
	Path    string
	Content []byte
	Mode    fs.FileMode
}

func (op *WriteFileIfNotExistsOp) Validate(ctx context.Context, force bool) error {
	dir := filepath.Dir(op.Path)

	// Create parent directory (side effect, but idempotent)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", dir, err)
	}

	// Check if file already exists - if so, skip without error
	if _, err := os.Stat(op.Path); err == nil {
		return nil // File exists, validation passes but Execute will skip
	}

	// Reject nil content (empty is OK)
	if op.Content == nil {
		return fmt.Errorf("content is nil for file: %s", op.Path)
	}

	return nil
}

func (op *WriteFileIfNotExistsOp) Execute(ctx context.Context) error {
	// Check if file already exists
	if _, err := os.Stat(op.Path); err == nil {
		return nil // File exists, skip creation
	}

	dir := filepath.Dir(op.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(op.Path, op.Content, op.Mode)
}

func (op *WriteFileIfNotExistsOp) Description() string {
	// Check if file exists
	if _, err := os.Stat(op.Path); err == nil {
		return fmt.Sprintf("Skip %s (already exists)", op.Path)
	}
	return fmt.Sprintf("Create %s (%d bytes)", op.Path, len(op.Content))
}
