package module

// ConfigField represents a single field in a module's configuration struct.
// It is used by modules to declaratively specify their configuration schema,
// which will be added to the project's internal/config/config.go file.
type ConfigField struct {
	// Name is the field name in PascalCase (e.g., "JWTSecret", "TokenExpiry")
	Name string

	// Type is the Go type as a string (e.g., "string", "int", "time.Duration", "*bool")
	// Qualified types are supported (e.g., "time.Duration", "pkg.Type")
	Type string

	// Tag is the struct tag, typically for YAML/JSON serialization
	// Example: `yaml:"jwt_secret" json:"jwt_secret"`
	Tag string

	// Doc is an optional documentation comment for the field
	// It will appear as a comment above the field in generated code
	Doc string
}
