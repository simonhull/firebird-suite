package generator_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

func TestExecute_DryRun(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	ops := []generator.Operation{
		&generator.WriteFileOp{
			Path:    filepath.Join(tmpDir, "test.txt"),
			Content: []byte("hello"),
			Mode:    0644,
		},
	}

	var buf bytes.Buffer
	err := generator.Execute(ctx, ops, generator.ExecuteOptions{
		DryRun: true,
		Writer: &buf,
	})

	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}

	// File should NOT be created
	if _, err := os.Stat(filepath.Join(tmpDir, "test.txt")); !os.IsNotExist(err) {
		t.Error("dry run created file")
	}

	// Output should show dry run
	output := buf.String()
	if !strings.Contains(output, "[DRY RUN]") {
		t.Errorf("output missing [DRY RUN] marker, got: %s", output)
	}
}

func TestExecute_RealRun(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	ops := []generator.Operation{
		&generator.WriteFileOp{
			Path:    filepath.Join(tmpDir, "test.txt"),
			Content: []byte("hello"),
			Mode:    0644,
		},
	}

	err := generator.Execute(ctx, ops, generator.ExecuteOptions{
		DryRun: false,
	})

	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	// File SHOULD be created
	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	if string(content) != "hello" {
		t.Errorf("wrong content: got %q, want %q", content, "hello")
	}
}

func TestExecute_ForceOverwrite(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	// Create existing file
	err := os.WriteFile(path, []byte("old"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	ops := []generator.Operation{
		&generator.WriteFileOp{
			Path:    path,
			Content: []byte("new"),
			Mode:    0644,
		},
	}

	// Without force - should fail
	err = generator.Execute(ctx, ops, generator.ExecuteOptions{
		Force: false,
	})
	if err == nil {
		t.Error("expected error when file exists without force")
	}

	// With force - should succeed
	err = generator.Execute(ctx, ops, generator.ExecuteOptions{
		Force: true,
	})
	if err != nil {
		t.Fatalf("execute with force failed: %v", err)
	}

	content, _ := os.ReadFile(path)
	if string(content) != "new" {
		t.Errorf("file not overwritten: got %q", content)
	}
}

func TestExecute_MultipleOperations(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	ops := []generator.Operation{
		&generator.WriteFileOp{
			Path:    filepath.Join(tmpDir, "file1.txt"),
			Content: []byte("content1"),
			Mode:    0644,
		},
		&generator.WriteFileOp{
			Path:    filepath.Join(tmpDir, "subdir", "file2.txt"),
			Content: []byte("content2"),
			Mode:    0644,
		},
		&generator.WriteFileOp{
			Path:    filepath.Join(tmpDir, "file3.txt"),
			Content: []byte("content3"),
			Mode:    0644,
		},
	}

	var buf bytes.Buffer
	err := generator.Execute(ctx, ops, generator.ExecuteOptions{
		Writer: &buf,
	})

	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	// All files should be created
	files := []string{
		filepath.Join(tmpDir, "file1.txt"),
		filepath.Join(tmpDir, "subdir", "file2.txt"),
		filepath.Join(tmpDir, "file3.txt"),
	}

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("file not created: %s", file)
		}
	}

	// Output should show all operations
	output := buf.String()
	if strings.Count(output, "✓") != 3 {
		t.Errorf("expected 3 checkmarks in output, got: %s", output)
	}
}

func TestExecute_ValidationBeforeExecution(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create an operation that will fail validation (nil content)
	ops := []generator.Operation{
		&generator.WriteFileOp{
			Path:    filepath.Join(tmpDir, "valid.txt"),
			Content: []byte("valid"),
			Mode:    0644,
		},
		&generator.WriteFileOp{
			Path:    filepath.Join(tmpDir, "invalid.txt"),
			Content: nil, // Nil content - should fail validation
			Mode:    0644,
		},
	}

	err := generator.Execute(ctx, ops, generator.ExecuteOptions{})

	if err == nil {
		t.Error("expected validation error for nil content")
	}

	// Neither file should be created (atomic validation)
	if _, err := os.Stat(filepath.Join(tmpDir, "valid.txt")); !os.IsNotExist(err) {
		t.Error("valid.txt was created despite validation failure in another operation")
	}
}

func TestWriteFileOp_Validate(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		op        *generator.WriteFileOp
		force     bool
		wantError bool
		setupFunc func() error
	}{
		{
			name: "valid operation",
			op: &generator.WriteFileOp{
				Path:    filepath.Join(tmpDir, "valid.txt"),
				Content: []byte("content"),
				Mode:    0644,
			},
			force:     false,
			wantError: false,
		},
		{
			name: "nil content fails",
			op: &generator.WriteFileOp{
				Path:    filepath.Join(tmpDir, "nil.txt"),
				Content: nil,
				Mode:    0644,
			},
			force:     false,
			wantError: true,
		},
		{
			name: "existing file without force fails",
			op: &generator.WriteFileOp{
				Path:    filepath.Join(tmpDir, "existing.txt"),
				Content: []byte("new content"),
				Mode:    0644,
			},
			force:     false,
			wantError: true,
			setupFunc: func() error {
				return os.WriteFile(filepath.Join(tmpDir, "existing.txt"), []byte("old"), 0644)
			},
		},
		{
			name: "existing file with force succeeds",
			op: &generator.WriteFileOp{
				Path:    filepath.Join(tmpDir, "existing_force.txt"),
				Content: []byte("new content"),
				Mode:    0644,
			},
			force:     true,
			wantError: false,
			setupFunc: func() error {
				return os.WriteFile(filepath.Join(tmpDir, "existing_force.txt"), []byte("old"), 0644)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				if err := tt.setupFunc(); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			err := tt.op.Validate(ctx, tt.force)

			if tt.wantError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWriteFileOp_Description(t *testing.T) {
	op := &generator.WriteFileOp{
		Path:    "/path/to/file.txt",
		Content: []byte("hello world"),
		Mode:    0644,
	}

	desc := op.Description()

	// Should include path and size
	if !strings.Contains(desc, "/path/to/file.txt") {
		t.Errorf("description missing path: %s", desc)
	}
	if !strings.Contains(desc, "11 bytes") {
		t.Errorf("description missing size: %s", desc)
	}
}

func TestExecute_CustomWriter(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	ops := []generator.Operation{
		&generator.WriteFileOp{
			Path:    filepath.Join(tmpDir, "test.txt"),
			Content: []byte("hello"),
			Mode:    0644,
		},
	}

	var buf bytes.Buffer
	err := generator.Execute(ctx, ops, generator.ExecuteOptions{
		Writer: &buf,
	})

	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	// Output should go to custom writer
	output := buf.String()
	if !strings.Contains(output, "✓") {
		t.Errorf("output missing checkmark: %s", output)
	}
}

func TestWriteFileOp_EmptyContent(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	op := &generator.WriteFileOp{
		Path:    filepath.Join(tmpDir, "empty.txt"),
		Content: []byte{}, // Empty but not nil
		Mode:    0644,
	}

	// Should validate successfully
	if err := op.Validate(ctx, false); err != nil {
		t.Errorf("empty content should be valid: %v", err)
	}

	// Should create empty file
	if err := op.Execute(ctx); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	content, err := os.ReadFile(op.Path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if len(content) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(content))
	}
}

func TestWriteFileOp_NilContent(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	op := &generator.WriteFileOp{
		Path:    filepath.Join(tmpDir, "test.txt"),
		Content: nil, // Nil content
		Mode:    0644,
	}

	// Should fail validation
	err := op.Validate(ctx, false)
	if err == nil {
		t.Error("nil content should fail validation")
	}

	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error should mention nil: %v", err)
	}
}

func TestWriteFileOp_NestedDirectories(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Deep nested path
	op := &generator.WriteFileOp{
		Path:    filepath.Join(tmpDir, "a", "b", "c", "deep.txt"),
		Content: []byte("nested"),
		Mode:    0644,
	}

	// Should validate (creates directories)
	if err := op.Validate(ctx, false); err != nil {
		t.Errorf("nested directory creation should succeed: %v", err)
	}

	// Should create file
	if err := op.Execute(ctx); err != nil {
		t.Fatalf("failed to create nested file: %v", err)
	}

	// Verify file exists
	content, err := os.ReadFile(op.Path)
	if err != nil {
		t.Fatalf("failed to read nested file: %v", err)
	}

	if string(content) != "nested" {
		t.Errorf("wrong content: got %q, want %q", content, "nested")
	}
}

func TestExecute_PartialValidationFailure(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create a file that will cause conflict
	existingPath := filepath.Join(tmpDir, "existing.txt")
	err := os.WriteFile(existingPath, []byte("old"), 0644)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	ops := []generator.Operation{
		&generator.WriteFileOp{
			Path:    filepath.Join(tmpDir, "new1.txt"),
			Content: []byte("content1"),
			Mode:    0644,
		},
		&generator.WriteFileOp{
			Path:    existingPath, // This will fail validation
			Content: []byte("new"),
			Mode:    0644,
		},
		&generator.WriteFileOp{
			Path:    filepath.Join(tmpDir, "new2.txt"),
			Content: []byte("content2"),
			Mode:    0644,
		},
	}

	// Should fail validation on second operation
	err = generator.Execute(ctx, ops, generator.ExecuteOptions{Force: false})
	if err == nil {
		t.Fatal("expected validation failure")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention file exists: %v", err)
	}

	// NO files should be created (atomic validation)
	if _, err := os.Stat(filepath.Join(tmpDir, "new1.txt")); !os.IsNotExist(err) {
		t.Error("new1.txt should not exist after validation failure")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "new2.txt")); !os.IsNotExist(err) {
		t.Error("new2.txt should not exist after validation failure")
	}
}

func TestWriteFileOp_ErrorMessageNoCLIHints(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create existing file
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("old"), 0644)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	op := &generator.WriteFileOp{
		Path:    path,
		Content: []byte("new"),
		Mode:    0644,
	}

	// Validate without force
	err = op.Validate(ctx, false)
	if err == nil {
		t.Fatal("expected validation error")
	}

	errMsg := err.Error()

	// Should NOT mention CLI flags
	if strings.Contains(errMsg, "--force") || strings.Contains(errMsg, "use ") {
		t.Errorf("error message should not mention CLI flags: %v", err)
	}

	// Should describe the problem
	if !strings.Contains(errMsg, "already exists") {
		t.Errorf("error message should describe the problem: %v", err)
	}
}
