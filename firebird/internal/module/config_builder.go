package module

import (
	"fmt"
	"strings"

	"github.com/simonhull/firebird-suite/fledge/astutil"
	"github.com/simonhull/firebird-suite/fledge/generator"
)

// ConfigBuilder provides a high-level API for adding module configuration
// to internal/config/config.go using safe AST modifications.
//
// It wraps the low-level astutil package with a module-friendly interface
// that handles common patterns like:
//   - Ensuring the Config struct has a Modules field
//   - Creating ModulesConfig if it doesn't exist
//   - Adding module-specific config structs
//   - Managing imports (e.g., time package)
//
// All operations are idempotent - safe to call multiple times.
type ConfigBuilder struct {
	filePath string
	mods     []astutil.ModificationSpec
}

// NewConfigBuilder creates a new config builder for the given file path.
// The file must be a valid Go source file containing a Config struct.
func NewConfigBuilder(filePath string) *ConfigBuilder {
	return &ConfigBuilder{
		filePath: filePath,
		mods:     []astutil.ModificationSpec{},
	}
}

// EnsureModulesField adds the Modules field to the Config struct if it doesn't exist.
// It also creates the ModulesConfig type if needed.
//
// This method is idempotent - calling it multiple times has the same effect as calling once.
//
// After calling this method, the config file will have:
//   type Config struct {
//       ...existing fields...
//       Modules ModulesConfig `yaml:"modules"`
//   }
//
//   type ModulesConfig struct {
//   }
func (b *ConfigBuilder) EnsureModulesField() error {
	// Check if Config.Modules field already exists
	hasModules, err := astutil.HasField(b.filePath, "Config", "Modules")
	if err != nil {
		return fmt.Errorf("checking for Modules field: %w", err)
	}

	if hasModules {
		// Field already exists, nothing to do (idempotent)
		return nil
	}

	// Add Modules field to Config struct
	b.mods = append(b.mods, astutil.ModificationSpec{
		Type: "add_struct_field",
		Params: map[string]interface{}{
			"struct_name": "Config",
			"field_name":  "Modules",
			"field_type":  "ModulesConfig",
			"tag":         `yaml:"modules"`,
		},
	})

	// Check if ModulesConfig type exists
	hasType, err := astutil.HasTypeDecl(b.filePath, "ModulesConfig")
	if err != nil {
		return fmt.Errorf("checking for ModulesConfig type: %w", err)
	}

	if !hasType {
		// Create empty ModulesConfig struct
		typeSpec := astutil.BuildEmptyStruct("ModulesConfig")

		b.mods = append(b.mods, astutil.ModificationSpec{
			Type: "add_type_decl",
			Params: map[string]interface{}{
				"type_spec":  typeSpec,
				"position":   astutil.PositionAfter,
				"after_type": "Config",
			},
		})
	}

	return nil
}

// AddModuleConfig adds a module-specific config struct and wires it into ModulesConfig.
//
// The moduleName should be in PascalCase (e.g., "Falcon", "Owl").
// A config struct named <ModuleName>Config will be created with the specified fields.
//
// This method is idempotent - if the module config already exists, it does nothing.
//
// Example:
//   builder.AddModuleConfig("Falcon", []ConfigField{
//       {Name: "JWTSecret", Type: "string", Tag: `yaml:"jwt_secret"`},
//       {Name: "TokenExpiry", Type: "time.Duration", Tag: `yaml:"token_expiry"`},
//   })
//
// Results in:
//   type ModulesConfig struct {
//       Falcon FalconConfig `yaml:"falcon"`
//   }
//
//   type FalconConfig struct {
//       JWTSecret   string        `yaml:"jwt_secret"`
//       TokenExpiry time.Duration `yaml:"token_expiry"`
//   }
func (b *ConfigBuilder) AddModuleConfig(moduleName string, fields []ConfigField) error {
	configTypeName := moduleName + "Config"

	// Check if module config already exists
	hasConfig, err := astutil.HasTypeDecl(b.filePath, configTypeName)
	if err != nil {
		return fmt.Errorf("checking for %s type: %w", configTypeName, err)
	}

	if hasConfig {
		// Config already exists, nothing to do (idempotent)
		return nil
	}

	// Add field to ModulesConfig struct
	moduleFieldName := moduleName
	yamlTag := toSnakeCase(moduleName)

	b.mods = append(b.mods, astutil.ModificationSpec{
		Type: "add_struct_field",
		Params: map[string]interface{}{
			"struct_name": "ModulesConfig",
			"field_name":  moduleFieldName,
			"field_type":  configTypeName,
			"tag":         fmt.Sprintf(`yaml:"%s"`, yamlTag),
		},
	})

	// Build the module config struct
	needsTimeImport := false

	configStruct := astutil.NewStruct(configTypeName).
		Doc(fmt.Sprintf("%s holds configuration for the %s module", configTypeName, moduleName))

	for _, cf := range fields {
		field := astutil.NewField(cf.Name).
			Type(cf.Type).
			Tag(cf.Tag)

		if cf.Doc != "" {
			field = field.Doc(cf.Doc)
		}

		configStruct.AddField(field.Build())

		// Check if we need to import time package
		if strings.Contains(cf.Type, "time.") {
			needsTimeImport = true
		}
	}

	b.mods = append(b.mods, astutil.ModificationSpec{
		Type: "add_type_decl",
		Params: map[string]interface{}{
			"type_spec":  configStruct.Build(),
			"position":   astutil.PositionAfter,
			"after_type": "ModulesConfig",
		},
	})

	// Add time import if needed
	if needsTimeImport {
		hasTimeImport, err := astutil.HasImport(b.filePath, "time")
		if err != nil {
			return fmt.Errorf("checking for time import: %w", err)
		}

		if !hasTimeImport {
			b.mods = append(b.mods, astutil.ModificationSpec{
				Type: "add_import",
				Params: map[string]interface{}{
					"path":  "time",
					"alias": "",
				},
			})
		}
	}

	return nil
}

// Build returns the generator operations to execute the config modifications.
// If no modifications have been queued (e.g., all operations were idempotent),
// it returns an empty slice.
func (b *ConfigBuilder) Build() ([]generator.Operation, error) {
	if len(b.mods) == 0 {
		// No modifications needed
		return []generator.Operation{}, nil
	}

	return []generator.Operation{
		&astutil.ASTModifyOp{
			Path:          b.filePath,
			Modifications: b.mods,
		},
	}, nil
}

// toSnakeCase converts PascalCase to snake_case for YAML tags.
//
// Examples:
//   "Falcon" -> "falcon"
//   "FalconAuth" -> "falcon_auth"
//   "APIKey" -> "a_p_i_key"
func toSnakeCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}
