# Firebird Suite

A collection of convention-over-configuration tools for Go developers.

## Structure

- **fledge/** - Foundation library for building CLI tools
- **firebird/** - Main web framework CLI (coming soon)

## Development

This is a Go workspace (monorepo). Requires Go 1.25+.
```bash
# Run tests for fledge
cd fledge
go test ./...