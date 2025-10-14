package project

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FirebirdConfig contains Firebird project configuration
type FirebirdConfig struct {
	ConfigPath string // Path to firebird.yml
	Database   string // Database driver (postgres, mysql, sqlite, none)
	Router     string // Router type (stdlib, chi, gin, echo, none)
}

// IsFirebirdProject checks if a directory contains firebird.yml
func IsFirebirdProject(rootPath string) bool {
	configPath := filepath.Join(rootPath, "firebird.yml")
	_, err := os.Stat(configPath)
	return err == nil
}

// DetectFirebirdProject checks for firebird.yml and parses basic config.
// Returns (found bool, config *FirebirdConfig, error).
func DetectFirebirdProject(rootPath string) (bool, *FirebirdConfig, error) {
	configPath := filepath.Join(rootPath, "firebird.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to read firebird.yml: %w", err)
	}

	var config struct {
		Application struct {
			Database struct {
				Driver string `yaml:"driver"`
			} `yaml:"database"`
			Router struct {
				Type string `yaml:"type"`
			} `yaml:"router"`
		} `yaml:"application"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return false, nil, fmt.Errorf("failed to parse firebird.yml: %w", err)
	}

	return true, &FirebirdConfig{
		ConfigPath: configPath,
		Database:   config.Application.Database.Driver,
		Router:     config.Application.Router.Type,
	}, nil
}

// LoadFirebirdConfig loads and fully parses firebird.yml.
// Use this when you need complete config details beyond basic detection.
func LoadFirebirdConfig(configPath string) (*FirebirdConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config struct {
		Application struct {
			Database struct {
				Driver string `yaml:"driver"`
			} `yaml:"database"`
			Router struct {
				Type string `yaml:"type"`
			} `yaml:"router"`
		} `yaml:"application"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &FirebirdConfig{
		ConfigPath: configPath,
		Database:   config.Application.Database.Driver,
		Router:     config.Application.Router.Type,
	}, nil
}
