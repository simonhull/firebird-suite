package conventions

// Registry holds all registered architectural patterns
type Registry struct {
	patterns []*Pattern
}

// NewRegistry creates a new pattern registry
func NewRegistry() *Registry {
	return &Registry{
		patterns: make([]*Pattern, 0),
	}
}

// Register adds a pattern to the registry
func (r *Registry) Register(pattern *Pattern) {
	r.patterns = append(r.patterns, pattern)
}

// Find returns a pattern by name
func (r *Registry) Find(name string) *Pattern {
	for _, p := range r.patterns {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// FindByID returns a pattern by ID
func (r *Registry) FindByID(id string) *Pattern {
	for _, p := range r.patterns {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// All returns all registered patterns
func (r *Registry) All() []*Pattern {
	return r.patterns
}
