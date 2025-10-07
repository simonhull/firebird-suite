package schema

import (
	"strings"
	"unicode"

	fledgeschema "github.com/simonhull/firebird-suite/fledge/schema"
)

// DefaultTableName converts resource name to snake_case plural
// Example: "User" -> "users", "BlogPost" -> "blog_posts"
func DefaultTableName(name string) string {
	// Convert PascalCase to snake_case
	snake := pascalToSnake(name)

	// Pluralize
	return Pluralize(snake)
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

// Pluralize converts singular to plural
// This is exported so other packages can use it
func Pluralize(s string) string {
	return fledgeschema.Pluralize(s)
}