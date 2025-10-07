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
	return GenerateMigrationNumberWithOffset(strategy, migrationsDir, time.Time{}, 0)
}

// GenerateMigrationNumberWithOffset creates a new migration number with optional base time and offset
// If baseTime is zero, uses time.Now(). Offset is added as seconds to the timestamp.
func GenerateMigrationNumberWithOffset(strategy NumberingStrategy, migrationsDir string, baseTime time.Time, offset int) (string, error) {
	switch strategy {
	case TimestampNumbering:
		return generateTimestampWithOffset(baseTime, offset), nil
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

// generateTimestampWithOffset creates a timestamp-based migration number with optional base time and offset
func generateTimestampWithOffset(baseTime time.Time, offset int) string {
	var t time.Time
	if baseTime.IsZero() {
		t = time.Now().UTC()
	} else {
		t = baseTime.UTC()
	}

	// Add offset seconds for sequential ordering
	if offset > 0 {
		t = t.Add(time.Duration(offset) * time.Second)
	}

	return t.Format("20060102150405")
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
