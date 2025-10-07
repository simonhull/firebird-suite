package shared

import (
	"context"
	"embed"
	"path/filepath"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

type Generator struct {
	projectPath string
	modulePath  string
	renderer    *generator.Renderer
}

func NewGenerator(projectPath, modulePath string) *Generator {
	return &Generator{
		projectPath: projectPath,
		modulePath:  modulePath,
		renderer:    generator.NewRenderer(),
	}
}

// Generate creates shared infrastructure files (errors, helpers, validation, CORS, middleware)
func (g *Generator) Generate() ([]generator.Operation, error) {
	var ops []generator.Operation

	// Generate errors package
	errorsOp, err := g.generateErrors()
	if err != nil {
		return nil, err
	}
	ops = append(ops, errorsOp)

	// Generate response helpers
	helpersOp, err := g.generateHelpers()
	if err != nil {
		return nil, err
	}
	ops = append(ops, helpersOp)

	// Generate validation helpers
	validationOp, err := g.generateValidation()
	if err != nil {
		return nil, err
	}
	ops = append(ops, validationOp)

	// Generate CORS middleware
	corsOp, err := g.generateCORS()
	if err != nil {
		return nil, err
	}
	ops = append(ops, corsOp)

	// Generate CORS config
	corsConfigOp, err := g.generateCORSConfig()
	if err != nil {
		return nil, err
	}
	ops = append(ops, corsConfigOp)

	// Generate request ID middleware
	requestIDOp, err := g.generateRequestID()
	if err != nil {
		return nil, err
	}
	ops = append(ops, requestIDOp)

	// Generate logger middleware
	loggerOp, err := g.generateLogger()
	if err != nil {
		return nil, err
	}
	ops = append(ops, loggerOp)

	// Generate rate limit config
	rateLimitConfigOp, err := g.generateRateLimitConfig()
	if err != nil {
		return nil, err
	}
	ops = append(ops, rateLimitConfigOp)

	// Generate rate limit middleware
	rateLimitOp, err := g.generateRateLimit()
	if err != nil {
		return nil, err
	}
	ops = append(ops, rateLimitOp)

	// Generate query helpers
	queryHelpersOp, err := g.generateQueryHelpers()
	if err != nil {
		return nil, err
	}
	ops = append(ops, queryHelpersOp)

	// Generate auth helpers
	authHelpersOp, err := g.generateAuthHelpers()
	if err != nil {
		return nil, err
	}
	ops = append(ops, authHelpersOp)

	// Generate UUID helpers
	uuidHelpersOp, err := g.generateUUIDHelpers()
	if err != nil {
		return nil, err
	}
	ops = append(ops, uuidHelpersOp)

	// Generate testing helpers
	testingHelpersOp, err := g.generateTestingHelpers()
	if err != nil {
		return nil, err
	}
	ops = append(ops, testingHelpersOp)

	// Generate request helpers (GetPathInt64, GetPathUUID, ParsePagination, etc.)
	requestHelpersOp, err := g.generateRequestHelpers()
	if err != nil {
		return nil, err
	}
	ops = append(ops, requestHelpersOp)

	return ops, nil
}

func (g *Generator) generateErrors() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "errors", "errors.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/errors.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateHelpers() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "helpers", "response.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/response.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateValidation() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "helpers", "validation.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/validation.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateCORS() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "middleware", "cors.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/cors.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateCORSConfig() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "config", "cors.go")

	data := map[string]interface{}{}

	content, err := g.renderer.RenderFS(templatesFS, "templates/cors_config.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateRequestID() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "middleware", "request_id.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/request_id.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateLogger() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "middleware", "logger.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/logger.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateRateLimitConfig() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "config", "rate_limit.go")

	data := map[string]interface{}{}

	content, err := g.renderer.RenderFS(templatesFS, "templates/rate_limit_config.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateRateLimit() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "middleware", "rate_limit.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/rate_limit.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateQueryHelpers() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "helpers", "query.go")

	data := map[string]interface{}{}

	content, err := g.renderer.RenderFS(templatesFS, "templates/query_helpers.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateAuthHelpers() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "helpers", "auth.go")

	data := map[string]interface{}{}

	content, err := g.renderer.RenderFS(templatesFS, "templates/auth_helpers.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateUUIDHelpers() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "helpers", "uuid.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/uuid_helpers.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateTestingHelpers() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "testhelpers", "testing.go")

	data := map[string]interface{}{}

	content, err := g.renderer.RenderFS(templatesFS, "templates/testing_helpers.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateRequestHelpers() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "handlers", "request.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/request_helpers.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

// ValidateOperation is a custom operation that validates the context
type ValidateOperation struct{}

func (op *ValidateOperation) Validate(ctx context.Context, force bool) error {
	return nil
}

func (op *ValidateOperation) Execute(ctx context.Context) error {
	return nil
}

func (op *ValidateOperation) Description() string {
	return "Validate shared infrastructure"
}
