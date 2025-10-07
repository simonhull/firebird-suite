package migration

import (
	"fmt"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
)

// Node represents a resource in the dependency graph
type Node struct {
	ResourceName string
	TableName    string
	DependsOn    []string // List of table names this resource depends on
}

// DependencyGraph tracks dependencies between resources
type DependencyGraph struct {
	nodes map[string]*Node
	edges map[string][]string // resource -> list of dependency resource names
}

// NewDependencyGraph creates a new empty graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*Node),
		edges: make(map[string][]string),
	}
}

// BuildDependencyGraph analyzes resources and builds a dependency graph
func BuildDependencyGraph(resources []*schema.Definition) (*DependencyGraph, error) {
	graph := NewDependencyGraph()

	// First pass: Create all nodes
	for _, resource := range resources {
		node := &Node{
			ResourceName: resource.Name,
			TableName:    resource.Spec.TableName,
			DependsOn:    []string{},
		}
		graph.nodes[resource.Name] = node
	}

	// Second pass: Identify FK dependencies
	for _, resource := range resources {
		node := graph.nodes[resource.Name]

		// Look for FK tags in fields
		for _, field := range resource.Spec.Fields {
			if fk := field.Tags["fk"]; fk != "" {
				// Parse "posts.id" or "users.id"
				parts := strings.Split(fk, ".")
				if len(parts) != 2 {
					continue
				}
				targetTable := parts[0]

				// Find resource with this table name
				targetResource := findResourceByTableName(resources, targetTable)
				if targetResource != nil {
					// Avoid duplicate dependencies
					if !contains(node.DependsOn, targetResource.Name) {
						node.DependsOn = append(node.DependsOn, targetResource.Name)
					}
				}
			}
		}

		graph.edges[resource.Name] = node.DependsOn
	}

	return graph, nil
}

// findResourceByTableName finds a resource by its table name
func findResourceByTableName(resources []*schema.Definition, tableName string) *schema.Definition {
	for _, r := range resources {
		if strings.EqualFold(r.Spec.TableName, tableName) {
			return r
		}
	}
	return nil
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// TopologicalSort sorts resources by dependencies using depth-first search
// Returns resources in order where dependencies come before dependents
func (g *DependencyGraph) TopologicalSort() ([]string, error) {
	var sorted []string
	visited := make(map[string]bool)
	visiting := make(map[string]bool) // For cycle detection

	var visit func(name string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("circular dependency detected involving resource '%s'", name)
		}
		if visited[name] {
			return nil
		}

		visiting[name] = true

		// Visit all dependencies first
		for _, dep := range g.edges[name] {
			if err := visit(dep); err != nil {
				return err
			}
		}

		visiting[name] = false
		visited[name] = true
		sorted = append(sorted, name) // Append (dependencies come first)

		return nil
	}

	// Visit all nodes
	for name := range g.nodes {
		if !visited[name] {
			if err := visit(name); err != nil {
				return nil, err
			}
		}
	}

	return sorted, nil
}

// GetDependencies returns the direct dependencies for a resource
func (g *DependencyGraph) GetDependencies(resourceName string) []string {
	if deps, ok := g.edges[resourceName]; ok {
		return deps
	}
	return []string{}
}

// HasCircularDependency checks if the graph contains any circular dependencies
func (g *DependencyGraph) HasCircularDependency() (bool, string) {
	_, err := g.TopologicalSort()
	if err != nil {
		return true, err.Error()
	}
	return false, ""
}

// GetNode returns the node for a given resource name
func (g *DependencyGraph) GetNode(resourceName string) *Node {
	return g.nodes[resourceName]
}
