package migrate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/simonhull/firebird-suite/fledge/exec"
	"github.com/simonhull/firebird-suite/fledge/output"
)

const migrationsDir = "./migrations"

// Migrator wraps golang-migrate commands
type Migrator struct {
	connectionString string
	executor         *exec.Executor
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
		executor:         exec.NewExecutor(nil),
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

// CheckMigrateInstalled checks if golang-migrate is installed
func CheckMigrateInstalled() error {
	executor := exec.NewExecutor(nil)
	err := executor.Run(context.Background(), "migrate", "-version")
	if err != nil {
		return fmt.Errorf("golang-migrate not found. Install it with:\n  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest")
	}
	return nil
}
