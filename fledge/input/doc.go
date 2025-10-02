// Package input provides interactive terminal input utilities.
//
// # Overview
//
// All tools in the Firebird Suite (Firebird, Talon, Hornbill, Plume, etc.)
// use this package for consistent user interaction when prompts are needed.
//
// # Usage
//
// Import the package and call the input functions:
//
//	import "github.com/simonhull/firebird-suite/fledge/input"
//
//	// Ask for text input with a default
//	modulePath := input.Prompt("Module path", "github.com/username/myapp")
//
//	// Ask yes/no question
//	if input.Confirm("Continue?", true) {
//	    // User said yes
//	}
//
// # Styling
//
// The package uses lipgloss for consistent terminal styling:
//   - Prompts are displayed in cyan and bold
//   - Hints (defaults, [Y/n]) are displayed in gray
//
// # Non-Interactive Mode
//
// In CI/CD or automated environments, you may want to skip prompts.
// Use flags in your CLI to bypass interactive prompts:
//
//	if moduleFlag != "" {
//	    modulePath = moduleFlag  // Use flag value
//	} else {
//	    modulePath = input.Prompt("Module path", "default")  // Prompt user
//	}
package input
