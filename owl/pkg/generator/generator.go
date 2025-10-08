package generator

import (
	"github.com/simonhull/firebird-suite/owl/pkg/analyzer"
)

// Generator generates documentation from analyzed projects
type Generator struct {
	outputPath string
	theme      string
}

// New creates a new documentation generator
func New(outputPath, theme string) *Generator {
	return &Generator{
		outputPath: outputPath,
		theme:      theme,
	}
}

// Generate creates documentation files
func (g *Generator) Generate(project *analyzer.Project) error {
	// Placeholder implementation
	// Future: Generate HTML/Markdown files organized by conventions
	return nil
}
