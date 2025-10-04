package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/generators/dto"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/handler"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/helpers"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/migration"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/model"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/query"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/repository"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/routes"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/scaffold"
	"github.com/simonhull/firebird-suite/firebird/internal/generators/service"
	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// GenerateCmd creates and returns the 'generate' command for code generation
func GenerateCmd() *cobra.Command {
	var force, skip, diff, dryRun bool
	var timestamps, softDeletes, generateAll bool
	var intID bool // NEW: Use int64 instead of UUID for primary key
	// Resource generator flags
	var skipModel, skipService, skipHandler, skipRoutes bool
	var skipHelpers, skipQueries, skipRepository, skipDTOs bool
	// Model generator flags
	var modelOutput, modelPackage, modelSchema string

	cmd := &cobra.Command{
		Use:   "generate [type] [name] [field:type[:modifier]...]",
		Short: "Generate code from schema",
		Long: `Generate code from .firebird.yml schema files.

Available types:
  scaffold   - Create schema file from field specifications
  model      - Generate Go struct from schema
  migration  - Generate SQL migration
  service    - Generate service layer
  handler    - Generate HTTP handler
  routes     - Generate route registration
  resource   - Generate complete CRUD stack (model + service + handler + routes)

Examples:
  # Atomic commands (generate individual components)
  firebird generate model User
  firebird generate service User
  firebird generate handler User
  firebird generate routes

  # Composite command (generate full stack)
  firebird generate resource Post
  firebird generate resource Article --skip-handler

  # Scaffold creates just the schema
  firebird generate scaffold Post title:string body:text

  # Custom model options
  firebird generate model User --output internal/domain --package domain
  firebird generate migration User

Field syntax for scaffold: name:type[:modifier]
  Modifiers: index, unique
  Supported types: string, text, int, int64, float64, bool, timestamp, date, time
  Third-party types: UUID, Decimal, NullString

Primary keys default to UUID. Use --int-id for int64 with auto-increment.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			genType := args[0]
			var name string
			if len(args) > 1 {
				name = args[1]
			}

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

				// Determine schema path
				schemaPath := modelSchema
				if schemaPath == "" {
					// Find schema file
					schemaPath, err = model.FindSchemaFile(name)
					if err != nil {
						output.Error(err.Error())
						os.Exit(1)
					}
				}

				// Generate model only
				if modelSchema != "" || modelOutput != "" || modelPackage != "" {
					opts := model.GenerateOptions{
						Name:       name,
						SchemaPath: modelSchema,
						OutputPath: modelOutput,
						Package:    modelPackage,
					}
					ops, err = gen.GenerateWithOptions(opts)
				} else {
					ops, err = gen.Generate(name)
				}

				if err != nil {
					output.Error(err.Error())
					os.Exit(1)
				}
			case "migration":
				gen := migration.NewGenerator()
				ops, err = gen.Generate(name)
			case "service":
				// Get module path
				modulePath, modErr := getModulePath(".")
				if modErr != nil {
					output.Error(fmt.Sprintf("Failed to detect module path: %v", modErr))
					os.Exit(1)
				}

				// Determine schema path
				schemaPath := modelSchema
				if schemaPath == "" {
					// Find schema file
					schemaPath, err = model.FindSchemaFile(name)
					if err != nil {
						output.Error(err.Error())
						os.Exit(1)
					}
				}

				output.Info("Generating service")

				serviceGen := service.New(".", schemaPath, modulePath)
				ops, err = serviceGen.Generate()
				if err != nil {
					output.Error(fmt.Sprintf("Failed to generate service: %v", err))
					os.Exit(1)
				}

				output.Success("Created service")
			case "handler":
				// Check router configuration
				routerType, err := getRouterConfig()
				if err != nil {
					output.Error(fmt.Sprintf("Failed to read router config: %v", err))
					os.Exit(1)
				}

				if routerType == "none" {
					output.Error("Handler generation is disabled (router: none in firebird.yml)")
					output.Info("To enable handlers, update your firebird.yml or run: firebird new --router stdlib")
					os.Exit(1)
				}

				// Get module path
				modulePath, modErr := getModulePath(".")
				if modErr != nil {
					output.Error(fmt.Sprintf("Failed to detect module path: %v", modErr))
					os.Exit(1)
				}

				// Determine schema path
				schemaPath := modelSchema
				if schemaPath == "" {
					schemaPath, err = model.FindSchemaFile(name)
					if err != nil {
						output.Error(err.Error())
						os.Exit(1)
					}
				}

				output.Info("Generating handler")

				handlerGen := handler.New(".", schemaPath, modulePath)
				ops, err = handlerGen.Generate()
				if err != nil {
					output.Error(fmt.Sprintf("Failed to generate handler: %v", err))
					os.Exit(1)
				}

				output.Success("Created handler")
			case "routes":
				// Check router configuration
				routerType, err := getRouterConfig()
				if err != nil {
					output.Error(fmt.Sprintf("Failed to read router config: %v", err))
					os.Exit(1)
				}

				if routerType == "none" {
					output.Error("Route generation is disabled (router: none in firebird.yml)")
					output.Info("To enable routes, update your firebird.yml or run: firebird new --router stdlib")
					os.Exit(1)
				}

				// Get module path
				modulePath, modErr := getModulePath(".")
				if modErr != nil {
					output.Error(fmt.Sprintf("Failed to detect module path: %v", modErr))
					os.Exit(1)
				}

				output.Info("Generating routes")

				routesGen := routes.New(".", modulePath, routerType)
				ops, err = routesGen.Generate()
				if err != nil {
					output.Error(fmt.Sprintf("Failed to generate routes: %v", err))
					os.Exit(1)
				}

				output.Success("Created routes")
			case "resource":
				ctx := context.Background()

				// Check router configuration
				routerType, err := getRouterConfig()
				if err != nil {
					output.Error(fmt.Sprintf("Failed to read router config: %v", err))
					os.Exit(1)
				}

				output.Info(fmt.Sprintf("Generating resource: %s", name))
				output.Step("This will generate: model → helpers → queries → repository → DTOs → service → handler → routes")

				// Get module path (needed for service, handler, routes)
				modulePath, modErr := getModulePath(".")
				if modErr != nil {
					output.Error(fmt.Sprintf("Failed to detect module path: %v", modErr))
					os.Exit(1)
				}

				// Find or use provided schema path
				schemaPath := modelSchema
				if schemaPath == "" {
					schemaPath, err = model.FindSchemaFile(name)
					if err != nil {
						output.Error(err.Error())
						os.Exit(1)
					}
				}

				// 1. Generate Model
				if !skipModel {
					output.Info("Generating model")
					modelGen := model.NewGenerator()

					var modelOps []generator.Operation
					if modelSchema != "" || modelOutput != "" || modelPackage != "" {
						opts := model.GenerateOptions{
							Name:       name,
							SchemaPath: modelSchema,
							OutputPath: modelOutput,
							Package:    modelPackage,
						}
						modelOps, err = modelGen.GenerateWithOptions(opts)
					} else {
						modelOps, err = modelGen.Generate(name)
					}

					if err != nil {
						output.Error(fmt.Sprintf("Failed to generate model: %v", err))
						os.Exit(1)
					}

					if err := generator.Execute(ctx, modelOps, generator.ExecuteOptions{
						DryRun: dryRun,
						Force:  force,
						Writer: cmd.OutOrStdout(),
					}); err != nil {
						output.Error(fmt.Sprintf("Failed to create model: %v", err))
						os.Exit(1)
					}

					output.Success("Created model")
				}

				// 2. Generate Helpers (if first resource)
				if !skipHelpers {
					output.Info("Generating helpers infrastructure")

					helpersGen := helpers.New(".", modulePath)
					helpersOps, helpersErr := helpersGen.Generate()
					if helpersErr != nil {
						output.Error(fmt.Sprintf("Failed to generate helpers: %v", helpersErr))
						os.Exit(1)
					}

					if err := generator.Execute(ctx, helpersOps, generator.ExecuteOptions{
						DryRun: dryRun,
						Force:  force,
						Writer: cmd.OutOrStdout(),
					}); err != nil {
						output.Error(fmt.Sprintf("Failed to create helpers: %v", err))
						os.Exit(1)
					}

					output.Success("Created helpers infrastructure")
				}

				// 3. Generate Queries
				if !skipQueries {
					output.Info("Generating queries")

					queryGen := query.New(".", schemaPath)
					queryOps, queryErr := queryGen.Generate()
					if queryErr != nil {
						output.Error(fmt.Sprintf("Failed to generate queries: %v", queryErr))
						os.Exit(1)
					}

					if err := generator.Execute(ctx, queryOps, generator.ExecuteOptions{
						DryRun: dryRun,
						Force:  force,
						Writer: cmd.OutOrStdout(),
					}); err != nil {
						output.Error(fmt.Sprintf("Failed to create queries: %v", err))
						os.Exit(1)
					}

					output.Success("Created queries")
					output.Info("💡 Run 'firebird db generate' to compile queries")
				}

				// 4. Generate Repository
				if !skipRepository {
					output.Info("Generating repository")

					repoGen := repository.New(".", schemaPath, modulePath)
					repoOps, repoErr := repoGen.Generate()
					if repoErr != nil {
						output.Error(fmt.Sprintf("Failed to generate repository: %v", repoErr))
						os.Exit(1)
					}

					if err := generator.Execute(ctx, repoOps, generator.ExecuteOptions{
						DryRun: dryRun,
						Force:  force,
						Writer: cmd.OutOrStdout(),
					}); err != nil {
						output.Error(fmt.Sprintf("Failed to create repository: %v", err))
						os.Exit(1)
					}

					output.Success("Created repository")
				}

				// 5. Generate DTOs
				if !skipDTOs {
					output.Info("Generating DTOs")

					dtoGen := dto.New(".", schemaPath, modulePath)
					dtoOps, dtoErr := dtoGen.Generate()
					if dtoErr != nil {
						output.Error(fmt.Sprintf("Failed to generate DTOs: %v", dtoErr))
						os.Exit(1)
					}

					if err := generator.Execute(ctx, dtoOps, generator.ExecuteOptions{
						DryRun: dryRun,
						Force:  force,
						Writer: cmd.OutOrStdout(),
					}); err != nil {
						output.Error(fmt.Sprintf("Failed to create DTOs: %v", err))
						os.Exit(1)
					}

					output.Success("Created DTOs")
				}

				// 6. Generate Service
				if !skipService {
					output.Info("Generating service")

					serviceGen := service.New(".", schemaPath, modulePath)
					serviceOps, serviceErr := serviceGen.Generate()
					if serviceErr != nil {
						output.Error(fmt.Sprintf("Failed to generate service: %v", serviceErr))
						os.Exit(1)
					}

					if err := generator.Execute(ctx, serviceOps, generator.ExecuteOptions{
						DryRun: dryRun,
						Force:  force,
						Writer: cmd.OutOrStdout(),
					}); err != nil {
						output.Error(fmt.Sprintf("Failed to create service: %v", err))
						os.Exit(1)
					}

					output.Success("Created service")
				}

				// 7. Generate Handler (if router != none)
				if !skipHandler && routerType != "none" {
					output.Info("Generating handler")

					handlerGen := handler.New(".", schemaPath, modulePath)
					handlerOps, handlerErr := handlerGen.Generate()
					if handlerErr != nil {
						output.Error(fmt.Sprintf("Failed to generate handler: %v", handlerErr))
						os.Exit(1)
					}

					if err := generator.Execute(ctx, handlerOps, generator.ExecuteOptions{
						DryRun: dryRun,
						Force:  force,
						Writer: cmd.OutOrStdout(),
					}); err != nil {
						output.Error(fmt.Sprintf("Failed to create handler: %v", err))
						os.Exit(1)
					}

					output.Success("Created handler")
				} else if routerType == "none" {
					output.Info("Skipping handler generation (router: none)")
				}

				// 8. Generate Routes (if router != none)
				if !skipRoutes && routerType != "none" {
					output.Info("Generating routes")

					routesGen := routes.New(".", modulePath, routerType)
					routesOps, routesErr := routesGen.Generate()
					if routesErr != nil {
						output.Error(fmt.Sprintf("Failed to generate routes: %v", routesErr))
						os.Exit(1)
					}

					if err := generator.Execute(ctx, routesOps, generator.ExecuteOptions{
						DryRun: dryRun,
						Force:  force,
						Writer: cmd.OutOrStdout(),
					}); err != nil {
						output.Error(fmt.Sprintf("Failed to create routes: %v", err))
						os.Exit(1)
					}

					output.Success("Created routes")
				} else if routerType == "none" {
					output.Info("Skipping route generation (router: none)")
				}

				// Summary
				if !dryRun {
					output.Success(fmt.Sprintf("\n✨ Resource complete: %s", name))
					output.Info("Generated components:")
					if !skipModel {
						output.Step("✓ Model (internal/models)")
					}
					if !skipHelpers {
						output.Step("✓ Helpers (internal/validation, internal/handlers)")
					}
					if !skipQueries {
						output.Step("✓ Queries (internal/queries)")
					}
					if !skipRepository {
						output.Step("✓ Repository (internal/repositories)")
					}
					if !skipDTOs {
						output.Step("✓ DTOs (internal/dto)")
					}
					if !skipService {
						output.Step("✓ Service (internal/services)")
					}
					if !skipHandler && routerType != "none" {
						output.Step("✓ Handler (internal/handlers)")
					}
					if !skipRoutes && routerType != "none" {
						output.Step("✓ Routes (internal/handlers/routes.go)")
					}
				}
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
				output.Step("service    - Generate service layer")
				output.Step("handler    - Generate HTTP handler")
				output.Step("routes     - Generate route registration")
				output.Step("resource   - Generate complete CRUD stack")
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
				} else if genType == "model" {
					output.Success(fmt.Sprintf("Generated model: %s", name))
					output.Info("\n💡 Next steps:")
					output.Step("firebird generate service " + name + " - Generate service layer")
					output.Step("firebird generate handler " + name + " - Generate HTTP handlers")
					output.Step("firebird generate routes - Wire up route registration")
					output.Step("firebird db generate - Compile SQLC queries (if using database)")
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
	// Resource generator flags
	cmd.Flags().BoolVar(&skipModel, "skip-model", false, "Skip model generation (resource only)")
	cmd.Flags().BoolVar(&skipService, "skip-service", false, "Skip service generation (resource only)")
	cmd.Flags().BoolVar(&skipHandler, "skip-handler", false, "Skip handler generation (resource only)")
	cmd.Flags().BoolVar(&skipRoutes, "skip-routes", false, "Skip routes generation (resource only)")
	cmd.Flags().BoolVar(&skipHelpers, "skip-helpers", false, "Skip helpers generation (resource only)")
	cmd.Flags().BoolVar(&skipQueries, "skip-queries", false, "Skip queries generation (resource only)")
	cmd.Flags().BoolVar(&skipRepository, "skip-repository", false, "Skip repository generation (resource only)")
	cmd.Flags().BoolVar(&skipDTOs, "skip-dtos", false, "Skip DTO generation (resource only)")
	// Model generator flags
	cmd.Flags().StringVar(&modelOutput, "output", "", "Custom output path for model file (model only)")
	cmd.Flags().StringVar(&modelPackage, "package", "", "Custom package name for model (model only)")
	cmd.Flags().StringVar(&modelSchema, "schema", "", "Custom schema file path (model only)")

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

// getRouterConfig reads router configuration from firebird.yml
func getRouterConfig() (string, error) {
	data, err := os.ReadFile("firebird.yml")
	if err != nil {
		return "", fmt.Errorf("reading firebird.yml: %w", err)
	}

	var config struct {
		Router string `yaml:"router"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("parsing firebird.yml: %w", err)
	}

	if config.Router == "" {
		return "stdlib", nil // Default to stdlib
	}

	return config.Router, nil
}
