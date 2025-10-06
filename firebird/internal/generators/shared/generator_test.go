package shared

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateErrors(t *testing.T) {
	tmpDir := t.TempDir()

	gen := NewGenerator(tmpDir, "github.com/test/myapp")
	ops, err := gen.Generate()

	require.NoError(t, err)
	require.Len(t, ops, 13) // errors, response, validation, cors, cors_config, request_id, logger, rate_limit_config, rate_limit, query, auth, uuid, testing

	// Execute operations
	ctx := context.Background()
	for _, op := range ops {
		err := op.Execute(ctx)
		require.NoError(t, err)
	}

	// Verify errors.go was created
	errorsPath := filepath.Join(tmpDir, "internal", "errors", "errors.go")
	assert.FileExists(t, errorsPath)

	content, err := os.ReadFile(errorsPath)
	require.NoError(t, err)

	// Check key functions exist
	assert.Contains(t, string(content), "func NewNotFoundError")
	assert.Contains(t, string(content), "func NewValidationError")
	assert.Contains(t, string(content), "func NewInternalError")
	assert.Contains(t, string(content), "func NewUnauthorizedError")
	assert.Contains(t, string(content), "func NewForbiddenError")
	assert.Contains(t, string(content), "func NewBadRequestError")
	assert.Contains(t, string(content), "func NewConflictError")

	// Check helper functions exist
	assert.Contains(t, string(content), "func IsNotFound")
	assert.Contains(t, string(content), "func IsValidationError")
	assert.Contains(t, string(content), "func IsConflict")

	// Check struct definition
	assert.Contains(t, string(content), "type AppError struct")
	assert.Contains(t, string(content), "Code       string")
	assert.Contains(t, string(content), "Message    string")
	assert.Contains(t, string(content), "StatusCode int")
	assert.Contains(t, string(content), "Details    map[string]interface{}")

	// Check methods exist
	assert.Contains(t, string(content), "func (e *AppError) Error()")
	assert.Contains(t, string(content), "func (e *AppError) Unwrap()")
	assert.Contains(t, string(content), "func (e *AppError) HTTPStatus()")
}

func TestGenerateHelpers(t *testing.T) {
	tmpDir := t.TempDir()

	gen := NewGenerator(tmpDir, "github.com/test/myapp")
	ops, err := gen.Generate()

	require.NoError(t, err)

	ctx := context.Background()
	for _, op := range ops {
		err := op.Execute(ctx)
		require.NoError(t, err)
	}

	// Verify response.go was created
	helpersPath := filepath.Join(tmpDir, "internal", "helpers", "response.go")
	assert.FileExists(t, helpersPath)

	content, err := os.ReadFile(helpersPath)
	require.NoError(t, err)

	// Check key functions exist
	assert.Contains(t, string(content), "func RespondJSON")
	assert.Contains(t, string(content), "func RespondError")
	assert.Contains(t, string(content), "func RespondSuccess")
	assert.Contains(t, string(content), "func RespondCreated")
	assert.Contains(t, string(content), "func RespondNoContent")
	assert.Contains(t, string(content), "func RespondAccepted")

	// Check struct definitions
	assert.Contains(t, string(content), "type ErrorResponse struct")
	assert.Contains(t, string(content), "type ErrorDetail struct")

	// Check import path is correct
	assert.Contains(t, string(content), `apperrors "github.com/test/myapp/internal/errors"`)
}

func TestGenerateWithDifferentModulePath(t *testing.T) {
	tmpDir := t.TempDir()

	gen := NewGenerator(tmpDir, "example.com/custom/path")
	ops, err := gen.Generate()

	require.NoError(t, err)

	ctx := context.Background()
	for _, op := range ops {
		err := op.Execute(ctx)
		require.NoError(t, err)
	}

	// Verify helpers contains correct module path
	helpersPath := filepath.Join(tmpDir, "internal", "helpers", "response.go")
	content, err := os.ReadFile(helpersPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), `apperrors "example.com/custom/path/internal/errors"`)
}

func TestOperationDescriptions(t *testing.T) {
	tmpDir := t.TempDir()

	gen := NewGenerator(tmpDir, "github.com/test/myapp")
	ops, err := gen.Generate()

	require.NoError(t, err)
	require.Len(t, ops, 13) // errors, response, validation, cors, cors_config, request_id, logger, rate_limit_config, rate_limit, query, auth, uuid, testing

	// Check operation descriptions are meaningful
	descriptions := make([]string, len(ops))
	for i, op := range ops {
		descriptions[i] = op.Description()
	}

	// Should have descriptions for all files
	assert.Contains(t, descriptions[0], "errors.go")
	assert.Contains(t, descriptions[1], "response.go")
	assert.Contains(t, descriptions[2], "validation.go")
	assert.Contains(t, descriptions[3], "cors.go")
	assert.Contains(t, descriptions[4], "cors.go")
}

func TestErrorTypesAreWellFormed(t *testing.T) {
	tmpDir := t.TempDir()

	gen := NewGenerator(tmpDir, "github.com/test/myapp")
	ops, err := gen.Generate()
	require.NoError(t, err)

	ctx := context.Background()
	for _, op := range ops {
		err := op.Execute(ctx)
		require.NoError(t, err)
	}

	errorsPath := filepath.Join(tmpDir, "internal", "errors", "errors.go")
	content, err := os.ReadFile(errorsPath)
	require.NoError(t, err)

	contentStr := string(content)

	// Verify NewNotFoundError creates correct error structure
	assert.Contains(t, contentStr, "Code:       \"NOT_FOUND\"")
	assert.Contains(t, contentStr, "StatusCode: http.StatusNotFound")

	// Verify NewValidationError supports details
	assert.Contains(t, contentStr, "Code:       \"VALIDATION_ERROR\"")
	assert.Contains(t, contentStr, "Details:    details")

	// Verify NewInternalError wraps errors
	assert.Contains(t, contentStr, "Code:       \"INTERNAL_ERROR\"")
	assert.Contains(t, contentStr, "Err:        err")

	// Verify all HTTP status codes are used correctly
	assert.Contains(t, contentStr, "http.StatusBadRequest")
	assert.Contains(t, contentStr, "http.StatusUnauthorized")
	assert.Contains(t, contentStr, "http.StatusForbidden")
	assert.Contains(t, contentStr, "http.StatusConflict")
	assert.Contains(t, contentStr, "http.StatusInternalServerError")
}

func TestResponseHelpersUsesAppError(t *testing.T) {
	tmpDir := t.TempDir()

	gen := NewGenerator(tmpDir, "github.com/test/myapp")
	ops, err := gen.Generate()
	require.NoError(t, err)

	ctx := context.Background()
	for _, op := range ops {
		err := op.Execute(ctx)
		require.NoError(t, err)
	}

	helpersPath := filepath.Join(tmpDir, "internal", "helpers", "response.go")
	content, err := os.ReadFile(helpersPath)
	require.NoError(t, err)

	contentStr := string(content)

	// Verify RespondError uses errors.As to detect AppError
	assert.Contains(t, contentStr, "var appErr *apperrors.AppError")
	assert.Contains(t, contentStr, "errors.As(err, &appErr)")

	// Verify conversion to AppError if not already
	assert.Contains(t, contentStr, "apperrors.NewInternalError(\"An unexpected error occurred\", err)")

	// Verify response structure
	assert.Contains(t, contentStr, "response := ErrorResponse")
	assert.Contains(t, contentStr, "Code:    appErr.Code")
	assert.Contains(t, contentStr, "Message: appErr.Message")
	assert.Contains(t, contentStr, "Details: appErr.Details")

	// Verify HTTP status comes from AppError
	assert.Contains(t, contentStr, "RespondJSON(w, appErr.HTTPStatus(), response)")
}

func TestGenerateValidation(t *testing.T) {
	tmpDir := t.TempDir()

	gen := NewGenerator(tmpDir, "github.com/test/myapp")
	ops, err := gen.Generate()
	require.NoError(t, err)

	ctx := context.Background()
	for _, op := range ops {
		err := op.Execute(ctx)
		require.NoError(t, err)
	}

	// Verify validation.go was created
	validationPath := filepath.Join(tmpDir, "internal", "helpers", "validation.go")
	assert.FileExists(t, validationPath)

	content, err := os.ReadFile(validationPath)
	require.NoError(t, err)
	contentStr := string(content)

	// Check key functions exist
	assert.Contains(t, contentStr, "func ValidateStruct")
	assert.Contains(t, contentStr, "func ValidationErrorResponse")
	assert.Contains(t, contentStr, "func formatValidationError")

	// Check validator initialization
	assert.Contains(t, contentStr, "var validate *validator.Validate")
	assert.Contains(t, contentStr, "validate = validator.New()")

	// Check validation error tags are handled
	assert.Contains(t, contentStr, `case "required"`)
	assert.Contains(t, contentStr, `case "email"`)
	assert.Contains(t, contentStr, `case "min"`)
	assert.Contains(t, contentStr, `case "max"`)
	assert.Contains(t, contentStr, `case "gte"`)
	assert.Contains(t, contentStr, `case "lte"`)
	assert.Contains(t, contentStr, `case "uuid"`)
}

func TestGenerateCORS(t *testing.T) {
	tmpDir := t.TempDir()

	gen := NewGenerator(tmpDir, "github.com/test/myapp")
	ops, err := gen.Generate()
	require.NoError(t, err)

	ctx := context.Background()
	for _, op := range ops {
		err := op.Execute(ctx)
		require.NoError(t, err)
	}

	// Verify cors.go was created
	corsPath := filepath.Join(tmpDir, "internal", "middleware", "cors.go")
	assert.FileExists(t, corsPath)

	content, err := os.ReadFile(corsPath)
	require.NoError(t, err)
	contentStr := string(content)

	// Check key functions exist
	assert.Contains(t, contentStr, "func CORS")
	assert.Contains(t, contentStr, "func isAllowedOrigin")

	// Check CORS headers are set
	assert.Contains(t, contentStr, "Access-Control-Allow-Origin")
	assert.Contains(t, contentStr, "Access-Control-Allow-Methods")
	assert.Contains(t, contentStr, "Access-Control-Allow-Headers")
	assert.Contains(t, contentStr, "Access-Control-Allow-Credentials")
	assert.Contains(t, contentStr, "Access-Control-Max-Age")
	assert.Contains(t, contentStr, "Access-Control-Expose-Headers")

	// Check preflight handling
	assert.Contains(t, contentStr, "http.MethodOptions")
	assert.Contains(t, contentStr, "http.StatusNoContent")

	// Check wildcard subdomain support
	assert.Contains(t, contentStr, `strings.HasPrefix(o, "*.")`)
}

func TestGenerateCORSConfig(t *testing.T) {
	tmpDir := t.TempDir()

	gen := NewGenerator(tmpDir, "github.com/test/myapp")
	ops, err := gen.Generate()
	require.NoError(t, err)

	ctx := context.Background()
	for _, op := range ops {
		err := op.Execute(ctx)
		require.NoError(t, err)
	}

	// Verify cors config was created
	corsConfigPath := filepath.Join(tmpDir, "internal", "config", "cors.go")
	assert.FileExists(t, corsConfigPath)

	content, err := os.ReadFile(corsConfigPath)
	require.NoError(t, err)
	contentStr := string(content)

	// Check struct definition
	assert.Contains(t, contentStr, "type CORSConfig struct")
	assert.Contains(t, contentStr, "Enabled          bool")
	assert.Contains(t, contentStr, "AllowedOrigins   []string")
	assert.Contains(t, contentStr, "AllowedMethods   []string")
	assert.Contains(t, contentStr, "AllowedHeaders   []string")
	assert.Contains(t, contentStr, "ExposedHeaders   []string")
	assert.Contains(t, contentStr, "AllowCredentials bool")
	assert.Contains(t, contentStr, "MaxAge           int")

	// Check default function
	assert.Contains(t, contentStr, "func DefaultCORSConfig()")

	// Check sensible defaults
	assert.Contains(t, contentStr, "http://localhost:3000")
	assert.Contains(t, contentStr, "http://localhost:5173") // Vite
	assert.Contains(t, contentStr, "Authorization")
	assert.Contains(t, contentStr, "Content-Type")
}
