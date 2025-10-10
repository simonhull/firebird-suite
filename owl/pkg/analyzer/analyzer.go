package analyzer

import (
	"fmt"
	"os"

	"github.com/simonhull/firebird-suite/fledge/filesystem"
	"github.com/simonhull/firebird-suite/fledge/project"
)

// Analyzer analyzes Go projects and extracts documentation
type Analyzer struct {
	parser   *Parser
	detector ConventionDetector
}

// NewAnalyzer creates a new Analyzer
func NewAnalyzer(detector ConventionDetector) *Analyzer {
	return &Analyzer{
		parser:   NewParser(),
		detector: detector,
	}
}

// ConventionDetector is an interface for detecting architectural conventions
type ConventionDetector interface {
	Detect(pkg *Package) []*Convention
}

// Analyze analyzes a Go project and returns a Project structure
func (a *Analyzer) Analyze(rootPath string) (*Project, error) {
	proj := &Project{
		RootPath: rootPath,
		Packages: make([]*Package, 0),
	}

	// Detect Firebird project using Fledge utility
	isFirebird, firebirdConfig, err := project.DetectFirebirdProject(rootPath)
	if err != nil {
		// Log warning but continue
		fmt.Printf("⚠️  Warning: error detecting Firebird project: %v\n", err)
	}

	if isFirebird && firebirdConfig != nil {
		fmt.Println("🔥 Firebird project detected!")
		fmt.Printf("   Database: %s, Router: %s\n",
			firebirdConfig.Database,
			firebirdConfig.Router)
		fmt.Println()

		// Convert to Owl's FirebirdConfig type
		proj.IsFirebirdProject = true
		proj.FirebirdConfig = &FirebirdConfig{
			ConfigPath: firebirdConfig.ConfigPath,
			Database:   firebirdConfig.Database,
			Router:     firebirdConfig.Router,
		}
	}

	// Walk the project directory using Fledge utility
	err = filesystem.Walk(rootPath, filesystem.WalkOptions{
		IgnoreDirs: []string{
			"node_modules", "vendor", ".git", ".svn",
			"dist", "build", "bin", "tmp", "temp", "testdata",
			".idea", ".vscode", ".vs",
		},
	}, func(path string, info os.FileInfo) error {
		// Only process directories (we'll parse all .go files in each directory)
		if !info.IsDir() {
			return nil
		}

		// Parse all Go files in this directory
		files, err := a.parser.ParseDirectory(path)
		if err != nil {
			// Log warning but continue
			fmt.Printf("⚠️  Warning: failed to parse %s: %v\n", path, err)
			return nil
		}

		if len(files) == 0 {
			return nil
		}

		// Extract package information
		pkg, err := a.parser.ParsePackage(files)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to extract package from %s: %v\n", path, err)
			return nil
		}

		if pkg != nil {
			pkg.Path = path

			// Detect conventions
			if a.detector != nil {
				a.applyConventions(pkg)
			}

			proj.Packages = append(proj.Packages, pkg)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("analyzing project: %w", err)
	}

	return proj, nil
}

// applyConventions applies convention detection to package types and functions
func (a *Analyzer) applyConventions(pkg *Package) {
	conventions := a.detector.Detect(pkg)

	// Note: The detector will have already assigned conventions to types/functions
	// This is just for tracking at the package level
	_ = conventions
}
