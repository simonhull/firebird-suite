package analyzer

import (
	"context"
	"fmt"
	"os"

	"github.com/simonhull/firebird-suite/fledge/filesystem"
	"github.com/simonhull/firebird-suite/fledge/project"
	"github.com/simonhull/firebird-suite/owl/pkg/logger"
)

// Analyzer analyzes Go projects and extracts documentation
type Analyzer struct {
	parser   *Parser
	detector ConventionDetector
	logger   logger.Logger
}

// NewAnalyzer creates a new Analyzer
func NewAnalyzer(detector ConventionDetector) *Analyzer {
	return &Analyzer{
		parser:   NewParser(),
		detector: detector,
		logger:   logger.Default(),
	}
}

// WithLogger returns a new Analyzer with the specified logger
func (a *Analyzer) WithLogger(log logger.Logger) *Analyzer {
	return &Analyzer{
		parser:   a.parser,
		detector: a.detector,
		logger:   log,
	}
}

// ConventionDetector is an interface for detecting architectural conventions
type ConventionDetector interface {
	Detect(pkg *Package) []*Convention
}

// Analyze analyzes a Go project and returns a Project structure
func (a *Analyzer) Analyze(rootPath string) (*Project, error) {
	return a.AnalyzeWithContext(context.Background(), rootPath)
}

// AnalyzeWithContext analyzes a Go project with context support for cancellation
func (a *Analyzer) AnalyzeWithContext(ctx context.Context, rootPath string) (*Project, error) {
	a.logger.Info("Starting project analysis", logger.F("path", rootPath))

	proj := &Project{
		RootPath: rootPath,
		Packages: make([]*Package, 0, 50), // Pre-allocate for typical project size
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Detect Firebird project using Fledge utility
	isFirebird, firebirdConfig, err := project.DetectFirebirdProject(rootPath)
	if err != nil {
		a.logger.Warn("Error detecting Firebird project", logger.F("error", err))
	}

	if isFirebird && firebirdConfig != nil {
		a.logger.Info("Firebird project detected",
			logger.F("database", firebirdConfig.Database),
			logger.F("router", firebirdConfig.Router))

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
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Only process directories (we'll parse all .go files in each directory)
		if !info.IsDir() {
			return nil
		}

		// Parse all Go files in this directory
		files, err := a.parser.ParseDirectory(path)
		if err != nil {
			a.logger.Warn("Failed to parse directory",
				logger.F("path", path),
				logger.F("error", err))
			return nil
		}

		if len(files) == 0 {
			return nil
		}

		// Extract package information
		pkg, err := a.parser.ParsePackage(files)
		if err != nil {
			a.logger.Warn("Failed to extract package",
				logger.F("path", path),
				logger.F("error", err))
			return nil
		}

		if pkg != nil {
			pkg.Path = path

			// Detect conventions
			if a.detector != nil {
				a.applyConventions(pkg)
			}

			proj.Packages = append(proj.Packages, pkg)
			a.logger.Debug("Parsed package",
				logger.F("name", pkg.Name),
				logger.F("types", len(pkg.Types)),
				logger.F("functions", len(pkg.Functions)))
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("analyzing project: %w", err)
	}

	a.logger.Info("Project analysis complete",
		logger.F("packages", len(proj.Packages)))

	return proj, nil
}

// applyConventions applies convention detection to package types and functions
func (a *Analyzer) applyConventions(pkg *Package) {
	conventions := a.detector.Detect(pkg)

	// Note: The detector will have already assigned conventions to types/functions
	// This is just for tracking at the package level
	_ = conventions
}
