package generator

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/simonhull/firebird-suite/owl/pkg/analyzer"
)

//go:embed assets
var assetsFS embed.FS

//go:embed assets/styles.css
var cssContent string

//go:embed templates/base.html
var baseTemplate string

//go:embed templates/index.html
var indexTemplate string

//go:embed templates/package.html
var packageTemplate string

//go:embed templates/package-base.html
var packageBaseTemplate string

// Generator generates HTML documentation from analyzed projects
type Generator struct {
	outputDir         string
	project           *analyzer.Project
	functionCallGraph map[string][]string // function ID -> callers
}

// NewGenerator creates a new HTML documentation generator
func NewGenerator(outputDir string) *Generator {
	return &Generator{
		outputDir: outputDir,
	}
}

// Generate creates HTML documentation from an analyzed project
func (g *Generator) Generate(project *analyzer.Project) error {
	// Store project for relationship building
	g.project = project

	// Build call graph relationships
	g.buildRelationships()

	// Convert analyzer.Project to SiteData
	siteData := g.convertToSiteData(project)

	// Create output directory
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create assets directory
	assetsDir := filepath.Join(g.outputDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return fmt.Errorf("failed to create assets directory: %w", err)
	}

	// Copy assets (CSS and Alpine.js)
	if err := g.copyAssets(assetsDir); err != nil {
		return fmt.Errorf("failed to copy assets: %w", err)
	}

	// Generate index.html
	if err := g.generateIndex(siteData); err != nil {
		return fmt.Errorf("failed to generate index.html: %w", err)
	}

	// Generate package pages
	if err := g.generatePackagePages(siteData); err != nil {
		return fmt.Errorf("failed to generate package pages: %w", err)
	}

	// Generate search index
	searchIndex := g.BuildSearchIndex(siteData)
	if err := g.WriteSearchIndex(g.outputDir, searchIndex); err != nil {
		return fmt.Errorf("failed to generate search index: %w", err)
	}

	fmt.Printf("✅ Documentation generated successfully at %s\n", g.outputDir)
	return nil
}

// convertToSiteData converts analyzer.Project to template-friendly SiteData
func (g *Generator) convertToSiteData(project *analyzer.Project) *SiteData {
	siteData := &SiteData{
		ProjectName:      g.extractProjectName(project),
		ModulePath:       g.extractModulePath(project),
		GoVersion:        g.extractGoVersion(project),
		GeneratedAt:      time.Now(),
		IsFirebird:       project.IsFirebirdProject,
		Packages:         make([]*PackageData, 0),
		AllTypes:         make([]*TypeData, 0),
		AllFunctions:     make([]*FunctionData, 0),
		ConventionGroups: make([]*ConventionGroup, 0),
		Stats:            &SiteStats{},
	}

	// Read project README
	if readmeHTML, found := ReadREADME(project.RootPath); found {
		siteData.ReadmeHTML = template.HTML(readmeHTML)
		siteData.HasReadme = true
	}

	// Convert Firebird config
	if project.FirebirdConfig != nil {
		siteData.FirebirdInfo = &FirebirdInfo{
			Database: project.FirebirdConfig.Database,
			Router:   project.FirebirdConfig.Router,
		}
	}

	// Track conventions for grouping
	conventionMap := make(map[string]*ConventionGroup)

	// Convert packages
	for _, pkg := range project.Packages {
		pkgData := &PackageData{
			Name:        pkg.Name,
			Path:        pkg.Path,
			ImportPath:  pkg.ImportPath,
			Description: pkg.Doc,
			Types:       make([]*TypeData, 0),
			Functions:   make([]*FunctionData, 0),
			Constants:   make([]*ConstantData, 0),
			Variables:   make([]*VariableData, 0),
		}

		// Read package documentation
		if docHTML, found := ReadPackageDoc(pkg.Path); found {
			pkgData.DocHTML = template.HTML(docHTML)
			pkgData.HasDoc = true
		}

		// Convert types
		for _, typ := range pkg.Types {
			typeData := g.convertType(typ, pkg.Name, pkg.ImportPath)
			pkgData.Types = append(pkgData.Types, typeData)
			siteData.AllTypes = append(siteData.AllTypes, typeData)

			// Track stats
			siteData.Stats.TotalTypes++
			if typ.Kind == "struct" {
				siteData.Stats.TotalStructs++
			} else if typ.Kind == "interface" {
				siteData.Stats.TotalInterfaces++
			}

			// Group by convention
			if typeData.PrimaryBadge != nil {
				convType := typeData.PrimaryBadge.Type
				if _, exists := conventionMap[convType]; !exists {
					conventionMap[convType] = &ConventionGroup{
						Name:  typeData.PrimaryBadge.Label + "s",
						Slug:  convType + "s",
						Types: make([]*TypeData, 0),
					}
				}
				conventionMap[convType].Types = append(conventionMap[convType].Types, typeData)
				conventionMap[convType].Count++

				// Track convention stats
				switch convType {
				case "handler":
					siteData.Stats.HandlerCount++
				case "service":
					siteData.Stats.ServiceCount++
				case "repository":
					siteData.Stats.RepositoryCount++
				case "dto":
					siteData.Stats.DTOCount++
				case "middleware":
					siteData.Stats.MiddlewareCount++
				case "model":
					siteData.Stats.ModelCount++
				}
			}
		}

		// Convert functions
		for _, fn := range pkg.Functions {
			fnData := g.convertFunction(fn, pkg.Name)
			pkgData.Functions = append(pkgData.Functions, fnData)
			siteData.AllFunctions = append(siteData.AllFunctions, fnData)
			siteData.Stats.TotalFunctions++
		}

		// Calculate package metrics
		pkgData.Metrics = CalculatePackageMetrics(pkg, project.Module)

		// Detect primary convention (most common)
		if len(pkgData.Metrics.Conventions) > 0 {
			pkgData.PrimaryConvention = pkgData.Metrics.Conventions[0].Name
		}

		// Generate call graph
		if len(pkg.Functions) > 0 {
			callGraph := g.GenerateCallGraph(pkg)
			if callGraph != nil {
				pkgData.CallGraph = template.HTML(callGraph.RenderCallGraphSVG())
				pkgData.HasGraph = len(pkgData.CallGraph) > 0
			}
		}

		siteData.Packages = append(siteData.Packages, pkgData)
		siteData.Stats.TotalPackages++
	}

	// Convert convention map to sorted slice
	// Order: handlers, services, repositories, dtos, middleware, models
	conventionOrder := []string{"handler", "service", "repository", "dto", "middleware", "model"}
	for _, convType := range conventionOrder {
		if group, exists := conventionMap[convType]; exists {
			siteData.ConventionGroups = append(siteData.ConventionGroups, group)
		}
	}

	return siteData
}

// convertType converts analyzer.Type to TypeData
func (g *Generator) convertType(typ *analyzer.Type, pkgName, importPath string) *TypeData {
	typeData := &TypeData{
		Name:         typ.Name,
		Kind:         typ.Kind,
		Package:      pkgName,
		ImportPath:   importPath,
		Description:  typ.Doc,
		Fields:       make([]*FieldData, 0),
		Methods:      make([]*MethodData, 0),
		UsedTypes:    typ.UsedTypes,
		File:         typ.FilePath,
		LineNumber:   typ.Line,
		RelativePath: g.makeRelativePath(typ.FilePath),
	}

	// Convert primary badge from convention
	if typ.Convention != nil {
		typeData.PrimaryBadge = &Badge{
			Type:       typ.Convention.Category,
			Label:      typ.Convention.Name,
			Confidence: typ.Convention.Confidence,
			Color:      "badge-" + typ.Convention.Category,
		}
	}

	// Convert fields
	for _, field := range typ.Fields {
		typeData.Fields = append(typeData.Fields, &FieldData{
			Name:        field.Name,
			Type:        field.Type,
			Tag:         field.Tag,
			Description: field.Doc,
			Exported:    isExported(field.Name),
		})
	}

	// Convert methods
	for _, method := range typ.Methods {
		typeData.Methods = append(typeData.Methods, &MethodData{
			Name:           method.Name,
			Receiver:       method.Receiver,
			Signature:      method.Signature,
			Description:    method.Doc,
			Parameters:     g.convertParameters(method.Parameters),
			Returns:        g.convertReturns(method.Returns),
			Exported:       isExported(method.Name),
			CallsFunctions: method.Calls,
			UsesTypes:      method.UsesTypes,
			File:           method.FilePath,
			LineNumber:     method.Line,
			RelativePath:   g.makeRelativePath(method.FilePath),
		})
	}

	return typeData
}

// convertFunction converts analyzer.Function to FunctionData
func (g *Generator) convertFunction(fn *analyzer.Function, pkgName string) *FunctionData {
	fnID := pkgName + "." + fn.Name

	// Get who calls this function
	calledBy := g.functionCallGraph[fnID]
	if calledBy == nil {
		calledBy = []string{}
	}

	return &FunctionData{
		Name:           fn.Name,
		Package:        pkgName,
		Signature:      fn.Signature,
		Description:    fn.Doc,
		Parameters:     g.convertParameters(fn.Parameters),
		Returns:        g.convertReturns(fn.Returns),
		Exported:       isExported(fn.Name),
		CallsFunctions: fn.Calls,
		CalledBy:       calledBy,
		UsesTypes:      fn.UsesTypes,
		File:           fn.FilePath,
		LineNumber:     fn.Line,
		RelativePath:   g.makeRelativePath(fn.FilePath),
	}
}

// convertParameters converts analyzer parameters to template parameters
func (g *Generator) convertParameters(params []*analyzer.Parameter) []*ParameterData {
	result := make([]*ParameterData, len(params))
	for i, p := range params {
		result[i] = &ParameterData{
			Name: p.Name,
			Type: p.Type,
		}
	}
	return result
}

// convertReturns converts analyzer returns to template returns
func (g *Generator) convertReturns(returns []*analyzer.Parameter) []*ReturnData {
	result := make([]*ReturnData, len(returns))
	for i, r := range returns {
		result[i] = &ReturnData{
			Name: r.Name,
			Type: r.Type,
		}
	}
	return result
}

// conventionTypeToLabel converts convention type to display label
func (g *Generator) conventionTypeToLabel(convType string) string {
	labels := map[string]string{
		"handler":    "Handler",
		"service":    "Service",
		"repository": "Repository",
		"dto":        "DTO",
		"middleware": "Middleware",
		"model":      "Model",
	}
	if label, ok := labels[convType]; ok {
		return label
	}
	return convType
}

// extractProjectName extracts a friendly project name
func (g *Generator) extractProjectName(project *analyzer.Project) string {
	if len(project.Packages) > 0 {
		// Try to use first package's import path
		importPath := project.Packages[0].ImportPath
		if importPath != "" {
			// Extract last part of import path
			parts := filepath.Base(importPath)
			return parts
		}
	}
	return "Project Documentation"
}

// extractModulePath extracts the Go module path
func (g *Generator) extractModulePath(project *analyzer.Project) string {
	if len(project.Packages) > 0 && project.Packages[0].ImportPath != "" {
		return project.Packages[0].ImportPath
	}
	return "unknown"
}

// extractGoVersion extracts the Go version
func (g *Generator) extractGoVersion(project *analyzer.Project) string {
	// TODO: Extract from go.mod when available
	return ""
}

// copyAssets copies CSS and Alpine.js to the assets directory
func (g *Generator) copyAssets(assetsDir string) error {
	// Copy CSS
	cssPath := filepath.Join(assetsDir, "styles.css")
	if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
		return fmt.Errorf("failed to write styles.css: %w", err)
	}
	fmt.Printf("   ✓ Copied styles.css (%d bytes)\n", len(cssContent))

	// Copy Alpine.js from embedded assets
	alpinePath := filepath.Join(assetsDir, "alpine.min.js")
	alpineContent, err := assetsFS.ReadFile("assets/alpine.min.js")
	if err != nil {
		return fmt.Errorf("failed to read alpine.min.js: %w", err)
	}
	if err := os.WriteFile(alpinePath, alpineContent, 0644); err != nil {
		return fmt.Errorf("failed to write alpine.min.js: %w", err)
	}
	fmt.Printf("   ✓ Copied alpine.min.js (%d bytes)\n", len(alpineContent))

	// Copy Fuse.js from embedded assets
	fusePath := filepath.Join(assetsDir, "fuse.min.js")
	fuseContent, err := assetsFS.ReadFile("assets/fuse.min.js")
	if err != nil {
		return fmt.Errorf("failed to read fuse.min.js: %w", err)
	}
	if err := os.WriteFile(fusePath, fuseContent, 0644); err != nil {
		return fmt.Errorf("failed to write fuse.min.js: %w", err)
	}
	fmt.Printf("   ✓ Copied fuse.min.js (%d bytes)\n", len(fuseContent))

	return nil
}

// generateIndex generates the index.html file
func (g *Generator) generateIndex(siteData *SiteData) error {
	// Create template with helper functions
	tmpl := template.New("base").Funcs(template.FuncMap{
		"len": func(v interface{}) int {
			switch val := v.(type) {
			case []*FieldData:
				return len(val)
			case []*MethodData:
				return len(val)
			case []*FunctionData:
				return len(val)
			case []*TypeData:
				return len(val)
			case []*ConstantData:
				return len(val)
			default:
				return 0
			}
		},
	})

	// Parse templates
	tmpl, err := tmpl.Parse(baseTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse base template: %w", err)
	}

	tmpl, err = tmpl.Parse(indexTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse index template: %w", err)
	}

	// Create output file
	outputPath := filepath.Join(g.outputDir, "index.html")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, siteData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// generatePackagePages generates individual package HTML pages
func (g *Generator) generatePackagePages(siteData *SiteData) error {
	// Create packages directory
	packagesDir := filepath.Join(g.outputDir, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		return fmt.Errorf("failed to create packages directory: %w", err)
	}

	// Generate a page for each package
	for _, pkg := range siteData.Packages {
		if err := g.generatePackagePage(pkg, packagesDir); err != nil {
			return fmt.Errorf("failed to generate page for package %s: %w", pkg.Name, err)
		}
	}

	fmt.Printf("   ✓ Generated %d package pages\n", len(siteData.Packages))
	return nil
}

// generatePackagePage generates a single package HTML page
func (g *Generator) generatePackagePage(pkg *PackageData, packagesDir string) error {
	// Create template with helper functions
	tmpl := template.New("package-base").Funcs(template.FuncMap{
		"len": func(v interface{}) int {
			switch val := v.(type) {
			case []*FieldData:
				return len(val)
			case []*MethodData:
				return len(val)
			case []*FunctionData:
				return len(val)
			case []*TypeData:
				return len(val)
			case []*ConstantData:
				return len(val)
			case []*ConventionCount:
				return len(val)
			default:
				return 0
			}
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"complexity_percent": func(count, totalFns, totalMethods int) float64 {
			total := totalFns + totalMethods
			if total == 0 {
				return 0
			}
			return float64(count) / float64(total) * 100
		},
	})

	// Parse templates
	tmpl, err := tmpl.Parse(packageBaseTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse package base template: %w", err)
	}

	tmpl, err = tmpl.Parse(packageTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse package template: %w", err)
	}

	// Create output file
	outputPath := filepath.Join(packagesDir, pkg.Name+".html")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create package file: %w", err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, pkg); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// isExported checks if a name is exported (starts with uppercase letter)
func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

// buildRelationships computes reverse relationships for functions
func (g *Generator) buildRelationships() {
	g.functionCallGraph = make(map[string][]string)

	// Build function call graph (who calls whom)
	for _, pkg := range g.project.Packages {
		for _, fn := range pkg.Functions {
			fnID := pkg.Name + "." + fn.Name

			// For each function this function calls
			for _, calledFn := range fn.Calls {
				if g.functionCallGraph[calledFn] == nil {
					g.functionCallGraph[calledFn] = []string{}
				}
				g.functionCallGraph[calledFn] = append(g.functionCallGraph[calledFn], fnID)
			}
		}

		// Also check method calls
		for _, typ := range pkg.Types {
			for _, method := range typ.Methods {
				methodID := pkg.Name + "." + typ.Name + "." + method.Name

				// Extract calls from method bodies
				for _, calledFn := range method.Calls {
					if g.functionCallGraph[calledFn] == nil {
						g.functionCallGraph[calledFn] = []string{}
					}
					g.functionCallGraph[calledFn] = append(g.functionCallGraph[calledFn], methodID)
				}
			}
		}
	}
}

// makeRelativePath makes a file path relative to the project root
func (g *Generator) makeRelativePath(fullPath string) string {
	if g.project == nil || g.project.RootPath == "" {
		return fullPath
	}
	rel, err := filepath.Rel(g.project.RootPath, fullPath)
	if err != nil {
		return fullPath
	}
	return rel
}
