package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureOutput captures stdout during test execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestSuccess(t *testing.T) {
	output := captureOutput(func() {
		Success("Test message")
	})

	if !strings.Contains(output, "🔥") {
		t.Error("Success output should contain fire emoji")
	}
	if !strings.Contains(output, "Test message") {
		t.Error("Success output should contain the message")
	}
}

func TestError(t *testing.T) {
	output := captureOutput(func() {
		Error("Error message")
	})

	if !strings.Contains(output, "❌") {
		t.Error("Error output should contain X emoji")
	}
	if !strings.Contains(output, "Error message") {
		t.Error("Error output should contain the message")
	}
}

func TestInfo(t *testing.T) {
	output := captureOutput(func() {
		Info("Info message")
	})

	if !strings.Contains(output, "ℹ️") {
		t.Error("Info output should contain info emoji")
	}
	if !strings.Contains(output, "Info message") {
		t.Error("Info output should contain the message")
	}
}

func TestStep(t *testing.T) {
	output := captureOutput(func() {
		Step("Step message")
	})

	if !strings.Contains(output, "   ") {
		t.Error("Step output should contain indentation")
	}
	if !strings.Contains(output, "Step message") {
		t.Error("Step output should contain the message")
	}
}

func TestVerbose(t *testing.T) {
	// Test with verbose mode off (default)
	output := captureOutput(func() {
		Verbose("Debug message")
	})

	if output != "" {
		t.Error("Verbose output should be empty when verbose mode is off")
	}

	// Test with verbose mode on
	SetVerbose(true)
	output = captureOutput(func() {
		Verbose("Debug message")
	})

	if !strings.Contains(output, "🔍") {
		t.Error("Verbose output should contain magnifying glass emoji when enabled")
	}
	if !strings.Contains(output, "Debug message") {
		t.Error("Verbose output should contain the message when enabled")
	}

	// Clean up
	SetVerbose(false)
}

func TestSetVerbose(t *testing.T) {
	// Test enabling verbose mode
	SetVerbose(true)
	if !verboseMode {
		t.Error("SetVerbose(true) should enable verbose mode")
	}

	// Test disabling verbose mode
	SetVerbose(false)
	if verboseMode {
		t.Error("SetVerbose(false) should disable verbose mode")
	}
}
