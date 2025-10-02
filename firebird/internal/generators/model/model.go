package model

import (
	"fmt"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

// Generator generates Go model structs from schemas
type Generator struct {
	resolver *generator.Resolver
}

// NewGenerator creates a new model generator with the given conflict resolver
func NewGenerator(resolver *generator.Resolver) *Generator {
	return &Generator{resolver: resolver}
}

// Generate generates a Go model struct for the given schema name
func (g *Generator) Generate(name string) error {
	return fmt.Errorf("model generator not yet implemented")
}
