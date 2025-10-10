package generator

import (
	"html/template"
	"time"
)

// SiteData represents the entire documentation site
type SiteData struct {
	// Project metadata
	ProjectName    string
	ModulePath     string
	GoVersion      string
	GeneratedAt    time.Time
	IsFirebird     bool
	FirebirdInfo   *FirebirdInfo

	// Documentation content
	Packages    []*PackageData
	AllTypes    []*TypeData
	AllFunctions []*FunctionData

	// Statistics
	Stats *SiteStats

	// Navigation
	ConventionGroups []*ConventionGroup

	// README content
	ReadmeHTML template.HTML // Parsed README content (marked as safe)
	HasReadme  bool          // Whether README exists
}

// PackageData represents a single Go package
type PackageData struct {
	Name        string
	Path        string
	ImportPath  string
	Description string

	// Content
	Types     []*TypeData
	Functions []*FunctionData
	Constants []*ConstantData
	Variables []*VariableData

	// Convention detection
	PrimaryConvention string // e.g., "handler", "service", "repository"

	// Documentation
	DocHTML template.HTML // Parsed package documentation (README or doc.go)
	HasDoc  bool          // Whether package has documentation

	// Metrics
	Metrics *PackageMetrics // Computed package metrics

	// Call Graph
	CallGraph template.HTML // Rendered SVG call graph
	HasGraph  bool          // Whether graph exists
}

// ConventionGroup groups types by architectural convention
type ConventionGroup struct {
	Name        string // e.g., "Handlers", "Services", "Repositories"
	Slug        string // e.g., "handlers", "services", "repositories"
	Description string
	Types       []*TypeData
	Count       int
}

// TypeData represents a Go type (struct, interface, etc.)
type TypeData struct {
	Name        string
	Kind        string // "struct", "interface", "type alias"
	Package     string
	ImportPath  string
	Description string

	// Convention detection - SINGLE BADGE (highest confidence)
	PrimaryBadge *Badge

	// Content
	Fields  []*FieldData
	Methods []*MethodData

	// Dependencies
	UsedTypes     []string // Types this type depends on
	UsedByTypes   []string // Types that depend on this type
	UsedFunctions []string // Functions this type calls

	// Source location
	File         string
	LineNumber   int
	RelativePath string
}

// FieldData represents a struct field
type FieldData struct {
	Name        string
	Type        string
	Tag         string
	Description string
	Exported    bool
}

// MethodData represents a type method
type MethodData struct {
	Name        string
	Receiver    string
	Signature   string
	Description string
	Parameters  []*ParameterData
	Returns     []*ReturnData
	Exported    bool

	// Dependencies
	CallsFunctions []string
	UsesTypes      []string

	// Source location
	File         string
	LineNumber   int
	RelativePath string
}

// FunctionData represents a package-level function
type FunctionData struct {
	Name        string
	Package     string
	Signature   string
	Description string
	Parameters  []*ParameterData
	Returns     []*ReturnData
	Exported    bool

	// Dependencies
	CallsFunctions []string // Functions this function calls
	CalledBy       []string // Functions that call this function
	UsesTypes      []string

	// Source location
	File         string
	LineNumber   int
	RelativePath string
}

// ParameterData represents a function parameter
type ParameterData struct {
	Name string
	Type string
}

// ReturnData represents a function return value
type ReturnData struct {
	Name string
	Type string
}

// ConstantData represents a package constant
type ConstantData struct {
	Name        string
	Type        string
	Value       string
	Description string
}

// VariableData represents a package variable
type VariableData struct {
	Name        string
	Type        string
	Description string
}

// Badge represents a convention match (single per type)
type Badge struct {
	Type       string  // "handler", "service", "repository", "dto", "middleware"
	Label      string  // Display text: "Handler", "Service", etc.
	Confidence float64 // 0.0-1.0
	Color      string  // CSS color class: "badge-handler", "badge-service"
}

// FirebirdInfo contains Firebird-specific metadata
type FirebirdInfo struct {
	Database string // "postgres", "mysql", "sqlite", "none"
	Router   string // "stdlib", "chi", "gin", "echo", "none"
	Version  string
}

// SiteStats contains project statistics
type SiteStats struct {
	TotalPackages   int
	TotalTypes      int
	TotalFunctions  int
	TotalInterfaces int
	TotalStructs    int

	// Convention counts
	HandlerCount    int
	ServiceCount    int
	RepositoryCount int
	DTOCount        int
	MiddlewareCount int
	ModelCount      int

	// Code metrics
	AverageMethods    float64
	AverageFields     float64
	ExportedRatio     float64
	DocumentationRatio float64
}

// PackageMetrics represents computed metrics for a package
type PackageMetrics struct {
	// Basic counts
	TotalTypes     int
	TotalFunctions int
	TotalMethods   int
	LinesOfCode    int

	// Exported vs internal
	ExportedCount  int
	InternalCount  int

	// Imports
	TotalImports    int
	InternalImports int  // Same module
	ExternalImports int  // External dependencies

	// Convention distribution
	Conventions []*ConventionCount

	// Complexity distribution
	SimpleFunctions  int  // 0-3 calls
	MediumFunctions  int  // 4-7 calls
	ComplexFunctions int  // 8+ calls
}

// ConventionCount represents count of items following a convention
type ConventionCount struct {
	Name       string  // e.g., "handler", "service"
	Count      int
	Percentage float64
}
