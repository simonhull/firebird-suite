package generator

import (
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/simonhull/firebird-suite/owl/pkg/analyzer"
)

//go:embed templates/interface-base.html
var interfaceBaseTemplate string

//go:embed templates/interface.html
var interfaceTemplate string

// InterfacePageData contains all data needed for the interface diagram page
type InterfacePageData struct {
	Title            string
	Analysis         *analyzer.InterfaceAnalysis
	InterfaceGroups  []InterfaceGroup
	Stats            analyzer.InterfaceStats
	HasCycles        bool
	UnusedInterfaces []*analyzer.Interface
}

// InterfaceGroup represents a grouped set of interfaces for display
type InterfaceGroup struct {
	Name         string
	Key          string
	Interfaces   []*InterfaceDisplayItem
	Count        int
	IsStdlib     bool
	TotalImpls   int
	AvgImpls     float64
}

// InterfaceDisplayItem contains display-ready interface data
type InterfaceDisplayItem struct {
	Interface        *analyzer.Interface
	Implementations  []*analyzer.Implementation
	AlmostImplements []*analyzer.AlmostImplementation
	IsUnused         bool
	ComplexityScore  float64
}

// GenerateInterfaceDiagram creates the interface implementation visualization page
func (g *Generator) GenerateInterfaceDiagram(analysis *analyzer.InterfaceAnalysis) error {
	interfacesDir := filepath.Join(g.outputDir, "interfaces")
	if err := os.MkdirAll(interfacesDir, 0755); err != nil {
		return fmt.Errorf("creating interfaces directory: %w", err)
	}

	// Build display data
	groups := g.buildInterfaceGroups(analysis)

	// Create template with helper functions
	tmpl := template.New("interface-base").Funcs(template.FuncMap{
		"len": func(v any) int {
			switch val := v.(type) {
			case []*analyzer.Implementation:
				return len(val)
			case []*analyzer.AlmostImplementation:
				return len(val)
			case []*analyzer.InterfaceMethod:
				return len(val)
			case []InterfaceGroup:
				return len(val)
			default:
				return 0
			}
		},
		"join": func(items []string, sep string) string {
			return strings.Join(items, sep)
		},
		"formatSignature": func(method *analyzer.InterfaceMethod) string {
			return method.Signature
		},
		"shortPath": func(path string) string {
			parts := strings.Split(path, "/")
			if len(parts) > 2 {
				return ".../" + strings.Join(parts[len(parts)-2:], "/")
			}
			return path
		},
		"getComplexityBadge": func(score float64) string {
			if score < 2 {
				return "simple"
			} else if score < 5 {
				return "moderate"
			}
			return "complex"
		},
		"toJSON": func(v any) template.JS {
			// For Alpine.js data binding
			return template.JS(fmt.Sprintf("%v", v))
		},
	})

	// Parse templates
	tmpl, err := tmpl.Parse(interfaceBaseTemplate)
	if err != nil {
		return fmt.Errorf("parsing interface base template: %w", err)
	}

	tmpl, err = tmpl.Parse(interfaceTemplate)
	if err != nil {
		return fmt.Errorf("parsing interface template: %w", err)
	}

	data := InterfacePageData{
		Title:            "Interface Implementations",
		Analysis:         analysis,
		InterfaceGroups:  groups,
		Stats:            analysis.Stats,
		UnusedInterfaces: analysis.UnusedInterfaces,
	}

	// Create output file
	outputPath := filepath.Join(interfacesDir, "implementations.html")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating interface file: %w", err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	g.logger.Info("Generated interface diagram page")
	return nil
}

// buildInterfaceGroups organizes interfaces into display groups
func (g *Generator) buildInterfaceGroups(analysis *analyzer.InterfaceAnalysis) []InterfaceGroup {
	groups := make(map[string]*InterfaceGroup)

	// Create groups: stdlib and project
	groups["stdlib"] = &InterfaceGroup{
		Name:       "Standard Library",
		Key:        "stdlib",
		Interfaces: make([]*InterfaceDisplayItem, 0),
		IsStdlib:   true,
	}

	groups["project"] = &InterfaceGroup{
		Name:       "Project Interfaces",
		Key:        "project",
		Interfaces: make([]*InterfaceDisplayItem, 0),
		IsStdlib:   false,
	}

	// Build unused interface map for quick lookup
	unusedMap := make(map[string]bool)
	for _, iface := range analysis.UnusedInterfaces {
		key := iface.PackagePath + "." + iface.Name
		unusedMap[key] = true
	}

	// Process each interface
	for _, iface := range analysis.Interfaces {
		key := iface.PackagePath + "." + iface.Name

		// Get implementations and almost-implementations
		impls := analysis.Implementations[key]
		almosts := analysis.AlmostImplements[key]

		// Calculate complexity score (based on method count and implementation count)
		complexityScore := float64(iface.MethodCount)
		if len(impls) > 0 {
			complexityScore += float64(len(impls)) * 0.5
		}

		item := &InterfaceDisplayItem{
			Interface:        iface,
			Implementations:  impls,
			AlmostImplements: almosts,
			IsUnused:         unusedMap[key],
			ComplexityScore:  complexityScore,
		}

		// Add to appropriate group
		if iface.IsStdlib {
			groups["stdlib"].Interfaces = append(groups["stdlib"].Interfaces, item)
			groups["stdlib"].TotalImpls += len(impls)
		} else {
			groups["project"].Interfaces = append(groups["project"].Interfaces, item)
			groups["project"].TotalImpls += len(impls)
		}
	}

	// Sort interfaces within each group by implementer count (descending)
	for _, group := range groups {
		sort.Slice(group.Interfaces, func(i, j int) bool {
			return len(group.Interfaces[i].Implementations) > len(group.Interfaces[j].Implementations)
		})

		group.Count = len(group.Interfaces)
		if group.Count > 0 {
			group.AvgImpls = float64(group.TotalImpls) / float64(group.Count)
		}
	}

	// Convert to slice and sort (project first, then stdlib)
	result := []InterfaceGroup{*groups["project"], *groups["stdlib"]}

	return result
}
