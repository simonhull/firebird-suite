package query

import (
	"testing"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"github.com/stretchr/testify/assert"
)

func TestCursorPaginationSupport(t *testing.T) {
	def := &schema.Definition{
		Name: "Post",
		Spec: schema.Spec{
			Fields: []schema.Field{
				{Name: "id", DBType: "UUID", PrimaryKey: true},
				{Name: "created_at", DBType: "TIMESTAMPTZ"},
			},
			Pagination: &schema.PaginationConfig{
				Type:         "cursor",
				CursorField:  "created_at",
				DefaultLimit: 20,
				MaxLimit:     100,
			},
		},
	}

	gen := New("/test/project", "/test/schema.firebird.yml")
	data := gen.templateData(def)

	assert.True(t, data["SupportsCursorPagination"].(bool))
	assert.Equal(t, "created_at", data["CursorField"])
	assert.Equal(t, "timestamp", data["CursorFieldType"])
}

func TestBothPaginationTypes(t *testing.T) {
	def := &schema.Definition{
		Name: "Post",
		Spec: schema.Spec{
			Fields: []schema.Field{
				{Name: "id", DBType: "BIGINT", PrimaryKey: true},
			},
			Pagination: &schema.PaginationConfig{
				Type: "both",
			},
		},
	}

	gen := New("/test/project", "/test/schema.firebird.yml")
	data := gen.templateData(def)

	assert.True(t, data["SupportsCursorPagination"].(bool))
	assert.Equal(t, "id", data["CursorField"])
	assert.Equal(t, "bigint", data["CursorFieldType"])
}

func TestNoPaginationConfig(t *testing.T) {
	def := &schema.Definition{
		Name: "Post",
		Spec: schema.Spec{
			Fields: []schema.Field{
				{Name: "id", DBType: "UUID", PrimaryKey: true},
			},
		},
	}

	gen := New("/test/project", "/test/schema.firebird.yml")
	data := gen.templateData(def)

	assert.False(t, data["SupportsCursorPagination"].(bool))
}

func TestGetCursorFieldType(t *testing.T) {
	tests := []struct {
		name         string
		def          *schema.Definition
		cursorField  string
		expectedType string
	}{
		{
			name: "UUID cursor field",
			def: &schema.Definition{
				Spec: schema.Spec{
					Fields: []schema.Field{
						{Name: "id", DBType: "UUID"},
					},
				},
			},
			cursorField:  "id",
			expectedType: "uuid",
		},
		{
			name: "BIGINT cursor field",
			def: &schema.Definition{
				Spec: schema.Spec{
					Fields: []schema.Field{
						{Name: "id", DBType: "BIGINT"},
					},
				},
			},
			cursorField:  "id",
			expectedType: "bigint",
		},
		{
			name: "TIMESTAMP cursor field",
			def: &schema.Definition{
				Spec: schema.Spec{
					Fields: []schema.Field{
						{Name: "created_at", DBType: "TIMESTAMPTZ"},
					},
				},
			},
			cursorField:  "created_at",
			expectedType: "timestamp",
		},
		{
			name: "Non-existent field (fallback)",
			def: &schema.Definition{
				Spec: schema.Spec{
					Fields: []schema.Field{
						{Name: "name", DBType: "VARCHAR"},
					},
				},
			},
			cursorField:  "id",
			expectedType: "bigint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCursorFieldType(tt.def, tt.cursorField)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestPermanentDeleteGeneration(t *testing.T) {
	def := &schema.Definition{
		Name: "Post",
		Spec: schema.Spec{
			SoftDeletes: true,
			Fields: []schema.Field{
				{Name: "id", DBType: "UUID", PrimaryKey: true},
			},
		},
	}

	gen := New("/test/project", "/test/schema.firebird.yml")
	data := gen.templateData(def)

	assert.True(t, data["SoftDeletes"].(bool))
	// The template will generate PermanentDelete query when SoftDeletes is true
}
