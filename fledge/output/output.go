// Package output provides beautiful, styled terminal output for CLI tools.
//
// All tools in the Firebird Suite use this package for consistent, delightful UX.
// Functions use lipgloss for styling but abstract away the details from callers.
package output

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("green")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("red")).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("cyan"))
	stepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	verboseMode bool
)

// SetVerbose enables or disables verbose output for debugging.
// This should be called by the CLI when the --verbose flag is set.
func SetVerbose(v bool) {
	verboseMode = v
}

// Success prints a success message with 🔥 emoji and green color.
// Use this for completed operations.
//
// Example:
//
//	output.Success("Created project: myapp")
func Success(msg string) {
	fmt.Println(successStyle.Render("🔥 " + msg))
}

// Error prints an error message with ❌ emoji and red color.
// Use this for failures that need user attention.
//
// Example:
//
//	output.Error("Failed to create project: permission denied")
func Error(msg string) {
	fmt.Println(errorStyle.Render("❌ " + msg))
}

// Info prints an informational message with ℹ️ emoji and cyan color.
// Use this for status updates or explanations.
//
// Example:
//
//	output.Info("Next steps:")
func Info(msg string) {
	fmt.Println(infoStyle.Render("ℹ️  " + msg))
}

// Step prints an indented step message in gray.
// Use this for actionable next steps or sub-items.
//
// Example:
//
//	output.Step("cd myapp")
//	output.Step("go mod tidy")
func Step(msg string) {
	fmt.Println(stepStyle.Render("   " + msg))
}

// Verbose prints a debug message with 🔍 emoji only if verbose mode is enabled.
// Use this for detailed debugging information.
//
// Example:
//
//	output.Verbose("Loading schema from: internal/schemas/user.firebird.yml")
func Verbose(msg string) {
	if verboseMode {
		fmt.Println(stepStyle.Render("🔍 " + msg))
	}
}
