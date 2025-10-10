package generator

import (
	"fmt"
	"strings"

	"github.com/simonhull/firebird-suite/owl/pkg/analyzer"
)

// CallGraphNode represents a node in the call graph
type CallGraphNode struct {
	Name       string
	ID         string // Sanitized ID for SVG
	X, Y       int    // Position in graph
	Width      int
	Height     int
	Complexity string // "simple", "medium", "complex"
	IsExternal bool   // If it's an external function call
}

// CallGraphEdge represents an edge (call relationship)
type CallGraphEdge struct {
	From string // Node ID
	To   string // Node ID
}

// CallGraph represents the complete graph
type CallGraph struct {
	Nodes  []*CallGraphNode
	Edges  []*CallGraphEdge
	Width  int
	Height int
}

// GenerateCallGraph creates a call graph for a package
func (g *Generator) GenerateCallGraph(pkg *analyzer.Package) *CallGraph {
	if len(pkg.Functions) == 0 {
		return nil
	}

	// Build node map
	nodeMap := make(map[string]*CallGraphNode)
	functionMap := make(map[string]*analyzer.Function)

	for _, fn := range pkg.Functions {
		functionMap[fn.Name] = fn

		complexity := calculateFunctionComplexity(fn)

		node := &CallGraphNode{
			Name:       fn.Name,
			ID:         sanitizeID(fn.Name),
			Width:      150,
			Height:     40,
			Complexity: complexity,
			IsExternal: false,
		}
		nodeMap[fn.Name] = node
	}

	// Build edges and identify external calls
	edges := []*CallGraphEdge{}
	externalNodes := make(map[string]*CallGraphNode)

	for _, fn := range pkg.Functions {
		for _, call := range fn.Calls {
			// Check if it's an internal function
			if _, exists := functionMap[call]; exists {
				edges = append(edges, &CallGraphEdge{
					From: sanitizeID(fn.Name),
					To:   sanitizeID(call),
				})
			} else {
				// External call - create external node if needed
				if _, exists := externalNodes[call]; !exists {
					externalNodes[call] = &CallGraphNode{
						Name:       call,
						ID:         sanitizeID(call),
						Width:      150,
						Height:     40,
						Complexity: "simple",
						IsExternal: true,
					}
				}
				edges = append(edges, &CallGraphEdge{
					From: sanitizeID(fn.Name),
					To:   sanitizeID(call),
				})
			}
		}
	}

	// Combine internal and external nodes
	nodes := []*CallGraphNode{}
	for _, node := range nodeMap {
		nodes = append(nodes, node)
	}
	for _, node := range externalNodes {
		nodes = append(nodes, node)
	}

	// Layout the graph (simple top-to-bottom approach)
	layoutGraph(nodes, edges)

	// Calculate total dimensions
	width, height := calculateGraphDimensions(nodes)

	return &CallGraph{
		Nodes:  nodes,
		Edges:  edges,
		Width:  width,
		Height: height,
	}
}

// layoutGraph positions nodes in the graph
func layoutGraph(nodes []*CallGraphNode, edges []*CallGraphEdge) {
	// Simple layout: arrange nodes in rows
	// More sophisticated layouts could use force-directed or hierarchical algorithms

	nodesPerRow := 3
	x, y := 50, 50
	spacing := 200
	rowHeight := 100

	for i, node := range nodes {
		node.X = x
		node.Y = y

		x += spacing

		// Move to next row after nodesPerRow nodes
		if (i+1)%nodesPerRow == 0 {
			x = 50
			y += rowHeight
		}
	}
}

// calculateGraphDimensions returns the total width and height needed
func calculateGraphDimensions(nodes []*CallGraphNode) (int, int) {
	if len(nodes) == 0 {
		return 400, 200
	}

	maxX, maxY := 0, 0
	for _, node := range nodes {
		if node.X+node.Width > maxX {
			maxX = node.X + node.Width
		}
		if node.Y+node.Height > maxY {
			maxY = node.Y + node.Height
		}
	}

	// Add padding
	return maxX + 50, maxY + 50
}

// sanitizeID creates a valid SVG ID from a function name
func sanitizeID(name string) string {
	// Replace special characters with underscores
	id := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)

	// Ensure it starts with a letter
	if len(id) > 0 && (id[0] >= '0' && id[0] <= '9') {
		id = "fn_" + id
	}

	return id
}

// RenderCallGraphSVG renders the call graph as SVG
func (g *CallGraph) RenderCallGraphSVG() string {
	if g == nil || len(g.Nodes) == 0 {
		return ""
	}

	var svg strings.Builder

	// SVG header
	svg.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg" class="call-graph">`,
		g.Width, g.Height, g.Width, g.Height))

	// Define arrow marker
	svg.WriteString(`
    <defs>
        <marker id="arrowhead" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
            <polygon points="0 0, 10 3, 0 6" fill="#64748b" />
        </marker>
    </defs>
    `)

	// Draw edges first (so they're behind nodes)
	for _, edge := range g.Edges {
		fromNode := g.findNode(edge.From)
		toNode := g.findNode(edge.To)

		if fromNode != nil && toNode != nil {
			// Calculate connection points (center-bottom to center-top)
			x1 := fromNode.X + fromNode.Width/2
			y1 := fromNode.Y + fromNode.Height
			x2 := toNode.X + toNode.Width/2
			y2 := toNode.Y

			svg.WriteString(fmt.Sprintf(
				`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#64748b" stroke-width="2" marker-end="url(#arrowhead)" class="graph-edge" />`,
				x1, y1, x2, y2))
		}
	}

	// Draw nodes
	for _, node := range g.Nodes {
		color := getComplexityColor(node.Complexity)
		if node.IsExternal {
			color = "#475569" // Darker gray for external
		}

		svg.WriteString(fmt.Sprintf(`
        <g class="graph-node" data-function="%s">
            <rect x="%d" y="%d" width="%d" height="%d" fill="%s" stroke="#cbd5e1" stroke-width="2" rx="6" class="node-rect" />
            <text x="%d" y="%d" fill="white" font-size="14" font-weight="600" text-anchor="middle" class="node-text">%s</text>
        </g>`,
			node.ID,
			node.X, node.Y, node.Width, node.Height,
			color,
			node.X+node.Width/2, node.Y+node.Height/2+5,
			truncateText(node.Name, 18)))
	}

	svg.WriteString("</svg>")

	return svg.String()
}

// findNode finds a node by ID
func (g *CallGraph) findNode(id string) *CallGraphNode {
	for _, node := range g.Nodes {
		if node.ID == id {
			return node
		}
	}
	return nil
}

// calculateFunctionComplexity determines complexity based on call count
func calculateFunctionComplexity(fn *analyzer.Function) string {
	callCount := len(fn.Calls)
	switch {
	case callCount <= 3:
		return "simple"
	case callCount <= 7:
		return "medium"
	default:
		return "complex"
	}
}

// getComplexityColor returns color for complexity level
func getComplexityColor(complexity string) string {
	switch complexity {
	case "simple":
		return "#10b981" // Green
	case "medium":
		return "#f59e0b" // Yellow
	case "complex":
		return "#ef4444" // Red
	default:
		return "#64748b" // Gray
	}
}

// truncateText truncates text to maxLen
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
