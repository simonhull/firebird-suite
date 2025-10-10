package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"

	"github.com/simonhull/firebird-suite/owl/pkg/analyzer"
)

// CalculatePackageMetrics computes comprehensive metrics for a package
func CalculatePackageMetrics(pkg *analyzer.Package, modulePath string) *PackageMetrics {
	metrics := &PackageMetrics{
		Conventions: make([]*ConventionCount, 0),
	}

	// Basic counts
	metrics.TotalTypes = len(pkg.Types)
	metrics.TotalFunctions = len(pkg.Functions)

	// Count methods across all types
	for _, typ := range pkg.Types {
		metrics.TotalMethods += len(typ.Methods)
	}

	// Exported vs internal counts
	for _, typ := range pkg.Types {
		if isExported(typ.Name) {
			metrics.ExportedCount++
		} else {
			metrics.InternalCount++
		}
	}
	for _, fn := range pkg.Functions {
		if isExported(fn.Name) {
			metrics.ExportedCount++
		} else {
			metrics.InternalCount++
		}
	}

	// Complexity distribution based on function calls
	for _, fn := range pkg.Functions {
		callCount := len(fn.Calls)
		categorizeComplexity(callCount, metrics)
	}
	for _, typ := range pkg.Types {
		for _, method := range typ.Methods {
			callCount := len(method.Calls)
			categorizeComplexity(callCount, metrics)
		}
	}

	// Convention distribution
	conventionMap := make(map[string]int)
	for _, typ := range pkg.Types {
		if typ.Convention != nil && typ.Convention.Category != "" {
			conventionMap[typ.Convention.Category]++
		}
	}
	for _, fn := range pkg.Functions {
		if fn.Convention != nil && fn.Convention.Category != "" {
			conventionMap[fn.Convention.Category]++
		}
	}

	// Convert to sorted slice with percentages
	totalItems := metrics.TotalTypes + metrics.TotalFunctions
	for name, count := range conventionMap {
		percentage := 0.0
		if totalItems > 0 {
			percentage = float64(count) / float64(totalItems) * 100
		}
		metrics.Conventions = append(metrics.Conventions, &ConventionCount{
			Name:       name,
			Count:      count,
			Percentage: percentage,
		})
	}

	// Sort conventions by count (descending)
	sort.Slice(metrics.Conventions, func(i, j int) bool {
		return metrics.Conventions[i].Count > metrics.Conventions[j].Count
	})

	// Count imports
	metrics.TotalImports = len(pkg.Imports)
	for _, imp := range pkg.Imports {
		if isInternalImport(imp, modulePath) {
			metrics.InternalImports++
		} else {
			metrics.ExternalImports++
		}
	}

	// Estimate lines of code
	metrics.LinesOfCode = estimatePackageLOC(pkg)

	return metrics
}

// categorizeComplexity categorizes a function based on call count
func categorizeComplexity(callCount int, metrics *PackageMetrics) {
	switch {
	case callCount <= 3:
		metrics.SimpleFunctions++
	case callCount <= 7:
		metrics.MediumFunctions++
	default:
		metrics.ComplexFunctions++
	}
}

// isInternalImport checks if an import is from the same module
func isInternalImport(importPath, modulePath string) bool {
	return strings.HasPrefix(importPath, modulePath)
}

// estimatePackageLOC estimates lines of code for a package
func estimatePackageLOC(pkg *analyzer.Package) int {
	loc := 0

	// Parse all Go files in the package directory
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkg.Path, func(fi os.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), ".go") && !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)

	if err != nil {
		return 0
	}

	// Count lines in each file
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			loc += countFileLines(fset, file)
		}
	}

	return loc
}

// countFileLines counts non-empty, non-comment lines in a file
func countFileLines(fset *token.FileSet, file *ast.File) int {
	// Simple heuristic: use the last position in the file
	endPos := fset.Position(file.End())
	return endPos.Line
}

