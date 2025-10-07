package main

import (
	"os"

	"github.com/simonhull/firebird-suite/firebird/internal/commands"
)

func main() {
	rootCmd := commands.RootCmd()

	// Always available commands
	rootCmd.AddCommand(commands.NewCmd())
	rootCmd.AddCommand(commands.GenerateCmd())
	rootCmd.AddCommand(commands.ModuleCmd())
	rootCmd.AddCommand(commands.RealtimeCmd())

	// Only register database commands if database is configured
	if commands.HasDatabaseConfigured() {
		rootCmd.AddCommand(commands.MigrateCmd())
		rootCmd.AddCommand(commands.DBCmd())
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
