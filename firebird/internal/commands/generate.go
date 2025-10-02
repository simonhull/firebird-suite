package commands

import (
	"fmt"
	"os"

	"github.com/simonhull/firebird-suite/firebird/internal/generators/migration"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/model"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/scaffold"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/spf13/cobra"
)

// Generator is the interface that all atomic generators implement
type Generator interface {
	Generate(name string) error
}

// GenerateCmd creates and returns the 'generate' command for code generation
func GenerateCmd() *cobra.Command {
	var force, skip, diff bool

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
  firebird generate migration User`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			genType := args[0]
			name := args[1]

			// Validate mutually exclusive flags
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

			// Create resolver with conflict resolution strategy
			resolver, err := generator.NewResolver(force, skip, diff)
			if err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}

			output.Verbose(fmt.Sprintf("Generating %s: %s", genType, name))

			// Route to appropriate generator based on type
			var gen Generator
			switch genType {
			case "model":
				gen = model.NewGenerator(resolver)
			case "migration":
				gen = migration.NewGenerator(resolver)
			case "scaffold":
				gen = scaffold.NewGenerator()
			default:
				output.Error(fmt.Sprintf("Unknown generator type: %s", genType))
				output.Info("Available types:")
				output.Step("scaffold   - Create empty schema file")
				output.Step("model      - Generate Go struct")
				output.Step("migration  - Generate SQL migration")
				os.Exit(1)
			}

			// Generate
			if err := gen.Generate(name); err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}

			output.Success(fmt.Sprintf("Generated %s: %s", genType, name))
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files without asking")
	cmd.Flags().BoolVar(&skip, "skip", false, "Skip existing files without asking")
	cmd.Flags().BoolVar(&diff, "diff", false, "Show diff before overwriting")

	return cmd
}
