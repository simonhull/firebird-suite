# 🦉 Owl Docs

**Convention-aware documentation generator for Go**

Owl Docs transforms your Go codebase into beautiful, intelligent documentation that developers actually want to read.

## Features

- 🔍 **Convention Detection** - Automatically identifies handlers, services, repositories, and more
- 📊 **Dependency Visualization** - See how your code fits together
- 🎨 **Beautiful UI** - Modern, responsive design with dark mode
- ⚡ **Live Reload** - Docs update as you code
- 🎯 **Zero Config** - Works great out of the box
- 🔧 **Highly Configurable** - Customize for your project's needs

## Installation

```bash
go install github.com/simonhull/firebird-suite/owldocs/cmd/owldocs@latest
```

## Quick Start

```bash
# Generate docs for your project
owldocs generate ./myproject

# Serve with live reload
owldocs serve --watch

# Open http://localhost:6060
```

## Status

**Version: 0.1.0 (Early Development)**

Currently bootstrapping the core architecture. Check back soon for updates!

## Architecture

Owl Docs is built on three core components:

1. **Analyzer** - Parses Go code and extracts documentation
2. **Convention Detector** - Identifies architectural patterns
3. **Generator** - Creates beautiful static HTML

## Roadmap

- [x] Project structure and interfaces
- [ ] AST parsing and analysis
- [ ] Convention detection (6+ patterns)
- [ ] HTML generation and theming
- [ ] Dev server with live reload
- [ ] Firebird framework integration

## License

MIT
