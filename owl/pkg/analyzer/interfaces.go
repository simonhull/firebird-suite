package analyzer

import (
	"sort"
	"strings"
)

// InterfaceAnalysis contains all interface implementation relationships
type InterfaceAnalysis struct {
	Interfaces       []*Interface
	Implementations  map[string][]*Implementation       // Interface key → implementations
	AlmostImplements map[string][]*AlmostImplementation // Interface key → almost implementations
	UnusedInterfaces []*Interface
	Stats            InterfaceStats
}

// Interface represents an interface type
type Interface struct {
	Name         string
	PackagePath  string
	PackageName  string
	Methods      []*InterfaceMethod
	MethodCount  int
	IsStdlib     bool
	IsProject    bool
	Doc          string
	Implementers int // Count of types that implement this
}

// InterfaceMethod represents a method in an interface
type InterfaceMethod struct {
	Name      string
	Signature string   // Full signature with params and returns
	Params    []string
	Returns   []string
}

// Implementation represents a type that implements an interface
type Implementation struct {
	TypeName       string
	PackagePath    string
	PackageName    string
	InterfaceName  string
	InterfacePath  string
	MatchedMethods []*InterfaceMethod
	IsExported     bool
}

// AlmostImplementation represents a type that's close to implementing an interface
type AlmostImplementation struct {
	TypeName       string
	PackagePath    string
	PackageName    string
	InterfaceName  string
	InterfacePath  string
	MissingMethods []*InterfaceMethod
	MatchedMethods []*InterfaceMethod
	MissingCount   int
}

// InterfaceStats provides summary metrics
type InterfaceStats struct {
	TotalInterfaces   int
	ProjectInterfaces int
	StdlibInterfaces  int
	TotalImplementers int
	AvgImplementers   float64
	UnusedCount       int
	AlmostCount       int
}

// Key stdlib interfaces we care about
var importantStdlibInterfaces = map[string]*Interface{
	"error": {
		Name:        "error",
		PackagePath: "builtin",
		PackageName: "",
		IsStdlib:    true,
		IsProject:   false,
		Methods: []*InterfaceMethod{
			{Name: "Error", Signature: "Error() string", Returns: []string{"string"}},
		},
		MethodCount: 1,
	},
	"fmt.Stringer": {
		Name:        "Stringer",
		PackagePath: "fmt",
		PackageName: "fmt",
		IsStdlib:    true,
		IsProject:   false,
		Methods: []*InterfaceMethod{
			{Name: "String", Signature: "String() string", Returns: []string{"string"}},
		},
		MethodCount: 1,
	},
	"io.Reader": {
		Name:        "Reader",
		PackagePath: "io",
		PackageName: "io",
		IsStdlib:    true,
		IsProject:   false,
		Methods: []*InterfaceMethod{
			{
				Name:      "Read",
				Signature: "Read(p []byte) (n int, err error)",
				Params:    []string{"[]byte"},
				Returns:   []string{"int", "error"},
			},
		},
		MethodCount: 1,
	},
	"io.Writer": {
		Name:        "Writer",
		PackagePath: "io",
		PackageName: "io",
		IsStdlib:    true,
		IsProject:   false,
		Methods: []*InterfaceMethod{
			{
				Name:      "Write",
				Signature: "Write(p []byte) (n int, err error)",
				Params:    []string{"[]byte"},
				Returns:   []string{"int", "error"},
			},
		},
		MethodCount: 1,
	},
	"io.Closer": {
		Name:        "Closer",
		PackagePath: "io",
		PackageName: "io",
		IsStdlib:    true,
		IsProject:   false,
		Methods: []*InterfaceMethod{
			{Name: "Close", Signature: "Close() error", Returns: []string{"error"}},
		},
		MethodCount: 1,
	},
}

// AnalyzeInterfaces performs interface implementation analysis
func AnalyzeInterfaces(project *Project) (*InterfaceAnalysis, error) {
	// Estimate sizes based on project
	estimatedInterfaces := len(project.Packages) * 2 // Rough estimate

	analysis := &InterfaceAnalysis{
		Interfaces:       make([]*Interface, 0, estimatedInterfaces+len(importantStdlibInterfaces)),
		Implementations:  make(map[string][]*Implementation, estimatedInterfaces),
		AlmostImplements: make(map[string][]*AlmostImplementation, estimatedInterfaces),
		UnusedInterfaces: make([]*Interface, 0, estimatedInterfaces/4),
	}

	// Step 1: Collect all interfaces (project + important stdlib)
	interfaceMap := make(map[string]*Interface, estimatedInterfaces+len(importantStdlibInterfaces))

	// Collect project interfaces
	for _, pkg := range project.Packages {
		for _, typ := range pkg.Types {
			if isInterfaceType(typ) {
				iface := &Interface{
					Name:        typ.Name,
					PackagePath: pkg.ImportPath,
					PackageName: pkg.Name,
					Methods:     extractInterfaceMethods(typ),
					IsStdlib:    false,
					IsProject:   true,
					Doc:         typ.Doc,
				}
				iface.MethodCount = len(iface.Methods)

				key := pkg.ImportPath + "." + typ.Name
				interfaceMap[key] = iface
				analysis.Interfaces = append(analysis.Interfaces, iface)
			}
		}
	}

	// Add important stdlib interfaces
	for key, iface := range importantStdlibInterfaces {
		// Create a copy to avoid modifying the original
		ifaceCopy := *iface
		interfaceMap[key] = &ifaceCopy
		analysis.Interfaces = append(analysis.Interfaces, &ifaceCopy)
	}

	// Step 2: Check each type against each interface
	for _, pkg := range project.Packages {
		for _, typ := range pkg.Types {
			if isInterfaceType(typ) {
				continue // Skip interfaces themselves
			}

			// Check against all interfaces
			for ifaceKey, iface := range interfaceMap {
				match := checkInterfaceImplementation(typ, iface)

				if match.IsComplete {
					// Perfect implementation
					impl := &Implementation{
						TypeName:       typ.Name,
						PackagePath:    pkg.ImportPath,
						PackageName:    pkg.Name,
						InterfaceName:  iface.Name,
						InterfacePath:  iface.PackagePath,
						MatchedMethods: match.Matched,
						IsExported:     isExported(typ.Name),
					}
					analysis.Implementations[ifaceKey] = append(
						analysis.Implementations[ifaceKey],
						impl,
					)
					iface.Implementers++

				} else if match.IsAlmost {
					// Almost implements (missing 1-2 methods)
					almost := &AlmostImplementation{
						TypeName:       typ.Name,
						PackagePath:    pkg.ImportPath,
						PackageName:    pkg.Name,
						InterfaceName:  iface.Name,
						InterfacePath:  iface.PackagePath,
						MissingMethods: match.Missing,
						MatchedMethods: match.Matched,
						MissingCount:   len(match.Missing),
					}
					analysis.AlmostImplements[ifaceKey] = append(
						analysis.AlmostImplements[ifaceKey],
						almost,
					)
				}
			}
		}
	}

	// Step 3: Identify unused interfaces
	for _, iface := range analysis.Interfaces {
		if iface.Implementers == 0 && iface.IsProject {
			analysis.UnusedInterfaces = append(analysis.UnusedInterfaces, iface)
		}
	}

	// Step 4: Calculate statistics
	analysis.Stats = calculateInterfaceStats(analysis)

	// Sort interfaces by implementer count (descending)
	sort.Slice(analysis.Interfaces, func(i, j int) bool {
		return analysis.Interfaces[i].Implementers > analysis.Interfaces[j].Implementers
	})

	return analysis, nil
}

// ImplementationMatch represents the result of checking type against interface
type ImplementationMatch struct {
	IsComplete bool
	IsAlmost   bool // Missing 1-2 methods
	Matched    []*InterfaceMethod
	Missing    []*InterfaceMethod
}

// checkInterfaceImplementation checks if a type implements an interface
func checkInterfaceImplementation(typ *Type, iface *Interface) ImplementationMatch {
	match := ImplementationMatch{
		Matched: make([]*InterfaceMethod, 0),
		Missing: make([]*InterfaceMethod, 0),
	}

	// Build method map for type
	typeMethods := make(map[string]*Function)
	for _, method := range typ.Methods {
		typeMethods[method.Name] = method
	}

	// Check each interface method
	for _, ifaceMethod := range iface.Methods {
		typeMethod, exists := typeMethods[ifaceMethod.Name]

		if !exists {
			match.Missing = append(match.Missing, ifaceMethod)
			continue
		}

		// Check signature compatibility (simplified)
		if methodSignaturesMatch(typeMethod, ifaceMethod) {
			match.Matched = append(match.Matched, ifaceMethod)
		} else {
			match.Missing = append(match.Missing, ifaceMethod)
		}
	}

	// Determine match type
	if len(match.Missing) == 0 {
		match.IsComplete = true
	} else if len(match.Missing) <= 2 && len(match.Matched) > 0 {
		match.IsAlmost = true
	}

	return match
}

// methodSignaturesMatch checks if method signatures are compatible (simplified)
func methodSignaturesMatch(typeMethod *Function, ifaceMethod *InterfaceMethod) bool {
	// Simplified signature matching
	if typeMethod.Name != ifaceMethod.Name {
		return false
	}

	// Check param count
	typeParamCount := len(typeMethod.Parameters)
	ifaceParamCount := len(ifaceMethod.Params)

	if typeParamCount != ifaceParamCount {
		return false
	}

	// Check return count
	typeReturnCount := len(typeMethod.Returns)
	ifaceReturnCount := len(ifaceMethod.Returns)

	if typeReturnCount != ifaceReturnCount {
		return false
	}

	return true
}

// isInterfaceType checks if a type is an interface
func isInterfaceType(typ *Type) bool {
	// An interface has methods but no fields, and kind is typically "interface"
	return strings.Contains(strings.ToLower(typ.Kind), "interface") ||
		(len(typ.Methods) > 0 && len(typ.Fields) == 0)
}

// extractInterfaceMethods extracts method signatures from an interface type
func extractInterfaceMethods(typ *Type) []*InterfaceMethod {
	methods := make([]*InterfaceMethod, 0, len(typ.Methods))

	for _, method := range typ.Methods {
		ifaceMethod := &InterfaceMethod{
			Name:      method.Name,
			Signature: method.Signature,
			Params:    make([]string, len(method.Parameters)),
			Returns:   make([]string, len(method.Returns)),
		}

		for i, param := range method.Parameters {
			ifaceMethod.Params[i] = param.Type
		}

		for i, ret := range method.Returns {
			ifaceMethod.Returns[i] = ret.Type
		}

		methods = append(methods, ifaceMethod)
	}

	return methods
}

// calculateInterfaceStats computes summary metrics
func calculateInterfaceStats(analysis *InterfaceAnalysis) InterfaceStats {
	stats := InterfaceStats{}

	for _, iface := range analysis.Interfaces {
		stats.TotalInterfaces++
		if iface.IsProject {
			stats.ProjectInterfaces++
		} else {
			stats.StdlibInterfaces++
		}
		stats.TotalImplementers += iface.Implementers
	}

	if stats.TotalInterfaces > 0 {
		stats.AvgImplementers = float64(stats.TotalImplementers) / float64(stats.TotalInterfaces)
	}

	stats.UnusedCount = len(analysis.UnusedInterfaces)

	for _, almosts := range analysis.AlmostImplements {
		stats.AlmostCount += len(almosts)
	}

	return stats
}

// isExported checks if a name is exported (starts with uppercase)
func isExported(name string) bool {
	if name == "" {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}
