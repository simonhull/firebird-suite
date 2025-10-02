package schema

import (
	"strings"
	"unicode"
)

// DefaultTableName converts resource name to snake_case plural
// Example: "User" -> "users", "BlogPost" -> "blog_posts"
func DefaultTableName(name string) string {
	// Convert PascalCase to snake_case
	snake := pascalToSnake(name)

	// Pluralize
	return pluralize(snake)
}

// pascalToSnake converts PascalCase to snake_case
func pascalToSnake(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	for i, r := range s {
		// If current rune is uppercase
		if unicode.IsUpper(r) {
			// Add underscore before uppercase letter if:
			// - Not the first character
			// - Previous character is lowercase OR
			// - Previous character is uppercase but next is lowercase (handling acronyms like "HTTPServer" -> "http_server")
			if i > 0 {
				prev := rune(s[i-1])
				if unicode.IsLower(prev) {
					result.WriteRune('_')
				} else if i+1 < len(s) && unicode.IsLower(rune(s[i+1])) {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// pluralize adds simple pluralization
// This is a basic implementation - could be enhanced with more rules
func pluralize(s string) string {
	if s == "" {
		return ""
	}

	// Handle some common irregular plurals
	irregulars := map[string]string{
		"person": "people",
		"child":  "children",
		"mouse":  "mice",
		"foot":   "feet",
		"tooth":  "teeth",
		"goose":  "geese",
		"man":    "men",
		"woman":  "women",
	}

	if plural, ok := irregulars[strings.ToLower(s)]; ok {
		return plural
	}

	// Handle words ending in 's', 'x', 'z', 'ch', 'sh' -> add 'es'
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") ||
		strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "sh") {
		return s + "es"
	}

	// Handle words ending in consonant + 'y' -> change 'y' to 'ies'
	if len(s) > 1 && strings.HasSuffix(s, "y") {
		prevChar := s[len(s)-2]
		if !isVowel(prevChar) {
			return s[:len(s)-1] + "ies"
		}
	}

	// Handle words ending in 'f' or 'fe' -> change to 'ves'
	if strings.HasSuffix(s, "f") {
		return s[:len(s)-1] + "ves"
	}
	if strings.HasSuffix(s, "fe") {
		return s[:len(s)-2] + "ves"
	}

	// Handle words ending in consonant + 'o' -> add 'es'
	if len(s) > 1 && strings.HasSuffix(s, "o") {
		prevChar := s[len(s)-2]
		if !isVowel(prevChar) {
			// Common exceptions that just add 's'
			exceptions := []string{"photo", "piano", "halo"}
			for _, exc := range exceptions {
				if strings.HasSuffix(s, exc[len(exc)-2:]) {
					return s + "s"
				}
			}
			return s + "es"
		}
	}

	// Default: just add 's'
	return s + "s"
}

// isVowel checks if a byte represents a vowel
func isVowel(c byte) bool {
	vowels := "aeiouAEIOU"
	return strings.ContainsRune(vowels, rune(c))
}