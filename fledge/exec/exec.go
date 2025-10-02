package exec

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Executor runs external commands with beautiful UX
type Executor struct {
	stdout io.Writer
	stderr io.Writer
	env    []string
	dir    string

	// For mocking in tests
	commandFunc func(name string, args ...string) *exec.Cmd
}

// Options configures command execution
type Options struct {
	Stdout      io.Writer
	Stderr      io.Writer
	Env         []string      // Additional environment variables
	Dir         string        // Working directory
	Timeout     time.Duration // Command timeout
	ShowCommand bool          // Print command before running
	Spinner     bool          // Show spinner for long-running commands
}

// NewExecutor creates an executor with sensible defaults
func NewExecutor(opts *Options) *Executor {
	if opts == nil {
		opts = &Options{
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
			Spinner: true,
		}
	}

	// Set defaults for nil fields
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	return &Executor{
		stdout:      opts.Stdout,
		stderr:      opts.Stderr,
		env:         opts.Env,
		dir:         opts.Dir,
		commandFunc: exec.Command, // Can be mocked for tests
	}
}

// Run executes a command with beautiful output
func (e *Executor) Run(ctx context.Context, name string, args ...string) error {
	cmd := e.commandFunc(name, args...)

	// Set working directory
	if e.dir != "" {
		cmd.Dir = e.dir
	}

	// Set environment
	if len(e.env) > 0 {
		cmd.Env = append(os.Environ(), e.env...)
	}

	// Connect output streams
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		// Check if command not found
		if isCommandNotFound(err) {
			return enhanceError(err, name)
		}
		return fmt.Errorf("failed to start %s: %w", name, err)
	}

	// Wait for completion
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Try graceful shutdown first
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return fmt.Errorf("%s cancelled: %w", name, ctx.Err())
	case err := <-errCh:
		if err != nil {
			// Check if command not found
			if isCommandNotFound(err) {
				return enhanceError(err, name)
			}
			return fmt.Errorf("%s failed: %w", name, err)
		}
		return nil
	}
}

// RunWithSpinner runs a command with a progress spinner
func (e *Executor) RunWithSpinner(ctx context.Context, message string, name string, args ...string) error {
	// Create pipes to capture output
	stdoutPipe, stdoutWriter := io.Pipe()
	stderrPipe, stderrWriter := io.Pipe()

	// Create a new executor with piped output
	execWithPipes := &Executor{
		stdout:      stdoutWriter,
		stderr:      stderrWriter,
		env:         e.env,
		dir:         e.dir,
		commandFunc: e.commandFunc,
	}

	// Channel to signal command completion
	done := make(chan error, 1)

	// Run command in background
	go func() {
		err := execWithPipes.Run(ctx, name, args...)
		stdoutWriter.Close()
		stderrWriter.Close()
		done <- err
	}()

	// Create spinner model
	m := newSpinnerModel(message)
	p := tea.NewProgram(m, tea.WithOutput(e.stderr))

	// Start the spinner
	go func() {
		if _, err := p.Run(); err != nil {
			// Silently ignore spinner errors
			_ = err
		}
	}()

	// Collect output
	go io.Copy(io.Discard, stdoutPipe)
	go io.Copy(io.Discard, stderrPipe)

	// Wait for command to complete
	err := <-done

	// Update spinner with result
	if err != nil {
		p.Send(spinnerDoneMsg{err: err})
	} else {
		p.Send(spinnerDoneMsg{})
	}

	// Give spinner time to render final state
	time.Sleep(50 * time.Millisecond)
	p.Quit()

	return err
}

// MustRun runs a command and panics on error (for use in main)
func (e *Executor) MustRun(ctx context.Context, name string, args ...string) {
	if err := e.Run(ctx, name, args...); err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

// spinnerModel is the bubbletea model for the spinner
type spinnerModel struct {
	spinner spinner.Model
	message string
	done    bool
	err     error
}

type spinnerDoneMsg struct {
	err error
}

func newSpinnerModel(message string) *spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return &spinnerModel{
		spinner: s,
		message: message,
	}
}

func (m *spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerDoneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		if !m.done {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *spinnerModel) View() string {
	if m.done {
		if m.err != nil {
			return fmt.Sprintf("❌ %s\n", m.message)
		}
		return fmt.Sprintf("✅ %s\n", m.message)
	}
	return fmt.Sprintf("%s %s...", m.spinner.View(), m.message)
}

// isCommandNotFound checks if an error indicates a command was not found
func isCommandNotFound(err error) bool {
	if err == nil {
		return false
	}
	// Check for exec.ErrNotFound
	return err == exec.ErrNotFound ||
		// Some systems return different errors
		contains(err.Error(), "executable file not found") ||
		contains(err.Error(), "command not found")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// enhanceError adds helpful message for missing commands
func enhanceError(err error, cmd string) error {
	return fmt.Errorf("%w\n💡 Command '%s' not found. Please install it and try again", err, cmd)
}

// GenericCommand provides a fluent API for building and executing commands
type GenericCommand struct {
	executor    *Executor
	command     string
	args        []string
	env         []string
	dir         string
	showSpinner bool
	spinnerMsg  string
}

// NewGenericCommand creates a new generic command builder
func NewGenericCommand(executor *Executor, command string) *GenericCommand {
	return &GenericCommand{
		executor:    executor,
		command:     command,
		args:        []string{},
		showSpinner: false,
	}
}

// WithArgs adds arguments to the command
func (g *GenericCommand) WithArgs(args ...string) *GenericCommand {
	g.args = append(g.args, args...)
	return g
}

// WithEnv adds environment variables
func (g *GenericCommand) WithEnv(env ...string) *GenericCommand {
	g.env = append(g.env, env...)
	return g
}

// WithDir sets the working directory
func (g *GenericCommand) WithDir(dir string) *GenericCommand {
	g.dir = dir
	return g
}

// WithSpinner enables spinner with the given message
func (g *GenericCommand) WithSpinner(message string) *GenericCommand {
	g.showSpinner = true
	g.spinnerMsg = message
	return g
}

// Run executes the command
func (g *GenericCommand) Run(ctx context.Context) error {
	// Create a new executor with the command-specific options
	cmdExecutor := &Executor{
		stdout:      g.executor.stdout,
		stderr:      g.executor.stderr,
		env:         append(g.executor.env, g.env...),
		dir:         g.dir,
		commandFunc: g.executor.commandFunc,
	}

	if g.dir == "" {
		cmdExecutor.dir = g.executor.dir
	}

	if g.showSpinner {
		return cmdExecutor.RunWithSpinner(ctx, g.spinnerMsg, g.command, g.args...)
	}
	return cmdExecutor.Run(ctx, g.command, g.args...)
}

// MustRun executes the command and panics on error
func (g *GenericCommand) MustRun(ctx context.Context) {
	if err := g.Run(ctx); err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

// String returns the command string representation for debugging
func (g *GenericCommand) String() string {
	parts := []string{g.command}
	parts = append(parts, g.args...)
	return strings.Join(parts, " ")
}