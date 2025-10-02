package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// Transaction represents a set of file operations that can be committed or rolled back
type Transaction struct {
	operations []fileOperation
	committed  bool
}

// fileOperation represents a single file write operation
type fileOperation struct {
	path    string
	content []byte
	mode    os.FileMode
}

// NewTransaction creates a new file operation transaction
func NewTransaction() *Transaction {
	return &Transaction{
		operations: make([]fileOperation, 0),
	}
}

// AddFile stages a file write operation (doesn't write yet)
func (t *Transaction) AddFile(path string, content []byte, mode os.FileMode) {
	t.operations = append(t.operations, fileOperation{
		path:    path,
		content: content,
		mode:    mode,
	})
}

// Commit writes all staged files to disk
// If any write fails, it attempts to rollback (delete) previously written files
func (t *Transaction) Commit() error {
	if t.committed {
		return fmt.Errorf("transaction already committed")
	}

	writtenFiles := make([]string, 0, len(t.operations))

	// Attempt to write all files
	for _, op := range t.operations {
		// Ensure directory exists
		dir := filepath.Dir(op.path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			// Rollback on error
			t.rollback(writtenFiles)
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write file
		if err := os.WriteFile(op.path, op.content, op.mode); err != nil {
			// Rollback on error
			t.rollback(writtenFiles)
			return fmt.Errorf("failed to write file %s: %w", op.path, err)
		}

		writtenFiles = append(writtenFiles, op.path)
	}

	t.committed = true
	return nil
}

// rollback attempts to delete all files that were written
func (t *Transaction) rollback(writtenFiles []string) {
	for _, path := range writtenFiles {
		os.Remove(path) // Best effort, ignore errors
	}
}

// Rollback manually triggers a rollback (for use in defer)
func (t *Transaction) Rollback() {
	if !t.committed {
		paths := make([]string, 0, len(t.operations))
		for _, op := range t.operations {
			// Check if file was written
			if _, err := os.Stat(op.path); err == nil {
				paths = append(paths, op.path)
			}
		}
		t.rollback(paths)
	}
}
