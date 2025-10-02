package exec

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// CommandWrapper is the interface that domain-specific commands must implement
// to be registered in the command registry
type CommandWrapper interface {
	// Name returns the command name for registry lookup
	Name() string
	// Description returns a brief description of what the command does
	Description() string
	// Execute runs the command with the given context and executor
	Execute(ctx context.Context, exec *Executor) error
}

// CommandRegistry manages registered command wrappers
type CommandRegistry struct {
	mu       sync.RWMutex
	commands map[string]CommandWrapper
}

// globalRegistry is the default registry instance
var globalRegistry = &CommandRegistry{
	commands: make(map[string]CommandWrapper),
}

// RegisterCommand registers a command wrapper in the global registry
func RegisterCommand(cmd CommandWrapper) error {
	return globalRegistry.Register(cmd)
}

// GetCommand retrieves a command wrapper from the global registry
func GetCommand(name string) (CommandWrapper, bool) {
	return globalRegistry.Get(name)
}

// ListCommands returns all registered command names from the global registry
func ListCommands() []string {
	return globalRegistry.List()
}

// ClearRegistry clears all registered commands (useful for testing)
func ClearRegistry() {
	globalRegistry.Clear()
}

// Execute runs a registered command from the global registry with the given executor
func Execute(ctx context.Context, name string, exec *Executor) error {
	return globalRegistry.Execute(ctx, name, exec)
}

// NewCommandRegistry creates a new command registry instance
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]CommandWrapper),
	}
}

// Register adds a command wrapper to the registry
func (r *CommandRegistry) Register(cmd CommandWrapper) error {
	if cmd == nil {
		return fmt.Errorf("cannot register nil command")
	}

	name := cmd.Name()
	if name == "" {
		return fmt.Errorf("cannot register command with empty name")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.commands[name]; exists {
		return fmt.Errorf("command '%s' is already registered", name)
	}

	r.commands[name] = cmd
	return nil
}

// Get retrieves a command wrapper by name
func (r *CommandRegistry) Get(name string) (CommandWrapper, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmd, ok := r.commands[name]
	return cmd, ok
}

// List returns all registered command names in sorted order
func (r *CommandRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// ListWithDescriptions returns all registered commands with their descriptions
func (r *CommandRegistry) ListWithDescriptions() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string, len(r.commands))
	for name, cmd := range r.commands {
		result[name] = cmd.Description()
	}
	return result
}

// Clear removes all registered commands
func (r *CommandRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.commands = make(map[string]CommandWrapper)
}

// Size returns the number of registered commands
func (r *CommandRegistry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.commands)
}

// Has checks if a command is registered
func (r *CommandRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.commands[name]
	return exists
}

// Unregister removes a command from the registry
func (r *CommandRegistry) Unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.commands[name]; exists {
		delete(r.commands, name)
		return true
	}
	return false
}

// Execute runs a command by name if it exists
func (r *CommandRegistry) Execute(ctx context.Context, name string, exec *Executor) error {
	cmd, ok := r.Get(name)
	if !ok {
		return fmt.Errorf("command '%s' not found in registry", name)
	}
	return cmd.Execute(ctx, exec)
}