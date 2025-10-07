package module

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

// Installer orchestrates module installation and removal
type Installer struct {
	projectPath   string
	projectModule string
}

// NewInstaller creates a new module installer
// projectPath: absolute path to project root (e.g., "/path/to/project")
// projectModule: Go module path (e.g., "github.com/user/project")
func NewInstaller(projectPath, projectModule string) *Installer {
	return &Installer{
		projectPath:   projectPath,
		projectModule: projectModule,
	}
}

// InstallOptions holds configuration for module installation
type InstallOptions struct {
	ModuleName    string                 // Module to install (e.g., "falcon")
	ModuleVersion string                 // Version to install (e.g., "1.0.0")
	ModuleConfig  map[string]interface{} // Module-specific configuration
	ConfigFields  []ConfigField          // Config fields to add to config.go
}

// Install installs a module into the project
// This is the main orchestration function that:
// 1. Updates config.go with module fields
// 2. Generates module wiring file
// 3. Regenerates orchestrator
// 4. Updates firebird.yml
func (i *Installer) Install(ctx context.Context, opts InstallOptions) error {
	// Validate options
	if opts.ModuleName == "" {
		return fmt.Errorf("module name is required")
	}
	if opts.ModuleVersion == "" {
		return fmt.Errorf("module version is required")
	}

	// Step 1: Update config.go with module fields (if provided)
	var configOps []generator.Operation
	if len(opts.ConfigFields) > 0 {
		ops, err := i.updateConfig(opts)
		if err != nil {
			return fmt.Errorf("updating config: %w", err)
		}
		configOps = ops
	}

	// Step 2: Update firebird.yml FIRST (orchestrator needs to read this)
	firebirdOps, err := i.updateFirebirdYml(opts)
	if err != nil {
		return fmt.Errorf("updating firebird.yml: %w", err)
	}

	// Execute firebird.yml update immediately so orchestrator can read it
	for _, op := range firebirdOps {
		if err := op.Execute(ctx); err != nil {
			return fmt.Errorf("executing firebird.yml update: %w", err)
		}
	}

	// Step 3: Generate module wiring
	wiringOps, err := i.generateWiring(opts)
	if err != nil {
		return fmt.Errorf("generating wiring: %w", err)
	}

	// Step 4: Regenerate orchestrator (reads from firebird.yml)
	orchestratorOps, err := i.regenerateOrchestrator()
	if err != nil {
		return fmt.Errorf("regenerating orchestrator: %w", err)
	}

	// Execute remaining operations
	allOps := append(configOps, wiringOps...)
	allOps = append(allOps, orchestratorOps...)

	for _, op := range allOps {
		if err := op.Execute(ctx); err != nil {
			return fmt.Errorf("executing operation: %w", err)
		}
	}

	return nil
}

// Uninstall removes a module from the project
// This is the reverse of Install:
// 1. Removes module fields from config.go
// 2. Deletes module wiring file
// 3. Regenerates orchestrator (without this module)
// 4. Removes module from firebird.yml
func (i *Installer) Uninstall(ctx context.Context, moduleName string) error {
	if moduleName == "" {
		return fmt.Errorf("module name is required")
	}

	// Step 1: Remove from firebird.yml FIRST (orchestrator needs to read this)
	firebirdOps, err := i.removeFromFirebirdYml(moduleName)
	if err != nil {
		return fmt.Errorf("removing from firebird.yml: %w", err)
	}

	// Execute firebird.yml update immediately so orchestrator can read it
	for _, op := range firebirdOps {
		if err := op.Execute(ctx); err != nil {
			return fmt.Errorf("executing firebird.yml update: %w", err)
		}
	}

	// Step 2: Remove from config.go
	configOps, err := i.removeFromConfig(moduleName)
	if err != nil {
		return fmt.Errorf("removing from config: %w", err)
	}

	// Step 3: Delete module wiring file
	wiringOps, err := i.deleteWiring(moduleName)
	if err != nil {
		return fmt.Errorf("deleting wiring: %w", err)
	}

	// Step 4: Regenerate orchestrator (reads from firebird.yml)
	orchestratorOps, err := i.regenerateOrchestrator()
	if err != nil {
		return fmt.Errorf("regenerating orchestrator: %w", err)
	}

	// Execute remaining operations
	allOps := append(configOps, wiringOps...)
	allOps = append(allOps, orchestratorOps...)

	for _, op := range allOps {
		if err := op.Execute(ctx); err != nil {
			return fmt.Errorf("executing operation: %w", err)
		}
	}

	return nil
}

// updateConfig updates config.go with module fields
func (i *Installer) updateConfig(opts InstallOptions) ([]generator.Operation, error) {
	configPath := filepath.Join(i.projectPath, "internal", "config", "config.go")
	builder := NewConfigBuilder(configPath)

	// Ensure Modules field exists in Config struct
	if err := builder.EnsureModulesField(); err != nil {
		return nil, fmt.Errorf("ensuring modules field: %w", err)
	}

	// Add module config fields
	if err := builder.AddModuleConfig(opts.ModuleName, opts.ConfigFields); err != nil {
		return nil, fmt.Errorf("adding module config: %w", err)
	}

	return builder.Build()
}

// generateWiring generates module wiring file
func (i *Installer) generateWiring(opts InstallOptions) ([]generator.Operation, error) {
	gen := NewWiringGenerator(i.projectPath, i.projectModule)

	// First, ensure registry exists
	registryOps, err := gen.GenerateRegistry()
	if err != nil {
		return nil, fmt.Errorf("generating registry: %w", err)
	}

	// Then generate module wiring
	moduleOps, err := gen.GenerateModuleWiring(opts.ModuleName, opts.ModuleVersion)
	if err != nil {
		return nil, fmt.Errorf("generating module wiring: %w", err)
	}

	return append(registryOps, moduleOps...), nil
}

// regenerateOrchestrator regenerates the orchestrator
func (i *Installer) regenerateOrchestrator() ([]generator.Operation, error) {
	gen := NewWiringGenerator(i.projectPath, i.projectModule)
	return gen.RegenerateOrchestrator()
}

// updateFirebirdYml updates firebird.yml with module info
func (i *Installer) updateFirebirdYml(opts InstallOptions) ([]generator.Operation, error) {
	configPath := filepath.Join(i.projectPath, "firebird.yml")

	// Create operation wrapper
	op := &firebirdYmlOperation{
		action: "add",
		path:   configPath,
		name:   opts.ModuleName,
		version: opts.ModuleVersion,
		config: opts.ModuleConfig,
	}

	return []generator.Operation{op}, nil
}

// removeFromConfig removes module fields from config.go
func (i *Installer) removeFromConfig(moduleName string) ([]generator.Operation, error) {
	// For Phase 4, config removal is not implemented
	// Module config fields remain in config.go even after module is uninstalled
	// This is acceptable for Phase 4 - full cleanup will be added in later phases
	return []generator.Operation{}, nil
}

// deleteWiring deletes module wiring file
func (i *Installer) deleteWiring(moduleName string) ([]generator.Operation, error) {
	filename := fmt.Sprintf("wiring_%s.go", moduleName)
	path := filepath.Join(i.projectPath, "internal", "modules", filename)

	op := &deleteFileOperation{
		path: path,
	}

	return []generator.Operation{op}, nil
}

// removeFromFirebirdYml removes module from firebird.yml
func (i *Installer) removeFromFirebirdYml(moduleName string) ([]generator.Operation, error) {
	configPath := filepath.Join(i.projectPath, "firebird.yml")

	op := &firebirdYmlOperation{
		action: "remove",
		path:   configPath,
		name:   moduleName,
	}

	return []generator.Operation{op}, nil
}

// firebirdYmlOperation wraps firebird.yml modifications as an Operation
type firebirdYmlOperation struct {
	action  string                 // "add" or "remove"
	path    string                 // Path to firebird.yml
	name    string                 // Module name
	version string                 // Module version (for add)
	config  map[string]interface{} // Module config (for add)
}

func (op *firebirdYmlOperation) Validate(ctx context.Context, force bool) error {
	// Check if firebird.yml exists
	if _, err := os.Stat(op.path); os.IsNotExist(err) {
		return fmt.Errorf("firebird.yml not found at %s", op.path)
	}
	return nil
}

func (op *firebirdYmlOperation) Execute(ctx context.Context) error {
	switch op.action {
	case "add":
		return AddModule(op.path, op.name, op.version, op.config)
	case "remove":
		return RemoveModule(op.path, op.name)
	default:
		return fmt.Errorf("unknown action: %s", op.action)
	}
}

func (op *firebirdYmlOperation) Description() string {
	switch op.action {
	case "add":
		return fmt.Sprintf("Add module %s v%s to firebird.yml", op.name, op.version)
	case "remove":
		return fmt.Sprintf("Remove module %s from firebird.yml", op.name)
	default:
		return "Unknown firebird.yml operation"
	}
}

// deleteFileOperation deletes a file
type deleteFileOperation struct {
	path string
}

func (op *deleteFileOperation) Validate(ctx context.Context, force bool) error {
	// No validation needed - deletion is idempotent
	return nil
}

func (op *deleteFileOperation) Execute(ctx context.Context) error {
	// Remove file (ignore if not exists)
	if err := os.Remove(op.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting file %s: %w", op.path, err)
	}
	return nil
}

func (op *deleteFileOperation) Description() string {
	return fmt.Sprintf("Delete %s", op.path)
}
