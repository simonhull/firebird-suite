package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	// Create temp file
	tempDir := t.TempDir()
	schemaPath := filepath.Join(tempDir, "test.yml")

	schemaContent := `apiVersion: v1
kind: TestResource
name: TestName
metadata:
  author: test
spec:
  field1: value1
  field2: 123
`

	err := os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Parse
	def, err := Parse(schemaPath)
	require.NoError(t, err)
	require.NotNil(t, def)

	assert.Equal(t, "v1", def.APIVersion)
	assert.Equal(t, "TestResource", def.Kind)
	assert.Equal(t, "TestName", def.Name)
	assert.NotNil(t, def.Metadata)
	assert.NotNil(t, def.Spec)
}

func TestParseBytes(t *testing.T) {
	schemaContent := []byte(`apiVersion: v1
kind: Resource
name: Test
spec:
  data: value
`)

	def, err := ParseBytes(schemaContent)
	require.NoError(t, err)
	assert.Equal(t, "v1", def.APIVersion)
	assert.Equal(t, "Resource", def.Kind)
	assert.Equal(t, "Test", def.Name)
}

func TestWrite(t *testing.T) {
	tempDir := t.TempDir()
	schemaPath := filepath.Join(tempDir, "output.yml")

	def := &Definition{
		APIVersion: "v1",
		Kind:       "TestKind",
		Name:       "TestName",
		Spec:       map[string]interface{}{"field": "value"},
	}

	err := Write(schemaPath, def)
	require.NoError(t, err)

	// Verify file was written
	_, err = os.Stat(schemaPath)
	assert.NoError(t, err)

	// Parse it back
	parsed, err := Parse(schemaPath)
	require.NoError(t, err)
	assert.Equal(t, def.APIVersion, parsed.APIVersion)
	assert.Equal(t, def.Kind, parsed.Kind)
	assert.Equal(t, def.Name, parsed.Name)
}

func TestValidateBasicStructure(t *testing.T) {
	tests := []struct {
		name    string
		def     *Definition
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid structure",
			def: &Definition{
				APIVersion: "v1",
				Kind:       "Resource",
				Name:       "Test",
			},
			wantErr: false,
		},
		{
			name: "missing apiVersion",
			def: &Definition{
				Kind: "Resource",
				Name: "Test",
			},
			wantErr: true,
			errMsg:  "apiVersion",
		},
		{
			name: "missing kind",
			def: &Definition{
				APIVersion: "v1",
				Name:       "Test",
			},
			wantErr: true,
			errMsg:  "kind",
		},
		{
			name: "missing name",
			def: &Definition{
				APIVersion: "v1",
				Kind:       "Resource",
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name:    "all missing",
			def:     &Definition{},
			wantErr: true,
			errMsg:  "3 validation errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBasicStructure(tt.def)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:      "spec.field",
		Message:    "is required",
		Suggestion: "add the field",
		Line:       10,
	}

	errStr := err.Error()
	assert.Contains(t, errStr, "spec.field")
	assert.Contains(t, errStr, "line 10")
	assert.Contains(t, errStr, "is required")
	assert.Contains(t, errStr, "add the field")
}

func TestValidationErrors(t *testing.T) {
	errors := ValidationErrors{
		ValidationError{Field: "field1", Message: "error 1"},
		ValidationError{Field: "field2", Message: "error 2"},
	}

	errStr := errors.Error()
	assert.Contains(t, errStr, "2 validation errors")
	assert.Contains(t, errStr, "field1")
	assert.Contains(t, errStr, "field2")
}