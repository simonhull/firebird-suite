package generator

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Operation represents a file system operation that can be validated and executed.
//
// Validate checks if the operation would succeed without executing it.
// Some operations may have side effects during validation (e.g., creating parent directories).
// force=true skips conflict checks (e.g., file already exists).
//
// Execute performs the actual operation. This should only be called after Validate succeeds.
//
// Description returns a human-readable description for output (e.g., "Create models/user.go (234 bytes)").
type Operation interface {
	Validate(ctx context.Context, force bool) error
	Execute(ctx context.Context) error
	Description() string
}

// WriteFileOp creates a new file with content.
//
// Validation behavior:
//   - Creates parent directories if they don't exist (via os.MkdirAll)
//   - Checks for file conflicts unless force=true
//   - Allows empty content (zero bytes) but rejects nil content
//
// Execution behavior:
//   - Creates parent directories if needed
//   - Writes file atomically with specified Mode
type WriteFileOp struct {
	Path    string      // File path to create
	Content []byte      // File content (can be empty, must not be nil)
	Mode    fs.FileMode // File permissions (e.g., 0644)
}

func (op *WriteFileOp) Validate(ctx context.Context, force bool) error {
	dir := filepath.Dir(op.Path)

	// Create parent directory (side effect, but idempotent)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", dir, err)
	}

	// Check file conflict unless force is enabled
	if !force {
		if _, err := os.Stat(op.Path); err == nil {
			return fmt.Errorf("file already exists: %s", op.Path)
		}
	}

	// Reject nil content (empty is OK)
	if op.Content == nil {
		return fmt.Errorf("content is nil for file: %s", op.Path)
	}

	return nil
}

func (op *WriteFileOp) Execute(ctx context.Context) error {
	dir := filepath.Dir(op.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(op.Path, op.Content, op.Mode)
}

func (op *WriteFileOp) Description() string {
	return fmt.Sprintf("Create %s (%d bytes)", op.Path, len(op.Content))
}

// WriteFileIfNotExistsOp creates a file only if it doesn't already exist.
//
// This is useful for scaffolding files that users may have customized.
// Unlike WriteFileOp which fails on existing files (unless force=true),
// this operation silently skips existing files.
//
// Validation behavior:
//   - Creates parent directories if they don't exist
//   - Passes validation even if file exists (Execute will skip)
//   - Checks content is not nil
//
// Execution behavior:
//   - Skips writing if file already exists
//   - Creates parent directories if needed
//   - Writes file with specified Mode if file doesn't exist
type WriteFileIfNotExistsOp struct {
	Path    string      // File path to create
	Content []byte      // File content (can be empty, must not be nil)
	Mode    fs.FileMode // File permissions (e.g., 0644)
}

func (op *WriteFileIfNotExistsOp) Validate(ctx context.Context, force bool) error {
	// Create parent directory (side effect, but idempotent)
	dir := filepath.Dir(op.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", dir, err)
	}

	// Check if file exists - validation still passes, but Execute will skip
	// (This is intentional: we don't want to fail validation for existing files)
	if _, err := os.Stat(op.Path); err == nil {
		return nil
	}

	// Reject nil content
	if op.Content == nil {
		return fmt.Errorf("content is nil for file: %s", op.Path)
	}

	return nil
}

func (op *WriteFileIfNotExistsOp) Execute(ctx context.Context) error {
	// Skip if file exists
	if _, err := os.Stat(op.Path); err == nil {
		return nil
	}

	// Create parent directory
	dir := filepath.Dir(op.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(op.Path, op.Content, op.Mode); err != nil {
		return fmt.Errorf("failed to write file %s: %w", op.Path, err)
	}

	return nil
}

func (op *WriteFileIfNotExistsOp) Description() string {
	if _, err := os.Stat(op.Path); err == nil {
		return fmt.Sprintf("Skip %s (already exists)", op.Path)
	}
	return fmt.Sprintf("Create %s (%d bytes)", op.Path, len(op.Content))
}
