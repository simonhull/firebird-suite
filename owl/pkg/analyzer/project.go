package analyzer

// Project represents a complete analyzed Go project
type Project struct {
	Name        string
	Module      string
	Version     string
	Description string
	Packages    []*Package
	Graph       *DependencyGraph
}

// DependencyGraph represents package and type dependencies
type DependencyGraph struct {
	Nodes []*GraphNode
	Edges []*GraphEdge
}

// GraphNode represents a package or type in the dependency graph
type GraphNode struct {
	ID       string
	Type     string // "package", "type", "function"
	Name     string
	Package  string
	FilePath string
}

// GraphEdge represents a dependency relationship
type GraphEdge struct {
	From     string
	To       string
	Type     string // "imports", "uses", "calls"
	Strength int    // Weight of the dependency
}
