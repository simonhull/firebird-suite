package project

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/generators/logging"
	appgen "github.com/simonhull/firebird-suite/firebird/internal/generators/main"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/middleware"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/sqlc"
	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	fledgeExec "github.com/simonhull/firebird-suite/fledge/exec"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/input"
	"github.com/simonhull/firebird-suite/fledge/output"
)

//go:embed templates/*
var templatesFS embed.FS

// DatabaseDriver represents the database driver choice
type DatabaseDriver string

const (
	DatabasePostgreSQL DatabaseDriver = "postgres"
	DatabaseMySQL      DatabaseDriver = "mysql"
	DatabaseSQLite     DatabaseDriver = "sqlite"
	DatabaseNone       DatabaseDriver = "none"
)

// RouterType represents the HTTP router choice
type RouterType string

const (
	RouterStdlib RouterType = "stdlib"
	RouterChi    RouterType = "chi"
	RouterGin    RouterType = "gin"
	RouterEcho   RouterType = "echo"
	RouterNone   RouterType = "none"
)

// IsValid checks if router type is valid
func (r RouterType) IsValid() bool {
	switch r {
	case RouterStdlib, RouterChi, RouterGin, RouterEcho, RouterNone:
		return true
	default:
		return false
	}
}

// Description returns human-readable description
func (r RouterType) Description() string {
	switch r {
	case RouterStdlib:
		return "Go 1.22+ standard library (net/http.ServeMux) - recommended"
	case RouterChi:
		return "Chi - lightweight and idiomatic"
	case RouterGin:
		return "Gin - fast and popular"
	case RouterEcho:
		return "Echo - high performance"
	case RouterNone:
		return "None - I'll write my own handlers"
	default:
		return string(r)
	}
}

// Scaffolder scaffolds new Firebird projects
type Scaffolder struct {
	renderer *generator.Renderer
}

// ScaffoldOptions contains options for project scaffolding
type ScaffoldOptions struct {
	ProjectName string
	Module      string
	Path        string
	SkipTidy    bool
	Interactive bool           // If false, skip interactive prompts
	Database    DatabaseDriver // Database driver choice
	Router      RouterType     // HTTP router choice
}

// NewScaffolder creates a new project scaffolder
func NewScaffolder() *Scaffolder {
	return &Scaffolder{
		renderer: generator.NewRenderer(),
	}
}

// Scaffold creates a new Firebird project with the given options
// Returns Operations to be executed by the caller, and post-execution metadata
func (s *Scaffolder) Scaffold(opts *ScaffoldOptions) ([]generator.Operation, *ScaffoldResult, error) {
	// 1. Resolve project path
	projectPath := filepath.Join(opts.Path, opts.ProjectName)

	// 2. Check if directory exists (early validation)
	if _, err := os.Stat(projectPath); err == nil {
		return nil, nil, fmt.Errorf("directory '%s' already exists. Choose a different name or location", opts.ProjectName)
	}

	// 2.5. Warn if creating inside an existing Go module (only in interactive mode)
	if opts.Interactive {
		if opts.Path == "." || opts.Path == "" {
			parentGoMod := "go.mod"
			if _, err := os.Stat(parentGoMod); err == nil {
				output.Info("⚠️  Creating project inside an existing Go module")
				output.Info("   Generated projects work best in their own directory")
				output.Info("   Consider using: firebird new " + opts.ProjectName + " --path ~/projects")
				output.Info("")

				// Give user a chance to cancel
				if !input.Confirm("Continue anyway?", false) {
					return nil, nil, fmt.Errorf("project creation cancelled")
				}
				output.Info("")
			}
		} else if opts.Path != "" && opts.Path != "." {
			// Check in the target path too
			parentGoMod := filepath.Join(opts.Path, "go.mod")
			if _, err := os.Stat(parentGoMod); err == nil {
				output.Info("⚠️  Target directory contains a go.mod file")
				output.Info("   Generated projects work best in their own directory")
				output.Info("")
			}
		}
	}

	// 3. Get module path (interactive or from flag)
	modulePath := opts.Module
	if modulePath == "" && opts.Interactive {
		defaultModule := fmt.Sprintf("github.com/username/%s", opts.ProjectName)
		modulePath = input.Prompt("Module path", defaultModule)
		output.Verbose(fmt.Sprintf("Using module path: %s", modulePath))
	} else if modulePath == "" {
		// Non-interactive: use sensible default
		modulePath = fmt.Sprintf("github.com/username/%s", opts.ProjectName)
	}

	// 4. Detect Go version
	goVersion := detectGoVersion()
	output.Verbose(fmt.Sprintf("Detected Go version: %s", goVersion))

	// 5. Prepare template data
	data := &ProjectData{
		Name:      opts.ProjectName,
		Module:    modulePath,
		GoVersion: goVersion,
		Database:  opts.Database,
		Router:    opts.Router,
	}

	// 6. Build operations for core directory structure
	ops, err := s.buildCoreDirectoryOperations(projectPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build directory operations: %w", err)
	}

	// 7. Build operations for core project files
	fileOps, err := s.buildCoreFileOperations(projectPath, data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build file operations: %w", err)
	}
	ops = append(ops, fileOps...)

	// 8. Conditionally add database-specific operations
	if opts.Database != DatabaseNone {
		dbOps, err := s.buildDatabaseOperations(projectPath, data)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build database operations: %w", err)
		}
		ops = append(ops, dbOps...)
	}

	// 8.5. Generate logging, middleware, and main
	loggingOps, err := s.buildLoggingOperations(projectPath, data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build logging operations: %w", err)
	}
	ops = append(ops, loggingOps...)

	// 9. Prepare result metadata
	result := &ScaffoldResult{
		ProjectPath:    projectPath,
		ShouldRunTidy:  !opts.SkipTidy && opts.Interactive && input.Confirm("Run go mod tidy?", true),
		Database:       opts.Database,
		InstallMigrate: opts.Database != DatabaseNone,
		InstallSQLC:    opts.Database != DatabaseNone,
	}

	return ops, result, nil
}

// ScaffoldResult contains metadata about the scaffolding operation
type ScaffoldResult struct {
	ProjectPath    string
	ShouldRunTidy  bool
	Database       DatabaseDriver
	InstallMigrate bool
	InstallSQLC    bool
}

// ProjectData is the data passed to templates
type ProjectData struct {
	Name            string         // Project name (e.g., "myapp")
	Module          string         // Go module path (e.g., "github.com/username/myapp")
	GoVersion       string         // Go version (e.g., "1.25")
	Database        DatabaseDriver // Database driver
	Router          RouterType     // HTTP router
	HasRealtime     bool           // true if any schema has realtime enabled
	RealtimeBackend string         // "memory" or "nats"
	NatsURL         string         // NATS server URL (only if backend=nats)
}

// buildCoreDirectoryOperations creates operations for core directories (always created)
func (s *Scaffolder) buildCoreDirectoryOperations(projectPath string) ([]generator.Operation, error) {
	var ops []generator.Operation

	// Create .gitkeep files to ensure core directories exist
	keepDirs := []string{
		filepath.Join(projectPath, "internal", "schemas"),
		filepath.Join(projectPath, "internal", "models"),
		filepath.Join(projectPath, "internal", "handlers"),
	}

	for _, dir := range keepDirs {
		keepPath := filepath.Join(dir, ".gitkeep")
		ops = append(ops, &generator.WriteFileOp{
			Path:    keepPath,
			Content: []byte{}, // Empty file
			Mode:    0644,
		})
	}

	return ops, nil
}

// buildCoreFileOperations generates operations for core project files (always created)
func (s *Scaffolder) buildCoreFileOperations(projectPath string, data *ProjectData) ([]generator.Operation, error) {
	var ops []generator.Operation

	files := map[string]string{
		"firebird.yml.tmpl": "firebird.yml",
		"go.mod.tmpl":       "go.mod",
		".air.toml.tmpl":    ".air.toml",
		".gitignore.tmpl":   ".gitignore",
		"main.go.tmpl":      "cmd/server/main.go",
		"config.go.tmpl":    "internal/config/config.go",
		"logger.go.tmpl":    "internal/config/logger.go",
		"routes.go.tmpl":    "internal/routes/routes.go",
	}

	for tmplName, outputPath := range files {
		fullPath := filepath.Join(projectPath, outputPath)

		// Render template
		content, err := s.renderer.RenderFS(templatesFS, "templates/"+tmplName, data)
		if err != nil {
			return nil, fmt.Errorf("failed to render %s: %w", tmplName, err)
		}

		// Build operation
		ops = append(ops, &generator.WriteFileOp{
			Path:    fullPath,
			Content: content,
			Mode:    0644,
		})
	}

	return ops, nil
}

// buildDatabaseOperations generates operations for database-specific files
func (s *Scaffolder) buildDatabaseOperations(projectPath string, data *ProjectData) ([]generator.Operation, error) {
	var ops []generator.Operation

	// Create db/migrations directory
	migrationsKeep := filepath.Join(projectPath, "db", "migrations", ".gitkeep")
	ops = append(ops, &generator.WriteFileOp{
		Path:    migrationsKeep,
		Content: []byte{},
		Mode:    0644,
	})

	// Create database.yml
	databaseYML := generateDatabaseYML(data.Database, data.Name)
	dbYMLPath := filepath.Join(projectPath, "config", "database.yml")
	ops = append(ops, &generator.WriteFileOp{
		Path:    dbYMLPath,
		Content: []byte(databaseYML),
		Mode:    0644,
	})

	// Initialize SQLC
	output.Info("Initializing SQLC")
	sqlcGen := sqlc.New(projectPath, data.Name, string(data.Database), data.Module)
	sqlcOps, err := sqlcGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("generating SQLC config: %w", err)
	}
	ops = append(ops, sqlcOps...)
	output.Success("SQLC initialized")

	return ops, nil
}

// generateDatabaseYML creates database.yml content based on driver
func generateDatabaseYML(driver DatabaseDriver, projectName string) string {
	switch driver {
	case DatabasePostgreSQL:
		return fmt.Sprintf(`development:
  driver: postgres
  host: localhost
  port: 5432
  database: %s_development
  username: postgres
  password: postgres
  sslmode: disable

test:
  driver: postgres
  host: localhost
  port: 5432
  database: %s_test
  username: postgres
  password: postgres
  sslmode: disable

production:
  driver: postgres
  url: ${DATABASE_URL}
`, projectName, projectName)

	case DatabaseMySQL:
		return fmt.Sprintf(`development:
  driver: mysql
  host: localhost
  port: 3306
  database: %s_development
  username: root
  password: root

test:
  driver: mysql
  host: localhost
  port: 3306
  database: %s_test
  username: root
  password: root

production:
  driver: mysql
  url: ${DATABASE_URL}
`, projectName, projectName)

	case DatabaseSQLite:
		return `development:
  driver: sqlite
  database: db/development.db

test:
  driver: sqlite
  database: db/test.db

production:
  driver: sqlite
  database: db/production.db
`

	default:
		return ""
	}
}

// detectGoVersion detects the installed Go version
func detectGoVersion() string {
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err != nil {
		return "1.25" // Fallback
	}

	// Parse "go version go1.23.4 linux/amd64" -> "1.23"
	versionStr := string(out)
	parts := strings.Fields(versionStr)
	if len(parts) < 3 {
		return "1.25" // Fallback
	}

	// Extract version (e.g., "go1.23.4" -> "1.23")
	fullVersion := strings.TrimPrefix(parts[2], "go")
	versionParts := strings.Split(fullVersion, ".")
	if len(versionParts) >= 2 {
		// Parse as integers for proper comparison
		major, err1 := strconv.Atoi(versionParts[0])
		minor, err2 := strconv.Atoi(versionParts[1])

		if err1 != nil || err2 != nil {
			return "1.25" // Fallback on parse error
		}

		// Ensure minimum version 1.25
		if major < 1 || (major == 1 && minor < 25) {
			return "1.25"
		}

		return fmt.Sprintf("%d.%d", major, minor)
	}

	return "1.25" // Fallback
}

// buildLoggingOperations generates operations for logging, middleware, and main
func (s *Scaffolder) buildLoggingOperations(projectPath string, data *ProjectData) ([]generator.Operation, error) {
	var ops []generator.Operation

	// Generate logging package
	output.Info("Generating logging package")
	loggingGen := logging.New(projectPath)
	loggingOps, err := loggingGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("generating logging: %w", err)
	}
	ops = append(ops, loggingOps...)
	output.Success("Logging package generated")

	// Generate middleware package
	output.Info("Generating middleware package")
	middlewareGen := middleware.New(projectPath)
	middlewareOps, err := middlewareGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("generating middleware: %w", err)
	}
	ops = append(ops, middlewareOps...)
	output.Success("Middleware package generated")

	// Generate main.go
	output.Info("Generating main.go")
	mainGenerator := appgen.New(projectPath, data.Module)
	mainOps, err := mainGenerator.Generate()
	if err != nil {
		return nil, fmt.Errorf("generating main.go: %w", err)
	}
	ops = append(ops, mainOps...)
	output.Success("Main.go generated")

	return ops, nil
}

// RunGoModTidy runs go mod tidy in the project directory
// This is exported so the CLI can call it after operations are executed
func (s *Scaffolder) RunGoModTidy(projectPath string) error {
	executor := fledgeExec.NewExecutor(&fledgeExec.Options{
		Dir:    projectPath,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	return executor.Run(context.Background(), "go", "mod", "tidy")
}

// detectRealtimeConfig scans all .firebird.yml files for realtime configuration
func (s *Scaffolder) detectRealtimeConfig(projectPath string) (bool, string, string) {
	schemasDir := filepath.Join(projectPath, "schemas")

	// Return early if schemas directory doesn't exist
	if _, err := os.Stat(schemasDir); os.IsNotExist(err) {
		return false, "", ""
	}

	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		return false, "", ""
	}

	hasRealtime := false
	backend := "memory"                   // default
	natsURL := "nats://localhost:4222"    // default

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".firebird.yml") {
			continue
		}

		schemaPath := filepath.Join(schemasDir, entry.Name())
		def, err := schema.Parse(schemaPath)
		if err != nil {
			continue
		}

		if def.Spec.Realtime != nil && def.Spec.Realtime.Enabled {
			hasRealtime = true
			if def.Spec.Realtime.Backend != "" {
				backend = def.Spec.Realtime.Backend
			}
			if def.Spec.Realtime.NatsURL != "" {
				natsURL = def.Spec.Realtime.NatsURL
			}
			break // Found one, that's enough
		}
	}

	return hasRealtime, backend, natsURL
}
