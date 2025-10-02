// Package schema provides generic YAML schema parsing for Aerie tools.
//
// This package defines the common structure that all Aerie schemas share:
// apiVersion, kind, name, metadata, and a generic spec field.
//
// # Schema Structure
//
// Every Aerie schema has this basic structure:
//
//	apiVersion: v1
//	kind: <ResourceType>
//	name: <ResourceName>
//	metadata:
//	  # Optional tool-specific metadata
//	spec:
//	  # Tool-specific specification
//
// # Domain-Specific Schemas
//
// Domain packages define their own specific schema structures:
//
// - Firebird: Database resources with tables, fields, migrations
// - Talon: Authentication configs with providers, OAuth settings
// - Plume: Frontend components with routes, templates, events
//
// Each tool uses fledge/schema for parsing but defines its own spec structure.
//
// # Example Usage
//
//	// Parse any Aerie schema
//	def, err := schema.Parse("resource.yml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate basic structure
//	if err := schema.ValidateBasicStructure(def); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Domain packages convert to their specific types
//	// (See firebird/internal/schema for example)
package schema