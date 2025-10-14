# 🦉 Owl Docs

Convention-Aware Documentation Generator for Go

## Overview

Owl analyzes Go projects to automatically detect architectural patterns and conventions, then generates beautiful, structured documentation that reflects your project's actual organization.

Instead of generic godoc output, Owl understands your architecture - handlers, services, repositories, DTOs, middleware - and organizes documentation accordingly.

## Features

- 🏗️ **Convention Detection** - Automatically identifies handlers, services, repositories, and more
- 🔍 **Deep Code Analysis** - Parses function bodies to understand dependencies and call graphs
- 📊 **Dependency Tracking** - Visualizes type usage and function calls
- 🎯 **Generic Support** - Full support for Go 1.18+ generics
- 🗂️ **Smart Organization** - Groups docs by architectural layer, not just package
- 🎨 **Beautiful Output** - Clean, modern documentation themes
- ⚡ **Live Reload** - Development server with auto-refresh (coming soon)
- 🔌 **Extensible** - Custom pattern detection and templates

## Installation

```bash
# From the firebird-suite workspace
cd owldocs
go install ./cmd/owldocs
```

Or build directly:

```bash
go build -o owldocs ./cmd/owldocs
```

## Quick Start

```bash
# Generate docs for a project
owldocs generate ./internal/handlers

# Generate docs for entire project
owldocs generate .

# Start development server (coming soon)
owldocs serve

# Initialize configuration
owldocs init
```

## Example Output

```
🦉 Analyzing project at ./internal/handlers...
✅ Found 5 packages

📊 Analysis Results:
   Types: 23
   Functions: 47

🏗️  Detected Conventions:
   Handler: 8
   Service: 5
   Repository: 4
   DTO: 12
   Middleware: 3

✅ Documentation generated successfully!
```

## Architecture

Owl is built with three main components:

- **Analyzer** - Parses Go code and extracts structure
- **Convention Detector** - Identifies architectural patterns
- **Generator** - Creates documentation from analysis

## Dependencies

Owl is built on [Fledge](../fledge), the Firebird suite foundation library.

Key Fledge packages used:
- `fledge/output` - Consistent terminal output
- `fledge/project` - Go module and Firebird project detection
- `fledge/filesystem` - Directory traversal and package discovery

## Roadmap

- [x] Core analyzer with AST parsing
- [x] Convention detection system
- [x] Deep function body analysis
- [x] Generic type support
- [x] Dependency tracking
- [ ] HTML documentation generation
- [ ] Dependency graph visualization
- [ ] Live reload server
- [ ] Custom pattern definitions
- [ ] Multiple output formats (Markdown, JSON)
- [ ] Search index generation

## License

MIT
