package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/project"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/spf13/cobra"
)

// NewCmd creates and returns the 'new' command for scaffolding projects
func NewCmd() *cobra.Command {
	var module string
	var path string
	var skipTidy bool
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new Firebird project",
		Long: `Creates a new Firebird project with:
• Go module initialization
• Standard directory structure
• Configuration (firebird.yml)
• Logging (slog)
• Hot reload (Air)

Example:
  firebird new myapp
  firebird new myapp --module github.com/username/myapp
  firebird new myapp --path ~/projects
  firebird new myapp --dry-run
  firebird new myapp --skip-tidy`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			projectName := args[0]

			output.Verbose(fmt.Sprintf("Creating new Firebird project: %s (dry-run=%v, force=%v)", projectName, dryRun, force))

			scaffolder := project.NewScaffolder()

			// Create scaffold options
			opts := &project.ScaffoldOptions{
				ProjectName: projectName,
				Module:      module,
				Path:        path,
				SkipTidy:    skipTidy,
				Interactive: !dryRun, // Disable prompts in dry-run mode
			}

			ops, result, err := scaffolder.Scaffold(opts)
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
					output.Info("\nTip: Use --force to overwrite existing files")
					os.Exit(1)
				}
				output.Error(err.Error())
				os.Exit(1)
			}

			// Add summary message
			if dryRun {
				fmt.Fprintln(writer, "\n✓ Dry-run complete. Run without --dry-run to create project.")
				return
			}

			// Run go mod tidy if requested (only in non-dry-run mode)
			if result.ShouldRunTidy {
				output.Info("Running go mod tidy...")
				if err := scaffolder.RunGoModTidy(result.ProjectPath); err != nil {
					output.Error("Failed to run go mod tidy (you can run it manually later)")
					output.Verbose(err.Error())
				} else {
					output.Verbose("Ran go mod tidy successfully")
				}
			}

			output.Success(fmt.Sprintf("Created Firebird project: %s", projectName))
			output.Info("Next steps:")

			// Show cd command if path was custom
			if path != "" && path != "." {
				output.Step(fmt.Sprintf("cd %s/%s", path, projectName))
			} else {
				output.Step(fmt.Sprintf("cd %s", projectName))
			}

			if skipTidy {
				output.Step("go mod tidy  # Skipped, run manually")
			}
			output.Step("air  # Start with hot reload")
		},
	}

	cmd.Flags().StringVar(&module, "module", "", "Go module path (e.g., github.com/username/myapp)")
	cmd.Flags().StringVar(&path, "path", ".", "Directory to create project in")
	cmd.Flags().BoolVar(&skipTidy, "skip-tidy", false, "Skip running go mod tidy")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be generated without creating files")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files without prompting")

	return cmd
}
