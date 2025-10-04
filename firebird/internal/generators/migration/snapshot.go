package migration

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"gopkg.in/yaml.v3"
)

// extractLastSnapshot finds the most recent migration for a model and extracts its schema snapshot
// Returns nil if no previous migration exists (first migration case)
func extractLastSnapshot(migrationsDir, modelName string) (*schema.Definition, error) {
	// Find the most recent migration file for this model
	migrationFile, err := findLastMigration(migrationsDir, modelName)
	if err != nil {
		return nil, err
	}
	if migrationFile == "" {
		// No previous migration exists - this is the first migration
		return nil, nil
	}

	// Open and read the migration file
	file, err := os.Open(migrationFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open migration file %s: %w", migrationFile, err)
	}
	defer file.Close()

	// Extract YAML from SQL comments
	var yamlLines []string
	inSnapshot := false
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Check for snapshot markers
		if strings.Contains(line, "-- FIREBIRD_SCHEMA_SNAPSHOT_BEGIN") {
			inSnapshot = true
			continue
		}
		if strings.Contains(line, "-- FIREBIRD_SCHEMA_SNAPSHOT_END") {
			break
		}

		// Extract YAML line from SQL comment
		if inSnapshot && strings.HasPrefix(line, "-- ") {
			yamlLine := strings.TrimPrefix(line, "-- ")
			yamlLines = append(yamlLines, yamlLine)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	// No snapshot found
	if len(yamlLines) == 0 {
		return nil, fmt.Errorf("no schema snapshot found in %s (migration may be from older Firebird version)", migrationFile)
	}

	// Parse YAML into schema definition
	yamlContent := strings.Join(yamlLines, "\n")
	var def schema.Definition
	if err := yaml.Unmarshal([]byte(yamlContent), &def); err != nil {
		return nil, fmt.Errorf("failed to parse schema snapshot: %w", err)
	}

	return &def, nil
}

// findLastMigration finds the most recent migration file for a given model
// Returns empty string if no migration exists
func findLastMigration(migrationsDir, modelName string) (string, error) {
	// Check if migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return "", nil // No migrations directory, no previous migration
	}

	// Read all migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return "", fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Build pattern for this model's migration files
	// Pattern: {number}_{create|alter}_{pluralized_model_name}.up.sql
	// We want to find the most recent one (highest number)
	tableName := strings.ToLower(pluralize(modelName))

	// Match both create and alter migrations
	pattern := regexp.MustCompile(`^(\d+)_(create|alter)_` + regexp.QuoteMeta(tableName) + `\.up\.sql$`)

	var latestFile string
	var latestNumber string

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		matches := pattern.FindStringSubmatch(file.Name())
		if len(matches) > 1 {
			number := matches[1]
			// Compare as strings (works for timestamp-based numbering)
			if number > latestNumber {
				latestNumber = number
				latestFile = filepath.Join(migrationsDir, file.Name())
			}
		}
	}

	return latestFile, nil
}

// pluralize converts a singular model name to plural table name
// This is a simple implementation - matches the logic in other generators
func pluralize(s string) string {
	s = strings.ToLower(s)

	// Handle special cases
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
	   strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") ||
	   strings.HasSuffix(s, "sh") {
		return s + "es"
	}

	if strings.HasSuffix(s, "y") && len(s) > 1 {
		// Check if preceded by consonant
		prevChar := s[len(s)-2]
		if prevChar != 'a' && prevChar != 'e' && prevChar != 'i' &&
		   prevChar != 'o' && prevChar != 'u' {
			return s[:len(s)-1] + "ies"
		}
	}

	return s + "s"
}
