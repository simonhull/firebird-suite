package repository

import (
	"embed"
	"fmt"
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

// RelationshipMethodData holds data for generating repository relationship methods
type RelationshipMethodData struct {
	Name             string // Relationship name (e.g., "Author", "Comments", "Tags")
	Type             string // "belongs_to", "has_many", or "many_to_many"
	Model            string // Target model (e.g., "User", "Comment", "Tag")
	ForeignKey       string // FK field name snake_case (e.g., "author_id")
	ForeignKeyField  string // FK field name PascalCase (e.g., "AuthorID")
	LoadMethod       string // Method name (e.g., "LoadAuthor", "LoadTags")
	LoadManyMethod   string // Batch method name (e.g., "LoadCommentsForMany", "LoadTagsForMany")
	GetQueryName     string // SQLC query name (e.g., "GetPostAuthor", "GetPostTags")
	GetManyQueryName string // SQLC batch query (e.g., "GetCommentsForPosts", "GetTagsForPosts")
	AddMethod        string // M2M add method (e.g., "AddTags")
	RemoveMethod     string // M2M remove method (e.g., "RemoveTags")
	SetMethod        string // M2M set method (e.g., "SetTags")
	AddQueryName     string // M2M SQLC add query (e.g., "AddPostTag")
	RemoveQueryName  string // M2M SQLC remove query (e.g., "RemovePostTag")
	RemoveAllQueryName string // M2M SQLC remove all query (e.g., "RemoveAllPostTags")
	ModelType        string // Go type (e.g., "db.User", "db.Tag")
	ForeignKeyType   string // FK Go type (e.g., "uuid.UUID", "int64")
	IsSingle         bool   // belongs_to flag
	IsMany           bool   // has_many flag
	IsM2M            bool   // many_to_many flag
	APILoadable      bool   // Allow loading via API includes (from schema)
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

	// NOTE: Interface generation removed to avoid naming conflict with user repository struct
	// The user repository struct (*MessageRepository) is what's used throughout the codebase
	// If an interface is needed for testing, users can create it manually

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

	return &generator.WriteFileIfNotExistsOp{
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

	// Prepare relationship data
	relationships := prepareRelationshipMethods(def)

	// Check if any relationships are API loadable
	hasAPILoadable := false
	for _, rel := range def.Spec.Relationships {
		if rel.APILoadable {
			hasAPILoadable = true
			break
		}
	}

	// Detect if UUID is used (for import)
	usesUUID := strings.Contains(pkType, "uuid.UUID")
	if !usesUUID {
		for _, rel := range relationships {
			if strings.Contains(rel.ForeignKeyType, "uuid.UUID") {
				usesUUID = true
				break
			}
		}
	}

	return map[string]interface{}{
		"ModelName":                   modelName,
		"TableName":                   tableName,
		"ModulePath":                  g.modulePath,
		"SoftDeletes":                 def.Spec.SoftDeletes,
		"PrimaryKeyType":              pkType,
		"Relationships":               relationships,
		"HasAPILoadableRelationships": hasAPILoadable,
		"UsesUUID":                    usesUUID,
	}
}

// prepareRelationshipMethods transforms relationships into repository method data
func prepareRelationshipMethods(def *schema.Definition) []RelationshipMethodData {
	var result []RelationshipMethodData

	for _, rel := range def.Spec.Relationships {
		// Find FK field to determine type (for belongs_to/has_many)
		fkType := "uuid.UUID" // Default for M2M
		if rel.Type != "many_to_many" {
			fkType = findForeignKeyType(def, rel.ForeignKey)
		} else {
			// For M2M, use primary key type
			for _, field := range def.Spec.Fields {
				if field.PrimaryKey {
					fkType = strings.TrimPrefix(field.Type, "*")
					break
				}
			}
		}

		data := RelationshipMethodData{
			Name:            rel.Name,
			Type:            rel.Type,
			Model:           rel.Model,
			ForeignKey:      rel.ForeignKey,
			ForeignKeyField: generator.PascalCase(rel.ForeignKey),

			// Method names
			LoadMethod:     fmt.Sprintf("Load%s", rel.Name),
			LoadManyMethod: fmt.Sprintf("Load%sForMany", rel.Name),

			// SQLC query names (must match query generator)
			GetQueryName:     fmt.Sprintf("Get%s%s", def.Name, rel.Name),
			GetManyQueryName: fmt.Sprintf("Get%sFor%s", generator.Pluralize(rel.Model), generator.Pluralize(def.Name)),

			// Go types
			ModelType:      fmt.Sprintf("db.%s", rel.Model),
			ForeignKeyType: fkType,

			// For return types
			IsSingle: rel.Type == "belongs_to",
			IsMany:   rel.Type == "has_many",
			IsM2M:    rel.Type == "many_to_many",

			// API access control
			APILoadable: rel.APILoadable,
		}

		// M2M specific methods
		if rel.Type == "many_to_many" {
			data.AddMethod = fmt.Sprintf("Add%s", rel.Name)
			data.RemoveMethod = fmt.Sprintf("Remove%s", rel.Name)
			data.SetMethod = fmt.Sprintf("Set%s", rel.Name)
			data.AddQueryName = fmt.Sprintf("Add%s%s", def.Name, rel.Model)
			data.RemoveQueryName = fmt.Sprintf("Remove%s%s", def.Name, rel.Model)
			data.RemoveAllQueryName = fmt.Sprintf("RemoveAll%s%s", def.Name, rel.Name)
			// For has_many, fix the GetManyQueryName  to match query generator
			data.GetManyQueryName = fmt.Sprintf("Get%sFor%s", rel.Name, generator.Pluralize(def.Name))
		}

		result = append(result, data)
	}

	return result
}

// findForeignKeyType returns the Go type for the foreign key field
func findForeignKeyType(def *schema.Definition, fkName string) string {
	for _, field := range def.Spec.Fields {
		if field.Name == fkName {
			return strings.TrimPrefix(field.Type, "*") // Remove pointer if present
		}
	}
	return "int64" // Fallback
}

// WriteFileIfNotExistsOp creates a file only if it doesn't already exist.
// This allows user customizations to be preserved across regenerations.
