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
	var module string
	var path string
	var skipTidy bool

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
  firebird new myapp --skip-tidy`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]

			output.Verbose(fmt.Sprintf("Creating new Firebird project: %s", projectName))

			scaffolder := project.NewScaffolder()

			// Create scaffold options
			opts := &project.ScaffoldOptions{
				ProjectName: projectName,
				Module:      module,
				Path:        path,
				SkipTidy:    skipTidy,
			}

			if err := scaffolder.Scaffold(opts); err != nil {
				output.Error(err.Error())
				os.Exit(1)
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

	return cmd
}
