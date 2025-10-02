package commands

import (
	"github.com/simonhull/firebird-suite/firebird"
	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/spf13/cobra"
)

// RootCmd creates and returns the root command for the Firebird CLI
func RootCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "firebird",
		Short: "Convention-over-configuration web framework for Go",
		Long: `Firebird generates clean, idiomatic Go code from schema definitions.

Built on modern Go practices with excellent DX, Firebird helps you:
• Scaffold projects with sensible defaults
• Generate models, migrations, handlers from schemas
• Build APIs faster without sacrificing control

Learn more: https://github.com/simonhull/firebird-suite`,
		Version: firebird.Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			output.SetVerbose(verbose)
		},
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output for debugging")

	return cmd
}
