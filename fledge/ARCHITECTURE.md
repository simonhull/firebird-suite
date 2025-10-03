# Fledge Architecture

## Design Principles

### 1. Domain Agnostic

Fledge has **zero knowledge** of:
- CLI frameworks (cobra, urfave/cli, etc.)
- Web frameworks (gin, echo, fiber, etc.)
- Specific use cases (models, migrations, etc.)

**Why:** Maximum reusability. Any tool can use Fledge without coupling.

**Example:**
```go
// L BAD - Fledge knows about CLI flags
func Execute(dryRun bool, force bool) error

//  GOOD - Fledge accepts options, CLI maps flags
func Execute(ctx context.Context, ops []Operation, opts ExecuteOptions) error
```

### 2. Two-Phase Execution

**Problem:** Partial file generation on errors creates inconsistent state.

**Solution:** Validate ALL operations before executing ANY.

```go
// Phase 1: Validate everything
for _, op := range ops {
    if err := op.Validate(ctx, opts.Force); err != nil {
        return err  // Stop before any writes
    }
}

// Phase 2: All valid, safe to execute
for _, op := range ops {
    if err := op.Execute(ctx); err != nil {
        return err
    }
}
```

**Benefits:**
- Atomic-ish behavior (all or nothing)
- Fast failure (catch errors early)
- Safe dry-run (validation without execution)

### 3. Operation Pattern

**Problem:** How to represent file operations that can be validated AND executed?

**Solution:** Operation interface with three methods.

```go
type Operation interface {
    Validate(ctx context.Context, force bool) error
    Execute(ctx context.Context) error
    Description() string
}
```

**Why this works:**
- `Validate()` checks preconditions without side effects (mostly)
- `Execute()` performs the actual work
- `Description()` provides human-readable output
- Interface allows custom operations

### 4. Validation Side Effects

**Design Decision:** `Validate()` may create directories via `os.MkdirAll`.

**Rationale:**
- `MkdirAll` is idempotent (safe to call multiple times)
- Directory existence is required to validate file writes
- Creating empty directories is harmless
- Alternative (permission checks only) is complex and OS-dependent

**Trade-off:** Dry-run may create directories, but this is acceptable.

### 5. Force Mode

**Problem:** How to intentionally overwrite existing files?

**Solution:** `force bool` parameter in `Validate()`.

```go
if !force {
    if _, err := os.Stat(op.Path); err == nil {
        return fmt.Errorf("file already exists: %s", op.Path)
    }
}
```

**Why not global flag?**
- Per-operation control (future: selective force)
- Testable without global state
- Explicit parameter vs hidden config

### 6. Injectable Writer

**Problem:** How to test output and support custom destinations?

**Solution:** `io.Writer` in `ExecuteOptions`.

```go
type ExecuteOptions struct {
    Writer io.Writer  // Defaults to os.Stdout
}
```

**Benefits:**
- Tests capture output in `bytes.Buffer`
- CLI can route to custom writers
- Silent mode possible (`io.Discard`)
- No global print functions

### 7. Error Handling

**Pattern:** Wrap errors with context, never lose information.

```go
//  GOOD - preserves underlying error
return fmt.Errorf("validation failed: %w", err)

// L BAD - loses error chain
return fmt.Errorf("validation failed: %v", err)
```

**Why `%w`:**
- Enables `errors.Is()` and `errors.As()`
- Preserves stack traces (with tools)
- Allows selective error handling

### 8. Context Propagation

**Pattern:** Accept `context.Context` in all I/O operations.

**Why:**
- Cancellation support (ctrl-c, timeouts)
- Request-scoped values (future: logging, tracing)
- Idiomatic Go for long-running operations

**Usage:**
```go
func (op *WriteFileOp) Execute(ctx context.Context) error {
    // Check cancellation before expensive work
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    return os.WriteFile(op.Path, op.Content, op.Mode)
}
```

## Package Organization

### generator/
Core operation system. Zero dependencies on other Fledge packages.

**Exports:**
- `Operation` interface
- `WriteFileOp` struct
- `Execute()` function
- `ExecuteOptions` struct

### schema/
Schema parsing and validation. Depends on internal YAML parsing.

### exec/
Command execution with solid UX. Zero dependencies on generator.

### output/
Styled terminal output. Used by other packages for consistent formatting.

### input/
Interactive prompts. Used for conflict resolution and user input.

## Extension Points

### Custom Operations

Implement the `Operation` interface:

```go
type CustomOp struct {
    // Your fields
}

func (op *CustomOp) Validate(ctx context.Context, force bool) error {
    // Your validation logic
    return nil
}

func (op *CustomOp) Execute(ctx context.Context) error {
    // Your execution logic
    return nil
}

func (op *CustomOp) Description() string {
    return "Custom operation description"
}
```

**Use cases:**
- API calls (create GitHub repo)
- Database operations (run migration)
- External tool execution (run `go fmt`)

### Custom Validators

Add validation logic in your generator:

```go
func Generate(schema *Schema) ([]generator.Operation, error) {
    // Custom validation
    if err := validateSchema(schema); err != nil {
        return nil, err
    }

    // Build operations
    ops := buildOperations(schema)
    return ops, nil
}
```

## Performance Considerations

### Validation Cost

Validation creates directories and stats files. For large operation sets:
- **Measured:** ~0.1ms per operation (SSD)
- **Acceptable:** <100ms for typical use (100-1000 operations)

### Memory Usage

Operations hold file content in memory. For large files:
- **Pattern:** Stream content or use temp files for >10MB files

### Parallelization

Currently sequential. Future: parallel validation + execution.

## Testing Strategy

### Unit Tests
- Test each `Operation` implementation
- Mock context for cancellation
- Use `t.TempDir()` for isolation

### Integration Tests
- Test complete execution flow
- Verify dry-run behavior
- Test force mode

### Property-Based Tests (Future)
- Generate random operation sets
- Verify validation catches all errors
- Verify execution succeeds after validation

## Future Enhancements

### Potential Features
- **Parallel Execution** - Validate/execute operations concurrently
- **Rollback Support** - Undo operations on failure
- **Progress Reporting** - Callbacks for long-running operations
- **Operation Batching** - Group related operations
- **Selective Force** - Per-file force decisions

### Non-Goals
- **CLI Integration** - That's the caller's job
- **Framework-Specific Logic** - Fledge stays generic
- **Complex Templating** - Use Go templates in generators
- **Database Operations** - Focus on file generation

## Comparison to Other Tools

### vs `go generate`
- **Fledge:** Validated, atomic, testable
- **go generate:** Simple, tool-agnostic, no validation

### vs Wire/Ent
- **Fledge:** General-purpose, bring-your-own-schema
- **Wire/Ent:** Domain-specific (DI, ORM)

### vs Custom Scripts
- **Fledge:** Structured, testable, reusable
- **Scripts:** Quick, flexible, error-prone

## Questions & Answers

**Q: Why not use `go:embed` for templates?**
A: Fledge doesn't assume templating. Generators can use `go:embed`, `text/template`, or string builders.

**Q: Can I use Fledge without schemas?**
A: Yes! The generator package has zero dependencies on schemas.

**Q: Is Fledge thread-safe?**
A: Operations are stateless. `Execute()` can be called concurrently with different operation sets.

**Q: Does Fledge work on Windows?**
A: Yes. Uses `filepath.Join` and `os.PathSeparator` for cross-platform paths.

**Q: Why not a plugin system?**
A: Fledge is a library, not a framework. Generators are just Go functions that return `[]Operation`.

**Q: How do I handle very large files?**
A: For files >10MB, consider streaming or creating custom operations that don't hold content in memory.
