package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the Owl configuration
type Config struct {
	Project     ProjectConfig     `yaml:"project"`
	Conventions ConventionConfig  `yaml:"conventions"`
	Structure   StructureConfig   `yaml:"structure"`
	Output      OutputConfig      `yaml:"output"`
	Features    FeatureFlags      `yaml:"features"`
	Server      ServerConfig      `yaml:"server"`
}

// ProjectConfig contains project-level settings
type ProjectConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	RootPaths   []string `yaml:"root_paths"`
	Exclude     []string `yaml:"exclude"`
}

// ConventionConfig contains convention detection settings
type ConventionConfig struct {
	Enabled        bool     `yaml:"enabled"`
	CustomPatterns []string `yaml:"custom_patterns"`
	IgnorePatterns []string `yaml:"ignore_patterns"`
}

// StructureConfig defines how documentation is organized
type StructureConfig struct {
	GroupBy      string   `yaml:"group_by"` // "layer", "package", "type"
	Sections     []string `yaml:"sections"`
	ShowInternal bool     `yaml:"show_internal"`
}

// OutputConfig defines output settings
type OutputConfig struct {
	Path   string `yaml:"path"`
	Format string `yaml:"format"` // "html", "markdown", "json"
	Theme  string `yaml:"theme"`
}

// FeatureFlags enables/disables features
type FeatureFlags struct {
	DependencyGraph bool `yaml:"dependency_graph"`
	SearchIndex     bool `yaml:"search_index"`
	LiveReload      bool `yaml:"live_reload"`
}

// ServerConfig contains dev server settings
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(path string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Project: ProjectConfig{
			Name:        "My Project",
			Description: "Project documentation",
			RootPaths:   []string{"."},
			Exclude:     []string{"vendor", "testdata"},
		},
		Conventions: ConventionConfig{
			Enabled:        true,
			CustomPatterns: []string{},
			IgnorePatterns: []string{},
		},
		Structure: StructureConfig{
			GroupBy:      "layer",
			Sections:     []string{"overview", "api", "architecture"},
			ShowInternal: false,
		},
		Output: OutputConfig{
			Path:   "./docs",
			Format: "html",
			Theme:  "default",
		},
		Features: FeatureFlags{
			DependencyGraph: true,
			SearchIndex:     true,
			LiveReload:      true,
		},
		Server: ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
	}
}
