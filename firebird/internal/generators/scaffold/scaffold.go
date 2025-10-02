package scaffold

import "fmt"

// Generator generates empty schema scaffolds
type Generator struct{}

// NewGenerator creates a new scaffold generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates an empty schema file for the given name
func (g *Generator) Generate(name string) error {
	return fmt.Errorf("scaffold generator not yet implemented")
}
