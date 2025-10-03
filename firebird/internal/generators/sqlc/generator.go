package sqlc

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates SQLC configuration and database helpers.
type Generator struct {
	projectPath string
	projectName string
	database    string
	modulePath  string
	renderer    *generator.Renderer
}

// New creates a new SQLC generator.
func New(projectPath, projectName, database, modulePath string) *Generator {
	return &Generator{
		projectPath: projectPath,
		projectName: projectName,
		database:    database,
		modulePath:  modulePath,
		renderer:    generator.NewRenderer(),
	}
}

// Generate creates all SQLC-related files.
func (g *Generator) Generate() ([]generator.Operation, error) {
	var ops []generator.Operation

	// Generate sqlc.yaml
	sqlcConfigOp, err := g.generateSQLCConfig()
	if err != nil {
		return nil, fmt.Errorf("generating sqlc.yaml: %w", err)
	}
	ops = append(ops, sqlcConfigOp)

	// Generate internal/db/db.go
	dbHelperOp, err := g.generateDBHelper()
	if err != nil {
		return nil, fmt.Errorf("generating db.go: %w", err)
	}
	ops = append(ops, dbHelperOp)

	// Create .gitkeep in queries directory (directory will be created automatically)
	gitkeepOp := &generator.WriteFileOp{
		Path:    filepath.Join(g.projectPath, "internal", "db", "queries", ".gitkeep"),
		Content: []byte(""),
		Mode:    0644,
	}
	ops = append(ops, gitkeepOp)

	return ops, nil
}

func (g *Generator) generateSQLCConfig() (generator.Operation, error) {
	data := g.templateData()

	content, err := g.renderer.RenderFS(templatesFS, "templates/sqlc.yaml.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    filepath.Join(g.projectPath, "sqlc.yaml"),
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateDBHelper() (generator.Operation, error) {
	data := g.templateData()

	content, err := g.renderer.RenderFS(templatesFS, "templates/db.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    filepath.Join(g.projectPath, "internal", "db", "db.go"),
		Content: content,
		Mode:    0644,
	}, nil
}

// templateData prepares data for templates based on database selection.
func (g *Generator) templateData() map[string]interface{} {
	var engine, driverName, driverImport string

	switch g.database {
	case "postgresql", "postgres":
		engine = "postgresql"
		driverName = "postgres"
		driverImport = `_ "github.com/lib/pq"`
	case "mysql":
		engine = "mysql"
		driverName = "mysql"
		driverImport = `_ "github.com/go-sql-driver/mysql"`
	case "sqlite":
		engine = "sqlite"
		driverName = "sqlite" // modernc.org/sqlite uses "sqlite" driver name
		driverImport = `_ "modernc.org/sqlite"`
	default:
		engine = "postgresql"
		driverName = "postgres"
		driverImport = `_ "github.com/lib/pq"`
	}

	return map[string]interface{}{
		"DatabaseEngine": engine,
		"DriverName":     driverName,
		"DriverImport":   driverImport,
		"ModulePath":     g.modulePath,
		"ProjectName":    g.projectName,
	}
}