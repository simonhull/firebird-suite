package schema

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Definition is the generic structure for all Aerie schemas
type Definition struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Name       string                 `yaml:"name"`
	Metadata   map[string]interface{} `yaml:"metadata,omitempty"`
	Spec       map[string]interface{} `yaml:"spec"` // Generic - tools define their own spec structure
}

// Parse reads and parses a YAML schema file
func Parse(path string) (*Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	return ParseBytes(data)
}

// ParseBytes parses schema from bytes
func ParseBytes(data []byte) (*Definition, error) {
	var def Definition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	return &def, nil
}

// Write writes a schema to a file
func Write(path string, def *Definition) error {
	data, err := yaml.Marshal(def)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// WriteBytes marshals a schema to bytes
func WriteBytes(def *Definition) ([]byte, error) {
	data, err := yaml.Marshal(def)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}
	return data, nil
}