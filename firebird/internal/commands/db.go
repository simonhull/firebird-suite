package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/spf13/cobra"
)

// DBCmd creates the db command for database operations.
func DBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database operations (queries, migrations)",
		Long: `Database operations including SQLC query generation and migration management.

Examples:
  firebird db generate    # Compile SQLC queries
  firebird db vet         # Validate SQLC queries`,
	}

	cmd.AddCommand(newDBGenerateCommand())
	cmd.AddCommand(newDBVetCommand())

	return cmd
}

func newDBGenerateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate type-safe Go code from SQL queries",
		Long:  `Runs 'sqlc generate' to compile SQL queries into type-safe Go code.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSQLCCommand("generate")
		},
	}
}

func newDBVetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "vet",
		Short: "Validate SQL queries",
		Long:  `Runs 'sqlc vet' to validate SQL queries for common errors.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSQLCCommand("vet")
		},
	}
}

func runSQLCCommand(command string) error {
	// Check if sqlc is installed
	if _, err := exec.LookPath("sqlc"); err != nil {
		return fmt.Errorf(`sqlc not found. Install it with:
  go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

Or visit: https://docs.sqlc.dev/en/latest/overview/install.html`)
	}

	// Check if we're in a Firebird project
	if _, err := os.Stat("sqlc.yaml"); os.IsNotExist(err) {
		return fmt.Errorf("not in a Firebird project (sqlc.yaml not found)")
	}

	output.Info(fmt.Sprintf("Running sqlc %s", command))

	// Run sqlc command
	sqlcCmd := exec.CommandContext(context.Background(), "sqlc", command)
	sqlcCmd.Stdout = os.Stdout
	sqlcCmd.Stderr = os.Stderr

	if err := sqlcCmd.Run(); err != nil {
		return fmt.Errorf("sqlc %s failed: %w", command, err)
	}

	output.Success(fmt.Sprintf("sqlc %s completed", command))
	return nil
}
