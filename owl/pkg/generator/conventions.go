package generator

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/simonhull/firebird-suite/owl/pkg/analyzer"
	"github.com/simonhull/firebird-suite/owl/pkg/logger"
)

//go:embed templates/convention-base.html
var conventionBaseTemplate string

//go:embed templates/convention.html
var conventionTemplate string

// ConventionPageGroup represents all items of a specific convention type for convention pages
type ConventionPageGroup struct {
	Type           string                 // "Handler", "Service", etc.
	TypeLower      string                 // "handler" for URLs
	Description    string                 // Human-readable description
	Items          []ConventionItem       // All items of this convention
	TotalCount     int                    // Total number of items
	PackageDistrib map[string]int         // Package name → count
	DocCoverage    float64                // % with godoc
	TotalMethods   int                    // Sum of all methods
	TotalInbound   int                    // Total calls into these items
	TotalOutbound  int                    // Total calls out
}

// ConventionItem represents a single instance (e.g., one Handler)
type ConventionItem struct {
	Name            string // Type name
	PackageName     string // Just parent package (e.g., "handlers")
	FullPackagePath string // Full path for linking
	TypePath        string // Relative path to type detail page
	MethodCount     int    // Number of methods
	InboundCalls    int    // Number of calls into this type
	OutboundCalls   int    // Number of calls out from this type
	HasDocs         bool   // Whether type has documentation
	Exported        bool   // Whether type is exported
}

// GenerateConventionPages creates HTML pages for each convention type
func (g *Generator) GenerateConventionPages(project *analyzer.Project) error {
	conventionDir := filepath.Join(g.outputDir, "conventions")
	if err := os.MkdirAll(conventionDir, 0755); err != nil {
		return fmt.Errorf("creating conventions directory: %w", err)
	}

	// Group all types by convention
	groups := g.buildConventionGroups(project)

	// Generate one page per convention type
	for _, group := range groups {
		if err := g.generateConventionPage(group, conventionDir); err != nil {
			return fmt.Errorf("generating %s page: %w", group.Type, err)
		}
	}

	g.logger.Info("Generated convention pages", logger.F("count", len(groups)))
	return nil
}

// buildConventionGroups organizes all types by their detected conventions
func (g *Generator) buildConventionGroups(project *analyzer.Project) []ConventionPageGroup {
	// Convention metadata
	convTypes := []struct {
		name string
		desc string
	}{
		{"Handler", "HTTP handlers and request processors"},
		{"Service", "Business logic and service layer components"},
		{"Repository", "Data access and persistence layer"},
		{"Model", "Data models and domain entities"},
		{"Middleware", "Request/response middleware and interceptors"},
		{"Util", "Utility functions and helpers"},
		{"Config", "Configuration structures and loaders"},
	}

	groups := make([]ConventionPageGroup, 0, len(convTypes))

	for _, ct := range convTypes {
		group := ConventionPageGroup{
			Type:           ct.name,
			TypeLower:      strings.ToLower(ct.name),
			Description:    ct.desc,
			Items:          make([]ConventionItem, 0),
			PackageDistrib: make(map[string]int),
		}

		// Collect all types matching this convention
		for _, pkg := range project.Packages {
			for _, typ := range pkg.Types {
				// Check if type has this convention
				if typ.Convention == nil || typ.Convention.Name != ct.name {
					continue
				}

				item := ConventionItem{
					Name:            typ.Name,
					PackageName:     pkg.Name,
					FullPackagePath: pkg.Path,
					TypePath:        fmt.Sprintf("../types/%s/%s.html", pkg.Name, typ.Name),
					MethodCount:     len(typ.Methods),
					HasDocs:         typ.Doc != "",
					Exported:        isExported(typ.Name),
				}

				// Calculate call graph metrics - for now use 0, we can enhance this later
				item.InboundCalls = 0
				item.OutboundCalls = 0

				group.Items = append(group.Items, item)
				group.PackageDistrib[item.PackageName]++

				if item.HasDocs {
					group.DocCoverage++
				}
				group.TotalMethods += item.MethodCount
				group.TotalInbound += item.InboundCalls
				group.TotalOutbound += item.OutboundCalls
			}
		}

		group.TotalCount = len(group.Items)
		if group.TotalCount > 0 {
			group.DocCoverage = (group.DocCoverage / float64(group.TotalCount)) * 100

			// Sort items by name for consistent output
			sort.Slice(group.Items, func(i, j int) bool {
				return group.Items[i].Name < group.Items[j].Name
			})
		}

		// Always add the group, even if empty
		groups = append(groups, group)
	}

	return groups
}

// generateConventionPage creates the HTML for one convention type
func (g *Generator) generateConventionPage(group ConventionPageGroup, outputDir string) error {
	// Create template with helper functions
	tmpl := template.New("base").Funcs(template.FuncMap{
		"len": func(v any) int {
			switch val := v.(type) {
			case []ConventionItem:
				return len(val)
			case map[string]int:
				return len(val)
			default:
				return 0
			}
		},
		"toJSON": toJSON,
	})

	// Parse base template
	tmpl, err := tmpl.Parse(conventionBaseTemplate)
	if err != nil {
		return fmt.Errorf("parsing base template: %w", err)
	}

	// Parse convention template
	tmpl, err = tmpl.Parse(conventionTemplate)
	if err != nil {
		return fmt.Errorf("parsing convention template: %w", err)
	}

	data := struct {
		Title string
		Group ConventionPageGroup
	}{
		Title: fmt.Sprintf("%ss - Conventions", group.Type),
		Group: group,
	}

	// Create output file - handle special pluralizations
	filename := strings.ToLower(group.Type) + "s"
	if group.Type == "Repository" {
		filename = "repositories"
	}
	outputPath := filepath.Join(outputDir, filename+".html")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating convention file: %w", err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
}

// toJSON converts a value to JSON for use in templates
func toJSON(v any) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("{}")
	}
	return template.JS(b)
}
