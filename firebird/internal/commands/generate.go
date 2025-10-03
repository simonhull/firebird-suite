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
	var timestamps, softDeletes, generateAll bool
	var intID bool // NEW: Use int64 instead of UUID for primary key

	cmd := &cobra.Command{
		Use:   "generate [type] [name] [field:type[:modifier]...]",
		Short: "Generate code from schema",
		Long: `Generate code from .firebird.yml schema files.

Available types:
  scaffold   - Create schema file from field specifications
  model      - Generate Go struct
  migration  - Generate SQL migration

Examples:
  firebird generate scaffold Post title:string body:text
  firebird generate scaffold User email:string:unique name:string --timestamps
  firebird generate scaffold Product name:string price:Decimal --int-id
  firebird generate scaffold Article title:string:unique published_at:timestamp:index --timestamps --generate
  firebird generate model User
  firebird generate model User --dry-run
  firebird generate migration User
  firebird generate migration User --force

Field syntax for scaffold: name:type[:modifier]
  Modifiers: index, unique
  Supported types: string, text, int, int64, float64, bool, timestamp, date, time
  Third-party types: UUID, Decimal, NullString

Primary keys default to UUID. Use --int-id for int64 with auto-increment.`,
		Args: cobra.MinimumNArgs(2),
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
				// Parse field specifications from remaining args
				fieldArgs := args[2:]
				fields, err := parseFields(fieldArgs)
				if err != nil {
					output.Error(fmt.Sprintf("Invalid field specification: %v", err))
					os.Exit(1)
				}

				// Build scaffold options
				opts := scaffold.Options{
					Name:        name,
					Fields:      fields,
					Timestamps:  timestamps,
					SoftDeletes: softDeletes,
					Generate:    generateAll,
					IntID:       intID, // NEW: Pass flag
				}

				// Generate operations
				gen := scaffold.NewGenerator()
				ops, err = gen.Generate(opts)
			default:
				output.Error(fmt.Sprintf("Unknown generator type: %s", genType))
				output.Info("Available types:")
				output.Step("scaffold   - Create schema file from field specifications")
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
				// Different message for scaffold with --generate
				if genType == "scaffold" && generateAll {
					fmt.Fprintln(writer, "\n✨ Scaffold complete! Generated schema, model, and migration.")
				} else if genType == "scaffold" {
					fmt.Fprintln(writer, "\n✨ Schema created! Run model and migration generators:")
					fmt.Fprintf(writer, "  firebird generate model %s\n", name)
					fmt.Fprintf(writer, "  firebird generate migration %s\n", name)
				} else {
					output.Success(fmt.Sprintf("Generated %s: %s", genType, name))
				}
			}
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be generated without creating files")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files without asking")
	cmd.Flags().BoolVar(&skip, "skip", false, "Skip existing files without asking")
	cmd.Flags().BoolVar(&diff, "diff", false, "Show diff before overwriting")
	cmd.Flags().BoolVar(&timestamps, "timestamps", false, "Add created_at and updated_at fields (scaffold only)")
	cmd.Flags().BoolVar(&softDeletes, "soft-deletes", false, "Add deleted_at field for soft deletes (scaffold only)")
	cmd.Flags().BoolVar(&generateAll, "generate", false, "Also generate model and migration files (scaffold only)")
	cmd.Flags().BoolVar(&intID, "int-id", false, "Use int64 with auto-increment instead of UUID for primary key (scaffold only)")

	return cmd
}

// parseFields parses field specifications from command line
func parseFields(fieldArgs []string) ([]scaffold.Field, error) {
	var fields []scaffold.Field

	for _, arg := range fieldArgs {
		parts := strings.Split(arg, ":")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid field format: %s (expected name:type[:modifier])", arg)
		}

		field := scaffold.Field{
			Name: parts[0],
			Type: parts[1],
		}

		// Parse optional modifier
		if len(parts) == 3 {
			modifier := parts[2]
			if modifier != "index" && modifier != "unique" {
				return nil, fmt.Errorf("invalid modifier: %s (valid: index, unique)", modifier)
			}
			field.Modifier = modifier
		} else if len(parts) > 3 {
			return nil, fmt.Errorf("invalid field format: %s (too many colons)", arg)
		}

		// Validate field type
		if !isValidType(field.Type) {
			return nil, fmt.Errorf("invalid type: %s (valid: string, text, int, int64, float64, bool, timestamp, date, time)", field.Type)
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// isValidType checks if a field type is supported
func isValidType(t string) bool {
	valid := map[string]bool{
		// Built-in types
		"string":    true,
		"text":      true,
		"int":       true,
		"int64":     true,
		"float64":   true,
		"bool":      true,
		"timestamp": true,
		"date":      true,
		"time":      true,
		// Third-party types
		"UUID":       true,
		"Decimal":    true,
		"NullString": true,
	}
	return valid[t]
}
