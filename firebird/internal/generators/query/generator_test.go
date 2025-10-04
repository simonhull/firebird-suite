package query

import (
	"testing"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"github.com/stretchr/testify/assert"
)

func TestPrepareRelationshipData(t *testing.T) {
	def := &schema.Definition{
		Name: "Post",
		Spec: schema.Spec{
			Fields: []schema.Field{
				{Name: "id", DBType: "UUID", PrimaryKey: true},
				{Name: "author_id", DBType: "UUID"},
			},
			Relationships: []schema.Relationship{
				{
					Name:       "Author",
					Type:       "belongs_to",
					Model:      "User",
					ForeignKey: "author_id",
				},
				{
					Name:       "Comments",
					Type:       "has_many",
					Model:      "Comment",
					ForeignKey: "post_id",
				},
			},
		},
	}

	relationships := prepareRelationshipData(def)

	assert.Len(t, relationships, 2)

	// Test belongs_to
	assert.Equal(t, "Author", relationships[0].Name)
	assert.Equal(t, "belongs_to", relationships[0].Type)
	assert.Equal(t, "GetPostAuthor", relationships[0].GetSingleQueryName)
	assert.Equal(t, "users", relationships[0].TargetTable)
	assert.Equal(t, "uuid", relationships[0].PrimaryKeyType)
	assert.Equal(t, "author_id", relationships[0].ForeignKey)

	// Test has_many
	assert.Equal(t, "Comments", relationships[1].Name)
	assert.Equal(t, "has_many", relationships[1].Type)
	assert.Equal(t, "GetPostComments", relationships[1].GetSingleQueryName)
	assert.Equal(t, "GetCommentsForPosts", relationships[1].GetManyQueryName)
	assert.Equal(t, "comments", relationships[1].TargetTable)
	assert.Equal(t, "post_id", relationships[1].ForeignKey)
}

func TestGetPrimaryKeyDBType(t *testing.T) {
	tests := []struct {
		name     string
		def      *schema.Definition
		expected string
	}{
		{
			name: "UUID primary key",
			def: &schema.Definition{
				Spec: schema.Spec{
					Fields: []schema.Field{
						{Name: "id", DBType: "UUID", PrimaryKey: true},
					},
				},
			},
			expected: "uuid",
		},
		{
			name: "BIGINT primary key",
			def: &schema.Definition{
				Spec: schema.Spec{
					Fields: []schema.Field{
						{Name: "id", DBType: "BIGINT", PrimaryKey: true},
					},
				},
			},
			expected: "bigint",
		},
		{
			name: "INTEGER primary key",
			def: &schema.Definition{
				Spec: schema.Spec{
					Fields: []schema.Field{
						{Name: "id", DBType: "INTEGER", PrimaryKey: true},
					},
				},
			},
			expected: "integer",
		},
		{
			name: "No primary key (fallback)",
			def: &schema.Definition{
				Spec: schema.Spec{
					Fields: []schema.Field{
						{Name: "name", DBType: "VARCHAR(255)", PrimaryKey: false},
					},
				},
			},
			expected: "bigint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPrimaryKeyDBType(tt.def)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrepareRelationshipDataWithTableName(t *testing.T) {
	def := &schema.Definition{
		Name: "Post",
		Spec: schema.Spec{
			TableName: "custom_posts",
			Fields: []schema.Field{
				{Name: "id", DBType: "BIGINT", PrimaryKey: true},
			},
			Relationships: []schema.Relationship{
				{
					Name:       "Author",
					Type:       "belongs_to",
					Model:      "User",
					ForeignKey: "author_id",
				},
			},
		},
	}

	relationships := prepareRelationshipData(def)

	assert.Len(t, relationships, 1)
	assert.Equal(t, "custom_posts", relationships[0].SourceTable)
	assert.Equal(t, "bigint", relationships[0].PrimaryKeyType)
}
