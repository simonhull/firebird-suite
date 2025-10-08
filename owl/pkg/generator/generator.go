package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/simonhull/firebird-suite/owldocs/pkg/analyzer"
)

// Generator generates static documentation
type Generator struct {
	outputPath string
	theme      string
}

// NewGenerator creates a new Generator
func NewGenerator(outputPath, theme string) *Generator {
	return &Generator{
		outputPath: outputPath,
		theme:      theme,
	}
}

// Generate creates static HTML documentation
func (g *Generator) Generate(project *analyzer.Project) error {
	// Create output directory
	if err := os.MkdirAll(g.outputPath, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// TODO: Implement HTML generation
	// This will use templates and generate the static site

	fmt.Printf("📦 Analyzed %d packages\n", len(project.Packages))
	fmt.Printf("📁 Output directory: %s\n", g.outputPath)

	return nil
}

// writeFile writes content to a file in the output directory
func (g *Generator) writeFile(relativePath, content string) error {
	fullPath := filepath.Join(g.outputPath, relativePath)

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}
