package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/simonhull/firebird-suite/owl/pkg/analyzer"
	"github.com/simonhull/firebird-suite/owl/pkg/conventions"
	"github.com/simonhull/firebird-suite/owl/pkg/generator"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "owl",
	Short: "Owl - Observation-Based Go Code Analyzer",
	Long: `Owl analyzes Go projects and reports observable patterns and facts.
It focuses on conclusive observations rather than assumptions about project type.`,
}

func generateCmd() *cobra.Command {
	var outputPath string
	var configPath string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "generate [path]",
		Short: "Generate documentation for a Go project",
		Long: `Analyzes a Go project and generates static HTML documentation.

Example:
  owl generate ./internal/handlers
  owl generate ../myproject
  owl generate ../firebird --verbose`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			fmt.Printf("🦉 Analyzing project: %s\n", projectPath)
			if verbose {
				fmt.Println("📊 Verbose mode enabled - detailed analysis output")
			}
			fmt.Println()

			// Create analyzer with convention detector
			detector := conventions.NewDetector()
			a := analyzer.NewAnalyzer(detector)

			// Analyze the project
			project, err := a.Analyze(projectPath)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			// Print analysis results
			printAnalysisResults(project, verbose)

			// Generate documentation
			gen := generator.New(outputPath, "default")
			if err := gen.Generate(project); err != nil {
				return fmt.Errorf("generation failed: %w", err)
			}

			fmt.Printf("\n✅ Documentation generated successfully!\n")
			fmt.Printf("📁 Output: %s\n", outputPath)

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "out", "o", "./docs", "Output directory for generated documentation")
	cmd.Flags().StringVarP(&configPath, "config", "c", "owl.yaml", "Path to configuration file")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed analysis information")

	return cmd
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve documentation with live reload",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🦉 Starting documentation server...")
		fmt.Println("📝 Server will watch for changes and auto-reload")
		fmt.Println("🌐 Visit http://localhost:8080")
		fmt.Println("\n⚠️  Server implementation coming soon!")
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Owl configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🦉 Initializing Owl configuration...")
		fmt.Println("📝 Creating .owl.yml with default settings")
		fmt.Println("✅ Configuration initialized!")
		fmt.Println("\n⚠️  Init implementation coming soon!")
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Owl v%s\n", version)
	},
}

// printAnalysisResults displays analysis results in terminal
func printAnalysisResults(project *analyzer.Project, verbose bool) {
	totalTypes := 0
	totalFunctions := 0
	categoryGroups := make(map[string]map[string]int) // category -> convention name -> count

	for _, pkg := range project.Packages {
		if len(pkg.Types) == 0 && len(pkg.Functions) == 0 {
			if verbose {
				fmt.Printf("📦 Package: %s (empty)\n", pkg.Name)
			}
			continue
		}

		fmt.Printf("📦 Package: %s\n", pkg.Name)
		if verbose {
			fmt.Printf("   Path: %s\n", pkg.Path)
			if pkg.ImportPath != "" {
				fmt.Printf("   Import: %s\n", pkg.ImportPath)
			}
		}

		// Print types
		if len(pkg.Types) > 0 {
			fmt.Println("  Types:")
			for _, typ := range pkg.Types {
				totalTypes++

				// Show based on confidence
				if typ.Convention != nil && typ.Convention.Confidence >= 0.9 {
					// High confidence - show as classification
					fmt.Printf("    ✓ %s (%s)\n", typ.Name, typ.Convention.Name)
					if verbose {
						fmt.Printf("      Confidence: %.0f%% - %s\n",
							typ.Convention.Confidence*100,
							typ.Convention.Reason)
					}

					// Track for summary
					if categoryGroups[typ.Convention.Category] == nil {
						categoryGroups[typ.Convention.Category] = make(map[string]int)
					}
					categoryGroups[typ.Convention.Category][typ.Convention.Name]++

				} else if typ.Convention != nil {
					// Lower confidence - show as observation
					fmt.Printf("    ~ %s (possibly %s)\n", typ.Name, typ.Convention.Name)
					if verbose {
						fmt.Printf("      Confidence: %.0f%% - %s\n",
							typ.Convention.Confidence*100,
							typ.Convention.Reason)
					}
				} else {
					fmt.Printf("    - %s", typ.Name)
					if verbose {
						fmt.Printf(" (%s)", typ.Kind)
					}
					fmt.Println()
				}

				// Verbose: Show fields
				if verbose && len(typ.Fields) > 0 && len(typ.Fields) <= 5 {
					for _, field := range typ.Fields {
						if field.Name != "" {
							fmt.Printf("      • %s: %s\n", field.Name, field.Type)
						}
					}
				} else if verbose && len(typ.Fields) > 5 {
					fmt.Printf("      • %d fields\n", len(typ.Fields))
				}

				// Verbose: Show methods
				if verbose && len(typ.Methods) > 0 {
					fmt.Printf("      • %d methods\n", len(typ.Methods))
				}

				// Verbose: Show type dependencies
				if verbose && len(typ.UsedTypes) > 0 && len(typ.UsedTypes) <= 3 {
					fmt.Printf("      Uses: %v\n", typ.UsedTypes)
				} else if verbose && len(typ.UsedTypes) > 3 {
					fmt.Printf("      Uses: %d types\n", len(typ.UsedTypes))
				}
			}
		}

		// Print functions
		standaloneFuncs := make([]*analyzer.Function, 0)
		for _, fn := range pkg.Functions {
			if fn.Receiver == "" {
				standaloneFuncs = append(standaloneFuncs, fn)
			}
		}

		if len(standaloneFuncs) > 0 {
			fmt.Println("  Functions:")
			for _, fn := range standaloneFuncs {
				totalFunctions++

				if fn.Convention != nil && fn.Convention.Confidence >= 0.9 {
					fmt.Printf("    ✓ %s (%s)\n", fn.Name, fn.Convention.Name)
					if verbose {
						fmt.Printf("      Confidence: %.0f%% - %s\n",
							fn.Convention.Confidence*100,
							fn.Convention.Reason)
					}

					// Track for summary
					if categoryGroups[fn.Convention.Category] == nil {
						categoryGroups[fn.Convention.Category] = make(map[string]int)
					}
					categoryGroups[fn.Convention.Category][fn.Convention.Name]++

				} else if fn.Convention != nil {
					fmt.Printf("    ~ %s (possibly %s)\n", fn.Name, fn.Convention.Name)
					if verbose {
						fmt.Printf("      Confidence: %.0f%% - %s\n",
							fn.Convention.Confidence*100,
							fn.Convention.Reason)
					}
				} else {
					fmt.Printf("    - %s", fn.Name)
					if verbose && fn.Signature != "" {
						fmt.Printf("(%s)", fn.Signature)
					}
					fmt.Println()
				}

				// Verbose: Show what function calls
				if verbose && len(fn.Calls) > 0 && len(fn.Calls) <= 3 {
					fmt.Printf("      Calls: %v\n", fn.Calls)
				} else if verbose && len(fn.Calls) > 3 {
					fmt.Printf("      Calls: %d functions\n", len(fn.Calls))
				}

				// Verbose: Show type usage
				if verbose && len(fn.UsesTypes) > 0 && len(fn.UsesTypes) <= 3 {
					fmt.Printf("      Uses types: %v\n", fn.UsesTypes)
				} else if verbose && len(fn.UsesTypes) > 3 {
					fmt.Printf("      Uses: %d types\n", len(fn.UsesTypes))
				}
			}
		}

		fmt.Println()
	}

	// Summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Summary:")
	fmt.Printf("  📊 %d packages analyzed\n", len(project.Packages))
	fmt.Printf("  📝 %d types found\n", totalTypes)
	fmt.Printf("  🔧 %d functions found\n", totalFunctions)

	if len(categoryGroups) > 0 {
		fmt.Println("\nDetected Conventions:")

		// Sort categories for consistent output
		categories := []string{"handlers", "services", "repositories", "models", "dtos",
			"config", "generators", "parsers", "templates", "validators",
			"middleware", "commands"}

		for _, cat := range categories {
			if patterns, exists := categoryGroups[cat]; exists {
				// Title case the category
				title := strings.Title(cat)
				fmt.Printf("  %s:\n", title)

				// Sort pattern names
				names := make([]string, 0, len(patterns))
				for name := range patterns {
					names = append(names, name)
				}
				sort.Strings(names)

				for _, name := range names {
					count := patterns[name]
					fmt.Printf("    • %s: %d\n", name, count)
				}
			}
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func main() {
	rootCmd.AddCommand(generateCmd())
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
