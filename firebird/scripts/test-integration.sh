#!/bin/bash
set -e

echo "🔥 Running Firebird Integration Tests"
echo "======================================"

# Build firebird binary first
echo "Building firebird..."
go build -o firebird ./cmd/firebird

# Run integration tests
echo ""
echo "Running integration tests..."
go test -v -tags=integration ./test/integration -timeout=10m

echo ""
echo "✅ All integration tests passed!"
