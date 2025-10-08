package conventions

import (
	"github.com/simonhull/firebird-suite/owl/pkg/analyzer"
)

// Detector detects architectural conventions in Go packages
type Detector struct {
	registry *Registry
}

// NewDetector creates a new detector with default patterns
func NewDetector() *Detector {
	registry := NewRegistry()
	RegisterDefaultPatterns(registry)
	return &Detector{
		registry: registry,
	}
}

// Detect analyzes a package and returns detected conventions
func (d *Detector) Detect(pkg *analyzer.Package) []*analyzer.Convention {
	conventions := make([]*analyzer.Convention, 0)

	// Apply pattern matching to types
	for _, t := range pkg.Types {
		for _, pattern := range d.registry.patterns {
			if pattern.MatchType != nil && pattern.MatchType(t) {
				conv := &analyzer.Convention{
					Name:       pattern.Name,
					Category:   pattern.Category,
					Layer:      "", // TODO: infer from category
					Confidence: pattern.Confidence,
					Reason:     pattern.Description,
					Tags:       pattern.Tags,
				}
				t.Convention = conv
				conventions = append(conventions, conv)
				break // Only match first pattern
			}
		}
	}

	// Apply pattern matching to functions
	for _, f := range pkg.Functions {
		for _, pattern := range d.registry.patterns {
			if pattern.MatchFunction != nil && pattern.MatchFunction(f) {
				conv := &analyzer.Convention{
					Name:       pattern.Name,
					Category:   pattern.Category,
					Layer:      "",
					Confidence: pattern.Confidence,
					Reason:     pattern.Description,
					Tags:       pattern.Tags,
				}
				f.Convention = conv
				conventions = append(conventions, conv)
				break // Only match first pattern
			}
		}
	}

	return conventions
}
