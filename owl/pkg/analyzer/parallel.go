package analyzer

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/simonhull/firebird-suite/fledge/project"
	"github.com/simonhull/firebird-suite/owl/pkg/logger"
)

// parseResult holds the result of parsing a directory
type parseResult struct {
	pkg *Package
	err error
}

// directoryJob represents a directory to be parsed
type directoryJob struct {
	path string
	info os.FileInfo
}

// AnalyzeParallel analyzes a Go project using parallel workers for better performance
func (a *Analyzer) AnalyzeParallel(ctx context.Context, rootPath string, numWorkers int) (*Project, error) {
	a.logger.Info("Starting parallel project analysis",
		logger.F("path", rootPath),
		logger.F("workers", numWorkers))

	proj := &Project{
		RootPath: rootPath,
		Packages: make([]*Package, 0, 50),
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Detect Firebird project using Fledge utility
	isFirebird, firebirdConfig, err := detectFirebirdProject(rootPath)
	if err != nil {
		a.logger.Warn("Error detecting Firebird project", logger.F("error", err))
	}

	if isFirebird && firebirdConfig != nil {
		a.logger.Info("Firebird project detected",
			logger.F("database", firebirdConfig.Database),
			logger.F("router", firebirdConfig.Router))

		proj.IsFirebirdProject = true
		proj.FirebirdConfig = firebirdConfig
	}

	// Collect all directories to parse
	directories := make([]directoryJob, 0, 100)
	err = collectDirectories(rootPath, &directories)
	if err != nil {
		return nil, err
	}

	a.logger.Debug("Collected directories", logger.F("count", len(directories)))

	// Set up worker pool
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	jobs := make(chan directoryJob, len(directories))
	results := make(chan parseResult, len(directories))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go a.parseWorker(ctx, jobs, results, &wg)
	}

	// Send jobs
	go func() {
		for _, dir := range directories {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- dir:
			}
		}
		close(jobs)
	}()

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	packagesProcessed := 0
	for result := range results {
		if result.err != nil {
			// Log but continue processing
			a.logger.Warn("Failed to parse package", logger.F("error", result.err))
			continue
		}

		if result.pkg != nil {
			// Detect conventions
			if a.detector != nil {
				a.applyConventions(result.pkg)
			}

			proj.Packages = append(proj.Packages, result.pkg)
			packagesProcessed++

			a.logger.Debug("Parsed package",
				logger.F("name", result.pkg.Name),
				logger.F("types", len(result.pkg.Types)),
				logger.F("functions", len(result.pkg.Functions)))
		}
	}

	// Check if context was cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	a.logger.Info("Parallel project analysis complete",
		logger.F("packages", len(proj.Packages)),
		logger.F("workers", numWorkers))

	return proj, nil
}

// parseWorker is a worker that processes directory parsing jobs
func (a *Analyzer) parseWorker(ctx context.Context, jobs <-chan directoryJob, results chan<- parseResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Parse directory
		files, err := a.parser.ParseDirectory(job.path)
		if err != nil {
			results <- parseResult{err: err}
			continue
		}

		if len(files) == 0 {
			continue
		}

		// Extract package information
		pkg, err := a.parser.ParsePackage(files)
		if err != nil {
			results <- parseResult{err: err}
			continue
		}

		if pkg != nil {
			pkg.Path = job.path
			results <- parseResult{pkg: pkg}
		}
	}
}

// collectDirectories walks the directory tree and collects all directories to parse
func collectDirectories(rootPath string, directories *[]directoryJob) error {
	ignoreDirs := map[string]bool{
		"node_modules": true,
		"vendor":       true,
		".git":         true,
		".svn":         true,
		"dist":         true,
		"build":        true,
		"bin":          true,
		"tmp":          true,
		"temp":         true,
		"testdata":     true,
		".idea":        true,
		".vscode":      true,
		".vs":          true,
	}

	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors but continue
		}

		if !info.IsDir() {
			return nil
		}

		// Check if this is an ignored directory
		if ignoreDirs[info.Name()] {
			return filepath.SkipDir
		}

		*directories = append(*directories, directoryJob{
			path: path,
			info: info,
		})

		return nil
	})
}

// detectFirebirdProject wraps the fledge project detection
func detectFirebirdProject(rootPath string) (bool, *FirebirdConfig, error) {
	isFirebird, firebirdConfig, err := project.DetectFirebirdProject(rootPath)
	if err != nil || !isFirebird || firebirdConfig == nil {
		return false, nil, err
	}

	return true, &FirebirdConfig{
		ConfigPath: firebirdConfig.ConfigPath,
		Database:   firebirdConfig.Database,
		Router:     firebirdConfig.Router,
	}, nil
}
