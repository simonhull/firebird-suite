package conventions

// Registry manages convention patterns
type Registry struct {
	patterns []Pattern
}

// NewRegistry creates a new Registry with default patterns
func NewRegistry() *Registry {
	r := &Registry{
		patterns: make([]Pattern, 0),
	}

	// Register default patterns
	for _, pattern := range DefaultPatterns() {
		r.Register(pattern)
	}

	return r
}

// Register adds a pattern to the registry
func (r *Registry) Register(pattern Pattern) {
	r.patterns = append(r.patterns, pattern)
}

// Patterns returns all registered patterns
func (r *Registry) Patterns() []Pattern {
	return r.patterns
}

// Find returns patterns matching the given predicate
func (r *Registry) Find(predicate func(Pattern) bool) []Pattern {
	var matches []Pattern
	for _, pattern := range r.patterns {
		if predicate(pattern) {
			matches = append(matches, pattern)
		}
	}
	return matches
}
