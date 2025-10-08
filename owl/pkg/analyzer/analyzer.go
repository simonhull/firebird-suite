package analyzer

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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
	project := &Project{
		Packages: make([]*Package, 0),
	}

	// Detect Firebird project
	project.IsFirebirdProject, project.FirebirdConfig = a.detectFirebirdProject(rootPath)

	if project.IsFirebirdProject {
		fmt.Println("🔥 Firebird project detected!")
		if project.FirebirdConfig != nil {
			fmt.Printf("   Database: %s, Router: %s\n",
				project.FirebirdConfig.Database,
				project.FirebirdConfig.Router)
		}
		fmt.Println()
	}

	// Walk the project directory
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and vendor/testdata
		if info.IsDir() {
			name := info.Name()
			if name[0] == '.' || name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
		}

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

			project.Packages = append(project.Packages, pkg)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("analyzing project: %w", err)
	}

	return project, nil
}

// applyConventions applies convention detection to package types and functions
func (a *Analyzer) applyConventions(pkg *Package) {
	conventions := a.detector.Detect(pkg)

	// Note: The detector will have already assigned conventions to types/functions
	// This is just for tracking at the package level
	_ = conventions
}

// detectFirebirdProject checks if this is a Firebird-generated project
func (a *Analyzer) detectFirebirdProject(rootPath string) (bool, *FirebirdConfig) {
	configPath := filepath.Join(rootPath, "firebird.yml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, nil
	}

	// Parse basic config
	var config struct {
		Application struct {
			Database struct {
				Driver string `yaml:"driver"`
			} `yaml:"database"`
			Router struct {
				Type string `yaml:"type"`
			} `yaml:"router"`
		} `yaml:"application"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return false, nil
	}

	return true, &FirebirdConfig{
		ConfigPath: configPath,
		Database:   config.Application.Database.Driver,
		Router:     config.Application.Router.Type,
	}
}
