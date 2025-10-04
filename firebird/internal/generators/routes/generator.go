package routes

import (
	"context"
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates route registration files
type Generator struct {
	projectPath string
	modulePath  string
	router      string
	renderer    *generator.Renderer
}

// New creates a new routes generator
func New(projectPath, modulePath, router string) *Generator {
	return &Generator{
		projectPath: projectPath,
		modulePath:  modulePath,
		router:      router,
		renderer:    generator.NewRenderer(),
	}
}

// HandlerInfo represents a discovered handler
type HandlerInfo struct {
	Name        string   // "PostHandler"
	VarName     string   // "postHandler"
	ModelName   string   // "Post"
	ModelPlural string   // "posts"
	FilePath    string   // "internal/handlers/post_handler.go"
	Methods     []string // ["Index", "Store", "Show", "Update", "Destroy"]
}

// Generate discovers handlers and generates routes file
func (g *Generator) Generate() ([]generator.Operation, error) {
	// Discover all handlers
	handlers, err := g.discoverHandlers()
	if err != nil {
		return nil, fmt.Errorf("discovering handlers: %w", err)
	}

	if len(handlers) == 0 {
		return nil, fmt.Errorf("no handlers found in internal/handlers/")
	}

	// Generate routes file based on router type
	var ops []generator.Operation

	switch g.router {
	case "stdlib":
		op, err := g.generateStdlibRoutes(handlers)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	case "chi":
		op, err := g.generateChiRoutes(handlers)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	case "gin":
		op, err := g.generateGinRoutes(handlers)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	case "echo":
		op, err := g.generateEchoRoutes(handlers)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	default:
		return nil, fmt.Errorf("unsupported router: %s", g.router)
	}

	return ops, nil
}

// discoverHandlers scans internal/handlers/ for handler files
func (g *Generator) discoverHandlers() ([]HandlerInfo, error) {
	handlersDir := filepath.Join(g.projectPath, "internal", "handlers")

	entries, err := os.ReadDir(handlersDir)
	if err != nil {
		return nil, fmt.Errorf("reading handlers directory: %w", err)
	}

	var handlers []HandlerInfo

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_handler.go") {
			continue
		}

		filePath := filepath.Join(handlersDir, entry.Name())
		handler, err := g.parseHandlerFile(filePath)
		if err != nil {
			// Skip files that don't parse as handlers
			continue
		}

		handlers = append(handlers, handler)
	}

	return handlers, nil
}

// parseHandlerFile extracts handler info from a Go file
func (g *Generator) parseHandlerFile(filePath string) (HandlerInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return HandlerInfo{}, err
	}

	var info HandlerInfo
	info.FilePath = filePath

	// Find the handler struct type and its methods
	ast.Inspect(node, func(n ast.Node) bool {
		// Look for type definitions
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if strings.HasSuffix(typeSpec.Name.Name, "Handler") {
				info.Name = typeSpec.Name.Name
				info.ModelName = strings.TrimSuffix(typeSpec.Name.Name, "Handler")
				info.VarName = toLowerCamel(info.Name)
				info.ModelPlural = pluralize(toSnakeCase(info.ModelName))
			}
		}

		// Look for methods on the handler
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				// Check if receiver is our handler
				if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok {
						if ident.Name == info.Name {
							// This is a method on our handler
							info.Methods = append(info.Methods, funcDecl.Name.Name)
						}
					}
				}
			}
		}

		return true
	})

	if info.Name == "" {
		return HandlerInfo{}, fmt.Errorf("no handler struct found")
	}

	return info, nil
}

func (g *Generator) generateStdlibRoutes(handlers []HandlerInfo) (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "handlers", "routes.go")

	data := RoutesTemplateData{
		ModulePath: g.modulePath,
		Router:     g.router,
		Handlers:   handlers,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/routes_stdlib.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateChiRoutes(handlers []HandlerInfo) (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "handlers", "routes.go")

	data := RoutesTemplateData{
		ModulePath: g.modulePath,
		Router:     g.router,
		Handlers:   handlers,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/routes_chi.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateGinRoutes(handlers []HandlerInfo) (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "handlers", "routes.go")

	data := RoutesTemplateData{
		ModulePath: g.modulePath,
		Router:     g.router,
		Handlers:   handlers,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/routes_gin.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateEchoRoutes(handlers []HandlerInfo) (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "handlers", "routes.go")

	data := RoutesTemplateData{
		ModulePath: g.modulePath,
		Router:     g.router,
		Handlers:   handlers,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/routes_echo.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &WriteFileIfNotExistsOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

// Template data structure
type RoutesTemplateData struct {
	ModulePath string
	Router     string
	Handlers   []HandlerInfo
}

// Helper functions
func toLowerCamel(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

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

// WriteFileIfNotExistsOp creates files only if they don't exist
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
