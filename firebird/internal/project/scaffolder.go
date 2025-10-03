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
	Interactive bool // If false, skip interactive prompts
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
	}

	// 6. Build operations for directory structure
	ops, err := s.buildDirectoryOperations(projectPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build directory operations: %w", err)
	}

	// 7. Build operations for files from templates
	fileOps, err := s.buildFileOperations(projectPath, data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build file operations: %w", err)
	}
	ops = append(ops, fileOps...)

	// 8. Prepare result metadata
	result := &ScaffoldResult{
		ProjectPath: projectPath,
		ShouldRunTidy: !opts.SkipTidy && opts.Interactive && input.Confirm("Run go mod tidy?", true),
	}

	return ops, result, nil
}

// ScaffoldResult contains metadata about the scaffolding operation
type ScaffoldResult struct {
	ProjectPath   string
	ShouldRunTidy bool
}

// ProjectData is the data passed to templates
type ProjectData struct {
	Name      string // Project name (e.g., "myapp")
	Module    string // Go module path (e.g., "github.com/username/myapp")
	GoVersion string // Go version (e.g., "1.25")
}

// buildDirectoryOperations creates operations for the standard Firebird project structure
// Note: WriteFileOp handles parent directory creation automatically, so we create
// .gitkeep files in each directory to ensure they exist
func (s *Scaffolder) buildDirectoryOperations(projectPath string) ([]generator.Operation, error) {
	var ops []generator.Operation

	// Create .gitkeep files to ensure directories exist
	// This is needed for migrations/ and internal/schemas/ which start empty
	keepDirs := []string{
		filepath.Join(projectPath, "migrations"),
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

// buildFileOperations generates operations for all project files from templates
func (s *Scaffolder) buildFileOperations(projectPath string, data *ProjectData) ([]generator.Operation, error) {
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
