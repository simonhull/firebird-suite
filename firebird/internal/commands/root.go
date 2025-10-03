package commands

import (
	"os"

	"github.com/simonhull/firebird-suite/firebird"
	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

// HasDatabaseConfigured checks if the current directory is a Firebird project
// with a database configured (driver != "none")
func HasDatabaseConfigured() bool {
	// Check if .firebird.yml exists
	data, err := os.ReadFile("firebird.yml")
	if err != nil {
		// Not in a Firebird project or file doesn't exist
		return false
	}

	// Parse the config
	var config struct {
		Application struct {
			Database struct {
				Driver string `yaml:"driver"`
			} `yaml:"database"`
		} `yaml:"application"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		// Invalid config file
		return false
	}

	// Database is configured if driver exists and is not "none"
	return config.Application.Database.Driver != "" && config.Application.Database.Driver != "none"
}
