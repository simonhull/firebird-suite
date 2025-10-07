package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/simonhull/firebird-suite/firebird/internal/module"
	"github.com/spf13/cobra"
)

// ModuleCmd returns the module command with add/remove subcommands
func ModuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module",
		Short: "Manage Firebird modules",
		Long:  "Install, remove, and manage Firebird modules in your project",
	}

	cmd.AddCommand(moduleAddCmd())
	cmd.AddCommand(moduleRemoveCmd())
	cmd.AddCommand(moduleListCmd())

	return cmd
}

// moduleAddCmd installs a module
func moduleAddCmd() *cobra.Command {
	var version string
	var configFields []string

	cmd := &cobra.Command{
		Use:   "add [module-name]",
		Short: "Install a Firebird module",
		Long: `Install a Firebird module into your project.

This command will:
1. Add module configuration fields to internal/config/config.go
2. Generate module wiring code in internal/modules/
3. Update firebird.yml with module metadata

Example:
  firebird module add falcon --version 1.0.0`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleName := args[0]

			// Get project info
			projectPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}

			// Load firebird.yml to get project module
			configPath := filepath.Join(projectPath, "firebird.yml")
			cfg, err := module.LoadFirebirdConfig(configPath)
			if err != nil {
				return fmt.Errorf("loading firebird.yml: %w", err)
			}

			// Create installer
			installer := module.NewInstaller(projectPath, cfg.Project.Module)

			// For Phase 5, we use empty config fields
			// In a real implementation, this would query a module registry
			// or read from a module specification file
			opts := module.InstallOptions{
				ModuleName:    moduleName,
				ModuleVersion: version,
				ModuleConfig:  nil,
				ConfigFields:  []module.ConfigField{},
			}

			// Install module
			ctx := context.Background()
			if err := installer.Install(ctx, opts); err != nil {
				return fmt.Errorf("installing module: %w", err)
			}

			fmt.Printf("✓ Module %s v%s installed successfully\n", moduleName, version)
			fmt.Println("\nGenerated files:")
			fmt.Printf("  - internal/modules/wiring_%s.go\n", moduleName)
			fmt.Printf("  - internal/modules/wiring_modules.go (updated)\n")
			fmt.Println("\nNext steps:")
			fmt.Println("  1. Review generated wiring code")
			fmt.Println("  2. Add module-specific initialization logic")
			fmt.Println("  3. Update your main.go to call modules.InitModules()")

			return nil
		},
	}

	cmd.Flags().StringVar(&version, "version", "latest", "Module version to install")
	cmd.Flags().StringSliceVar(&configFields, "config", nil, "Config fields (name:type:tag format)")

	return cmd
}

// moduleRemoveCmd uninstalls a module
func moduleRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [module-name]",
		Short: "Remove a Firebird module",
		Long: `Remove a Firebird module from your project.

This command will:
1. Delete module wiring code
2. Update orchestrator to remove module initialization
3. Remove module from firebird.yml

Note: Module config fields in config.go are NOT removed for safety.
You can manually remove them if desired.

Example:
  firebird module remove falcon`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleName := args[0]

			// Get project info
			projectPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}

			// Load firebird.yml to get project module
			configPath := filepath.Join(projectPath, "firebird.yml")
			cfg, err := module.LoadFirebirdConfig(configPath)
			if err != nil {
				return fmt.Errorf("loading firebird.yml: %w", err)
			}

			// Check if module is installed
			if _, exists := cfg.Modules[moduleName]; !exists {
				return fmt.Errorf("module %s is not installed", moduleName)
			}

			// Create installer
			installer := module.NewInstaller(projectPath, cfg.Project.Module)

			// Uninstall module
			ctx := context.Background()
			if err := installer.Uninstall(ctx, moduleName); err != nil {
				return fmt.Errorf("removing module: %w", err)
			}

			fmt.Printf("✓ Module %s removed successfully\n", moduleName)
			fmt.Println("\nDeleted files:")
			fmt.Printf("  - internal/modules/wiring_%s.go\n", moduleName)
			fmt.Println("\nUpdated files:")
			fmt.Println("  - internal/modules/wiring_modules.go")
			fmt.Println("  - firebird.yml")

			return nil
		},
	}

	return cmd
}

// moduleListCmd lists installed modules
func moduleListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed modules",
		Long:  "Show all installed Firebird modules and their versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get project info
			projectPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}

			// Load firebird.yml
			configPath := filepath.Join(projectPath, "firebird.yml")
			cfg, err := module.LoadFirebirdConfig(configPath)
			if err != nil {
				return fmt.Errorf("loading firebird.yml: %w", err)
			}

			if len(cfg.Modules) == 0 {
				fmt.Println("No modules installed")
				return nil
			}

			fmt.Println("Installed modules:")
			for name, modCfg := range cfg.Modules {
				fmt.Printf("  - %s (v%s)\n", name, modCfg.Version)
			}

			return nil
		},
	}

	return cmd
}
