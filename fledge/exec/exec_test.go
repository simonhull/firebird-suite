package exec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCommand returns a command that prints predetermined output
func mockCommand(name string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", name}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// mockCommandWithEnv returns a command with specific environment variables
func mockCommandWithEnv(env []string, name string, args ...string) *exec.Cmd {
	cmd := mockCommand(name, args...)
	cmd.Env = append(cmd.Env, env...)
	return cmd
}

// TestHelperProcess is the mock command executor for generic commands only
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Read command from args
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "no command specified\n")
		os.Exit(1)
	}

	cmd := args[0]

	// Mock generic commands only
	switch cmd {
	case "echo":
		// Simple echo for testing output
		if len(args) > 1 {
			fmt.Println(strings.Join(args[1:], " "))
		}
		os.Exit(0)
	case "sleep":
		// For testing timeouts
		if len(args) > 1 && args[1] == "10" {
			time.Sleep(10 * time.Second)
		}
		os.Exit(0)
	case "error":
		// For testing error handling
		fmt.Fprintf(os.Stderr, "error occurred\n")
		os.Exit(1)
	case "success":
		// For testing successful execution
		fmt.Println("command succeeded")
		os.Exit(0)
	case "notfound":
		// Simulate command not found
		os.Exit(127)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func TestNewExecutor(t *testing.T) {
	// Test with nil options
	executor := NewExecutor(nil)
	assert.NotNil(t, executor)
	assert.Equal(t, os.Stdout, executor.stdout)
	assert.Equal(t, os.Stderr, executor.stderr)
	assert.NotNil(t, executor.commandFunc)

	// Test with custom options
	var stdout, stderr bytes.Buffer
	executor = NewExecutor(&Options{
		Stdout: &stdout,
		Stderr: &stderr,
		Env:    []string{"TEST=1"},
		Dir:    "/tmp",
	})
	assert.Equal(t, &stdout, executor.stdout)
	assert.Equal(t, &stderr, executor.stderr)
	assert.Equal(t, []string{"TEST=1"}, executor.env)
	assert.Equal(t, "/tmp", executor.dir)
}

func TestExecutor_Run(t *testing.T) {
	var stdout bytes.Buffer

	executor := NewExecutor(&Options{
		Stdout: &stdout,
	})
	executor.commandFunc = mockCommand

	err := executor.Run(context.Background(), "echo", "hello", "world")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "hello world")
}

func TestExecutor_RunWithError(t *testing.T) {
	var stderr bytes.Buffer

	executor := NewExecutor(&Options{
		Stderr: &stderr,
	})
	executor.commandFunc = mockCommand

	err := executor.Run(context.Background(), "error")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error failed")
	assert.Contains(t, stderr.String(), "error occurred")
}

func TestExecutor_Timeout(t *testing.T) {
	executor := NewExecutor(nil)
	executor.commandFunc = mockCommand

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := executor.Run(ctx, "sleep", "10")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestExecutor_WithEnvironment(t *testing.T) {
	var stdout bytes.Buffer

	executor := NewExecutor(&Options{
		Stdout: &stdout,
		Env:    []string{"TEST_VAR=test_value"},
	})
	executor.commandFunc = func(name string, args ...string) *exec.Cmd {
		return mockCommandWithEnv([]string{"TEST_VAR=test_value"}, name, args...)
	}

	err := executor.Run(context.Background(), "echo", "test")
	require.NoError(t, err)
}

func TestExecutor_WithWorkingDirectory(t *testing.T) {
	var stdout bytes.Buffer

	executor := NewExecutor(&Options{
		Stdout: &stdout,
		Dir:    "/tmp",
	})
	executor.commandFunc = mockCommand

	err := executor.Run(context.Background(), "echo", "test")
	require.NoError(t, err)
}

func TestExecutor_RunWithSpinner(t *testing.T) {
	// This test is basic because spinner requires a terminal
	// In CI/test environment, it should gracefully handle non-terminal
	executor := NewExecutor(nil)
	executor.commandFunc = mockCommand

	err := executor.RunWithSpinner(context.Background(), "Testing", "echo", "test")
	assert.NoError(t, err)
}

func TestGenericCommand(t *testing.T) {
	var stdout bytes.Buffer

	executor := NewExecutor(&Options{
		Stdout: &stdout,
	})
	executor.commandFunc = mockCommand

	t.Run("basic command", func(t *testing.T) {
		cmd := NewGenericCommand(executor, "echo").
			WithArgs("hello", "world")

		err := cmd.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "hello world")
	})

	t.Run("with environment", func(t *testing.T) {
		stdout.Reset()
		cmd := NewGenericCommand(executor, "echo").
			WithArgs("test").
			WithEnv("TEST=1", "FOO=bar")

		err := cmd.Run(context.Background())
		require.NoError(t, err)
	})

	t.Run("with directory", func(t *testing.T) {
		stdout.Reset()
		cmd := NewGenericCommand(executor, "echo").
			WithArgs("test").
			WithDir("/tmp")

		err := cmd.Run(context.Background())
		require.NoError(t, err)
	})

	t.Run("with spinner", func(t *testing.T) {
		stdout.Reset()
		cmd := NewGenericCommand(executor, "success").
			WithSpinner("Processing")

		err := cmd.Run(context.Background())
		require.NoError(t, err)
	})

	t.Run("string representation", func(t *testing.T) {
		cmd := NewGenericCommand(executor, "git").
			WithArgs("commit", "-m", "test message")

		assert.Equal(t, "git commit -m test message", cmd.String())
	})

	t.Run("fluent chaining", func(t *testing.T) {
		stdout.Reset()
		err := NewGenericCommand(executor, "echo").
			WithArgs("hello").
			WithEnv("TEST=1").
			WithDir("/tmp").
			WithSpinner("Running").
			Run(context.Background())

		require.NoError(t, err)
	})
}

// testCommandWrapper is a test implementation of CommandWrapper
type testCommandWrapper struct {
	name        string
	description string
	executeFunc func(context.Context, *Executor) error
}

func (t *testCommandWrapper) Name() string        { return t.name }
func (t *testCommandWrapper) Description() string { return t.description }
func (t *testCommandWrapper) Execute(ctx context.Context, exec *Executor) error {
	if t.executeFunc != nil {
		return t.executeFunc(ctx, exec)
	}
	return nil
}

func TestCommandRegistry(t *testing.T) {
	// Verify interface implementation
	var _ CommandWrapper = (*testCommandWrapper)(nil)

	t.Run("register and get command", func(t *testing.T) {
		registry := NewCommandRegistry()

		cmd := &testCommandWrapper{
			name:        "test-cmd",
			description: "A test command",
		}

		err := registry.Register(cmd)
		require.NoError(t, err)

		retrieved, ok := registry.Get("test-cmd")
		assert.True(t, ok)
		assert.Equal(t, "test-cmd", retrieved.Name())
		assert.Equal(t, "A test command", retrieved.Description())
	})

	t.Run("register duplicate command", func(t *testing.T) {
		registry := NewCommandRegistry()

		cmd1 := &testCommandWrapper{name: "duplicate", description: "First"}
		cmd2 := &testCommandWrapper{name: "duplicate", description: "Second"}

		err := registry.Register(cmd1)
		require.NoError(t, err)

		err = registry.Register(cmd2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("register nil command", func(t *testing.T) {
		registry := NewCommandRegistry()
		err := registry.Register(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot register nil")
	})

	t.Run("register command with empty name", func(t *testing.T) {
		registry := NewCommandRegistry()
		cmd := &testCommandWrapper{name: "", description: "No name"}
		err := registry.Register(cmd)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty name")
	})

	t.Run("list commands", func(t *testing.T) {
		registry := NewCommandRegistry()

		commands := []*testCommandWrapper{
			{name: "cmd-c", description: "Command C"},
			{name: "cmd-a", description: "Command A"},
			{name: "cmd-b", description: "Command B"},
		}

		for _, cmd := range commands {
			err := registry.Register(cmd)
			require.NoError(t, err)
		}

		list := registry.List()
		assert.Equal(t, []string{"cmd-a", "cmd-b", "cmd-c"}, list) // Should be sorted
	})

	t.Run("list with descriptions", func(t *testing.T) {
		registry := NewCommandRegistry()

		cmd1 := &testCommandWrapper{name: "cmd1", description: "First command"}
		cmd2 := &testCommandWrapper{name: "cmd2", description: "Second command"}

		registry.Register(cmd1)
		registry.Register(cmd2)

		descriptions := registry.ListWithDescriptions()
		assert.Equal(t, "First command", descriptions["cmd1"])
		assert.Equal(t, "Second command", descriptions["cmd2"])
	})

	t.Run("clear registry", func(t *testing.T) {
		registry := NewCommandRegistry()

		cmd := &testCommandWrapper{name: "test", description: "Test"}
		registry.Register(cmd)

		assert.Equal(t, 1, registry.Size())

		registry.Clear()
		assert.Equal(t, 0, registry.Size())

		_, ok := registry.Get("test")
		assert.False(t, ok)
	})

	t.Run("has command", func(t *testing.T) {
		registry := NewCommandRegistry()
		cmd := &testCommandWrapper{name: "exists", description: "Test"}

		assert.False(t, registry.Has("exists"))

		registry.Register(cmd)
		assert.True(t, registry.Has("exists"))
	})

	t.Run("unregister command", func(t *testing.T) {
		registry := NewCommandRegistry()
		cmd := &testCommandWrapper{name: "temp", description: "Temporary"}

		registry.Register(cmd)
		assert.True(t, registry.Has("temp"))

		removed := registry.Unregister("temp")
		assert.True(t, removed)
		assert.False(t, registry.Has("temp"))

		removed = registry.Unregister("temp") // Try again
		assert.False(t, removed)
	})

	t.Run("execute command", func(t *testing.T) {
		registry := NewCommandRegistry()
		executed := false

		cmd := &testCommandWrapper{
			name:        "exec-test",
			description: "Execute test",
			executeFunc: func(ctx context.Context, exec *Executor) error {
				executed = true
				return nil
			},
		}

		registry.Register(cmd)

		executor := NewExecutor(nil)
		err := registry.Execute(context.Background(), "exec-test", executor)
		require.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("execute non-existent command", func(t *testing.T) {
		registry := NewCommandRegistry()

		executor := NewExecutor(nil)
		err := registry.Execute(context.Background(), "non-existent", executor)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("global registry functions", func(t *testing.T) {
		// Clear global registry first
		ClearRegistry()

		cmd := &testCommandWrapper{
			name:        "global-test",
			description: "Global test command",
		}

		err := RegisterCommand(cmd)
		require.NoError(t, err)

		retrieved, ok := GetCommand("global-test")
		assert.True(t, ok)
		assert.Equal(t, "global-test", retrieved.Name())

		list := ListCommands()
		assert.Contains(t, list, "global-test")

		ClearRegistry()
		list = ListCommands()
		assert.Empty(t, list)
	})

	t.Run("global execute function", func(t *testing.T) {
		ClearRegistry()

		executed := false
		cmd := &testCommandWrapper{
			name:        "global-exec",
			description: "Global execution test",
			executeFunc: func(ctx context.Context, exec *Executor) error {
				executed = true
				return nil
			},
		}

		RegisterCommand(cmd)

		executor := NewExecutor(nil)
		err := Execute(context.Background(), "global-exec", executor)
		require.NoError(t, err)
		assert.True(t, executed)

		ClearRegistry()
	})
}

func TestEnhanceError(t *testing.T) {
	err := fmt.Errorf("command not found")

	enhanced := enhanceError(err, "some-command")
	assert.Contains(t, enhanced.Error(), "Command 'some-command' not found")
	assert.Contains(t, enhanced.Error(), "Please install it")
}

func TestContains(t *testing.T) {
	assert.True(t, contains("hello world", "world"))
	assert.True(t, contains("hello world", "hello"))
	assert.False(t, contains("hello world", "foo"))
	assert.True(t, contains("", ""))
	assert.False(t, contains("hello", "hello world"))
}

func TestStreamingWriter(t *testing.T) {
	var output bytes.Buffer
	writer := NewStreamingWriter(&output, "[prefix] ", "205")

	// Write partial line
	n, err := writer.Write([]byte("Hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Empty(t, output.String()) // Not written yet (no newline)

	// Complete the line
	n, err = writer.Write([]byte(" World\n"))
	assert.NoError(t, err)
	assert.Equal(t, 7, n)
	assert.Contains(t, output.String(), "[prefix] Hello World")

	// Write multiple lines at once
	output.Reset()
	n, err = writer.Write([]byte("Line1\nLine2\nPartial"))
	assert.NoError(t, err)
	assert.Equal(t, 19, n)
	assert.Contains(t, output.String(), "[prefix] Line1")
	assert.Contains(t, output.String(), "[prefix] Line2")
	assert.NotContains(t, output.String(), "Partial") // Not written yet

	// Flush remaining
	err = writer.Flush()
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "[prefix] Partial")
}

func TestPrefixWriter(t *testing.T) {
	var output bytes.Buffer
	writer := NewPrefixWriter(&output, ">>> ")

	// Write single line
	n, err := writer.Write([]byte("Hello World\n"))
	assert.NoError(t, err)
	assert.Equal(t, 12, n)
	assert.Equal(t, ">>> Hello World\n", output.String())

	// Write partial line
	output.Reset()
	n, err = writer.Write([]byte("Partial"))
	assert.NoError(t, err)
	assert.Equal(t, 7, n)
	assert.Empty(t, output.String()) // Buffered

	// Complete the line
	n, err = writer.Write([]byte(" Line\n"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, ">>> Partial Line\n", output.String())
}

func TestTeeWriter(t *testing.T) {
	var output1, output2 bytes.Buffer
	writer := NewTeeWriter(&output1, &output2)

	n, err := writer.Write([]byte("Hello World"))
	assert.NoError(t, err)
	assert.Equal(t, 11, n)
	assert.Equal(t, "Hello World", output1.String())
	assert.Equal(t, "Hello World", output2.String())
}

// Example usage for documentation
func ExampleNewExecutor() {
	executor := NewExecutor(&Options{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	ctx := context.Background()
	if err := executor.Run(ctx, "echo", "Hello, World!"); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func ExampleGenericCommand() {
	executor := NewExecutor(nil)

	// Build and run a command with fluent API
	err := NewGenericCommand(executor, "echo").
		WithArgs("Hello", "World").
		WithEnv("USER=test").
		WithDir("/tmp").
		WithSpinner("Processing").
		Run(context.Background())

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

// ExampleMyCommand shows how domain packages would implement CommandWrapper
type ExampleMyCommand struct{
	// Domain packages might store configuration here
	args []string
}

func (m ExampleMyCommand) Name() string        { return "my-command" }
func (m ExampleMyCommand) Description() string { return "Does something useful" }
func (m ExampleMyCommand) Execute(ctx context.Context, exec *Executor) error {
	// Now the command can actually execute real commands using the provided executor
	return exec.Run(ctx, "echo", "Executing my command")
}

func ExampleCommandRegistry() {
	// This example shows how domain packages would register their commands
	// The exec package itself doesn't know about these specific commands

	// Register the command
	if err := RegisterCommand(ExampleMyCommand{args: []string{"hello", "world"}}); err != nil {
		fmt.Printf("Registration failed: %v\n", err)
	}

	// Later, retrieve and execute with an executor
	if cmd, ok := GetCommand("my-command"); ok {
		executor := NewExecutor(nil)
		if err := cmd.Execute(context.Background(), executor); err != nil {
			fmt.Printf("Execution failed: %v\n", err)
		}
	}
}

// ExampleMigrateCommand shows how a real domain command would be implemented
type ExampleMigrateCommand struct {
	migrationsDir string
	databaseURL   string
}

func (m ExampleMigrateCommand) Name() string {
	return "migrate"
}

func (m ExampleMigrateCommand) Description() string {
	return "Run database migrations"
}

func (m ExampleMigrateCommand) Execute(ctx context.Context, exec *Executor) error {
	// Use the provided executor to run the actual migrate command
	return NewGenericCommand(exec, "migrate").
		WithArgs("-path", m.migrationsDir, "-database", m.databaseURL, "up").
		WithSpinner("Running migrations").
		Run(ctx)
}