package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValidMinimal(t *testing.T) {
	path := filepath.Join("testdata", "valid_minimal.firebird.yml")
	def, err := Parse(path)

	require.NoError(t, err)
	require.NotNil(t, def)

	assert.Equal(t, "v1", def.APIVersion)
	assert.Equal(t, "Resource", def.Kind)
	assert.Equal(t, "User", def.Name)
	assert.Empty(t, def.Spec.TableName)
	require.Len(t, def.Spec.Fields, 1)

	field := def.Spec.Fields[0]
	assert.Equal(t, "id", field.Name)
	assert.Equal(t, "string", field.Type)
	assert.Equal(t, "UUID", field.DBType)
	assert.True(t, field.PrimaryKey)
}

func TestParseValidComplete(t *testing.T) {
	path := filepath.Join("testdata", "valid_complete.firebird.yml")
	def, err := Parse(path)

	require.NoError(t, err)
	require.NotNil(t, def)

	assert.Equal(t, "v1", def.APIVersion)
	assert.Equal(t, "Resource", def.Kind)
	assert.Equal(t, "BlogPost", def.Name)
	assert.Equal(t, "posts", def.Spec.TableName)
	require.Len(t, def.Spec.Fields, 11)

	// Check a few key fields
	idField := def.Spec.Fields[0]
	assert.Equal(t, "id", idField.Name)
	assert.Equal(t, "string", idField.Type)
	assert.Equal(t, "UUID", idField.DBType)
	assert.True(t, idField.PrimaryKey)

	titleField := def.Spec.Fields[1]
	assert.Equal(t, "title", titleField.Name)
	assert.Equal(t, "string", titleField.Type)
	assert.Equal(t, "VARCHAR(255)", titleField.DBType)
	assert.Equal(t, []string{"required", "min=3", "max=255"}, titleField.Validation)

	createdAtField := def.Spec.Fields[9]
	assert.Equal(t, "created_at", createdAtField.Name)
	assert.Equal(t, "time.Time", createdAtField.Type)
	assert.Equal(t, "TIMESTAMP", createdAtField.DBType)
	assert.True(t, createdAtField.AutoNowAdd)

	updatedAtField := def.Spec.Fields[10]
	assert.Equal(t, "updated_at", updatedAtField.Name)
	assert.Equal(t, "*time.Time", updatedAtField.Type)
	assert.Equal(t, "TIMESTAMP", updatedAtField.DBType)
	assert.True(t, updatedAtField.AutoNow)
	assert.True(t, updatedAtField.Nullable)
}

func TestParseBytes(t *testing.T) {
	data := []byte(`
apiVersion: v1
kind: Resource
name: Product
spec:
  fields:
    - name: id
      type: string
      db_type: UUID
      primary_key: true
    - name: name
      type: string
      db_type: VARCHAR(100)
    - name: price
      type: float64
      db_type: DECIMAL(10,2)
`)

	def, err := ParseBytes(data)

	require.NoError(t, err)
	require.NotNil(t, def)

	assert.Equal(t, "v1", def.APIVersion)
	assert.Equal(t, "Resource", def.Kind)
	assert.Equal(t, "Product", def.Name)
	require.Len(t, def.Spec.Fields, 3)
}

func TestParseMissingAPIVersion(t *testing.T) {
	path := filepath.Join("testdata", "invalid_missing_apiversion.firebird.yml")
	def, err := Parse(path)

	assert.Nil(t, def)
	require.Error(t, err)

	// Check that the error message contains validation error
	assert.Contains(t, err.Error(), "validation error")
	assert.Contains(t, err.Error(), "apiVersion")
	assert.Contains(t, err.Error(), "required")
}

func TestParseUnknownFields(t *testing.T) {
	path := filepath.Join("testdata", "invalid_unknown_field.firebird.yml")
	def, err := Parse(path)

	assert.Nil(t, def)
	require.Error(t, err)

	// Strict parsing should catch unknown fields
	assert.Contains(t, err.Error(), "unknown")
}

func TestParseBadType(t *testing.T) {
	path := filepath.Join("testdata", "invalid_bad_type.firebird.yml")
	def, err := Parse(path)

	assert.Nil(t, def)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "invalid Go type")
	assert.Contains(t, err.Error(), "InvalidType")
}

func TestParseNoPrimaryKey(t *testing.T) {
	path := filepath.Join("testdata", "invalid_no_primary_key.firebird.yml")
	def, err := Parse(path)

	assert.Nil(t, def)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "primary_key")
	assert.Contains(t, err.Error(), "at least one field must have")
}

func TestParseAutoNowBadType(t *testing.T) {
	path := filepath.Join("testdata", "invalid_auto_now_bad_type.firebird.yml")
	def, err := Parse(path)

	assert.Nil(t, def)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "auto_now")
	assert.Contains(t, err.Error(), "time.Time")
}

func TestValidateEmptyDefinition(t *testing.T) {
	def := &Definition{}
	err := Validate(def)

	require.Error(t, err)
	verrs, ok := err.(ValidationErrors)
	require.True(t, ok)

	// Should have multiple errors
	assert.True(t, len(verrs) > 0)

	// Check for specific required field errors
	errStr := err.Error()
	assert.Contains(t, errStr, "apiVersion")
	assert.Contains(t, errStr, "kind")
	assert.Contains(t, errStr, "name")
	assert.Contains(t, errStr, "fields")
}

func TestValidateInvalidAPIVersion(t *testing.T) {
	def := &Definition{
		APIVersion: "v2",
		Kind:       "Resource",
		Name:       "User",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "string", DBType: "UUID", PrimaryKey: true},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid apiVersion 'v2'")
	assert.Contains(t, err.Error(), "use 'v1'")
}

func TestValidateInvalidKind(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Model",
		Name:       "User",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "string", DBType: "UUID", PrimaryKey: true},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid kind 'Model'")
	assert.Contains(t, err.Error(), "use 'Resource'")
}

func TestValidateNonPascalCaseName(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "user_model",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "string", DBType: "UUID", PrimaryKey: true},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "should be in PascalCase")
}

func TestValidateFieldMissingName(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "User",
		Spec: Spec{
			Fields: []Field{
				{Type: "string", DBType: "UUID", PrimaryKey: true},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field name is required")
}

func TestValidateFieldMissingDBType(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "User",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "string", PrimaryKey: true},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db_type is required")
}

func TestValidateNullableWithNonPointer(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "User",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "string", DBType: "UUID", PrimaryKey: true},
				{Name: "age", Type: "int", DBType: "INTEGER", Nullable: true},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nullable field should use pointer type")
	assert.Contains(t, err.Error(), "*int")
}

func TestValidateValidationRules(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "User",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "string", DBType: "UUID", PrimaryKey: true},
				{
					Name:       "email",
					Type:       "string",
					DBType:     "VARCHAR(255)",
					Validation: []string{"email", "required", "max=255"},
				},
			},
		},
	}

	err := Validate(def)
	assert.NoError(t, err)
}

func TestValidateInvalidValidationRules(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "User",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "string", DBType: "UUID", PrimaryKey: true},
				{
					Name:       "email",
					Type:       "string",
					DBType:     "VARCHAR(255)",
					Validation: []string{"==invalid", ""},
				},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation")
}

func TestDefaultTableName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "users"},
		{"BlogPost", "blog_posts"},
		{"HTTPServer", "http_servers"},
		{"APIKey", "api_keys"},
		{"Person", "people"},
		{"Child", "children"},
		{"Category", "categories"},
		{"Box", "boxes"},
		{"Buzz", "buzzes"},
		{"Dish", "dishes"},
		{"Match", "matches"},
		{"City", "cities"},
		{"Wolf", "wolves"},
		{"Knife", "knives"},
		{"Hero", "heroes"},
		{"Photo", "photos"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := DefaultTableName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidGoType(t *testing.T) {
	validTypes := []string{
		"string", "*string",
		"int", "int32", "int64",
		"*int", "*int32", "*int64",
		"uint", "uint32", "uint64",
		"*uint", "*uint32", "*uint64",
		"float32", "float64",
		"*float32", "*float64",
		"bool", "*bool",
		"time.Time", "*time.Time",
		"[]byte",
	}

	for _, typ := range validTypes {
		t.Run(typ, func(t *testing.T) {
			assert.True(t, IsValidGoType(typ), "Type %s should be valid", typ)
		})
	}

	invalidTypes := []string{
		"InvalidType",
		"map[string]string",
		"[]string",
		"interface{}",
		"",
		"Time",
		"*",
	}

	for _, typ := range invalidTypes {
		t.Run(typ, func(t *testing.T) {
			assert.False(t, IsValidGoType(typ), "Type %s should be invalid", typ)
		})
	}
}

func TestIsPointerType(t *testing.T) {
	assert.True(t, IsPointerType("*string"))
	assert.True(t, IsPointerType("*int"))
	assert.True(t, IsPointerType("*time.Time"))
	assert.False(t, IsPointerType("string"))
	assert.False(t, IsPointerType("int"))
	assert.False(t, IsPointerType("[]byte"))
	assert.False(t, IsPointerType(""))
}

func TestValidateValidationRulesDetailed(t *testing.T) {
	tests := []struct {
		name    string
		rules   []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid common rules",
			rules:   []string{"required", "email", "min=3", "max=255"},
			wantErr: false,
		},
		{
			name:    "valid numeric rules",
			rules:   []string{"gt=0", "lte=100", "ne=50"},
			wantErr: false,
		},
		{
			name:    "empty rule",
			rules:   []string{""},
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "double equals",
			rules:   []string{"min==5"},
			wantErr: true,
			errMsg:  "invalid '=='",
		},
		{
			name:    "starts with equals",
			rules:   []string{"=value"},
			wantErr: true,
			errMsg:  "misplaced '='",
		},
		{
			name:    "ends with equals",
			rules:   []string{"min="},
			wantErr: true,
			errMsg:  "misplaced '='",
		},
		{
			name:    "unbalanced quotes",
			rules:   []string{"oneof='a' 'b"},
			wantErr: true,
			errMsg:  "unbalanced",
		},
		{
			name:    "space without equals",
			rules:   []string{"invalid rule"},
			wantErr: true,
			errMsg:  "contains spaces",
		},
		{
			name:    "custom validator",
			rules:   []string{"customValidator", "myCustomRule"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateValidationRules(tt.rules)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationErrorFormatting(t *testing.T) {
	// Single error
	err := ValidationError{
		Field:      "spec.fields[0].name",
		Message:    "field name is required",
		Suggestion: "provide a name for the field",
		Line:       10,
	}

	errStr := err.Error()
	assert.Contains(t, errStr, "spec.fields[0].name")
	assert.Contains(t, errStr, "line 10")
	assert.Contains(t, errStr, "field name is required")
	assert.Contains(t, errStr, "provide a name for the field")

	// Single error without line number
	err2 := ValidationError{
		Field:   "apiVersion",
		Message: "is required",
	}

	errStr2 := err2.Error()
	assert.Contains(t, errStr2, "apiVersion")
	assert.Contains(t, errStr2, "is required")
	assert.NotContains(t, errStr2, "line")

	// Multiple errors
	errs := ValidationErrors{
		ValidationError{
			Field:   "apiVersion",
			Message: "is required",
			Line:    1,
		},
		ValidationError{
			Field:   "kind",
			Message: "invalid value",
			Line:    2,
		},
		ValidationError{
			Field:   "spec.fields",
			Message: "at least one field is required",
			Line:    5,
		},
	}

	errsStr := errs.Error()
	assert.Contains(t, errsStr, "3 validation errors")
	assert.Contains(t, errsStr, "1.")
	assert.Contains(t, errsStr, "2.")
	assert.Contains(t, errsStr, "3.")
	assert.Contains(t, errsStr, "apiVersion")
	assert.Contains(t, errsStr, "kind")
	assert.Contains(t, errsStr, "spec.fields")
}

func TestParseNonExistentFile(t *testing.T) {
	def, err := Parse("nonexistent.yml")

	assert.Nil(t, def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

func TestParseMalformedYAML(t *testing.T) {
	// Create a temporary file with malformed YAML
	tempDir := t.TempDir()
	malformedPath := filepath.Join(tempDir, "malformed.yml")

	malformedYAML := []byte(`
apiVersion: v1
kind: Resource
name: User
spec:
  fields:
    - name: id
      type: string
      db_type UUID  # Missing colon
      primary_key: true
`)

	err := os.WriteFile(malformedPath, malformedYAML, 0644)
	require.NoError(t, err)

	def, err := Parse(malformedPath)

	assert.Nil(t, def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestValidateMultipleFields(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Order",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "string", DBType: "UUID", PrimaryKey: true},
				{Name: "customer_id", Type: "string", DBType: "UUID", Index: true},
				{Name: "total", Type: "float64", DBType: "DECIMAL(10,2)"},
				{Name: "status", Type: "string", DBType: "VARCHAR(50)", Default: "pending"},
				{Name: "notes", Type: "*string", DBType: "TEXT", Nullable: true},
				{Name: "created_at", Type: "time.Time", DBType: "TIMESTAMP", AutoNowAdd: true},
				{Name: "updated_at", Type: "time.Time", DBType: "TIMESTAMP", AutoNow: true},
			},
		},
	}

	err := Validate(def)
	assert.NoError(t, err)
}

func TestValidateCustomTableName(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Person",
		Spec: Spec{
			TableName: "employees",
			Fields: []Field{
				{Name: "id", Type: "string", DBType: "UUID", PrimaryKey: true},
			},
		},
	}

	err := Validate(def)
	assert.NoError(t, err)
	assert.Equal(t, "employees", def.Spec.TableName)

	// DefaultTableName should still work
}

// ============================================================================
// Relationship Tests
// ============================================================================

func TestValidateBelongsToRelationship(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Post",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "author_id", Type: "uuid.UUID", DBType: "UUID"},
				{Name: "title", Type: "string", DBType: "VARCHAR(255)"},
			},
			Relationships: []Relationship{
				{Name: "Author", Type: "belongs_to", Model: "User", ForeignKey: "author_id"},
			},
		},
	}

	err := Validate(def)
	assert.NoError(t, err)
}

func TestValidateHasManyRelationship(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "User",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "email", Type: "string", DBType: "VARCHAR(255)"},
			},
			Relationships: []Relationship{
				{Name: "Posts", Type: "has_many", Model: "Post", ForeignKey: "author_id"},
			},
		},
	}

	err := Validate(def)
	assert.NoError(t, err)
}

func TestValidateMultipleRelationships(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Comment",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "post_id", Type: "uuid.UUID", DBType: "UUID"},
				{Name: "author_id", Type: "uuid.UUID", DBType: "UUID"},
				{Name: "content", Type: "string", DBType: "TEXT"},
			},
			Relationships: []Relationship{
				{Name: "Post", Type: "belongs_to", Model: "Post", ForeignKey: "post_id"},
				{Name: "Author", Type: "belongs_to", Model: "User", ForeignKey: "author_id"},
			},
		},
	}

	err := Validate(def)
	assert.NoError(t, err)
}

func TestRelationshipMissingName(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Post",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "author_id", Type: "uuid.UUID", DBType: "UUID"},
			},
			Relationships: []Relationship{
				{Type: "belongs_to", Model: "User", ForeignKey: "author_id"},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "relationship name is required")
}

func TestRelationshipInvalidType(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Post",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "author_id", Type: "uuid.UUID", DBType: "UUID"},
			},
			Relationships: []Relationship{
				{Name: "Author", Type: "has_one", Model: "User", ForeignKey: "author_id"},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid relationship type")
	assert.Contains(t, err.Error(), "belongs_to")
	assert.Contains(t, err.Error(), "has_many")
}

func TestRelationshipNotPascalCase(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Post",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "author_id", Type: "uuid.UUID", DBType: "UUID"},
			},
			Relationships: []Relationship{
				{Name: "author", Type: "belongs_to", Model: "User", ForeignKey: "author_id"},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "should be in PascalCase")
}

func TestRelationshipMissingModel(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Post",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "author_id", Type: "uuid.UUID", DBType: "UUID"},
			},
			Relationships: []Relationship{
				{Name: "Author", Type: "belongs_to", ForeignKey: "author_id"},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")
}

func TestRelationshipModelNotPascalCase(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Post",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "author_id", Type: "uuid.UUID", DBType: "UUID"},
			},
			Relationships: []Relationship{
				{Name: "Author", Type: "belongs_to", Model: "user", ForeignKey: "author_id"},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "should be in PascalCase")
}

func TestRelationshipMissingForeignKey(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Post",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "author_id", Type: "uuid.UUID", DBType: "UUID"},
			},
			Relationships: []Relationship{
				{Name: "Author", Type: "belongs_to", Model: "User"},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "foreign_key is required")
}

func TestBelongsToForeignKeyFieldNotFound(t *testing.T) {
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Post",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "title", Type: "string", DBType: "VARCHAR(255)"},
			},
			Relationships: []Relationship{
				{Name: "Author", Type: "belongs_to", Model: "User", ForeignKey: "author_id"},
			},
		},
	}

	err := Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "foreign key field 'author_id' not found")
	assert.Contains(t, err.Error(), "add a field named 'author_id'")
}

func TestRelationshipNamingConventionWarning(t *testing.T) {
	// belongs_to with plural name (informational warning in validation logic)
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "Post",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "author_id", Type: "uuid.UUID", DBType: "UUID"},
			},
			Relationships: []Relationship{
				{Name: "Authors", Type: "belongs_to", Model: "User", ForeignKey: "author_id"},
			},
		},
	}

	// This should still pass validation (naming convention is informational only)
	err := Validate(def)
	assert.NoError(t, err)
}

func TestHasManyRelationshipNoFKValidation(t *testing.T) {
	// has_many doesn't validate FK field exists (it's in the related model)
	def := &Definition{
		APIVersion: "v1",
		Kind:       "Resource",
		Name:       "User",
		Spec: Spec{
			Fields: []Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
			},
			Relationships: []Relationship{
				{Name: "Posts", Type: "has_many", Model: "Post", ForeignKey: "author_id"},
			},
		},
	}

	err := Validate(def)
	assert.NoError(t, err)
}

func TestValidateDuplicateRelationshipNames(t *testing.T) {
	schema := `apiVersion: v1
kind: Resource
name: Post
spec:
  fields:
    - name: id
      type: int64
      db_type: BIGINT
      primary_key: true
    - name: author_id
      type: int64
      db_type: BIGINT
    - name: editor_id
      type: int64
      db_type: BIGINT
  relationships:
    - name: Author
      type: belongs_to
      model: User
      foreign_key: author_id
    - name: Author
      type: belongs_to
      model: User
      foreign_key: editor_id`

	def, err := ParseBytes([]byte(schema))
	assert.Nil(t, def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate relationship name 'Author'")
}

func TestValidateRelationshipNameConflictsWithField(t *testing.T) {
	schema := `apiVersion: v1
kind: Resource
name: Post
spec:
  fields:
    - name: id
      type: int64
      db_type: BIGINT
      primary_key: true
    - name: Author
      type: string
      db_type: VARCHAR(255)
    - name: author_id
      type: int64
      db_type: BIGINT
  relationships:
    - name: Author
      type: belongs_to
      model: User
      foreign_key: author_id`

	def, err := ParseBytes([]byte(schema))
	assert.Nil(t, def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "relationship name 'Author' conflicts with field name")
}