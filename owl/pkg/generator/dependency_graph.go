package generator

import (
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/simonhull/firebird-suite/owl/pkg/analyzer"
)

//go:embed templates/dependency-base.html
var dependencyBaseTemplate string

//go:embed templates/dependency.html
var dependencyTemplate string

// GenerateDependencyGraph creates the package dependency visualization page
func (g *Generator) GenerateDependencyGraph(graph *analyzer.PackageDependencyGraph) error {
	depsDir := filepath.Join(g.outputDir, "dependencies")
	if err := os.MkdirAll(depsDir, 0755); err != nil {
		return fmt.Errorf("creating dependencies directory: %w", err)
	}

	// Generate SVG
	svg := g.renderDependencySVG(graph)

	// Create template with helper functions
	tmpl := template.New("dependency-base").Funcs(template.FuncMap{
		"len": func(v any) int {
			switch val := v.(type) {
			case [][]string:
				return len(val)
			default:
				return 0
			}
		},
		"iterate": func(n int) []int {
			result := make([]int, n+1)
			for i := range result {
				result[i] = i
			}
			return result
		},
	})

	// Parse templates
	tmpl, err := tmpl.Parse(dependencyBaseTemplate)
	if err != nil {
		return fmt.Errorf("parsing dependency base template: %w", err)
	}

	tmpl, err = tmpl.Parse(dependencyTemplate)
	if err != nil {
		return fmt.Errorf("parsing dependency template: %w", err)
	}

	data := struct {
		Title string
		Graph *analyzer.PackageDependencyGraph
		SVG   template.HTML
	}{
		Title: "Package Dependencies",
		Graph: graph,
		SVG:   template.HTML(svg),
	}

	// Create output file
	outputPath := filepath.Join(depsDir, "packages.html")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating dependency file: %w", err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	fmt.Printf("   ✓ Generated dependency graph page\n")
	return nil
}

// NodePosition represents x,y coordinates for a node
type NodePosition struct {
	X float64
	Y float64
}

// renderDependencySVG generates the SVG visualization
func (g *Generator) renderDependencySVG(graph *analyzer.PackageDependencyGraph) string {
	const (
		nodeWidth    = 140.0
		nodeHeight   = 60.0
		layerSpacing = 200.0
		nodeSpacing  = 30.0
		svgPadding   = 50.0
	)

	// Calculate node positions using hierarchical layout
	positions := g.calculateHierarchicalLayout(graph, layerSpacing, nodeSpacing, nodeWidth, nodeHeight)

	// Calculate SVG dimensions
	var maxX, maxY float64
	for _, pos := range positions {
		if pos.X > maxX {
			maxX = pos.X
		}
		if pos.Y > maxY {
			maxY = pos.Y
		}
	}

	width := maxX + nodeWidth + 2*svgPadding
	height := maxY + nodeHeight + 2*svgPadding

	// Start building SVG
	var svg strings.Builder
	svg.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" class="dependency-graph">`, width, height))
	svg.WriteString(`<defs>`)
	svg.WriteString(`<marker id="arrowhead" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">`)
	svg.WriteString(`<polygon points="0 0, 10 3, 0 6" fill="#94a3b8" />`)
	svg.WriteString(`</marker>`)
	svg.WriteString(`<marker id="arrowhead-cycle" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">`)
	svg.WriteString(`<polygon points="0 0, 10 3, 0 6" fill="#ef4444" />`)
	svg.WriteString(`</marker>`)
	svg.WriteString(`</defs>`)

	// Draw edges first (so they appear behind nodes)
	for _, edge := range graph.Edges {
		fromPos, fromExists := positions[edge.From]
		toPos, toExists := positions[edge.To]

		if !fromExists || !toExists {
			continue
		}

		// Calculate edge start/end points (center of nodes)
		x1 := fromPos.X + nodeWidth/2
		y1 := fromPos.Y + nodeHeight/2
		x2 := toPos.X + nodeWidth/2
		y2 := toPos.Y + nodeHeight/2

		// Edge styling
		class := "edge"
		marker := "arrowhead"
		if edge.IsCycle {
			class += " edge-cycle"
			marker = "arrowhead-cycle"
		} else if !edge.IsInternal {
			class += " edge-external"
		}

		svg.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" class="%s" marker-end="url(#%s)" data-from="%s" data-to="%s"/>`,
			x1, y1, x2, y2, class, marker, escapeAttr(edge.From), escapeAttr(edge.To),
		))
	}

	// Draw nodes
	nodeMap := make(map[string]*analyzer.PackageNode)
	for _, node := range graph.Nodes {
		nodeMap[node.Path] = node
	}

	// Calculate max dependents for sizing
	maxDeps := 0
	for _, n := range graph.Nodes {
		if n.DependentCount > maxDeps {
			maxDeps = n.DependentCount
		}
	}

	for path, pos := range positions {
		node := nodeMap[path]
		if node == nil {
			continue
		}

		// Calculate node size based on centrality
		sizeFactor := 1.0
		if maxDeps > 0 {
			sizeFactor = 0.6 + 0.4*float64(node.DependentCount)/float64(maxDeps)
		}

		w := nodeWidth * sizeFactor
		h := nodeHeight * sizeFactor

		// Node styling
		class := "node"
		if !node.IsInternal {
			class += " node-external"
		}
		if node.DependentCount > 5 {
			class += " node-core"
		}

		// Draw node rectangle
		svg.WriteString(fmt.Sprintf(
			`<g class="node-group" data-package="%s" data-layer="%d">`,
			escapeAttr(node.Path), node.Layer,
		))
		svg.WriteString(fmt.Sprintf(
			`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" class="%s" rx="8"/>`,
			pos.X, pos.Y, w, h, class,
		))

		// Node label
		label := node.Name
		if len(label) > 15 {
			label = label[:12] + "..."
		}

		svg.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" class="node-label">%s</text>`,
			pos.X+w/2, pos.Y+h/2-8, escapeText(label),
		))

		// Node metrics
		if node.IsInternal {
			svg.WriteString(fmt.Sprintf(
				`<text x="%.1f" y="%.1f" class="node-metric">%d deps</text>`,
				pos.X+w/2, pos.Y+h/2+8, node.DependentCount,
			))
		}

		svg.WriteString(`</g>`)
	}

	svg.WriteString(`</svg>`)
	return svg.String()
}

// calculateHierarchicalLayout positions nodes in layers
func (g *Generator) calculateHierarchicalLayout(
	graph *analyzer.PackageDependencyGraph,
	layerSpacing float64,
	nodeSpacing float64,
	nodeWidth float64,
	nodeHeight float64,
) map[string]NodePosition {
	positions := make(map[string]NodePosition)

	// Group nodes by layer
	layers := make(map[int][]*analyzer.PackageNode)
	maxLayer := 0

	for _, node := range graph.Nodes {
		if !node.IsInternal {
			continue // Skip external packages in hierarchical layout
		}

		layers[node.Layer] = append(layers[node.Layer], node)
		if node.Layer > maxLayer {
			maxLayer = node.Layer
		}
	}

	// Position nodes layer by layer
	for layer := 0; layer <= maxLayer; layer++ {
		nodes := layers[layer]
		if len(nodes) == 0 {
			continue
		}

		startX := 50.0 // Left padding

		// Position each node in the layer
		for i, node := range nodes {
			x := startX + float64(i)*(nodeWidth+nodeSpacing)
			y := 50.0 + float64(layer)*layerSpacing

			positions[node.Path] = NodePosition{X: x, Y: y}
		}
	}

	// Position external packages separately (to the right)
	externalX := 50.0 + float64(len(layers))*300
	externalY := 50.0
	for _, node := range graph.Nodes {
		if node.IsExternal {
			positions[node.Path] = NodePosition{X: externalX, Y: externalY}
			externalY += nodeHeight + nodeSpacing
		}
	}

	return positions
}

// escapeAttr escapes strings for use in XML attributes
func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// escapeText escapes strings for use in XML text content
func escapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
