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

	fledgeExec "github.com/simonhull/firebird-suite/fledge/exec"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/input"
	"github.com/simonhull/firebird-suite/fledge/output"
)

//go:embed templates/*
var templatesFS embed.FS

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
}

// NewScaffolder creates a new project scaffolder
func NewScaffolder() *Scaffolder {
	return &Scaffolder{
		renderer: generator.NewRenderer(),
	}
}

// Scaffold creates a new Firebird project with the given options
func (s *Scaffolder) Scaffold(opts *ScaffoldOptions) error {
	// 1. Resolve project path
	projectPath := filepath.Join(opts.Path, opts.ProjectName)

	// 2. Check if directory exists
	if _, err := os.Stat(projectPath); err == nil {
		return fmt.Errorf("directory '%s' already exists. Choose a different name or location", opts.ProjectName)
	}

	// 2.5. Warn if creating inside an existing Go module
	if opts.Path == "." || opts.Path == "" {
		parentGoMod := "go.mod"
		if _, err := os.Stat(parentGoMod); err == nil {
			output.Info("⚠️  Creating project inside an existing Go module")
			output.Info("   Generated projects work best in their own directory")
			output.Info("   Consider using: firebird new " + opts.ProjectName + " --path ~/projects")
			output.Info("")

			// Give user a chance to cancel
			if !input.Confirm("Continue anyway?", false) {
				return fmt.Errorf("project creation cancelled")
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

	// 3. Get module path (interactive or from flag)
	modulePath := opts.Module
	if modulePath == "" {
		defaultModule := fmt.Sprintf("github.com/username/%s", opts.ProjectName)
		modulePath = input.Prompt("Module path", defaultModule)
		output.Verbose(fmt.Sprintf("Using module path: %s", modulePath))
	}

	// 4. Detect Go version
	goVersion := detectGoVersion()
	output.Verbose(fmt.Sprintf("Detected Go version: %s", goVersion))

	// 5. Prepare template data
	data := &ProjectData{
		Name:      opts.ProjectName,
		Module:    modulePath,
		GoVersion: goVersion,
	}

	// 6. Create directory structure
	if err := s.createDirectories(projectPath); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}
	output.Verbose("Created directory structure")

	// 7. Generate files from templates
	if err := s.generateFiles(projectPath, data); err != nil {
		return fmt.Errorf("failed to generate files: %w", err)
	}
	output.Verbose("Generated project files")

	// 8. Run go mod tidy (unless skipped)
	if !opts.SkipTidy {
		if input.Confirm("Run go mod tidy?", true) {
			if err := s.runGoModTidy(projectPath); err != nil {
				output.Error("Failed to run go mod tidy (you can run it manually later)")
				output.Verbose(err.Error())
			} else {
				output.Verbose("Ran go mod tidy successfully")
			}
		}
	}

	return nil
}

// ProjectData is the data passed to templates
type ProjectData struct {
	Name      string // Project name (e.g., "myapp")
	Module    string // Go module path (e.g., "github.com/username/myapp")
	GoVersion string // Go version (e.g., "1.25")
}

// createDirectories creates the standard Firebird project structure
func (s *Scaffolder) createDirectories(projectPath string) error {
	dirs := []string{
		projectPath,
		filepath.Join(projectPath, "cmd", "server"),
		filepath.Join(projectPath, "internal", "config"),
		filepath.Join(projectPath, "internal", "models"),
		filepath.Join(projectPath, "internal", "handlers"),
		filepath.Join(projectPath, "internal", "routes"),
		filepath.Join(projectPath, "internal", "schemas"),
		filepath.Join(projectPath, "migrations"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// generateFiles generates all project files from templates
func (s *Scaffolder) generateFiles(projectPath string, data *ProjectData) error {
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
			return fmt.Errorf("failed to render %s: %w", tmplName, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", outputPath, err)
		}
	}

	return nil
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

// runGoModTidy runs go mod tidy in the project directory
func (s *Scaffolder) runGoModTidy(projectPath string) error {
	executor := fledgeExec.NewExecutor(&fledgeExec.Options{
		Dir:    projectPath,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	return executor.Run(context.Background(), "go", "mod", "tidy")
}
