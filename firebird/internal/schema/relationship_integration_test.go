package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRealSchemaWithRelationships(t *testing.T) {
	// Create a temporary schema file with relationships
	tmpDir := t.TempDir()

	userSchema := `apiVersion: v1
kind: Resource
name: User
spec:
  fields:
    - name: id
      type: uuid.UUID
      db_type: UUID
      primary_key: true
    - name: email
      type: string
      db_type: VARCHAR(255)
  relationships:
    - name: Posts
      type: has_many
      model: Post
      foreign_key: author_id
  timestamps: true
`

	userPath := filepath.Join(tmpDir, "User.firebird.yml")
	err := os.WriteFile(userPath, []byte(userSchema), 0644)
	require.NoError(t, err)

	// Parse the schema
	def, err := Parse(userPath)
	require.NoError(t, err)
	require.NotNil(t, def)

	// Verify basic structure
	assert.Equal(t, "User", def.Name)
	assert.Len(t, def.Spec.Fields, 2)
	assert.Len(t, def.Spec.Relationships, 1)

	// Verify relationship
	rel := def.Spec.Relationships[0]
	assert.Equal(t, "Posts", rel.Name)
	assert.Equal(t, "has_many", rel.Type)
	assert.Equal(t, "Post", rel.Model)
	assert.Equal(t, "author_id", rel.ForeignKey)
}

func TestParseRealSchemaWithBelongsTo(t *testing.T) {
	tmpDir := t.TempDir()

	postSchema := `apiVersion: v1
kind: Resource
name: Post
spec:
  fields:
    - name: id
      type: uuid.UUID
      db_type: UUID
      primary_key: true
    - name: author_id
      type: uuid.UUID
      db_type: UUID
    - name: title
      type: string
      db_type: VARCHAR(255)
  relationships:
    - name: Author
      type: belongs_to
      model: User
      foreign_key: author_id
  timestamps: true
`

	postPath := filepath.Join(tmpDir, "Post.firebird.yml")
	err := os.WriteFile(postPath, []byte(postSchema), 0644)
	require.NoError(t, err)

	// Parse the schema
	def, err := Parse(postPath)
	require.NoError(t, err)
	require.NotNil(t, def)

	// Verify basic structure
	assert.Equal(t, "Post", def.Name)
	assert.Len(t, def.Spec.Fields, 3)
	assert.Len(t, def.Spec.Relationships, 1)

	// Verify relationship
	rel := def.Spec.Relationships[0]
	assert.Equal(t, "Author", rel.Name)
	assert.Equal(t, "belongs_to", rel.Type)
	assert.Equal(t, "User", rel.Model)
	assert.Equal(t, "author_id", rel.ForeignKey)
}

func TestParseSchemaWithInvalidRelationship(t *testing.T) {
	tmpDir := t.TempDir()

	// Schema with belongs_to but missing foreign key field
	invalidSchema := `apiVersion: v1
kind: Resource
name: Post
spec:
  fields:
    - name: id
      type: uuid.UUID
      db_type: UUID
      primary_key: true
    - name: title
      type: string
      db_type: VARCHAR(255)
  relationships:
    - name: Author
      type: belongs_to
      model: User
      foreign_key: author_id
`

	schemaPath := filepath.Join(tmpDir, "Post.firebird.yml")
	err := os.WriteFile(schemaPath, []byte(invalidSchema), 0644)
	require.NoError(t, err)

	// Parse should fail
	def, err := Parse(schemaPath)
	assert.Nil(t, def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "foreign key field 'author_id' not found")
}

func TestParseSchemaWithMultipleRelationships(t *testing.T) {
	tmpDir := t.TempDir()

	commentSchema := `apiVersion: v1
kind: Resource
name: Comment
spec:
  fields:
    - name: id
      type: uuid.UUID
      db_type: UUID
      primary_key: true
    - name: post_id
      type: uuid.UUID
      db_type: UUID
    - name: author_id
      type: uuid.UUID
      db_type: UUID
    - name: content
      type: string
      db_type: TEXT
  relationships:
    - name: Post
      type: belongs_to
      model: Post
      foreign_key: post_id
    - name: Author
      type: belongs_to
      model: User
      foreign_key: author_id
  timestamps: true
`

	commentPath := filepath.Join(tmpDir, "Comment.firebird.yml")
	err := os.WriteFile(commentPath, []byte(commentSchema), 0644)
	require.NoError(t, err)

	// Parse the schema
	def, err := Parse(commentPath)
	require.NoError(t, err)
	require.NotNil(t, def)

	// Verify basic structure
	assert.Equal(t, "Comment", def.Name)
	assert.Len(t, def.Spec.Fields, 4)
	assert.Len(t, def.Spec.Relationships, 2)

	// Verify first relationship
	rel1 := def.Spec.Relationships[0]
	assert.Equal(t, "Post", rel1.Name)
	assert.Equal(t, "belongs_to", rel1.Type)
	assert.Equal(t, "Post", rel1.Model)
	assert.Equal(t, "post_id", rel1.ForeignKey)

	// Verify second relationship
	rel2 := def.Spec.Relationships[1]
	assert.Equal(t, "Author", rel2.Name)
	assert.Equal(t, "belongs_to", rel2.Type)
	assert.Equal(t, "User", rel2.Model)
	assert.Equal(t, "author_id", rel2.ForeignKey)
}
