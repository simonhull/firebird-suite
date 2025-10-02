package schema

import (
	"fmt"
	"strings"
)

// validGoTypes contains all valid Go types for schema fields
var validGoTypes = []string{
	"string", "*string",
	"int", "int8", "int16", "int32", "int64",
	"*int", "*int8", "*int16", "*int32", "*int64",
	"uint", "uint8", "uint16", "uint32", "uint64",
	"*uint", "*uint8", "*uint16", "*uint32", "*uint64",
	"float32", "float64", "*float32", "*float64",
	"bool", "*bool",
	"time.Time", "*time.Time",
	"uuid.UUID", "*uuid.UUID",
	"[]byte",
}

// IsValidGoType checks if a type string is a valid Go type
func IsValidGoType(typeStr string) bool {
	for _, validType := range validGoTypes {
		if typeStr == validType {
			return true
		}
	}
	return false
}

// IsPointerType checks if a type is a pointer (starts with *)
func IsPointerType(typeStr string) bool {
	return strings.HasPrefix(typeStr, "*")
}

// ValidateValidationRules performs basic validation of validator tags
// Just check for obviously invalid syntax, don't validate semantics
func ValidateValidationRules(rules []string) error {
	for _, rule := range rules {
		if err := validateSingleRule(rule); err != nil {
			return err
		}
	}
	return nil
}

// validateSingleRule checks a single validation rule for basic syntax errors
func validateSingleRule(rule string) error {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return fmt.Errorf("validation rule cannot be empty")
	}

	// Check for common validator syntax patterns
	// These are common go-playground/validator tags
	knownPrefixes := []string{
		"required",
		"min=", "max=",
		"len=",
		"eq=", "ne=", "gt=", "gte=", "lt=", "lte=",
		"email", "url", "uri",
		"alpha", "alphanum", "numeric",
		"uuid", "uuid3", "uuid4", "uuid5",
		"ascii", "printascii",
		"lowercase", "uppercase",
		"contains=", "containsany=", "containsrune=",
		"excludes=", "excludesall=", "excludesrune=",
		"startswith=", "endswith=",
		"ip", "ipv4", "ipv6",
		"cidr", "cidrv4", "cidrv6",
		"mac",
		"hostname", "fqdn",
		"unique",
		"oneof=",
		"json",
		"base64",
	}

	// Check if the rule starts with a known prefix or is a known standalone rule
	isKnown := false
	for _, prefix := range knownPrefixes {
		if strings.HasPrefix(rule, prefix) || rule == prefix {
			isKnown = true
			break
		}
	}

	// Also allow custom validators (they usually start with a letter)
	if !isKnown && isValidCustomValidator(rule) {
		isKnown = true
	}

	// Basic syntax checks
	if strings.Contains(rule, " ") && !strings.Contains(rule, "=") {
		return fmt.Errorf("validation rule '%s' contains spaces but no '=' operator", rule)
	}

	// Check for unbalanced quotes
	if strings.Count(rule, "'")%2 != 0 {
		return fmt.Errorf("validation rule '%s' has unbalanced single quotes", rule)
	}
	if strings.Count(rule, "\"")%2 != 0 {
		return fmt.Errorf("validation rule '%s' has unbalanced double quotes", rule)
	}

	// Check for obviously malformed rules
	if strings.HasPrefix(rule, "=") || strings.HasSuffix(rule, "=") {
		return fmt.Errorf("validation rule '%s' has misplaced '=' operator", rule)
	}

	// Check for multiple consecutive equals
	if strings.Contains(rule, "==") {
		return fmt.Errorf("validation rule '%s' has invalid '==' operator (use single '=')", rule)
	}

	return nil
}

// isValidCustomValidator checks if a string could be a custom validator name
func isValidCustomValidator(s string) bool {
	if s == "" {
		return false
	}

	// Custom validators typically start with a letter and contain only letters, numbers, and underscores
	for i, c := range s {
		if i == 0 {
			// First character should be a letter
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
				return false
			}
		} else {
			// Subsequent characters can be letters, numbers, or underscores
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
				// Could be a validator with parameters (e.g., "customvalidator=param")
				if c == '=' {
					return i > 0 // Must have something before the equals
				}
				return false
			}
		}
	}
	return true
}