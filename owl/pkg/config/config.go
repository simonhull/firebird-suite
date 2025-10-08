package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents owldocs.yaml configuration
type Config struct {
	Project     ProjectConfig    `yaml:"project"`
	Framework   string           `yaml:"framework"`
	Conventions ConventionConfig `yaml:"conventions"`
	Structure   StructureConfig  `yaml:"structure"`
	Output      OutputConfig     `yaml:"output"`
	Features    FeatureFlags     `yaml:"features"`
	Server      ServerConfig     `yaml:"server"`
}

// ProjectConfig holds project metadata
type ProjectConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// ConventionConfig holds convention detection settings
type ConventionConfig struct {
	Enabled bool               `yaml:"enabled"`
	Builtin []string           `yaml:"builtin"`
	Custom  []CustomConvention `yaml:"custom"`
}

// CustomConvention represents a user-defined convention pattern
type CustomConvention struct {
	Name     string `yaml:"name"`
	Pattern  string `yaml:"pattern"`
	Category string `yaml:"category"`
	Layer    string `yaml:"layer"`
}

// StructureConfig defines documentation structure
type StructureConfig struct {
	Guides       string `yaml:"guides"`
	Architecture string `yaml:"architecture"`
	Examples     string `yaml:"examples"`
}

// OutputConfig defines output settings
type OutputConfig struct {
	Path  string `yaml:"path"`
	Theme string `yaml:"theme"`
}

// FeatureFlags controls optional features
type FeatureFlags struct {
	Search       bool `yaml:"search"`
	Diagrams     bool `yaml:"diagrams"`
	Examples     bool `yaml:"examples"`
	Dependencies bool `yaml:"dependencies"`
	DarkMode     bool `yaml:"dark_mode"`
}

// ServerConfig holds dev server settings
type ServerConfig struct {
	Port       int  `yaml:"port"`
	Watch      bool `yaml:"watch"`
	LiveReload bool `yaml:"livereload"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// Return default config if file doesn't exist
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return &cfg, nil
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Framework: "auto",
		Conventions: ConventionConfig{
			Enabled: true,
			Builtin: []string{
				"handlers",
				"services",
				"repositories",
				"middleware",
			},
		},
		Structure: StructureConfig{
			Guides:       "docs/guides",
			Architecture: "docs/architecture",
			Examples:     "docs/examples",
		},
		Output: OutputConfig{
			Path:  "./docs-site",
			Theme: "default",
		},
		Features: FeatureFlags{
			Search:       true,
			Diagrams:     true,
			Examples:     true,
			Dependencies: true,
			DarkMode:     true,
		},
		Server: ServerConfig{
			Port:       6060,
			Watch:      true,
			LiveReload: true,
		},
	}
}

// SaveConfig writes configuration to a YAML file
func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
