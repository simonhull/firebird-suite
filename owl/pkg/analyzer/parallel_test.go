package analyzer

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/simonhull/firebird-suite/owl/pkg/logger"
)

func TestAnalyzer_AnalyzeParallel(t *testing.T) {
	// Create temporary directory with multiple test packages
	tmpDir := t.TempDir()

	// Create package 1
	pkg1Dir := filepath.Join(tmpDir, "pkg1")
	if err := os.MkdirAll(pkg1Dir, 0755); err != nil {
		t.Fatal(err)
	}
	pkg1File := filepath.Join(pkg1Dir, "file1.go")
	pkg1Code := `package pkg1

type Type1 struct {
	Field1 string
}

func Func1() {}
`
	if err := os.WriteFile(pkg1File, []byte(pkg1Code), 0644); err != nil {
		t.Fatal(err)
	}

	// Create package 2
	pkg2Dir := filepath.Join(tmpDir, "pkg2")
	if err := os.MkdirAll(pkg2Dir, 0755); err != nil {
		t.Fatal(err)
	}
	pkg2File := filepath.Join(pkg2Dir, "file2.go")
	pkg2Code := `package pkg2

type Type2 struct {
	Field2 int
}

func Func2() {}
`
	if err := os.WriteFile(pkg2File, []byte(pkg2Code), 0644); err != nil {
		t.Fatal(err)
	}

	// Create package 3
	pkg3Dir := filepath.Join(tmpDir, "pkg3")
	if err := os.MkdirAll(pkg3Dir, 0755); err != nil {
		t.Fatal(err)
	}
	pkg3File := filepath.Join(pkg3Dir, "file3.go")
	pkg3Code := `package pkg3

type Type3 struct {
	Field3 bool
}

func Func3() {}
`
	if err := os.WriteFile(pkg3File, []byte(pkg3Code), 0644); err != nil {
		t.Fatal(err)
	}

	// Analyze with parallel processing
	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())

	numWorkers := 2
	project, err := analyzer.AnalyzeParallel(context.Background(), tmpDir, numWorkers)
	if err != nil {
		t.Fatalf("AnalyzeParallel failed: %v", err)
	}

	// Verify results
	if project == nil {
		t.Fatal("project is nil")
	}

	if len(project.Packages) != 3 {
		t.Errorf("expected 3 packages, got: %d", len(project.Packages))
	}

	// Verify each package
	pkgNames := make(map[string]bool)
	for _, pkg := range project.Packages {
		pkgNames[pkg.Name] = true

		if len(pkg.Types) != 1 {
			t.Errorf("package %s: expected 1 type, got: %d", pkg.Name, len(pkg.Types))
		}

		if len(pkg.Functions) != 1 {
			t.Errorf("package %s: expected 1 function, got: %d", pkg.Name, len(pkg.Functions))
		}
	}

	// Check all packages were found
	expectedPkgs := []string{"pkg1", "pkg2", "pkg3"}
	for _, name := range expectedPkgs {
		if !pkgNames[name] {
			t.Errorf("package %s not found", name)
		}
	}
}

func TestAnalyzer_AnalyzeParallel_Cancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())

	_, err := analyzer.AnalyzeParallel(ctx, tmpDir, 2)
	if err == nil {
		t.Error("expected error from cancelled context")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestAnalyzer_AnalyzeParallel_DefaultWorkers(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testCode := `package test

type TestType struct{}
`
	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())

	// Use 0 workers to test default (NumCPU)
	project, err := analyzer.AnalyzeParallel(context.Background(), tmpDir, 0)
	if err != nil {
		t.Fatalf("AnalyzeParallel failed: %v", err)
	}

	if project == nil {
		t.Fatal("project is nil")
	}
}

func TestCollectDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	dirs := []string{
		"pkg1",
		"pkg2",
		"pkg3/subpkg",
		"node_modules/ignored", // Should be ignored
		"vendor/ignored",       // Should be ignored
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	var directories []directoryJob
	err := collectDirectories(tmpDir, &directories)
	if err != nil {
		t.Fatalf("collectDirectories failed: %v", err)
	}

	// Should include: tmpDir, pkg1, pkg2, pkg3, pkg3/subpkg
	// Should NOT include: node_modules, vendor, and their subdirectories
	if len(directories) < 5 {
		t.Errorf("expected at least 5 directories, got: %d", len(directories))
	}

	// Verify node_modules and vendor are not included
	for _, dir := range directories {
		if filepath.Base(dir.path) == "node_modules" || filepath.Base(dir.path) == "vendor" {
			t.Errorf("ignored directory included: %s", dir.path)
		}
	}
}

func BenchmarkAnalyzer_AnalyzeParallel(b *testing.B) {
	// Create a larger test project
	tmpDir := b.TempDir()

	// Create 10 packages
	for i := 0; i < 10; i++ {
		pkgDir := filepath.Join(tmpDir, filepath.Join("pkg", string(rune('a'+i))))
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			b.Fatal(err)
		}

		pkgFile := filepath.Join(pkgDir, "file.go")
		pkgCode := `package test

type TestType struct {
	Field string
}

func TestFunc() {}
`
		if err := os.WriteFile(pkgFile, []byte(pkgCode), 0644); err != nil {
			b.Fatal(err)
		}
	}

	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.AnalyzeParallel(context.Background(), tmpDir, runtime.NumCPU())
	}
}

func BenchmarkAnalyzer_AnalyzeSequential(b *testing.B) {
	// Create a larger test project
	tmpDir := b.TempDir()

	// Create 10 packages
	for i := 0; i < 10; i++ {
		pkgDir := filepath.Join(tmpDir, filepath.Join("pkg", string(rune('a'+i))))
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			b.Fatal(err)
		}

		pkgFile := filepath.Join(pkgDir, "file.go")
		pkgCode := `package test

type TestType struct {
	Field string
}

func TestFunc() {}
`
		if err := os.WriteFile(pkgFile, []byte(pkgCode), 0644); err != nil {
			b.Fatal(err)
		}
	}

	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.Analyze(tmpDir)
	}
}

func TestAnalyzer_ParallelVsSequential(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance comparison in short mode")
	}

	// Create a test project with multiple packages
	tmpDir := t.TempDir()

	numPackages := 20
	for i := 0; i < numPackages; i++ {
		pkgDir := filepath.Join(tmpDir, filepath.Join("pkg", string(rune('a'+i))))
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatal(err)
		}

		pkgFile := filepath.Join(pkgDir, "file.go")
		pkgCode := `package test

type TestType struct {
	Field string
}

func TestFunc() {}
`
		if err := os.WriteFile(pkgFile, []byte(pkgCode), 0644); err != nil {
			t.Fatal(err)
		}
	}

	detector := &mockDetector{}
	analyzer := NewAnalyzer(detector).WithLogger(logger.NewSilentLogger())

	// Time sequential
	startSeq := time.Now()
	projSeq, err := analyzer.Analyze(tmpDir)
	seqDuration := time.Since(startSeq)
	if err != nil {
		t.Fatalf("Sequential analysis failed: %v", err)
	}

	// Time parallel
	startPar := time.Now()
	projPar, err := analyzer.AnalyzeParallel(context.Background(), tmpDir, runtime.NumCPU())
	parDuration := time.Since(startPar)
	if err != nil {
		t.Fatalf("Parallel analysis failed: %v", err)
	}

	// Verify same results
	if len(projSeq.Packages) != len(projPar.Packages) {
		t.Errorf("different package counts: seq=%d, par=%d",
			len(projSeq.Packages), len(projPar.Packages))
	}

	t.Logf("Sequential: %v, Parallel: %v, Speedup: %.2fx",
		seqDuration, parDuration, float64(seqDuration)/float64(parDuration))
}
