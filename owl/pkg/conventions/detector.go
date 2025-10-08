package conventions

import (
	"github.com/simonhull/firebird-suite/owldocs/pkg/analyzer"
)

// Detector implements convention detection
type Detector struct {
	registry *Registry
}

// NewDetector creates a new Detector with default patterns
func NewDetector() *Detector {
	return &Detector{
		registry: NewRegistry(),
	}
}

// Detect identifies conventions in a package
func (d *Detector) Detect(pkg *analyzer.Package) []*analyzer.Convention {
	var conventions []*analyzer.Convention

	// TODO: Implement convention detection logic
	// This will match types/functions against registered patterns

	return conventions
}

// RegisterPattern adds a custom pattern to the detector
func (d *Detector) RegisterPattern(pattern Pattern) {
	d.registry.Register(pattern)
}
