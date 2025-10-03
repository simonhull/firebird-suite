package migrate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	fledgeexec "github.com/simonhull/firebird-suite/fledge/exec"
	"github.com/simonhull/firebird-suite/fledge/output"
)

const migrationsDir = "./migrations"

// Migrator wraps golang-migrate commands
type Migrator struct {
	connectionString string
	executor         *fledgeexec.Executor
}

// NewMigrator creates a new migrator
func NewMigrator() (*Migrator, error) {
	// Load database config
	cfg, err := LoadDatabaseConfig()
	if err != nil {
		return nil, err
	}

	// Build connection string
	connStr, err := BuildConnectionString(cfg)
	if err != nil {
		return nil, err
	}

	return &Migrator{
		connectionString: connStr,
		executor:         fledgeexec.NewExecutor(nil),
	}, nil
}

// Up applies all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	output.Info("Applying migrations...")
	output.Verbose(fmt.Sprintf("Migrations directory: %s", migrationsDir))
	output.Verbose(fmt.Sprintf("Database: %s", m.maskPassword(m.connectionString)))

	err := m.executor.Run(ctx,
		"migrate",
		"-path", migrationsDir,
		"-database", m.connectionString,
		"up",
	)

	if err != nil {
		// Check if it's "no change" error (not a real error)
		if strings.Contains(err.Error(), "no change") {
			output.Info("No pending migrations")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	output.Success("Migrations applied successfully")
	return nil
}

// Down rolls back the last n migrations
func (m *Migrator) Down(ctx context.Context, steps int) error {
	if steps < 1 {
		return fmt.Errorf("steps must be at least 1")
	}

	output.Info(fmt.Sprintf("Rolling back %d migration(s)...", steps))
	output.Verbose(fmt.Sprintf("Migrations directory: %s", migrationsDir))
	output.Verbose(fmt.Sprintf("Database: %s", m.maskPassword(m.connectionString)))

	err := m.executor.Run(ctx,
		"migrate",
		"-path", migrationsDir,
		"-database", m.connectionString,
		"down",
		strconv.Itoa(steps),
	)

	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	output.Success(fmt.Sprintf("Rolled back %d migration(s)", steps))
	return nil
}

// Status shows current migration status
func (m *Migrator) Status(ctx context.Context) error {
	output.Info("Migration status:")
	output.Verbose(fmt.Sprintf("Migrations directory: %s", migrationsDir))
	output.Verbose(fmt.Sprintf("Database: %s", m.maskPassword(m.connectionString)))

	err := m.executor.Run(ctx,
		"migrate",
		"-path", migrationsDir,
		"-database", m.connectionString,
		"version",
	)

	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	return nil
}

// Force sets the migration version without running migrations
func (m *Migrator) Force(ctx context.Context, version string) error {
	output.Info(fmt.Sprintf("Forcing migration version to: %s", version))
	output.Verbose(fmt.Sprintf("Database: %s", m.maskPassword(m.connectionString)))
	output.Info("⚠️  Warning: This is a recovery tool. Use with caution!")

	err := m.executor.Run(ctx,
		"migrate",
		"-path", migrationsDir,
		"-database", m.connectionString,
		"force",
		version,
	)

	if err != nil {
		return fmt.Errorf("force failed: %w", err)
	}

	output.Success(fmt.Sprintf("Migration version forced to: %s", version))
	return nil
}

// maskPassword masks the password in connection string for display
func (m *Migrator) maskPassword(connStr string) string {
	// Simple masking: replace password between : and @
	parts := strings.Split(connStr, ":")
	if len(parts) < 3 {
		return connStr
	}

	// Find the password part (between second : and @)
	for i := 2; i < len(parts); i++ {
		if idx := strings.Index(parts[i], "@"); idx != -1 {
			parts[i] = "****" + parts[i][idx:]
			break
		}
	}

	return strings.Join(parts, ":")
}

// List shows all migrations with their status
func (m *Migrator) List(ctx context.Context) error {
	output.Info("Migration list:")
	output.Verbose(fmt.Sprintf("Migrations directory: %s", migrationsDir))
	output.Verbose(fmt.Sprintf("Database: %s", m.maskPassword(m.connectionString)))

	// Get current version first using exec.Command to capture output
	cmd := exec.Command("migrate",
		"-path", migrationsDir,
		"-database", m.connectionString,
		"version",
	)

	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = os.Stderr

	currentVersion := "none"
	if err := cmd.Run(); err == nil {
		currentVersion = strings.TrimSpace(outBuf.String())
	}

	output.Verbose(fmt.Sprintf("Current version: %s", currentVersion))

	// List all migrations using os.ReadDir
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Parse migration files
	migrations := parseMigrationEntries(entries, currentVersion)

	// Display migrations
	if len(migrations) == 0 {
		output.Info("No migrations found")
		return nil
	}

	output.Info(fmt.Sprintf("Found %d migration(s):\n", len(migrations)/2))
	for _, mig := range migrations {
		if mig.Direction == "up" {
			status := "pending"
			symbol := "○"
			if mig.Applied {
				status = "applied"
				symbol = "✓"
			}
			output.Info(fmt.Sprintf("  %s %s - %s", symbol, mig.Version, status))
		}
	}

	return nil
}

// Migration represents a single migration file
type Migration struct {
	Version   string
	Name      string
	Direction string // "up" or "down"
	Applied   bool
}

// parseMigrationEntries parses directory entries and determines which migrations are applied
func parseMigrationEntries(entries []os.DirEntry, currentVersion string) []Migration {
	var migrations []Migration

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Migration files are in format: {version}_{name}.{up|down}.sql
		// Example: 20250102030405_create_users.up.sql
		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			continue
		}

		version := parts[0]
		rest := strings.Join(parts[1:], "_")

		// Extract direction (up or down)
		var direction string
		if strings.HasSuffix(rest, ".up.sql") {
			direction = "up"
			rest = strings.TrimSuffix(rest, ".up.sql")
		} else if strings.HasSuffix(rest, ".down.sql") {
			direction = "down"
			rest = strings.TrimSuffix(rest, ".down.sql")
		} else {
			continue
		}

		// Check if migration is applied
		// Migration is applied if its version <= current version
		applied := false
		if currentVersion != "none" && version <= currentVersion {
			applied = true
		}

		migrations = append(migrations, Migration{
			Version:   version,
			Name:      rest,
			Direction: direction,
			Applied:   applied,
		})
	}

	return migrations
}

// CheckMigrateInstalled checks if golang-migrate is installed
func CheckMigrateInstalled() error {
	executor := fledgeexec.NewExecutor(nil)
	err := executor.Run(context.Background(), "migrate", "-version")
	if err != nil {
		return fmt.Errorf("golang-migrate not found. Install it with:\n  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest")
	}
	return nil
}
