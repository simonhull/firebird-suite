package migration

import (
	"fmt"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

// Generator generates SQL migrations from schemas
type Generator struct {
	resolver *generator.Resolver
}

// NewGenerator creates a new migration generator with the given conflict resolver
func NewGenerator(resolver *generator.Resolver) *Generator {
	return &Generator{resolver: resolver}
}

// Generate generates a SQL migration for the given schema name
func (g *Generator) Generate(name string) error {
	return fmt.Errorf("migration generator not yet implemented")
}
