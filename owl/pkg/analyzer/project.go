package analyzer

// Project represents an analyzed Go project
type Project struct {
	Name        string
	Module      string
	Version     string
	Description string
	Packages    []*Package
	Graph       *DependencyGraph

	// Firebird-specific (if detected)
	IsFirebirdProject bool
	FirebirdConfig    *FirebirdConfig
}

// FirebirdConfig holds Firebird project metadata
type FirebirdConfig struct {
	ConfigPath string
	Database   string
	Router     string
	Resources  []string
}

// DependencyGraph represents the dependency relationships in the project
type DependencyGraph struct {
	Nodes []*GraphNode
	Edges []*GraphEdge
}

// GraphNode represents a type or function in the dependency graph
type GraphNode struct {
	ID       string
	Type     string // "type" or "function"
	Name     string
	Package  string
	Layer    string // from convention detection
	FilePath string
}

// GraphEdge represents a dependency relationship
type GraphEdge struct {
	From       string // Node ID
	To         string // Node ID
	Type       string // "uses", "calls", "contains"
	Strength   int    // Frequency or importance
	SourceFile string
}
