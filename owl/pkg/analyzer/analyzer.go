package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
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
			return fmt.Errorf("parsing directory %s: %w", path, err)
		}

		if len(files) == 0 {
			return nil
		}

		// Extract package information
		pkg, err := a.parser.ParsePackage(files)
		if err != nil {
			return fmt.Errorf("parsing package in %s: %w", path, err)
		}

		if pkg != nil {
			// Detect conventions
			if a.detector != nil {
				pkg.Types = a.applyConventions(pkg)
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

// applyConventions applies convention detection to package types
func (a *Analyzer) applyConventions(pkg *Package) []*Type {
	conventions := a.detector.Detect(pkg)

	// TODO: Match conventions to types
	// This is a placeholder - will be implemented when we build the detector

	return pkg.Types
}
