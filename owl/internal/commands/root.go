package commands

import (
	"fmt"

	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/simonhull/firebird-suite/owl"
	"github.com/spf13/cobra"
)

var (
	verbose bool
)

// RootCmd is the root command for Owl
var RootCmd = &cobra.Command{
	Use:   "owl",
	Short: "Owl - Convention-Aware Go Code Analyzer",
	Long: `Owl analyzes Go projects and reports observable patterns and facts.
It focuses on conclusive observations rather than assumptions about project type.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.SetVerbose(verbose)
	},
}

// Execute runs the root command
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed analysis information")

	// Add version command
	RootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Owl v%s\n", owl.Version)
		},
	})
}
