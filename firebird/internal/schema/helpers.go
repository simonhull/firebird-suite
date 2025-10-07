package schema

import (
	"github.com/simonhull/firebird-suite/fledge/generator"
	fledgeschema "github.com/simonhull/firebird-suite/fledge/schema"
)

// DefaultTableName converts resource name to snake_case plural
// Example: "User" -> "users", "BlogPost" -> "blog_posts"
func DefaultTableName(name string) string {
	// Convert PascalCase to snake_case
	snake := generator.SnakeCase(name)

	// Pluralize
	return Pluralize(snake)
}

// Pluralize converts singular to plural
// This is exported so other packages can use it
func Pluralize(s string) string {
	return fledgeschema.Pluralize(s)
}