package model

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindSchemaFile searches for a schema file by resource name
// Looks in: internal/schemas/<name>.firebird.yml, then ./<name>.firebird.yml
func FindSchemaFile(name string) (string, error) {
	// Try internal/schemas first (convention)
	primaryPath := filepath.Join("internal", "schemas", name+".firebird.yml")
	if _, err := os.Stat(primaryPath); err == nil {
		return primaryPath, nil
	}

	// Fallback to current directory
	fallbackPath := name + ".firebird.yml"
	if _, err := os.Stat(fallbackPath); err == nil {
		return fallbackPath, nil
	}

	// Not found
	return "", fmt.Errorf("schema file not found for '%s'. Expected in:\n  - %s\n  - %s",
		name, primaryPath, fallbackPath)
}
