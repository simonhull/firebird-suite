// Package generator provides utilities for template-based code generation
// with conflict resolution, rollback support, and excellent DX.
//
// # Features
//
//   - Template rendering with helper functions
//   - Conflict resolution (interactive, --force, --skip, --diff)
//   - Myers diff algorithm for file comparison
//   - Transaction support for atomic file operations
//
// # Transactions
//
// Use transactions to ensure all files are written atomically:
//
//	tx := generator.NewTransaction()
//	tx.AddFile("file1.go", content1, 0644)
//	tx.AddFile("file2.go", content2, 0644)
//
//	if err := tx.Commit(); err != nil {
//	    // All files rolled back automatically on error
//	    return err
//	}
//
// If any file write fails, all previously written files are automatically
// cleaned up, ensuring no partial state is left on disk.
package generator
