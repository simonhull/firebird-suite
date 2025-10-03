package migration

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// NumberingStrategy determines how migrations are numbered
type NumberingStrategy string

const (
	TimestampNumbering  NumberingStrategy = "timestamp"
	SequentialNumbering NumberingStrategy = "sequential"
)

// GenerateMigrationNumber creates a new migration number
func GenerateMigrationNumber(strategy NumberingStrategy, migrationsDir string) (string, error) {
	switch strategy {
	case TimestampNumbering:
		return generateTimestamp(), nil
	case SequentialNumbering:
		return generateSequential(migrationsDir)
	default:
		return "", fmt.Errorf("unknown numbering strategy: %s", strategy)
	}
}

// generateTimestamp creates a timestamp-based migration number
func generateTimestamp() string {
	return time.Now().UTC().Format("20060102150405")
}

// generateSequential finds the highest sequential number and increments
func generateSequential(migrationsDir string) (string, error) {
	// Find all migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No migrations directory yet, start at 1
			return "000001", nil
		}
		return "", fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Extract numbers from filenames
	sequentialPattern := regexp.MustCompile(`^(\d{6})_`)
	maxNumber := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		matches := sequentialPattern.FindStringSubmatch(file.Name())
		if len(matches) > 1 {
			num, err := strconv.Atoi(matches[1])
			if err == nil && num > maxNumber {
				maxNumber = num
			}
		}
	}

	// Increment and format
	nextNumber := maxNumber + 1
	return fmt.Sprintf("%06d", nextNumber), nil
}

// MigrationExists checks if a migration with this name already exists
func MigrationExists(migrationsDir, name string) (bool, error) {
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	// Check if any file contains this migration name
	for _, file := range files {
		if strings.Contains(file.Name(), "_"+name+".") {
			return true, nil
		}
	}

	return false, nil
}

// GetMigrationFilenames returns the up and down filenames
func GetMigrationFilenames(number, name string) (string, string) {
	baseName := fmt.Sprintf("%s_%s", number, name)
	return baseName + ".up.sql", baseName + ".down.sql"
}
