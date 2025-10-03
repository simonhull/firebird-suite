package migrate

import (
	"fmt"
	"net/url"
)

// BuildConnectionString creates a golang-migrate compatible connection string
func BuildConnectionString(cfg *DatabaseConfig) (string, error) {
	switch cfg.Driver {
	case "postgres", "postgresql":
		return buildPostgreSQLConnectionString(cfg), nil
	case "mysql":
		return buildMySQLConnectionString(cfg), nil
	case "sqlite", "sqlite3":
		return buildSQLiteConnectionString(cfg), nil
	default:
		return "", fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
}

// buildPostgreSQLConnectionString builds a PostgreSQL connection string
func buildPostgreSQLConnectionString(cfg *DatabaseConfig) string {
	// Format: postgres://user:password@host:port/dbname?sslmode=disable
	sslmode := cfg.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   cfg.Name,
	}

	query := url.Values{}
	query.Set("sslmode", sslmode)
	u.RawQuery = query.Encode()

	return u.String()
}

// buildMySQLConnectionString builds a MySQL connection string
func buildMySQLConnectionString(cfg *DatabaseConfig) string {
	// Format: mysql://user:password@tcp(host:port)/dbname
	return fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
}

// buildSQLiteConnectionString builds a SQLite connection string
func buildSQLiteConnectionString(cfg *DatabaseConfig) string {
	// Format: sqlite3://path/to/database.db
	return fmt.Sprintf("sqlite3://%s", cfg.Name)
}
