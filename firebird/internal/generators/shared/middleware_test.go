package shared

import (
	"testing"

	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddlewareGeneration(t *testing.T) {
	gen := NewGenerator("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find request_id.go operation
	var requestIDOp *generator.WriteFileOp
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/middleware/request_id.go" {
				requestIDOp = writeOp
				break
			}
		}
	}

	require.NotNil(t, requestIDOp, "request_id.go operation should be generated")
	assert.Contains(t, string(requestIDOp.Content), "func RequestID")
	assert.Contains(t, string(requestIDOp.Content), "func GetRequestID")
	assert.Contains(t, string(requestIDOp.Content), "X-Request-ID")
	assert.Contains(t, string(requestIDOp.Content), "uuid.New()")
}

func TestLoggerMiddlewareGeneration(t *testing.T) {
	gen := NewGenerator("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find logger.go operation
	var loggerOp *generator.WriteFileOp
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/middleware/logger.go" {
				loggerOp = writeOp
				break
			}
		}
	}

	require.NotNil(t, loggerOp, "logger.go operation should be generated")
	assert.Contains(t, string(loggerOp.Content), "func Logger")
	assert.Contains(t, string(loggerOp.Content), "type responseWriter struct")
	assert.Contains(t, string(loggerOp.Content), "GetRequestID")
	assert.Contains(t, string(loggerOp.Content), "slog.String")
	assert.Contains(t, string(loggerOp.Content), "duration_ms")
}

func TestRateLimitConfigGeneration(t *testing.T) {
	gen := NewGenerator("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find rate_limit.go config operation
	var rateLimitConfigOp *generator.WriteFileOp
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/config/rate_limit.go" {
				rateLimitConfigOp = writeOp
				break
			}
		}
	}

	require.NotNil(t, rateLimitConfigOp, "rate_limit.go config operation should be generated")
	assert.Contains(t, string(rateLimitConfigOp.Content), "type RateLimitConfig struct")
	assert.Contains(t, string(rateLimitConfigOp.Content), "RequestsPerMin")
	assert.Contains(t, string(rateLimitConfigOp.Content), "Burst")
	assert.Contains(t, string(rateLimitConfigOp.Content), "func DefaultRateLimitConfig")
}

func TestRateLimitMiddlewareGeneration(t *testing.T) {
	gen := NewGenerator("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find rate_limit.go operation
	var rateLimitOp *generator.WriteFileOp
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/middleware/rate_limit.go" {
				rateLimitOp = writeOp
				break
			}
		}
	}

	require.NotNil(t, rateLimitOp, "rate_limit.go operation should be generated")
	assert.Contains(t, string(rateLimitOp.Content), "func RateLimit")
	assert.Contains(t, string(rateLimitOp.Content), "type RateLimiter struct")
	assert.Contains(t, string(rateLimitOp.Content), "func NewRateLimiter")
	assert.Contains(t, string(rateLimitOp.Content), "func getIP")
	assert.Contains(t, string(rateLimitOp.Content), "X-Forwarded-For")
	assert.Contains(t, string(rateLimitOp.Content), "rate.Limiter")
}

func TestRateLimitErrorGeneration(t *testing.T) {
	gen := NewGenerator("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find errors.go operation
	var errorsOp *generator.WriteFileOp
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/errors/errors.go" {
				errorsOp = writeOp
				break
			}
		}
	}

	require.NotNil(t, errorsOp, "errors.go operation should be generated")
	assert.Contains(t, string(errorsOp.Content), "func NewRateLimitError")
	assert.Contains(t, string(errorsOp.Content), "RATE_LIMIT_EXCEEDED")
	assert.Contains(t, string(errorsOp.Content), "http.StatusTooManyRequests")
}

func TestMiddlewareGenerationCount(t *testing.T) {
	gen := NewGenerator("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Should generate 13 files total:
	// 1. errors.go
	// 2. response.go
	// 3. validation.go
	// 4. cors.go
	// 5. cors_config.go
	// 6. request_id.go
	// 7. logger.go
	// 8. rate_limit_config.go
	// 9. rate_limit.go
	// 10. query.go (NEW)
	// 11. auth.go (NEW)
	// 12. uuid.go (NEW)
	// 13. testing.go (NEW)
	assert.Equal(t, 13, len(ops), "Should generate 13 files")
}
