package analyzer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/simonhull/firebird-suite/owl/pkg/logger"
)

// mockDetector implements ConventionDetector for testing
type mockDetector struct {
	detectFunc func(*Package) []*Convention
}

func (m *mockDetector) Detect(pkg *Package) []*Convention {
	if m.detectFunc != nil {
		return m.detectFunc(pkg)
	}
	return nil
}

func TestAnalyzer_New(t *testing.T) {
	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector)

	if analyzer == nil {
		t.Fatal("NewAnalyzer returned nil")
	}

	if analyzer.parser == nil {
		t.Error("Analyzer.parser is nil")
	}

	if analyzer.detector == nil {
		t.Error("Analyzer.detector is nil")
	}

	if analyzer.logger == nil {
		t.Error("Analyzer.logger is nil")
	}
}

func TestAnalyzer_WithLogger(t *testing.T) {
	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector)

	customLogger := logger.NewSilentLogger()
	newAnalyzer := analyzer.WithLogger(customLogger)

	if newAnalyzer.logger != customLogger {
		t.Error("WithLogger did not set custom logger")
	}

	// Original should be unchanged
	if analyzer.logger == customLogger {
		t.Error("WithLogger modified original analyzer")
	}
}

func TestAnalyzer_AnalyzeWithContext_Cancellation(t *testing.T) {
	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := analyzer.AnalyzeWithContext(ctx, "./testdata")
	if err == nil {
		t.Error("expected error from cancelled context")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestAnalyzer_AnalyzeWithContext_Timeout(t *testing.T) {
	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond) // Ensure timeout

	_, err := analyzer.AnalyzeWithContext(ctx, "./testdata")
	if err == nil {
		t.Error("expected error from timed out context")
	}
}

func TestAnalyzer_Analyze_NonexistentPath(t *testing.T) {
	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())

	_, err := analyzer.Analyze("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

// TestAnalyzer_Analyze_WithTestData tests analysis of actual Go code
func TestAnalyzer_Analyze_WithTestData(t *testing.T) {
	// Create temporary directory with test Go files
	tmpDir := t.TempDir()

	// Create a simple test package
	testFile := filepath.Join(tmpDir, "test.go")
	testCode := `package testpkg

// TestType is a test type
type TestType struct {
	Name string
	Age  int
}

// TestFunc is a test function
func TestFunc() string {
	return "test"
}

const TestConst = "constant"

var TestVar = "variable"
`

	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Analyze the test directory
	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())

	project, err := analyzer.Analyze(tmpDir)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if project == nil {
		t.Fatal("project is nil")
	}

	if len(project.Packages) == 0 {
		t.Error("expected at least one package")
	}

	// Verify package contents
	pkg := project.Packages[0]
	if pkg.Name != "testpkg" {
		t.Errorf("expected package name 'testpkg', got: %s", pkg.Name)
	}

	// Check types
	if len(pkg.Types) != 1 {
		t.Errorf("expected 1 type, got: %d", len(pkg.Types))
	} else {
		if pkg.Types[0].Name != "TestType" {
			t.Errorf("expected type name 'TestType', got: %s", pkg.Types[0].Name)
		}
	}

	// Check functions
	if len(pkg.Functions) != 1 {
		t.Errorf("expected 1 function, got: %d", len(pkg.Functions))
	} else {
		if pkg.Functions[0].Name != "TestFunc" {
			t.Errorf("expected function name 'TestFunc', got: %s", pkg.Functions[0].Name)
		}
	}

	// Check constants
	if len(pkg.Constants) != 1 {
		t.Errorf("expected 1 constant, got: %d", len(pkg.Constants))
	}

	// Check variables
	if len(pkg.Variables) != 1 {
		t.Errorf("expected 1 variable, got: %d", len(pkg.Variables))
	}
}

func TestAnalyzer_ConventionDetection(t *testing.T) {
	// Create temporary directory with test Go files
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "handler.go")
	testCode := `package handlers

type UserHandler struct{}

func (h *UserHandler) HandleRequest() {}
`

	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Track convention detection
	detectionCalled := false
	detector := &mockDetector{
		detectFunc: func(pkg *Package) []*Convention {
			detectionCalled = true
			return []*Convention{
				{
					Name:       "Handler",
					Category:   "handlers",
					Layer:      "presentation",
					Confidence: 1.0,
					Reason:     "Test detection",
				},
			}
		},
	}

	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())
	_, err := analyzer.Analyze(tmpDir)

	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !detectionCalled {
		t.Error("convention detector was not called")
	}
}

func BenchmarkAnalyzer_Analyze(b *testing.B) {
	// Create a small test package
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench.go")
	testCode := `package bench

type BenchType struct {
	Field string
}

func BenchFunc() {}
`

	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		b.Fatalf("failed to write test file: %v", err)
	}

	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.Analyze(tmpDir)
	}
}
