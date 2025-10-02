package commands

import (
	"fmt"
	"os"

	"github.com/simonhull/firebird-suite/firebird/internal/project"
	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/spf13/cobra"
)

// NewCmd creates and returns the 'new' command for scaffolding projects
func NewCmd() *cobra.Command {
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
  firebird new myapp`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]

			output.Verbose(fmt.Sprintf("Creating new Firebird project: %s", projectName))

			scaffolder := project.NewScaffolder()
			if err := scaffolder.Scaffold(projectName); err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}

			output.Success(fmt.Sprintf("Created Firebird project: %s", projectName))
			output.Info("Next steps:")
			output.Step(fmt.Sprintf("cd %s", projectName))
			output.Step("go mod tidy")
			output.Step("air  # Start with hot reload")
		},
	}

	return cmd
}
