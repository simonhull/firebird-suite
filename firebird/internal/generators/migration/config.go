package migration

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// DatabaseDialect represents the SQL dialect to generate
type DatabaseDialect string

const (
	PostgreSQL DatabaseDialect = "postgres"
	MySQL      DatabaseDialect = "mysql"
	SQLite     DatabaseDialect = "sqlite"
)

// DetectDatabaseDialect reads firebird.yml and returns the database dialect
func DetectDatabaseDialect() (DatabaseDialect, error) {
	// Check if firebird.yml exists
	if _, err := os.Stat("firebird.yml"); os.IsNotExist(err) {
		// Default to PostgreSQL if no config
		return PostgreSQL, nil
	}

	// Load config
	v := viper.New()
	v.SetConfigName("firebird")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return "", fmt.Errorf("failed to read firebird.yml: %w", err)
	}

	// Get database driver
	driver := v.GetString("application.database.driver")
	if driver == "" {
		// Default to PostgreSQL
		return PostgreSQL, nil
	}

	// Validate driver
	switch driver {
	case "postgres", "postgresql":
		return PostgreSQL, nil
	case "mysql":
		return MySQL, nil
	case "sqlite", "sqlite3":
		return SQLite, nil
	default:
		return "", fmt.Errorf("unsupported database driver: %s (supported: postgres, mysql, sqlite)", driver)
	}
}
