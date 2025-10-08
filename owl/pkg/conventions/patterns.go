package conventions

import (
	"strings"

	"github.com/simonhull/firebird-suite/owldocs/pkg/analyzer"
)

// Pattern represents a convention pattern
type Pattern struct {
	Name     string
	Category string
	Layer    string
	Matcher  func(t *analyzer.Type) bool
}

// DefaultPatterns returns the built-in convention patterns
func DefaultPatterns() []Pattern {
	return []Pattern{
		{
			Name:     "HTTP Handler",
			Category: "handlers",
			Layer:    "presentation",
			Matcher: func(t *analyzer.Type) bool {
				return strings.HasSuffix(t.Name, "Handler") ||
					strings.HasPrefix(t.Name, "Handle")
			},
		},
		{
			Name:     "Service",
			Category: "services",
			Layer:    "business",
			Matcher: func(t *analyzer.Type) bool {
				return strings.HasSuffix(t.Name, "Service")
			},
		},
		{
			Name:     "Repository",
			Category: "repositories",
			Layer:    "data",
			Matcher: func(t *analyzer.Type) bool {
				return strings.HasSuffix(t.Name, "Repository") ||
					strings.HasSuffix(t.Name, "Store")
			},
		},
		{
			Name:     "Middleware",
			Category: "middleware",
			Layer:    "presentation",
			Matcher: func(t *analyzer.Type) bool {
				return strings.HasSuffix(t.Name, "Middleware") ||
					strings.Contains(strings.ToLower(t.Name), "middleware")
			},
		},
		{
			Name:     "DTO",
			Category: "dto",
			Layer:    "presentation",
			Matcher: func(t *analyzer.Type) bool {
				return strings.HasSuffix(t.Name, "Request") ||
					strings.HasSuffix(t.Name, "Response") ||
					strings.HasSuffix(t.Name, "DTO")
			},
		},
		{
			Name:     "Validator",
			Category: "validation",
			Layer:    "business",
			Matcher: func(t *analyzer.Type) bool {
				return strings.HasSuffix(t.Name, "Validator")
			},
		},
	}
}
