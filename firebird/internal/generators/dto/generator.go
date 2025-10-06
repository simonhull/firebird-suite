package dto

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

// Generator generates DTO files from schemas
type Generator struct {
	projectPath string
	schemaPath  string
	modulePath  string
	renderer    *generator.Renderer
}

// New creates a new DTO generator
func New(projectPath, schemaPath, modulePath string) *Generator {
	return &Generator{
		projectPath: projectPath,
		schemaPath:  schemaPath,
		modulePath:  modulePath,
		renderer:    generator.NewRenderer(),
	}
}

// Generate creates DTO files (CreateInput, UpdateInput, Response)
func (g *Generator) Generate() ([]generator.Operation, error) {
	// Parse schema
	spec, err := schema.Parse(g.schemaPath)
	if err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}

	var ops []generator.Operation

	// Generate CreateInput DTO
	createOp, err := g.generateCreateInput(spec)
	if err != nil {
		return nil, fmt.Errorf("generating CreateInput: %w", err)
	}
	ops = append(ops, createOp)

	// Generate UpdateInput DTO
	updateOp, err := g.generateUpdateInput(spec)
	if err != nil {
		return nil, fmt.Errorf("generating UpdateInput: %w", err)
	}
	ops = append(ops, updateOp)

	// Generate Response DTO
	responseOp, err := g.generateResponse(spec)
	if err != nil {
		return nil, fmt.Errorf("generating Response: %w", err)
	}
	ops = append(ops, responseOp)

	return ops, nil
}

func (g *Generator) generateCreateInput(def *schema.Definition) (generator.Operation, error) {
	data := g.prepareCreateInputData(def)

	path := filepath.Join(
		g.projectPath,
		"internal",
		"dto",
		fmt.Sprintf("%s_create_input.go", strings.ToLower(def.Name)),
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/create_input.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	// User-owned file - only create if doesn't exist
	return &WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateUpdateInput(def *schema.Definition) (generator.Operation, error) {
	data := g.prepareUpdateInputData(def)

	path := filepath.Join(
		g.projectPath,
		"internal",
		"dto",
		fmt.Sprintf("%s_update_input.go", strings.ToLower(def.Name)),
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/update_input.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateResponse(def *schema.Definition) (generator.Operation, error) {
	data := g.prepareResponseData(def)

	path := filepath.Join(
		g.projectPath,
		"internal",
		"dto",
		fmt.Sprintf("%s_response.go", strings.ToLower(def.Name)),
	)

	content, err := g.renderer.RenderFS(templatesFS, "templates/response.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

// prepareCreateInputData prepares template data for CreateInput DTO
func (g *Generator) prepareCreateInputData(def *schema.Definition) CreateInputData {
	var fields []FieldData
	var excluded []string

	for _, field := range def.Spec.Fields {
		// Skip auto-generated fields
		if shouldExcludeFromCreate(field) {
			excluded = append(excluded, field.Name)
			continue
		}

		fields = append(fields, FieldData{
			Name:       toGoName(field.Name),
			Type:       cleanType(field.Type),
			JSONTag:    getJSONTag(field),
			Validation: buildValidationTags(field, false),
		})
	}

	return CreateInputData{
		ModelName:      def.Name,
		ModulePath:     g.modulePath,
		HasUUID:        hasTypeInFields(fields, "uuid.UUID"),
		HasTime:        hasTypeInFields(fields, "time.Time"),
		ExcludedFields: excluded,
		Fields:         fields,
	}
}

// prepareUpdateInputData prepares template data for UpdateInput DTO
func (g *Generator) prepareUpdateInputData(def *schema.Definition) UpdateInputData {
	var fields []FieldData
	var excluded []string

	for _, field := range def.Spec.Fields {
		// Skip immutable fields
		if shouldExcludeFromUpdate(field) {
			excluded = append(excluded, field.Name)
			continue
		}

		// All fields are pointers in update (optional)
		fieldType := cleanType(field.Type)

		fields = append(fields, FieldData{
			Name:       toGoName(field.Name),
			Type:       fieldType,
			JSONTag:    getJSONTag(field),
			Validation: buildValidationTags(field, true),
		})
	}

	return UpdateInputData{
		ModelName:      def.Name,
		ModulePath:     g.modulePath,
		HasUUID:        hasTypeInFields(fields, "uuid.UUID"),
		HasTime:        hasTypeInFields(fields, "time.Time"),
		ExcludedFields: excluded,
		Fields:         fields,
	}
}

// prepareResponseData prepares template data for Response DTO
func (g *Generator) prepareResponseData(def *schema.Definition) ResponseData {
	var fields []ResponseFieldData
	var excluded []string

	// Always include ID
	for _, field := range def.Spec.Fields {
		if field.PrimaryKey {
			fields = append(fields, ResponseFieldData{
				Name:        toGoName(field.Name),
				Type:        cleanType(field.Type),
				JSONTag:     getJSONTag(field),
				DBFieldName: toGoName(field.Name),
				Omitempty:   false,
			})
			break
		}
	}

	// Include other fields
	for _, field := range def.Spec.Fields {
		if shouldExcludeFromResponse(field) {
			excluded = append(excluded, field.Name)
			continue
		}

		if field.PrimaryKey {
			continue // Already added
		}

		fields = append(fields, ResponseFieldData{
			Name:        toGoName(field.Name),
			Type:        cleanType(field.Type),
			JSONTag:     getJSONTag(field),
			DBFieldName: toGoName(field.Name),
			Omitempty:   field.Nullable || strings.HasPrefix(field.Type, "*"),
		})
	}

	// Add timestamps if enabled
	if def.Spec.Timestamps {
		fields = append(fields,
			ResponseFieldData{
				Name:        "CreatedAt",
				Type:        "time.Time",
				JSONTag:     "created_at",
				DBFieldName: "CreatedAt",
				Omitempty:   false,
			},
			ResponseFieldData{
				Name:        "UpdatedAt",
				Type:        "time.Time",
				JSONTag:     "updated_at",
				DBFieldName: "UpdatedAt",
				Omitempty:   false,
			},
		)
	}

	// Prepare relationship fields
	relationships := prepareRelationshipFields(def)

	return ResponseData{
		ModelName:      def.Name,
		ModulePath:     g.modulePath,
		HasUUID:        hasUUIDInResponse(fields),
		HasTime:        hasTimeInResponse(fields),
		ExcludedFields: excluded,
		Fields:         fields,
		Relationships:  relationships,
	}
}

// shouldExcludeFromCreate checks if field should be excluded from CreateInput
func shouldExcludeFromCreate(field schema.Field) bool {
	return field.PrimaryKey ||
		field.Name == "created_at" ||
		field.Name == "updated_at" ||
		field.Name == "deleted_at"
}

// shouldExcludeFromUpdate checks if field should be excluded from UpdateInput
func shouldExcludeFromUpdate(field schema.Field) bool {
	return field.PrimaryKey ||
		field.Name == "created_at" ||
		field.Name == "deleted_at"
	// Note: Unique fields are NOT automatically excluded - users can update them
	// Whether a unique field should be updateable is a business logic decision.
	// If you want to prevent updates to specific unique fields, add validation
	// in your custom DTO or add business logic in the service layer
}

// shouldExcludeFromResponse checks if field should be excluded from Response
func shouldExcludeFromResponse(field schema.Field) bool {
	// Exclude soft delete timestamp and sensitive fields
	return field.Name == "deleted_at" ||
		field.Name == "password_hash" ||
		field.Name == "password" ||
		strings.Contains(field.Name, "secret")
}

// buildValidationTags converts schema validation rules to go-playground/validator tags
func buildValidationTags(field schema.Field, isUpdate bool) string {
	// No explicit validation rules
	if len(field.Validation) == 0 {
		// For CreateInput: required if explicitly marked OR (non-nullable AND no default)
		// Fields with database defaults should be optional in the API
		if !isUpdate && (field.Required || (!field.Nullable && field.Default == nil)) {
			return "required"
		}
		return ""
	}

	// Build validation tags from schema rules
	tags := make([]string, 0, len(field.Validation)+1)

	if isUpdate {
		// For updates, make all validations conditional
		tags = append(tags, "omitempty")
	}

	for _, rule := range field.Validation {
		tags = append(tags, rule)
	}

	return strings.Join(tags, ",")
}

// toGoName converts snake_case to PascalCase following Go conventions
func toGoName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if len(part) > 0 {
			// Handle common Go initialisms (ID, URL, HTTP, etc.)
			upperPart := strings.ToUpper(part)
			if isCommonInitialism(upperPart) {
				parts[i] = upperPart
			} else {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
	}
	return strings.Join(parts, "")
}

// isCommonInitialism checks if a string is a common Go initialism that should be all caps
func isCommonInitialism(s string) bool {
	// Common Go initialisms per https://github.com/golang/lint/blob/master/lint.go
	initialisms := map[string]bool{
		"API": true, "ASCII": true, "CPU": true, "CSS": true, "DNS": true,
		"EOF": true, "GUID": true, "HTML": true, "HTTP": true, "HTTPS": true,
		"ID": true, "IP": true, "JSON": true, "LHS": true, "QPS": true,
		"RAM": true, "RHS": true, "RPC": true, "SLA": true, "SMTP": true,
		"SQL": true, "SSH": true, "TCP": true, "TLS": true, "TTL": true,
		"UDP": true, "UI": true, "UID": true, "UUID": true, "URI": true,
		"URL": true, "UTF8": true, "VM": true, "XML": true, "XMPP": true,
		"XSRF": true, "XSS": true,
	}
	return initialisms[s]
}

// getJSONTag gets the JSON tag for a field
func getJSONTag(field schema.Field) string {
	if field.JSON != "" {
		return field.JSON
	}
	return field.Name
}

// cleanType removes pointer prefix from type
func cleanType(t string) string {
	return strings.TrimPrefix(t, "*")
}

// hasTypeInFields checks if any field has a specific type
func hasTypeInFields(fields []FieldData, typeName string) bool {
	for _, field := range fields {
		if strings.Contains(field.Type, typeName) {
			return true
		}
	}
	return false
}

// hasUUIDInResponse checks for UUID in response fields
func hasUUIDInResponse(fields []ResponseFieldData) bool {
	for _, field := range fields {
		if strings.Contains(field.Type, "uuid.UUID") {
			return true
		}
	}
	return false
}

// hasTimeInResponse checks for time.Time in response fields
func hasTimeInResponse(fields []ResponseFieldData) bool {
	for _, field := range fields {
		if strings.Contains(field.Type, "time.Time") {
			return true
		}
	}
	return false
}

// prepareRelationshipFields transforms relationships into DTO field data
func prepareRelationshipFields(def *schema.Definition) []RelationshipFieldData {
	var result []RelationshipFieldData

	for _, rel := range def.Spec.Relationships {
		field := RelationshipFieldData{
			Name:    rel.Name,
			JSONTag: toSnakeCase(rel.Name),
		}

		// Determine response type
		if rel.Type == "belongs_to" {
			// Single relationship - pointer to allow nil
			field.ResponseType = fmt.Sprintf("*%sResponse", rel.Model)
		} else {
			// Collection relationship - slice
			field.ResponseType = fmt.Sprintf("[]*%sResponse", rel.Model)
		}

		result = append(result, field)
	}

	return result
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		if r >= 'A' && r <= 'Z' {
			result = append(result, r-'A'+'a')
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// Template data structures

type CreateInputData struct {
	ModelName      string
	ModulePath     string
	HasUUID        bool
	HasTime        bool
	ExcludedFields []string
	Fields         []FieldData
}

type UpdateInputData struct {
	ModelName      string
	ModulePath     string
	HasUUID        bool
	HasTime        bool
	ExcludedFields []string
	Fields         []FieldData
}

type ResponseData struct {
	ModelName      string
	ModulePath     string
	HasUUID        bool
	HasTime        bool
	ExcludedFields []string
	Fields         []ResponseFieldData
	Relationships  []RelationshipFieldData
}

type FieldData struct {
	Name       string
	Type       string
	JSONTag    string
	Validation string
}

type ResponseFieldData struct {
	Name        string
	Type        string
	JSONTag     string
	DBFieldName string
	Omitempty   bool
}

type RelationshipFieldData struct {
	Name         string // Field name (e.g., "Author", "Posts")
	JSONTag      string // JSON tag (e.g., "author", "posts")
	ResponseType string // Go type (e.g., "*UserResponse", "[]*PostResponse")
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
