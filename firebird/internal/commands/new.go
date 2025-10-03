package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/project"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/input"
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
	var database string

	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new Firebird project",
		Long: `Creates a new Firebird project with:
• Go module initialization
• Standard directory structure
• Configuration (firebird.yml)
• Optional database setup
• Logging (slog)
• Hot reload (Air)

Example:
  firebird new myapp
  firebird new myapp --module github.com/username/myapp
  firebird new myapp --database postgres
  firebird new myapp --database none
  firebird new myapp --path ~/projects
  firebird new myapp --dry-run
  firebird new myapp --skip-tidy`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			projectName := args[0]
			writer := cmd.OutOrStdout()

			output.Verbose(fmt.Sprintf("Creating new Firebird project: %s (dry-run=%v, force=%v)", projectName, dryRun, force))

			// Get database choice
			var dbDriver project.DatabaseDriver
			if database != "" {
				// Non-interactive: use flag
				dbDriver = project.DatabaseDriver(database)
				if err := validateDatabaseChoice(dbDriver); err != nil {
					output.Error(err.Error())
					os.Exit(1)
				}
			} else if !dryRun {
				// Interactive: prompt user
				dbDriver = promptForDatabase(writer)
			} else {
				// Dry-run without flag: default to postgres
				dbDriver = project.DatabasePostgreSQL
			}

			scaffolder := project.NewScaffolder()

			// Create scaffold options
			opts := &project.ScaffoldOptions{
				ProjectName: projectName,
				Module:      module,
				Path:        path,
				SkipTidy:    skipTidy,
				Interactive: !dryRun, // Disable prompts in dry-run mode
				Database:    dbDriver,
			}

			ops, result, err := scaffolder.Scaffold(opts)
			if err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}

			// Execute operations through Fledge
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
				fmt.Fprintln(writer, "\n📦 Installing dependencies...")
				if err := scaffolder.RunGoModTidy(result.ProjectPath); err != nil {
					output.Error("Failed to run go mod tidy (you can run it manually later)")
					output.Verbose(err.Error())
				} else {
					fmt.Fprintln(writer, "✓ go mod tidy complete")
				}
			}

			// Print success message with database-specific info
			printSuccessMessage(writer, result, path, skipTidy)
		},
	}

	cmd.Flags().StringVar(&module, "module", "", "Go module path (e.g., github.com/username/myapp)")
	cmd.Flags().StringVar(&path, "path", ".", "Directory to create project in")
	cmd.Flags().BoolVar(&skipTidy, "skip-tidy", false, "Skip running go mod tidy")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be generated without creating files")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files without prompting")
	cmd.Flags().StringVar(&database, "database", "", "Database driver: postgres, mysql, sqlite, none")

	return cmd
}

// promptForDatabase prompts the user to select a database driver
func promptForDatabase(writer io.Writer) project.DatabaseDriver {
	fmt.Fprintln(writer, "\n🗄️  Select database:")
	fmt.Fprintln(writer, "  1. PostgreSQL (recommended for production)")
	fmt.Fprintln(writer, "  2. MySQL")
	fmt.Fprintln(writer, "  3. SQLite (great for development/testing)")
	fmt.Fprintln(writer, "  4. None (API-only, no database)")

	choiceStr := input.Prompt("\nChoice [1-4]", "1")

	switch choiceStr {
	case "1":
		return project.DatabasePostgreSQL
	case "2":
		return project.DatabaseMySQL
	case "3":
		return project.DatabaseSQLite
	case "4":
		return project.DatabaseNone
	default:
		fmt.Fprintln(writer, "Invalid choice, defaulting to PostgreSQL")
		return project.DatabasePostgreSQL
	}
}

// validateDatabaseChoice validates the database driver string
func validateDatabaseChoice(db project.DatabaseDriver) error {
	valid := map[project.DatabaseDriver]bool{
		project.DatabasePostgreSQL: true,
		project.DatabaseMySQL:      true,
		project.DatabaseSQLite:     true,
		project.DatabaseNone:       true,
	}

	if !valid[db] {
		return fmt.Errorf("invalid database: %s (valid: postgres, mysql, sqlite, none)", db)
	}
	return nil
}

// printSuccessMessage prints a database-aware success message
func printSuccessMessage(writer io.Writer, result *project.ScaffoldResult, path string, skipTidy bool) {
	projectName := filepath.Base(result.ProjectPath)

	fmt.Fprintf(writer, "\n✨ Project %s created successfully!\n\n", projectName)

	if result.Database != project.DatabaseNone {
		fmt.Fprintf(writer, "📊 Database: %s\n", result.Database)
		fmt.Fprintln(writer, "\nNext steps:")

		// Show cd command
		if path != "" && path != "." {
			fmt.Fprintf(writer, "  1. cd %s/%s\n", path, projectName)
		} else {
			fmt.Fprintf(writer, "  1. cd %s\n", projectName)
		}

		if skipTidy {
			fmt.Fprintln(writer, "  2. go mod tidy  # Skipped, run manually")
		}

		fmt.Fprintln(writer, "  2. Update config/database.yml with your credentials")
		fmt.Fprintln(writer, "  3. Run: firebird generate migration <ModelName>")
		fmt.Fprintln(writer, "  4. Run: firebird migrate up")
		fmt.Fprintln(writer, "  5. Run: firebird serve")
	} else {
		fmt.Fprintln(writer, "🚀 No database configured (API-only mode)")
		fmt.Fprintln(writer, "\nNext steps:")

		// Show cd command
		if path != "" && path != "." {
			fmt.Fprintf(writer, "  1. cd %s/%s\n", path, projectName)
		} else {
			fmt.Fprintf(writer, "  1. cd %s\n", projectName)
		}

		if skipTidy {
			fmt.Fprintln(writer, "  2. go mod tidy  # Skipped, run manually")
		}

		fmt.Fprintln(writer, "  2. Run: firebird serve")
		fmt.Fprintln(writer, "\n💡 To add a database later, edit firebird.yml")
	}
}
