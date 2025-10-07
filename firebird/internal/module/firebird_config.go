package module

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FirebirdConfig represents the firebird.yml structure
type FirebirdConfig struct {
	Project ProjectConfig           `yaml:"project"`
	Modules map[string]ModuleConfig `yaml:"modules,omitempty"`
}

// ProjectConfig holds project-level configuration
type ProjectConfig struct {
	Name   string `yaml:"name"`
	Module string `yaml:"module"`
}

// ModuleConfig holds per-module configuration
type ModuleConfig struct {
	Version string                 `yaml:"version"`
	Config  map[string]interface{} `yaml:"config,omitempty"`
}

// LoadFirebirdConfig loads firebird.yml from disk
func LoadFirebirdConfig(path string) (*FirebirdConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg FirebirdConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Initialize modules map if nil
	if cfg.Modules == nil {
		cfg.Modules = make(map[string]ModuleConfig)
	}

	return &cfg, nil
}

// SaveFirebirdConfig writes firebird.yml to disk
func SaveFirebirdConfig(path string, cfg *FirebirdConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// AddModule adds a module entry to firebird.yml
func AddModule(path, name, version string, config map[string]interface{}) error {
	cfg, err := LoadFirebirdConfig(path)
	if err != nil {
		return err
	}

	if cfg.Modules == nil {
		cfg.Modules = make(map[string]ModuleConfig)
	}

	cfg.Modules[name] = ModuleConfig{
		Version: version,
		Config:  config,
	}

	return SaveFirebirdConfig(path, cfg)
}

// RemoveModule removes a module entry from firebird.yml
func RemoveModule(path, name string) error {
	cfg, err := LoadFirebirdConfig(path)
	if err != nil {
		return err
	}

	delete(cfg.Modules, name)

	return SaveFirebirdConfig(path, cfg)
}
