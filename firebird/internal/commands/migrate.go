package commands

import (
	"context"
	"os"
	"strconv"

	"github.com/simonhull/firebird-suite/firebird/internal/migrate"
	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/spf13/cobra"
)

// MigrateCmd creates the migrate command with subcommands
func MigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands",
		Long: `Run database migrations using golang-migrate.

Firebird automatically configures migrations from firebird.yml.

Examples:
  firebird migrate up              # Apply all pending migrations
  firebird migrate down            # Rollback last migration
  firebird migrate down 3          # Rollback last 3 migrations
  firebird migrate status          # Show current version
  firebird migrate list            # List all migrations
  firebird migrate force 20250102  # Force version (recovery)
  firebird migrate create add_bio  # Create manual migration`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Check if golang-migrate is installed
			if err := migrate.CheckMigrateInstalled(); err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}
		},
	}

	// Add subcommands
	cmd.AddCommand(migrateUpCmd())
	cmd.AddCommand(migrateDownCmd())
	cmd.AddCommand(migrateStatusCmd())
	cmd.AddCommand(migrateListCmd())
	cmd.AddCommand(migrateForceCmd())
	cmd.AddCommand(migrateCreateCmd())

	return cmd
}

// migrateUpCmd applies all pending migrations
func migrateUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		Long:  "Applies all pending migrations to the database.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			migrator, err := migrate.NewMigrator()
			if err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}

			if err := migrator.Up(context.Background()); err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}
		},
	}
}

// migrateDownCmd rolls back migrations
func migrateDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down [steps]",
		Short: "Rollback migrations",
		Long:  "Rolls back the last N migrations. Default is 1.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			steps := 1
			if len(args) > 0 {
				var err error
				steps, err = strconv.Atoi(args[0])
				if err != nil || steps < 1 {
					output.Error("Steps must be a positive integer")
					os.Exit(1)
				}
			}

			migrator, err := migrate.NewMigrator()
			if err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}

			if err := migrator.Down(context.Background(), steps); err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}
		},
	}
}

// migrateStatusCmd shows migration status
func migrateStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		Long:  "Shows the current migration version.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			migrator, err := migrate.NewMigrator()
			if err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}

			if err := migrator.Status(context.Background()); err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}
		},
	}
}

// migrateForceCmd forces migration version
func migrateForceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "force <version>",
		Short: "Force migration version (recovery tool)",
		Long: `Forces the migration version without running migrations.

This is a recovery tool for fixing broken migration state.
Use with caution!`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			version := args[0]

			migrator, err := migrate.NewMigrator()
			if err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}

			if err := migrator.Force(context.Background(), version); err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}
		},
	}
}

// migrateListCmd lists all migrations with their status
func migrateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all migrations",
		Long:  "Shows all migrations in the migrations directory with their status (applied/pending).",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			migrator, err := migrate.NewMigrator()
			if err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}

			if err := migrator.List(context.Background()); err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}
		},
	}
}

// migrateCreateCmd creates manual migration files
func migrateCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new migration file",
		Long: `Creates empty migration files for manual editing.

The files are created with timestamp prefixes and include helpful comments.

Example:
  firebird migrate create add_user_bio`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			if err := migrate.CreateManualMigration(name); err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}
		},
	}
}
