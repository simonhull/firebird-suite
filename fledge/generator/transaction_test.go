package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTransaction_Success(t *testing.T) {
	tempDir := t.TempDir()

	tx := NewTransaction()
	tx.AddFile(filepath.Join(tempDir, "file1.txt"), []byte("content1"), 0644)
	tx.AddFile(filepath.Join(tempDir, "file2.txt"), []byte("content2"), 0644)

	err := tx.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify files exist
	content1, err := os.ReadFile(filepath.Join(tempDir, "file1.txt"))
	if err != nil || string(content1) != "content1" {
		t.Error("file1.txt not written correctly")
	}

	content2, err := os.ReadFile(filepath.Join(tempDir, "file2.txt"))
	if err != nil || string(content2) != "content2" {
		t.Error("file2.txt not written correctly")
	}
}

func TestTransaction_RollbackOnError(t *testing.T) {
	tempDir := t.TempDir()

	tx := NewTransaction()
	tx.AddFile(filepath.Join(tempDir, "file1.txt"), []byte("content1"), 0644)

	// Add a file to an invalid path (should fail)
	invalidPath := filepath.Join(tempDir, "\x00invalid", "file2.txt")
	tx.AddFile(invalidPath, []byte("content2"), 0644)

	err := tx.Commit()
	if err == nil {
		t.Fatal("Expected commit to fail with invalid path")
	}

	// Verify file1.txt was rolled back (deleted)
	if _, err := os.Stat(filepath.Join(tempDir, "file1.txt")); !os.IsNotExist(err) {
		t.Error("file1.txt should have been rolled back")
	}
}

func TestTransaction_CannotCommitTwice(t *testing.T) {
	tempDir := t.TempDir()

	tx := NewTransaction()
	tx.AddFile(filepath.Join(tempDir, "file1.txt"), []byte("content1"), 0644)

	err := tx.Commit()
	if err != nil {
		t.Fatalf("First commit failed: %v", err)
	}

	err = tx.Commit()
	if err == nil {
		t.Fatal("Expected second commit to fail")
	}
}

func TestTransaction_ManualRollback(t *testing.T) {
	tempDir := t.TempDir()

	tx := NewTransaction()
	file1Path := filepath.Join(tempDir, "file1.txt")
	tx.AddFile(file1Path, []byte("content1"), 0644)

	// Commit
	err := tx.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(file1Path); err != nil {
		t.Fatal("file1.txt should exist after commit")
	}

	// Manual rollback should not affect committed transaction
	tx.Rollback()

	// File should still exist
	if _, err := os.Stat(file1Path); err != nil {
		t.Error("file1.txt should still exist after rollback of committed transaction")
	}
}
