// Package output provides beautiful, styled terminal output for CLI tools.
//
// # Overview
//
// All tools in the Firebird Suite (Firebird, Talon, Hornbill, Plume, etc.)
// use this package for consistent, delightful terminal output.
//
// # Usage
//
// Import the package and call the output functions:
//
//	import "github.com/simonhull/firebird-suite/fledge/output"
//
//	output.Success("Operation completed!")
//	output.Info("Next steps:")
//	output.Step("cd myproject")
//	output.Error("Something went wrong")
//
// # Verbose Mode
//
// Enable verbose output for debugging:
//
//	output.SetVerbose(true)
//	output.Verbose("This only prints in verbose mode")
//
// # Styling
//
// The package uses lipgloss for terminal styling, but abstracts
// these details away from callers. All styling is consistent across
// the Firebird Suite:
//
//   - Success: 🔥 green bold
//   - Error: ❌ red bold
//   - Info: ℹ️ cyan
//   - Step: indented gray
//   - Verbose: 🔍 gray (when enabled)
package output
