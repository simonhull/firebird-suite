package project

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// ModuleInfo contains information from go.mod
type ModuleInfo struct {
	Path      string // Module path (e.g., "github.com/user/repo")
	Version   string // Module version (if specified)
	GoVersion string // Go version requirement (e.g., "1.21")
}

// DetectModule reads go.mod and returns module information.
// Returns an error if go.mod doesn't exist or is invalid.
func DetectModule(rootPath string) (*ModuleInfo, error) {
	modPath := filepath.Join(rootPath, "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("go.mod not found in %s", rootPath)
		}
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}

	modFile, err := modfile.Parse(modPath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	info := &ModuleInfo{
		Path: modFile.Module.Mod.Path,
	}

	if modFile.Module.Mod.Version != "" {
		info.Version = modFile.Module.Mod.Version
	}

	if modFile.Go != nil {
		info.GoVersion = modFile.Go.Version
	}

	return info, nil
}
