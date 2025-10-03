package migrate

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// DatabaseConfig holds database connection information
type DatabaseConfig struct {
	Driver   string
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

// LoadDatabaseConfig reads database config from firebird.yml
func LoadDatabaseConfig() (*DatabaseConfig, error) {
	// Check if firebird.yml exists
	if _, err := os.Stat("firebird.yml"); os.IsNotExist(err) {
		return nil, fmt.Errorf("firebird.yml not found. Are you in a Firebird project directory?")
	}

	// Load config
	v := viper.New()
	v.SetConfigName("firebird")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	// Enable environment variable overrides
	v.AutomaticEnv()
	v.SetEnvPrefix("APPLICATION_DATABASE")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read firebird.yml: %w", err)
	}

	// Extract database config
	cfg := &DatabaseConfig{
		Driver:   v.GetString("application.database.driver"),
		Host:     v.GetString("application.database.host"),
		Port:     v.GetInt("application.database.port"),
		Name:     v.GetString("application.database.name"),
		User:     v.GetString("application.database.user"),
		Password: v.GetString("application.database.password"),
		SSLMode:  v.GetString("application.database.sslmode"),
	}

	// Validate required fields
	if cfg.Driver == "" {
		return nil, fmt.Errorf("database driver not specified in firebird.yml")
	}
	if cfg.Name == "" {
		return nil, fmt.Errorf("database name not specified in firebird.yml")
	}

	return cfg, nil
}
