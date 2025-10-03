package main

import (
	"os"

	"github.com/simonhull/firebird-suite/firebird/internal/commands"
)

func main() {
	rootCmd := commands.RootCmd()
	rootCmd.AddCommand(commands.NewCmd())
	rootCmd.AddCommand(commands.GenerateCmd())
	rootCmd.AddCommand(commands.MigrateCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
