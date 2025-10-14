package main

import (
	"os"

	"github.com/simonhull/firebird-suite/owl/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
