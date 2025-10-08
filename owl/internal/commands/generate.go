package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/simonhull/firebird-suite/owl/pkg/analyzer"
	"github.com/simonhull/firebird-suite/owl/pkg/conventions"
	"github.com/simonhull/firebird-suite/owl/pkg/generator"
	"github.com/spf13/cobra"
)

var (
	outputPath string
	configPath string
)

var generateCmd = &cobra.Command{
	Use:   "generate [path]",
	Short: "Generate documentation for a Go project",
	Long: `Analyzes a Go project and generates static HTML documentation.

Example:
  owl generate ./internal/handlers
  owl generate ../myproject
  owl generate ../firebird --verbose`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().StringVarP(&outputPath, "out", "o", "./docs", "Output directory for generated documentation")
	generateCmd.Flags().StringVarP(&configPath, "config", "c", "owl.yaml", "Path to configuration file")

	RootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	fmt.Printf("🦉 Analyzing project: %s\n", projectPath)
	if verbose {
		output.Verbose("📊 Verbose mode enabled - detailed analysis output")
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
	printAnalysisResults(project)

	// Generate documentation
	gen := generator.New(outputPath, "default")
	if err := gen.Generate(project); err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	fmt.Println()
	output.Success("✅ Documentation generated successfully!")
	fmt.Printf("📁 Output: %s\n", outputPath)

	return nil
}

// printAnalysisResults displays analysis results in terminal
func printAnalysisResults(project *analyzer.Project) {
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
			output.Verbose(fmt.Sprintf("   Path: %s", pkg.Path))
			if pkg.ImportPath != "" {
				output.Verbose(fmt.Sprintf("   Import: %s", pkg.ImportPath))
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
						output.Verbose(fmt.Sprintf("      Confidence: %.0f%% - %s",
							typ.Convention.Confidence*100,
							typ.Convention.Reason))
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
						output.Verbose(fmt.Sprintf("      Confidence: %.0f%% - %s",
							typ.Convention.Confidence*100,
							typ.Convention.Reason))
					}
				} else {
					if verbose {
						fmt.Printf("    - %s (%s)\n", typ.Name, typ.Kind)
					} else {
						fmt.Printf("    - %s\n", typ.Name)
					}
				}

				// Verbose: Show fields
				if verbose && len(typ.Fields) > 0 && len(typ.Fields) <= 5 {
					for _, field := range typ.Fields {
						if field.Name != "" {
							output.Verbose(fmt.Sprintf("      • %s: %s", field.Name, field.Type))
						}
					}
				} else if verbose && len(typ.Fields) > 5 {
					output.Verbose(fmt.Sprintf("      • %d fields", len(typ.Fields)))
				}

				// Verbose: Show methods
				if verbose && len(typ.Methods) > 0 {
					output.Verbose(fmt.Sprintf("      • %d methods", len(typ.Methods)))
				}

				// Verbose: Show type dependencies
				if verbose && len(typ.UsedTypes) > 0 && len(typ.UsedTypes) <= 3 {
					output.Verbose(fmt.Sprintf("      Uses: %v", typ.UsedTypes))
				} else if verbose && len(typ.UsedTypes) > 3 {
					output.Verbose(fmt.Sprintf("      Uses: %d types", len(typ.UsedTypes)))
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
						output.Verbose(fmt.Sprintf("      Confidence: %.0f%% - %s",
							fn.Convention.Confidence*100,
							fn.Convention.Reason))
					}

					// Track for summary
					if categoryGroups[fn.Convention.Category] == nil {
						categoryGroups[fn.Convention.Category] = make(map[string]int)
					}
					categoryGroups[fn.Convention.Category][fn.Convention.Name]++

				} else if fn.Convention != nil {
					fmt.Printf("    ~ %s (possibly %s)\n", fn.Name, fn.Convention.Name)
					if verbose {
						output.Verbose(fmt.Sprintf("      Confidence: %.0f%% - %s",
							fn.Convention.Confidence*100,
							fn.Convention.Reason))
					}
				} else {
					if verbose && fn.Signature != "" {
						fmt.Printf("    - %s(%s)\n", fn.Name, fn.Signature)
					} else {
						fmt.Printf("    - %s\n", fn.Name)
					}
				}

				// Verbose: Show what function calls
				if verbose && len(fn.Calls) > 0 && len(fn.Calls) <= 3 {
					output.Verbose(fmt.Sprintf("      Calls: %v", fn.Calls))
				} else if verbose && len(fn.Calls) > 3 {
					output.Verbose(fmt.Sprintf("      Calls: %d functions", len(fn.Calls)))
				}

				// Verbose: Show type usage
				if verbose && len(fn.UsesTypes) > 0 && len(fn.UsesTypes) <= 3 {
					output.Verbose(fmt.Sprintf("      Uses types: %v", fn.UsesTypes))
				} else if verbose && len(fn.UsesTypes) > 3 {
					output.Verbose(fmt.Sprintf("      Uses: %d types", len(fn.UsesTypes)))
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
		fmt.Println()
		fmt.Println("Detected Conventions:")

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
