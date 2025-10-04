package service

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

// Generator generates service files from schemas
type Generator struct {
	projectPath string
	schemaPath  string
	modulePath  string
	renderer    *generator.Renderer
}

// New creates a new service generator
func New(projectPath, schemaPath, modulePath string) *Generator {
	return &Generator{
		projectPath: projectPath,
		schemaPath:  schemaPath,
		modulePath:  modulePath,
		renderer:    generator.NewRenderer(),
	}
}

// Generate creates service files (interface, implementation, helpers, test, shared)
func (g *Generator) Generate() ([]generator.Operation, error) {
	// Parse schema
	spec, err := schema.Parse(g.schemaPath)
	if err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}

	var ops []generator.Operation

	// Generate shared files first (errors.go, types.go) - once only
	sharedOps, err := g.generateSharedFiles()
	if err != nil {
		return nil, fmt.Errorf("generating shared files: %w", err)
	}
	ops = append(ops, sharedOps...)

	// Generate service interface (always regenerated)
	interfaceOp, err := g.generateInterface(spec)
	if err != nil {
		return nil, fmt.Errorf("generating interface: %w", err)
	}
	ops = append(ops, interfaceOp)

	// Generate user service implementation (created once, user-owned)
	serviceOp, err := g.generateService(spec)
	if err != nil {
		return nil, fmt.Errorf("generating service: %w", err)
	}
	ops = append(ops, serviceOp)

	// Generate helpers if relationships exist (always regenerated)
	if len(spec.Spec.Relationships) > 0 {
		helpersOp, err := g.generateHelpers(spec)
		if err != nil {
			return nil, fmt.Errorf("generating helpers: %w", err)
		}
		ops = append(ops, helpersOp)
	}

	// Generate test file (always regenerated)
	testOp, err := g.generateTest(spec)
	if err != nil {
		return nil, fmt.Errorf("generating test: %w", err)
	}
	ops = append(ops, testOp)

	return ops, nil
}

func (g *Generator) generateSharedFiles() ([]generator.Operation, error) {
	var ops []generator.Operation

	// Generate errors.go
	errorsPath := filepath.Join(g.projectPath, "internal", "services", "errors.go")
	errorsContent, err := g.renderer.RenderFS(templatesFS, "templates/errors.go.tmpl", nil)
	if err != nil {
		return nil, err
	}
	ops = append(ops, &WriteFileIfNotExistsOp{
		Path:    errorsPath,
		Content: errorsContent,
		Mode:    0644,
	})

	// Generate types.go
	typesPath := filepath.Join(g.projectPath, "internal", "services", "types.go")
	typesContent, err := g.renderer.RenderFS(templatesFS, "templates/types.go.tmpl", nil)
	if err != nil {
		return nil, err
	}
	ops = append(ops, &WriteFileIfNotExistsOp{
		Path:    typesPath,
		Content: typesContent,
		Mode:    0644,
	})

	return ops, nil
}

func (g *Generator) generateInterface(def *schema.Definition) (generator.Operation, error) {
	data := g.prepareTemplateData(def)

	path := filepath.Join(
		g.projectPath,
		"internal",
		"services",
		fmt.Sprintf("%s_service_interface.go", strings.ToLower(def.Name)),
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/service_interface.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateHelpers(def *schema.Definition) (generator.Operation, error) {
	data := g.prepareTemplateData(def)

	path := filepath.Join(
		g.projectPath,
		"internal",
		"services",
		"generated",
		fmt.Sprintf("%s_service_helpers.go", strings.ToLower(def.Name)),
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/service_helpers.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateService(def *schema.Definition) (generator.Operation, error) {
	data := g.prepareTemplateData(def)

	path := filepath.Join(
		g.projectPath,
		"internal",
		"services",
		fmt.Sprintf("%s_service.go", strings.ToLower(def.Name)),
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/service_impl.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateTest(def *schema.Definition) (generator.Operation, error) {
	data := g.prepareTemplateData(def)

	path := filepath.Join(
		g.projectPath,
		"internal",
		"services",
		fmt.Sprintf("%s_service_test.go", strings.ToLower(def.Name)),
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/service_test.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

// prepareTemplateData builds template data from schema
func (g *Generator) prepareTemplateData(def *schema.Definition) ServiceTemplateData {
	modelName := def.Name
	modelNameLower := strings.ToLower(modelName)
	repoFieldName := modelNameLower + "Repo"

	// Map DTO fields to DB param fields
	createFields := g.buildFieldMappings(def, false)
	updateFields := g.buildFieldMappings(def, true)

	// Prepare relationship data
	relationships := prepareRelationshipHelpers(def)

	return ServiceTemplateData{
		ModelName:      modelName,
		ModelNameLower: modelNameLower,
		ModulePath:     g.modulePath,
		PrimaryKeyType: detectPrimaryKeyType(def),
		RepoFieldName:  repoFieldName,
		SoftDeletes:    def.Spec.SoftDeletes,
		CreateFields:   createFields,
		UpdateFields:   updateFields,
		Relationships:  relationships,
	}
}

func (g *Generator) buildFieldMappings(def *schema.Definition, isUpdate bool) []FieldMapping {
	var mappings []FieldMapping

	for _, field := range def.Spec.Fields {
		// Skip auto-generated fields
		if field.PrimaryKey || field.Name == "created_at" ||
			field.Name == "updated_at" || field.Name == "deleted_at" {
			continue
		}

		if isUpdate && field.Name == "created_at" {
			continue // Can't update created_at
		}

		mappings = append(mappings, FieldMapping{
			DTOField: toGoName(field.Name),
			DBField:  toGoName(field.Name),
		})
	}

	return mappings
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

// toGoName converts snake_case to PascalCase
func toGoName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// cleanType removes pointer prefix from type
func cleanType(t string) string {
	return strings.TrimPrefix(t, "*")
}

// prepareRelationshipHelpers transforms relationships into helper method data
func prepareRelationshipHelpers(def *schema.Definition) []RelationshipHelperData {
	var result []RelationshipHelperData

	for _, rel := range def.Spec.Relationships {
		data := RelationshipHelperData{
			Name:           rel.Name,
			Type:           rel.Type,
			Model:          rel.Model,
			LoadMethod:     fmt.Sprintf("Load%s", rel.Name),
			LoadManyMethod: fmt.Sprintf("Load%sForMany", rel.Name),
			IsSingle:       rel.Type == "belongs_to",
			IsMany:         rel.Type == "has_many",
		}
		result = append(result, data)
	}

	return result
}

// Template data structures

type ServiceTemplateData struct {
	ModelName      string
	ModelNameLower string
	ModulePath     string
	PrimaryKeyType string
	RepoFieldName  string
	SoftDeletes    bool
	CreateFields   []FieldMapping
	UpdateFields   []FieldMapping
	Relationships  []RelationshipHelperData
}

type FieldMapping struct {
	DTOField string // Field name in DTO
	DBField  string // Field name in DB params
}

type RelationshipHelperData struct {
	Name           string // Relationship name (e.g., "Author", "Posts")
	Type           string // "belongs_to" or "has_many"
	Model          string // Target model (e.g., "User", "Post")
	LoadMethod     string // Method name (e.g., "LoadAuthor")
	LoadManyMethod string // Batch method name (e.g., "LoadPostsForMany")
	IsSingle       bool   // belongs_to flag
	IsMany         bool   // has_many flag
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
