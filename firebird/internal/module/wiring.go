package module

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/wiring/*.tmpl
var wiringTemplates embed.FS

// WiringGenerator generates module wiring code
type WiringGenerator struct {
	projectPath   string
	projectModule string
}

// NewWiringGenerator creates a new wiring generator
// projectPath: path to project root (e.g., "/path/to/project")
// projectModule: Go module path (e.g., "github.com/user/project")
func NewWiringGenerator(projectPath, projectModule string) *WiringGenerator {
	return &WiringGenerator{
		projectPath:   projectPath,
		projectModule: projectModule,
	}
}

// ModuleWiringData holds template data for module initialization
type ModuleWiringData struct {
	ModuleName     string   // "falcon"
	ModuleVersion  string   // "1.0.0"
	GeneratedAt    string   // "2025-10-07 15:30:00"
	ProjectModule  string   // "github.com/user/project"
	InitFunc       string   // "Falcon" (PascalCase)
	Imports        []string // Additional imports needed
	InitBody       string   // Initialization code
	ValidateConfig bool     // Whether to generate validation function
	ValidationBody string   // Config validation code
}

// OrchestratorData holds template data for the orchestrator
type OrchestratorData struct {
	GeneratedAt   string       // "2025-10-07 15:30:00"
	ProjectModule string       // "github.com/user/project"
	Modules       []ModuleInfo // List of installed modules
}

// ModuleInfo represents a module in the orchestrator
type ModuleInfo struct {
	Name     string // "falcon"
	Version  string // "1.0.0"
	InitFunc string // "Falcon" (PascalCase)
}

// GenerateRegistry creates the ModuleRegistry type
// This is called once when the first module is installed
// Uses WriteFileIfNotExistsOp so user modifications are preserved
func (g *WiringGenerator) GenerateRegistry() ([]generator.Operation, error) {
	// Render template
	content, err := g.renderTemplate("wiring/registry.go.tmpl", map[string]interface{}{
		"GeneratedAt": time.Now().Format("2006-01-02 15:04:05"),
	})
	if err != nil {
		return nil, fmt.Errorf("rendering registry template: %w", err)
	}

	// Create operations
	ops := []generator.Operation{
		// Write registry file (if not exists)
		&generator.WriteFileIfNotExistsOp{
			Path:    filepath.Join(g.projectPath, "internal", "modules", "registry.go"),
			Content: content,
			Mode:    0644,
		},
	}

	return ops, nil
}

// GenerateModuleWiring creates the Init<Module>() function
// This is called when a module is installed
// The file is regenerated on each install (overwrites existing)
func (g *WiringGenerator) GenerateModuleWiring(moduleName, moduleVersion string) ([]generator.Operation, error) {
	initFunc := toPascalCase(moduleName)

	// Build template data
	data := ModuleWiringData{
		ModuleName:     moduleName,
		ModuleVersion:  moduleVersion,
		GeneratedAt:    time.Now().Format("2006-01-02 15:04:05"),
		ProjectModule:  g.projectModule,
		InitFunc:       initFunc,
		Imports:        []string{}, // Empty for Phase 3
		InitBody:       g.generateInitBody(moduleName),
		ValidateConfig: false, // No validation in Phase 3
		ValidationBody: "",
	}

	// Render template
	content, err := g.renderTemplate("wiring/module.go.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("rendering module template: %w", err)
	}

	// Create operation
	filename := fmt.Sprintf("wiring_%s.go", moduleName)
	ops := []generator.Operation{
		&generator.WriteFileOp{
			Path:    filepath.Join(g.projectPath, "internal", "modules", filename),
			Content: content,
			Mode:    0644,
		},
	}

	return ops, nil
}

// generateInitBody creates the initialization code for a module
// This is a placeholder - Phase 4 will implement the full logic
func (g *WiringGenerator) generateInitBody(moduleName string) string {
	// For Phase 3, generate a simple comment
	// Phase 4 will use provider.Install() to get actual init code
	return fmt.Sprintf("// TODO: Initialize %s services\n\t// This will be implemented in Phase 4", moduleName)
}

// RegenerateOrchestrator updates the InitModules() function
// This is called whenever a module is added or removed
// Reads firebird.yml to determine which modules are installed
func (g *WiringGenerator) RegenerateOrchestrator() ([]generator.Operation, error) {
	// Load firebird.yml
	cfg, err := LoadFirebirdConfig(filepath.Join(g.projectPath, "firebird.yml"))
	if err != nil {
		return nil, fmt.Errorf("loading firebird.yml: %w", err)
	}

	// Build module list
	modules := []ModuleInfo{}
	for name, modCfg := range cfg.Modules {
		modules = append(modules, ModuleInfo{
			Name:     name,
			Version:  modCfg.Version,
			InitFunc: toPascalCase(name),
		})
	}

	// Sort for deterministic output
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})

	// Build template data
	data := OrchestratorData{
		GeneratedAt:   time.Now().Format("2006-01-02 15:04:05"),
		ProjectModule: g.projectModule,
		Modules:       modules,
	}

	// Render template
	content, err := g.renderTemplate("wiring/orchestrator.go.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("rendering orchestrator template: %w", err)
	}

	// Create operation
	ops := []generator.Operation{
		&generator.WriteFileOp{
			Path:    filepath.Join(g.projectPath, "internal", "modules", "wiring_modules.go"),
			Content: content,
			Mode:    0644,
		},
	}

	return ops, nil
}

// renderTemplate renders a template from the embedded FS
func (g *WiringGenerator) renderTemplate(name string, data interface{}) ([]byte, error) {
	tmplContent, err := wiringTemplates.ReadFile("templates/" + name)
	if err != nil {
		return nil, fmt.Errorf("reading template %s: %w", name, err)
	}

	tmpl, err := template.New(name).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template %s: %w", name, err)
	}

	return buf.Bytes(), nil
}

// toPascalCase converts snake_case or lowercase to PascalCase
// e.g., "falcon" -> "Falcon", "falcon_auth" -> "FalconAuth"
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
