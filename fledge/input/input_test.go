package input

import (
	"testing"
)

// Note: These tests are for documentation purposes.
// Interactive input functions require manual testing in a real terminal.

func TestPrompt_Documentation(t *testing.T) {
	t.Skip("Manual testing required - run: go run examples/prompt_example.go")

	// Example usage for documentation:
	// result := Prompt("Enter your name", "John Doe")
	// fmt.Printf("You entered: %s\n", result)
}

func TestConfirm_Documentation(t *testing.T) {
	t.Skip("Manual testing required - run: go run examples/confirm_example.go")

	// Example usage for documentation:
	// if Confirm("Continue?", true) {
	//     fmt.Println("User confirmed")
	// } else {
	//     fmt.Println("User declined")
	// }
}

// TODO: Add integration tests with mocked stdin for automated testing
// For now, manual testing in a real terminal is required
