package shared

import (
	"net/url"
	"testing"

	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/stretchr/testify/assert"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name     string
		query    url.Values
		expected struct {
			Page    int
			PerPage int
			Offset  int
		}
	}{
		{
			name:  "defaults",
			query: url.Values{},
			expected: struct {
				Page    int
				PerPage int
				Offset  int
			}{
				Page:    1,
				PerPage: 20,
				Offset:  0,
			},
		},
		{
			name: "custom page",
			query: url.Values{
				"page":     []string{"3"},
				"per_page": []string{"50"},
			},
			expected: struct {
				Page    int
				PerPage int
				Offset  int
			}{
				Page:    3,
				PerPage: 50,
				Offset:  100,
			},
		},
		{
			name: "exceeds max",
			query: url.Values{
				"per_page": []string{"500"},
			},
			expected: struct {
				Page    int
				PerPage int
				Offset  int
			}{
				Page:    1,
				PerPage: 100, // capped at max
				Offset:  0,
			},
		},
		{
			name: "invalid values default",
			query: url.Values{
				"page":     []string{"invalid"},
				"per_page": []string{"-5"},
			},
			expected: struct {
				Page    int
				PerPage int
				Offset  int
			}{
				Page:    1,
				PerPage: 20,
				Offset:  0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate query helpers to get the Pagination struct
			gen := NewGenerator("/test/project", "github.com/test/project")
			ops, err := gen.Generate()
			assert.NoError(t, err)

			// Find query.go operation
			var queryOp *generator.WriteFileOp
			for _, op := range ops {
				if writeOp, ok := op.(*generator.WriteFileOp); ok {
					if writeOp.Path == "/test/project/internal/helpers/query.go" {
						queryOp = writeOp
						break
					}
				}
			}

			assert.NotNil(t, queryOp, "query.go operation should be generated")
			assert.Contains(t, string(queryOp.Content), "func ParsePagination")
			assert.Contains(t, string(queryOp.Content), "type Pagination struct")
		})
	}
}

func TestParseSortGeneration(t *testing.T) {
	gen := NewGenerator("/test/project", "github.com/test/project")
	ops, err := gen.Generate()
	assert.NoError(t, err)

	// Find query.go operation
	var queryOp *generator.WriteFileOp
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/helpers/query.go" {
				queryOp = writeOp
				break
			}
		}
	}

	assert.NotNil(t, queryOp, "query.go operation should be generated")
	content := string(queryOp.Content)

	// Test ParseSort function exists with correct features
	assert.Contains(t, content, "func ParseSort")
	assert.Contains(t, content, "type SortOrder struct")
	assert.Contains(t, content, "Field     string")
	assert.Contains(t, content, "Direction string")

	// Test prefix notation support
	assert.Contains(t, content, `strings.HasPrefix(part, "-")`)
	assert.Contains(t, content, `strings.HasPrefix(part, "+")`)

	// Test colon notation support
	assert.Contains(t, content, `strings.Contains(part, ":")`)

	// Test validation
	assert.Contains(t, content, "allowedFields")
	assert.Contains(t, content, `direction != "ASC" && direction != "DESC"`)
}

func TestParseFiltersGeneration(t *testing.T) {
	gen := NewGenerator("/test/project", "github.com/test/project")
	ops, err := gen.Generate()
	assert.NoError(t, err)

	// Find query.go operation
	var queryOp *generator.WriteFileOp
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/helpers/query.go" {
				queryOp = writeOp
				break
			}
		}
	}

	assert.NotNil(t, queryOp, "query.go operation should be generated")
	content := string(queryOp.Content)

	// Test ParseFilters function
	assert.Contains(t, content, "func ParseFilters")
	assert.Contains(t, content, "type Filter struct")
	assert.Contains(t, content, "Field    string")
	assert.Contains(t, content, "Operator string")
	assert.Contains(t, content, "Value    interface{}")

	// Test operators
	assert.Contains(t, content, `// "=", "!=", ">", "<", ">=", "<=", "LIKE", "IN"`)

	// Test IN operator handling
	assert.Contains(t, content, `if operator == "IN"`)
	assert.Contains(t, content, "strings.Split")
}

func TestParseSearchGeneration(t *testing.T) {
	gen := NewGenerator("/test/project", "github.com/test/project")
	ops, err := gen.Generate()
	assert.NoError(t, err)

	// Find query.go operation
	var queryOp *generator.WriteFileOp
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/helpers/query.go" {
				queryOp = writeOp
				break
			}
		}
	}

	assert.NotNil(t, queryOp, "query.go operation should be generated")
	content := string(queryOp.Content)

	// Test ParseSearch function
	assert.Contains(t, content, "func ParseSearch")
	assert.Contains(t, content, "type SearchFilter struct")
	assert.Contains(t, content, "Query  string")
	assert.Contains(t, content, "Fields []string")

	// Test query parameter support
	assert.Contains(t, content, `values.Get("q")`)
	assert.Contains(t, content, `values.Get("search_fields")`)
}

func TestBuildOrderByClauseGeneration(t *testing.T) {
	gen := NewGenerator("/test/project", "github.com/test/project")
	ops, err := gen.Generate()
	assert.NoError(t, err)

	// Find query.go operation
	var queryOp *generator.WriteFileOp
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/helpers/query.go" {
				queryOp = writeOp
				break
			}
		}
	}

	assert.NotNil(t, queryOp, "query.go operation should be generated")
	content := string(queryOp.Content)

	// Test BuildOrderByClause function
	assert.Contains(t, content, "func BuildOrderByClause")
	assert.Contains(t, content, "ORDER BY")
	assert.Contains(t, content, "strings.Join")
}
