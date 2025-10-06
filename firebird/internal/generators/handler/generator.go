package handler

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

// Generator generates handler files from schemas
type Generator struct {
	projectPath string
	schemaPath  string
	modulePath  string
	renderer    *generator.Renderer
}

// New creates a new handler generator
func New(projectPath, schemaPath, modulePath string) *Generator {
	return &Generator{
		projectPath: projectPath,
		schemaPath:  schemaPath,
		modulePath:  modulePath,
		renderer:    generator.NewRenderer(),
	}
}

// Generate creates handler files
func (g *Generator) Generate() ([]generator.Operation, error) {
	// Parse schema
	spec, err := schema.Parse(g.schemaPath)
	if err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}

	var ops []generator.Operation

	// Generate handler file
	handlerOp, err := g.generateHandler(spec)
	if err != nil {
		return nil, fmt.Errorf("generating handler: %w", err)
	}
	ops = append(ops, handlerOp)

	return ops, nil
}

func (g *Generator) generateHandler(def *schema.Definition) (generator.Operation, error) {
	data := g.prepareTemplateData(def)

	path := filepath.Join(
		g.projectPath,
		"internal",
		"handlers",
		fmt.Sprintf("%s_handler.go", toSnakeCase(def.Name)),
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/handler.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

// prepareTemplateData builds template data from schema
func (g *Generator) prepareTemplateData(def *schema.Definition) HandlerTemplateData {
	modelName := def.Name
	modelNameLower := toLowerCamel(modelName)
	modelPlural := pluralize(toSnakeCase(modelName))

	// Check for soft deletes
	hasSoftDelete := def.Spec.SoftDeletes

	// Check for relationships
	hasRelationships := len(def.Spec.Relationships) > 0

	// Check if any relationships are API loadable
	hasAPILoadable := false
	for _, rel := range def.Spec.Relationships {
		if rel.APILoadable {
			hasAPILoadable = true
			break
		}
	}

	// Determine primary key type
	pkType := detectPrimaryKeyType(def)

	return HandlerTemplateData{
		ModulePath:                  g.modulePath,
		ModelName:                   modelName,
		ModelNameLower:              modelNameLower,
		ModelPlural:                 modelPlural,
		Package:                     "handlers",
		HasSoftDelete:               hasSoftDelete,
		HasRelationships:            hasRelationships,
		HasAPILoadableRelationships: hasAPILoadable,
		PrimaryKeyType:              pkType,
	}
}

// detectPrimaryKeyType determines the primary key type from schema
func detectPrimaryKeyType(def *schema.Definition) string {
	for _, field := range def.Spec.Fields {
		if field.PrimaryKey {
			return cleanType(field.Type)
		}
	}
	return "uuid.UUID" // Default
}

// toLowerCamel converts PascalCase to lowerCamelCase
func toLowerCamel(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && isUpper(r) {
			result = append(result, '_')
		}
		result = append(result, toLowerRune(r))
	}
	return string(result)
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func toLowerRune(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}

// pluralize converts singular to plural (simple English rules)
func pluralize(s string) string {
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	return s + "s"
}

// cleanType removes pointer prefix from type
func cleanType(t string) string {
	return strings.TrimPrefix(t, "*")
}

// Template data structures

type HandlerTemplateData struct {
	ModulePath                  string
	ModelName                   string
	ModelNameLower              string
	ModelPlural                 string
	Package                     string
	HasSoftDelete               bool
	HasRelationships            bool
	HasAPILoadableRelationships bool
	PrimaryKeyType              string
}

// WriteFileIfNotExistsOp is a custom operation that only creates files if they don't exist
type WriteFileIfNotExistsOp struct {
	Path    string
	Content []byte
	Mode    fs.FileMode
}

func (op *WriteFileIfNotExistsOp) Validate(ctx context.Context, force bool) error {
	dir := filepath.Dir(op.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", dir, err)
	}
	if _, err := os.Stat(op.Path); err == nil {
		return nil // File exists, validation passes but Execute will skip
	}
	if op.Content == nil {
		return fmt.Errorf("content is nil for file: %s", op.Path)
	}
	return nil
}

func (op *WriteFileIfNotExistsOp) Execute(ctx context.Context) error {
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
	if _, err := os.Stat(op.Path); err == nil {
		return fmt.Sprintf("Skip %s (already exists)", op.Path)
	}
	return fmt.Sprintf("Create %s (%d bytes)", op.Path, len(op.Content))
}
