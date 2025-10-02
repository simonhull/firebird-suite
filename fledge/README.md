# Fledge

The foundation library for building Firebird CLI tools.

## What is Fledge?

Fledge is a **domain-agnostic** Go library that provides the core utilities needed to build convention-over-configuration CLI tools. It powers Firebird and all other tools in the Firebird Suite.

## Philosophy

- **Domain Agnostic**: Zero knowledge of specific tools or domains
- **Beautiful UX**: Spinners, colors, interactive prompts out of the box
- **Composable**: Small, focused packages that work together
- **Testable**: Full mocking support for all components

## Packages

### `fledge/generator`
Template-based code generation with conflict resolution.

- Render templates with helper functions (pascalCase, snakeCase, plural, etc.)
- Interactive conflict resolution (skip/overwrite/diff)
- Myers diff algorithm for comparing files
- Caching for performance

### `fledge/exec`
Execute external commands with beautiful UX.

- Run commands with context support (cancellation, timeouts)
- Beautiful spinners using bubbletea
- Plugin registry for domain commands
- Fluent API for building commands
- Mock-friendly design for testing

### `fledge/schema`
Generic YAML schema parsing for Firebird tools.

- Common schema structure (apiVersion, kind, name, spec)
- Validation framework with helpful error messages
- Each tool defines its own spec structure

## Installation
```bash
go get github.com/simonhull/firebird-suite/fledge
```
## Usage
```go
import (
    "github.com/simonhull/firebird-suite/fledge/generator"
    "github.com/simonhull/firebird-suite/fledge/exec"
    "github.com/simonhull/firebird-suite/fledge/schema"
)
```
See individual package documentation for detailed examples.

## Testing
```bash
go test ./...
```

## Architecture
Fledge provides the foundation. Domain packages (like Firebird) build on top:
```
Domain Package (Firebird)
    ↓ uses
Fledge (Foundation)
    ↓ provides
Generator + Exec + Schema
```
