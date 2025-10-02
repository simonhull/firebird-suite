// Package input provides interactive terminal input utilities.
//
// All tools in the Firebird Suite use this package for consistent
// user interaction when prompts are needed.
package input

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("cyan")).Bold(true)
	hintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// Prompt asks the user for text input with an optional default value.
// If the user presses Enter without typing anything, the default is returned.
//
// Example:
//
//	modulePath := input.Prompt("Module path", "github.com/username/myapp")
//	// Displays: Module path (github.com/username/myapp): _
func Prompt(message, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)

	// Format prompt with default hint
	if defaultValue != "" {
		fmt.Print(promptStyle.Render(message) + " " +
			hintStyle.Render(fmt.Sprintf("(%s)", defaultValue)) + ": ")
	} else {
		fmt.Print(promptStyle.Render(message) + ": ")
	}

	// Read input
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	// Trim whitespace
	input = strings.TrimSpace(input)

	// Return default if empty
	if input == "" {
		return defaultValue
	}

	return input
}

// Confirm asks the user a yes/no question.
// Returns true if the user answers yes (y/Y/yes/YES), false otherwise.
// If defaultYes is true, pressing Enter returns true. Otherwise, returns false.
//
// Example:
//
//	if input.Confirm("Run go mod tidy?", true) {
//	    // User said yes (or pressed Enter with defaultYes=true)
//	}
//	// Displays: Run go mod tidy? [Y/n]: _
func Confirm(message string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)

	// Format prompt with [Y/n] or [y/N] hint
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}

	fmt.Print(promptStyle.Render(message) + " " +
		hintStyle.Render(hint) + ": ")

	// Read input
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	// Trim whitespace and convert to lowercase
	input = strings.TrimSpace(strings.ToLower(input))

	// Empty input returns default
	if input == "" {
		return defaultYes
	}

	// Check for yes answers
	return input == "y" || input == "yes"
}
