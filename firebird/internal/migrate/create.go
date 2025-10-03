package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/simonhull/firebird-suite/fledge/output"
)

// CreateManualMigration creates empty migration files for manual editing
func CreateManualMigration(name string) error {
	// Generate timestamp
	timestamp := time.Now().UTC().Format("20060102150405")

	// Build filenames
	upFile := fmt.Sprintf("%s_%s.up.sql", timestamp, name)
	downFile := fmt.Sprintf("%s_%s.down.sql", timestamp, name)

	upPath := filepath.Join(migrationsDir, upFile)
	downPath := filepath.Join(migrationsDir, downFile)

	output.Verbose(fmt.Sprintf("Creating migration: %s", name))

	// Ensure migrations directory exists
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Create up migration
	upContent := fmt.Sprintf(`-- Migration: %s
-- Created: %s
--
-- Write your UP migration here

`, name, time.Now().Format("2006-01-02 15:04:05"))

	if err := os.WriteFile(upPath, []byte(upContent), 0644); err != nil {
		return fmt.Errorf("failed to write up migration: %w", err)
	}
	output.Verbose(fmt.Sprintf("Created: %s", upPath))

	// Create down migration
	downContent := fmt.Sprintf(`-- Migration: %s (ROLLBACK)
-- Created: %s
--
-- Write your DOWN migration here (to undo the UP migration)

`, name, time.Now().Format("2006-01-02 15:04:05"))

	if err := os.WriteFile(downPath, []byte(downContent), 0644); err != nil {
		return fmt.Errorf("failed to write down migration: %w", err)
	}
	output.Verbose(fmt.Sprintf("Created: %s", downPath))

	output.Success(fmt.Sprintf("Created migration files: %s", name))
	output.Info("Edit the migration files:")
	output.Step(upPath)
	output.Step(downPath)

	return nil
}
