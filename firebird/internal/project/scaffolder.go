package project

import "fmt"

// Scaffolder scaffolds new Firebird projects
type Scaffolder struct{}

// NewScaffolder creates a new project scaffolder
func NewScaffolder() *Scaffolder {
	return &Scaffolder{}
}

// Scaffold creates a new Firebird project with the given name
func (s *Scaffolder) Scaffold(projectName string) error {
	return fmt.Errorf("project scaffolder not yet implemented")
}
