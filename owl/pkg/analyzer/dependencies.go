package analyzer

import (
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// PackageDependencyGraph represents the import relationships between packages
type PackageDependencyGraph struct {
	Nodes  []*PackageNode
	Edges  []*DependencyEdge
	Cycles [][]string      // List of circular dependency chains
	Layers map[string]int  // Package path → layer depth
	Stats  DependencyStats
}

// PackageNode represents a single package in the graph
type PackageNode struct {
	Path           string // Full import path
	Name           string // Package name (last segment)
	IsInternal     bool   // Whether it's part of the project
	IsExternal     bool   // Whether it's a third-party package
	ImportCount    int    // Number of packages this imports
	DependentCount int    // Number of packages that import this
	Layer          int    // Inferred architectural layer (0 = top)
	LOC            int    // Lines of code (for internal packages)
	TypeCount      int    // Number of types (for internal packages)
}

// DependencyEdge represents an import relationship
type DependencyEdge struct {
	From       string // Importing package path
	To         string // Imported package path
	IsInternal bool   // Both packages are internal
	IsCycle    bool   // Part of a circular dependency
}

// DependencyStats provides summary metrics
type DependencyStats struct {
	TotalPackages    int
	InternalPackages int
	ExternalPackages int
	TotalImports     int
	InternalImports  int
	ExternalImports  int
	CycleCount       int
	MaxLayerDepth    int
	AvgDependents    float64
}

// AnalyzeDependencies builds the complete dependency graph
func AnalyzeDependencies(project *Project) (*PackageDependencyGraph, error) {
	graph := &PackageDependencyGraph{
		Nodes:  make([]*PackageNode, 0),
		Edges:  make([]*DependencyEdge, 0),
		Cycles: make([][]string, 0),
		Layers: make(map[string]int),
	}

	// Build node map for quick lookup
	nodeMap := make(map[string]*PackageNode)

	// Get module path for identifying internal vs external packages
	modulePath := project.Module

	// Create nodes for all internal packages
	for _, pkg := range project.Packages {
		node := &PackageNode{
			Path:       pkg.ImportPath,
			Name:       pkg.Name,
			IsInternal: true,
			IsExternal: false,
			LOC:        len(pkg.Types) + len(pkg.Functions), // Simple heuristic
			TypeCount:  len(pkg.Types),
		}
		nodeMap[pkg.ImportPath] = node
		graph.Nodes = append(graph.Nodes, node)
	}

	// Parse imports for each package
	for _, pkg := range project.Packages {
		imports, err := extractImports(pkg.Path)
		if err != nil {
			continue // Skip packages with parse errors
		}

		for _, imp := range imports {
			// Create edge
			edge := &DependencyEdge{
				From: pkg.ImportPath,
				To:   imp,
			}

			// Check if target is internal (same module)
			isInternal := strings.HasPrefix(imp, modulePath)

			if targetNode, exists := nodeMap[imp]; exists {
				edge.IsInternal = true
				targetNode.DependentCount++
			} else {
				// External or not-yet-seen dependency
				if _, exists := nodeMap[imp]; !exists {
					externalNode := &PackageNode{
						Path:       imp,
						Name:       getPackageName(imp),
						IsInternal: isInternal,
						IsExternal: !isInternal,
					}
					nodeMap[imp] = externalNode
					graph.Nodes = append(graph.Nodes, externalNode)
				}
				edge.IsInternal = isInternal
			}

			graph.Edges = append(graph.Edges, edge)

			if fromNode, exists := nodeMap[pkg.ImportPath]; exists {
				fromNode.ImportCount++
			}
		}
	}

	// Detect circular dependencies
	graph.Cycles = detectCycles(graph, nodeMap)

	// Mark cycle edges
	for _, cycle := range graph.Cycles {
		markCycleEdges(graph.Edges, cycle)
	}

	// Infer architectural layers
	inferLayers(graph, nodeMap)

	// Calculate statistics
	graph.Stats = calculateStats(graph)

	return graph, nil
}

// extractImports parses Go files in a package and extracts import paths
func extractImports(pkgPath string) ([]string, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgPath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	importSet := make(map[string]bool)
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, imp := range file.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				importSet[path] = true
			}
		}
	}

	imports := make([]string, 0, len(importSet))
	for imp := range importSet {
		imports = append(imports, imp)
	}
	sort.Strings(imports)

	return imports, nil
}

// detectCycles finds all circular dependencies using DFS
func detectCycles(graph *PackageDependencyGraph, nodeMap map[string]*PackageNode) [][]string {
	cycles := make([][]string, 0)
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	// Build adjacency list (internal packages only)
	adjList := make(map[string][]string)
	for _, edge := range graph.Edges {
		if edge.IsInternal {
			adjList[edge.From] = append(adjList[edge.From], edge.To)
		}
	}

	// DFS from each unvisited node
	var dfs func(node string, path []string) bool
	dfs = func(node string, path []string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range adjList[node] {
			if !visited[neighbor] {
				if dfs(neighbor, path) {
					return true
				}
			} else if recStack[neighbor] {
				// Found a cycle - extract it from path
				cycleStart := -1
				for i, p := range path {
					if p == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart != -1 {
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycles = append(cycles, cycle)
				}
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for node := range nodeMap {
		if !visited[node] && nodeMap[node].IsInternal {
			dfs(node, []string{})
		}
	}

	return cycles
}

// markCycleEdges marks edges that are part of circular dependencies
func markCycleEdges(edges []*DependencyEdge, cycle []string) {
	for i := 0; i < len(cycle); i++ {
		from := cycle[i]
		to := cycle[(i+1)%len(cycle)]

		for _, edge := range edges {
			if edge.From == from && edge.To == to {
				edge.IsCycle = true
			}
		}
	}
}

// inferLayers assigns layer depths using topological sorting
func inferLayers(graph *PackageDependencyGraph, nodeMap map[string]*PackageNode) {
	// Build adjacency list (internal only, excluding cycles)
	adjList := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, node := range graph.Nodes {
		if node.IsInternal {
			inDegree[node.Path] = 0
		}
	}

	for _, edge := range graph.Edges {
		if edge.IsInternal && !edge.IsCycle {
			adjList[edge.To] = append(adjList[edge.To], edge.From)
			inDegree[edge.From]++
		}
	}

	// Topological sort with layer assignment
	queue := make([]string, 0)
	for path, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, path)
			if node, ok := nodeMap[path]; ok {
				node.Layer = 0
				graph.Layers[path] = 0
			}
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		currentNode, ok := nodeMap[current]
		if !ok {
			continue
		}
		currentLayer := currentNode.Layer

		for _, neighbor := range adjList[current] {
			inDegree[neighbor]--

			// Update neighbor's layer to be one deeper than current
			if neighborNode, ok := nodeMap[neighbor]; ok {
				if neighborNode.Layer < currentLayer+1 {
					neighborNode.Layer = currentLayer + 1
					graph.Layers[neighbor] = currentLayer + 1
				}
			}

			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}
}

// calculateStats computes summary metrics for the graph
func calculateStats(graph *PackageDependencyGraph) DependencyStats {
	stats := DependencyStats{}

	for _, node := range graph.Nodes {
		stats.TotalPackages++
		if node.IsInternal {
			stats.InternalPackages++
			stats.AvgDependents += float64(node.DependentCount)
			if node.Layer > stats.MaxLayerDepth {
				stats.MaxLayerDepth = node.Layer
			}
		} else {
			stats.ExternalPackages++
		}
	}

	for _, edge := range graph.Edges {
		stats.TotalImports++
		if edge.IsInternal {
			stats.InternalImports++
		} else {
			stats.ExternalImports++
		}
	}

	stats.CycleCount = len(graph.Cycles)

	if stats.InternalPackages > 0 {
		stats.AvgDependents = stats.AvgDependents / float64(stats.InternalPackages)
	}

	return stats
}

// getPackageName extracts the last segment of an import path
func getPackageName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
