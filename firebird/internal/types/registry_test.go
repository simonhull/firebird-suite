package types_test

import (
	"testing"

	"github.com/simonhull/firebird-suite/firebird/internal/types"
)

func TestLookup(t *testing.T) {
	tests := []struct {
		typeName string
		wantOK   bool
	}{
		{"string", true},
		{"UUID", true},
		{"Decimal", true},
		{"NullString", true},
		{"timestamp", true},
		{"int64", true},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			_, ok := types.Lookup(tt.typeName)
			if ok != tt.wantOK {
				t.Errorf("Lookup(%q) ok = %v, want %v", tt.typeName, ok, tt.wantOK)
			}
		})
	}
}

func TestGetDBType(t *testing.T) {
	tests := []struct {
		typeName string
		driver   string
		want     string
		wantErr  bool
	}{
		{"string", "postgres", "VARCHAR(255)", false},
		{"string", "mysql", "VARCHAR(255)", false},
		{"string", "sqlite", "TEXT", false},
		{"UUID", "postgres", "UUID", false},
		{"UUID", "mysql", "CHAR(36)", false},
		{"UUID", "sqlite", "TEXT", false},
		{"Decimal", "postgres", "NUMERIC(19,4)", false},
		{"Decimal", "mysql", "DECIMAL(19,4)", false},
		{"Decimal", "sqlite", "TEXT", false},
		{"text", "postgres", "TEXT", false},
		{"int64", "postgres", "BIGINT", false},
		{"bool", "mysql", "TINYINT(1)", false},
		{"timestamp", "sqlite", "DATETIME", false},
		{"unknown", "postgres", "", true},
		{"string", "invaliddriver", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.typeName+"_"+tt.driver, func(t *testing.T) {
			got, err := types.GetDBType(tt.typeName, tt.driver)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDBType(%q, %q) error = %v, wantErr %v", tt.typeName, tt.driver, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetDBType(%q, %q) = %q, want %q", tt.typeName, tt.driver, got, tt.want)
			}
		})
	}
}

func TestGetPrimaryKeyType(t *testing.T) {
	tests := []struct {
		typeName string
		driver   string
		want     string
		wantErr  bool
	}{
		{"UUID", "postgres", "UUID", false},
		{"UUID", "mysql", "CHAR(36)", false},
		{"UUID", "sqlite", "TEXT", false},
		{"int64", "postgres", "BIGSERIAL", false},
		{"int64", "mysql", "BIGINT AUTO_INCREMENT", false},
		{"int64", "sqlite", "INTEGER", false},
		{"string", "postgres", "", true},  // Not an ID type
		{"unknown", "postgres", "", true}, // Unknown type
	}

	for _, tt := range tests {
		t.Run(tt.typeName+"_"+tt.driver, func(t *testing.T) {
			got, err := types.GetPrimaryKeyType(tt.typeName, tt.driver)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPrimaryKeyType(%q, %q) error = %v, wantErr %v", tt.typeName, tt.driver, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetPrimaryKeyType(%q, %q) = %q, want %q", tt.typeName, tt.driver, got, tt.want)
			}
		})
	}
}

func TestGetGoType(t *testing.T) {
	tests := []struct {
		typeName   string
		wantType   string
		wantImport string
		wantErr    bool
	}{
		{"string", "string", "", false},
		{"text", "string", "", false},
		{"int", "int", "", false},
		{"int64", "int64", "", false},
		{"bool", "bool", "", false},
		{"UUID", "uuid.UUID", "github.com/google/uuid", false},
		{"Decimal", "decimal.Decimal", "github.com/shopspring/decimal", false},
		{"NullString", "sql.NullString", "database/sql", false},
		{"timestamp", "time.Time", "time", false},
		{"date", "time.Time", "time", false},
		{"time", "time.Time", "time", false},
		{"unknown", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			gotType, gotImport, err := types.GetGoType(tt.typeName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGoType(%q) unexpected error: %v", tt.typeName, err)
				return
			}
			if gotType != tt.wantType {
				t.Errorf("GetGoType(%q) type = %q, want %q", tt.typeName, gotType, tt.wantType)
			}
			if gotImport != tt.wantImport {
				t.Errorf("GetGoType(%q) import = %q, want %q", tt.typeName, gotImport, tt.wantImport)
			}
		})
	}
}

func TestCollectImports(t *testing.T) {
	tests := []struct {
		name      string
		typeNames []string
		want      []string
	}{
		{
			name:      "no imports",
			typeNames: []string{"string", "int", "bool"},
			want:      []string{},
		},
		{
			name:      "single import",
			typeNames: []string{"string", "UUID"},
			want:      []string{"github.com/google/uuid"},
		},
		{
			name:      "multiple imports",
			typeNames: []string{"string", "UUID", "Decimal", "timestamp"},
			want: []string{
				"github.com/google/uuid",
				"github.com/shopspring/decimal",
				"time",
			},
		},
		{
			name:      "duplicate types",
			typeNames: []string{"UUID", "UUID", "timestamp", "date", "time"},
			want: []string{
				"github.com/google/uuid",
				"time",
			},
		},
		{
			name:      "with unknown types",
			typeNames: []string{"string", "UUID", "unknown", "Decimal"},
			want: []string{
				"github.com/google/uuid",
				"github.com/shopspring/decimal",
			},
		},
		{
			name:      "empty list",
			typeNames: []string{},
			want:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := types.CollectImports(tt.typeNames)

			if len(got) != len(tt.want) {
				t.Errorf("CollectImports(%v) got %d imports, want %d\nGot: %v\nWant: %v",
					tt.typeNames, len(got), len(tt.want), got, tt.want)
				return
			}

			for i, imp := range got {
				if imp != tt.want[i] {
					t.Errorf("imports[%d] = %q, want %q", i, imp, tt.want[i])
				}
			}
		})
	}
}

func TestTypeInfoIDTypes(t *testing.T) {
	// Verify that only appropriate types can be used as IDs
	idTypes := []string{"UUID", "int64"}
	nonIDTypes := []string{"string", "text", "int", "bool", "Decimal", "timestamp"}

	for _, typeName := range idTypes {
		info, ok := types.Lookup(typeName)
		if !ok {
			t.Errorf("ID type %q not found in registry", typeName)
			continue
		}
		if !info.IsIDType {
			t.Errorf("Type %q should be marked as IsIDType", typeName)
		}
	}

	for _, typeName := range nonIDTypes {
		info, ok := types.Lookup(typeName)
		if !ok {
			continue // Unknown types are fine to skip
		}
		if info.IsIDType {
			t.Errorf("Type %q should not be marked as IsIDType", typeName)
		}
	}
}

func TestTypeInfoDBTypesComplete(t *testing.T) {
	// Verify all types have db_type entries for all supported drivers
	drivers := []string{"postgres", "mysql", "sqlite"}

	for typeName, info := range types.Registry {
		for _, driver := range drivers {
			if _, ok := info.DBTypes[driver]; !ok {
				t.Errorf("Type %q missing db_type for driver %q", typeName, driver)
			}
		}
	}
}
