package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SearchItem represents a searchable item
type SearchItem struct {
	Type        string   `json:"type"`        // "package", "type", "function"
	Name        string   `json:"name"`        // Display name
	Package     string   `json:"package"`     // Package name
	Path        string   `json:"path"`        // URL path
	Description string   `json:"description"` // Brief description
	Tags        []string `json:"tags"`        // Searchable tags (convention, complexity)
}

// SearchIndex contains all searchable items
type SearchIndex struct {
	Items []*SearchItem `json:"items"`
}

// BuildSearchIndex creates search index from site data
func (g *Generator) BuildSearchIndex(data *SiteData) *SearchIndex {
	index := &SearchIndex{
		Items: []*SearchItem{},
	}

	// Index packages
	for _, pkg := range data.Packages {
		index.Items = append(index.Items, &SearchItem{
			Type:        "package",
			Name:        pkg.Name,
			Package:     pkg.Name,
			Path:        fmt.Sprintf("packages/%s.html", pkg.Name),
			Description: fmt.Sprintf("%d types, %d functions", len(pkg.Types), len(pkg.Functions)),
			Tags:        []string{"package"},
		})

		// Index types
		for _, typ := range pkg.Types {
			tags := []string{"type", typ.Kind}
			if typ.PrimaryBadge != nil {
				tags = append(tags, typ.PrimaryBadge.Label)
			}

			index.Items = append(index.Items, &SearchItem{
				Type:        "type",
				Name:        typ.Name,
				Package:     pkg.Name,
				Path:        fmt.Sprintf("types/%s/%s.html", pkg.Name, typ.Name),
				Description: fmt.Sprintf("%s in %s", typ.Kind, pkg.Name),
				Tags:        tags,
			})
		}

		// Index functions
		for _, fn := range pkg.Functions {
			complexity := "simple"
			callCount := len(fn.CallsFunctions)
			if callCount > 3 {
				complexity = "medium"
			}
			if callCount > 7 {
				complexity = "complex"
			}

			index.Items = append(index.Items, &SearchItem{
				Type:        "function",
				Name:        fn.Name,
				Package:     pkg.Name,
				Path:        fmt.Sprintf("packages/%s.html#func-%s", pkg.Name, fn.Name),
				Description: fmt.Sprintf("Function in %s", pkg.Name),
				Tags:        []string{"function", complexity},
			})
		}
	}

	return index
}

// WriteSearchIndex writes search index as JSON
func (g *Generator) WriteSearchIndex(outputDir string, index *SearchIndex) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling search index: %w", err)
	}

	assetsDir := filepath.Join(outputDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return fmt.Errorf("creating assets directory: %w", err)
	}

	path := filepath.Join(assetsDir, "search-index.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing search index: %w", err)
	}

	return nil
}
