# Fledge

A domain-agnostic code generation foundation library for Go.

## What is Fledge?

Fledge provides the building blocks for creating code generators with built-in validation, dry-run support, and atomic execution. It's designed to be used by higher-level tools but remains completely independent of any specific framework or CLI.

**Key Features:**
- Two-phase execution (validate-all, then execute-all)
- Dry-run mode for safe operation preview
- Force mode for intentional overwrites
- Schema parsing and validation
- File diffing with conflict resolution
- Transaction-based code generation
- Zero dependencies on CLI frameworks

## Philosophy

Fledge follows these core principles:

1. **Domain Agnostic:** No knowledge of web frameworks, CLIs, or specific use cases
2. **Validation First:** Catch errors before any files are written
3. **Testability:** All I/O is injectable and mockable
4. **Simplicity:** Minimal API surface, maximum flexibility
5. **Convention Over Magic:** Explicit operations, no hidden behavior

## Quick Start

### Installation

```bash
go get github.com/simonhull/firebird-suite/fledge
```

### Basic Usage

```go
package main

import (
    "context"
    "os"

    "github.com/simonhull/firebird-suite/fledge/generator"
)

func main() {
    ctx := context.Background()

    // Build operations
    ops := []generator.Operation{
        &generator.WriteFileOp{
            Path:    "output/hello.txt",
            Content: []byte("Hello, Fledge!"),
            Mode:    0644,
        },
    }

    // Execute with validation
    err := generator.Execute(ctx, ops, generator.ExecuteOptions{
        DryRun: false,
        Force:  false,
        Writer: os.Stdout,
    })
    if err != nil {
        panic(err)
    }
}
```

**Output:**
```
✓ Create output/hello.txt (14 bytes)
```

### Dry-Run Mode

```go
err := generator.Execute(ctx, ops, generator.ExecuteOptions{
    DryRun: true,  // Validates but doesn't write
    Writer: os.Stdout,
})
```

**Output:**
```
✓ [DRY RUN] Create output/hello.txt (14 bytes)
```

## Project Structure

- **generator/** - Core operation system (validate + execute)
- **schema/** - Schema parsing and validation
- **exec/** - Command execution with solid UX
- **output/** - Styled terminal output
- **input/** - Interactive prompts
- **project/** - Go module and Firebird project detection
- **filesystem/** - Directory traversal with smart ignores

## Documentation

- [Architecture & Design Decisions](ARCHITECTURE.md)

## Example: Building a Code Generator

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/simonhull/firebird-suite/fledge/generator"
)

// Generate creates operations for a Go model struct
func Generate(modelName string) ([]generator.Operation, error) {
    var ops []generator.Operation

    // Generate model file
    fileName := strings.ToLower(modelName) + ".go"
    content := fmt.Sprintf(`package models

type %s struct {
    ID        int64
    CreatedAt time.Time
}
`, modelName)

    ops = append(ops, &generator.WriteFileOp{
        Path:    "models/" + fileName,
        Content: []byte(content),
        Mode:    0644,
    })

    return ops, nil
}

func main() {
    ctx := context.Background()

    // Build operations
    ops, err := Generate("User")
    if err != nil {
        panic(err)
    }

    // Execute
    if err := generator.Execute(ctx, ops, generator.ExecuteOptions{
        DryRun: false,
        Force:  false,
        Writer: os.Stdout,
    }); err != nil {
        panic(err)
    }
}
```

## Testing Your Generator

```go
func TestGenerate(t *testing.T) {
    ops, err := Generate("User")
    if err != nil {
        t.Fatalf("Generate failed: %v", err)
    }

    if len(ops) != 1 {
        t.Errorf("expected 1 operation, got %d", len(ops))
    }

    // Verify operation describes the file
    desc := ops[0].Description()
    if !strings.Contains(desc, "user.go") {
        t.Errorf("unexpected description: %s", desc)
    }
}

func TestExecute(t *testing.T) {
    tmpDir := t.TempDir()

    ops := []generator.Operation{
        &generator.WriteFileOp{
            Path:    filepath.Join(tmpDir, "test.txt"),
            Content: []byte("hello"),
            Mode:    0644,
        },
    }

    var buf bytes.Buffer
    err := generator.Execute(context.Background(), ops, generator.ExecuteOptions{
        Writer: &buf,
    })

    if err != nil {
        t.Fatalf("execution failed: %v", err)
    }

    // Verify file was created
    content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
    if err != nil {
        t.Fatalf("file not created: %v", err)
    }

    if string(content) != "hello" {
        t.Errorf("wrong content: got %q, want %q", content, "hello")
    }
}
```

## Real-World Usage

Fledge is used by:

- **Firebird** - Convention-over-configuration web framework for Go
- **Owl** - Convention-aware documentation generator for Go

## Contributing

We welcome contributions! Please ensure:

- All tests pass (`go test ./...`)
- Code follows Go conventions (`go fmt`, `go vet`)
- New features include documentation
- Public APIs have godoc comments

## License

MIT License - see [LICENSE](../LICENSE) for details.
