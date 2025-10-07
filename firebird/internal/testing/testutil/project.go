package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestProject represents a temporary Firebird project for testing
type TestProject struct {
	Root string
	Name string
	t    *testing.T
}

// NewTestProject creates a temporary project directory
func NewTestProject(t *testing.T, name string) *TestProject {
	t.Helper()

	tmpDir := t.TempDir()

	return &TestProject{
		Root: tmpDir,
		Name: name,
		t:    t,
	}
}

// RunFirebird executes a firebird command in the project
func (p *TestProject) RunFirebird(args ...string) error {
	p.t.Helper()

	// Look for firebird binary in current working directory (built by test script)
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Navigate up to find firebird binary (tests run from test/integration)
	firebirdPath := filepath.Join(cwd, "..", "..", "firebird")
	cmd := exec.Command(firebirdPath, args...)

	// For "new" command, run from temp directory
	if len(args) > 0 && args[0] == "new" {
		cmd.Dir = p.Root
	} else {
		// For other commands, run from project directory (Root/Name)
		cmd.Dir = filepath.Join(p.Root, p.Name)
	}

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		p.t.Logf("Firebird command failed: %s\nOutput: %s", err, string(output))
		return err
	}

	p.t.Logf("Firebird output: %s", string(output))
	return nil
}

// Build runs go mod tidy and go build
func (p *TestProject) Build() error {
	p.t.Helper()

	projectDir := filepath.Join(p.Root, p.Name)

	// go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectDir
	if output, err := cmd.CombinedOutput(); err != nil {
		p.t.Logf("go mod tidy failed: %s\nOutput: %s", err, string(output))
		return err
	}

	// go build
	cmd = exec.Command("go", "build", "-o", "server", "./cmd/server")
	cmd.Dir = projectDir
	if output, err := cmd.CombinedOutput(); err != nil {
		p.t.Logf("go build failed: %s\nOutput: %s", err, string(output))
		return err
	}

	return nil
}

// FileExists checks if a file exists in the project
func (p *TestProject) FileExists(path string) bool {
	p.t.Helper()

	fullPath := filepath.Join(p.Root, p.Name, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// WriteSchema writes a schema file to internal/schemas/ directory
func (p *TestProject) WriteSchema(name, content string) error {
	p.t.Helper()

	schemaDir := filepath.Join(p.Root, p.Name, "internal", "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(schemaDir, name+".firebird.yml")
	return os.WriteFile(path, []byte(content), 0644)
}

// ReadFile reads a file from the project
func (p *TestProject) ReadFile(path string) (string, error) {
	p.t.Helper()

	fullPath := filepath.Join(p.Root, p.Name, path)
	content, err := os.ReadFile(fullPath)
	return string(content), err
}
