// Package exec provides utilities for executing external commands with beautiful UX.
//
// The exec package is completely domain-agnostic and provides three main components:
//
// 1. Executor - Runs system commands with context support, streaming output, and spinners
// 2. CommandRegistry - Plugin system for domain packages to register custom command wrappers
// 3. GenericCommand - Fluent API for building and executing commands
//
// # Basic Usage
//
// Create an executor and run commands:
//
//	executor := exec.NewExecutor(nil)
//	err := executor.Run(ctx, "echo", "Hello, World!")
//
// # Command Registry Pattern
//
// Domain packages (like Firebird) implement CommandWrapper to register commands:
//
//	type MigrateCommand struct {
//	    migrationsDir string
//	    databaseURL   string
//	}
//
//	func (m MigrateCommand) Name() string { return "migrate" }
//	func (m MigrateCommand) Description() string { return "Run database migrations" }
//	func (m MigrateCommand) Execute(ctx context.Context, exec *exec.Executor) error {
//	    return exec.RunWithSpinner(ctx, "Running migrations", "migrate", "up")
//	}
//
// Commands receive an Executor at execution time (not construction), enabling:
// - Clean dependency injection
// - Easy testing with mocked executors
// - No tight coupling between packages
//
// Register commands globally:
//
//	exec.RegisterCommand(MigrateCommand{migrationsDir: "./migrations", databaseURL: "..."})
//
// Execute registered commands:
//
//	exec.Execute(ctx, "migrate", executor)
//
// # Design Principles
//
// - Domain Agnostic: This package knows nothing about specific tools (migrate, sqlc, etc.)
// - Dependency Injection: Commands receive executors, they don't store them
// - Beautiful UX: Spinners, colored output, and helpful error messages built-in
// - Testable: Full mocking support for testing command execution
package exec