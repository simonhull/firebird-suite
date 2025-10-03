package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/generators/migration"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/model"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/scaffold"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/spf13/cobra"
)

// GenerateCmd creates and returns the 'generate' command for code generation
func GenerateCmd() *cobra.Command {
	var force, skip, diff, dryRun bool

	cmd := &cobra.Command{
		Use:   "generate [type] [name]",
		Short: "Generate code from schema",
		Long: `Generate code from .firebird.yml schema files.

Available types:
  scaffold   - Create empty schema file
  model      - Generate Go struct
  migration  - Generate SQL migration

Examples:
  firebird generate scaffold User
  firebird generate model User
  firebird generate model User --dry-run
  firebird generate migration User
  firebird generate migration User --force`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			genType := args[0]
			name := args[1]

			// Validate mutually exclusive flags (force, skip, diff)
			// Note: --dry-run can be combined with these
			flagCount := 0
			conflictingFlags := []string{}
			if force {
				flagCount++
				conflictingFlags = append(conflictingFlags, "--force")
			}
			if skip {
				flagCount++
				conflictingFlags = append(conflictingFlags, "--skip")
			}
			if diff {
				flagCount++
				conflictingFlags = append(conflictingFlags, "--diff")
			}

			if flagCount > 1 {
				output.Error(fmt.Sprintf("Conflicting flags: %v are mutually exclusive", conflictingFlags))
				os.Exit(1)
			}

			output.Verbose(fmt.Sprintf("Generating %s: %s (dry-run=%v, force=%v)", genType, name, dryRun, force))

			// Route to appropriate generator based on type
			var ops []generator.Operation
			var err error

			switch genType {
			case "model":
				gen := model.NewGenerator()
				ops, err = gen.Generate(name)
			case "migration":
				gen := migration.NewGenerator()
				ops, err = gen.Generate(name)
			case "scaffold":
				// Scaffold generator still uses old pattern (writes directly)
				// TODO: Update scaffold generator to return operations
				gen := scaffold.NewGenerator()
				if err := gen.Generate(name); err != nil {
					output.Error(err.Error())
					os.Exit(1)
				}
				output.Success(fmt.Sprintf("Generated %s: %s", genType, name))
				return
			default:
				output.Error(fmt.Sprintf("Unknown generator type: %s", genType))
				output.Info("Available types:")
				output.Step("scaffold   - Create empty schema file")
				output.Step("model      - Generate Go struct")
				output.Step("migration  - Generate SQL migration")
				os.Exit(1)
			}

			if err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}

			// Execute operations through Fledge
			writer := cmd.OutOrStdout()
			if err := generator.Execute(ctx, ops, generator.ExecuteOptions{
				DryRun: dryRun,
				Force:  force,
				Writer: writer,
			}); err != nil {
				// Enhance error messages at CLI layer
				if strings.Contains(err.Error(), "already exists") && !force && !dryRun {
					output.Error(err.Error())
					output.Info("\nTip: Use --force to overwrite, --skip to skip, or --diff to review changes")
					os.Exit(1)
				}
				output.Error(err.Error())
				os.Exit(1)
			}

			// Add summary message
			if dryRun {
				fmt.Fprintln(writer, "\n✓ Dry-run complete. Run without --dry-run to create files.")
			} else {
				output.Success(fmt.Sprintf("Generated %s: %s", genType, name))
			}
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be generated without creating files")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files without asking")
	cmd.Flags().BoolVar(&skip, "skip", false, "Skip existing files without asking")
	cmd.Flags().BoolVar(&diff, "diff", false, "Show diff before overwriting")

	return cmd
}
