package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
	"gopkg.in/yaml.v3"
)

// IsRealtimeInitialized checks if realtime has been wired into the project
func IsRealtimeInitialized() bool {
	configPath := "firebird.yml" // In project root
	content, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}

	var config struct {
		Realtime struct {
			Initialized bool `yaml:"initialized"`
		} `yaml:"realtime"`
	}

	if err := yaml.Unmarshal(content, &config); err != nil {
		return false
	}

	return config.Realtime.Initialized
}

// DetectRealtimeFromSchemas scans schemas/ directory for realtime config
func DetectRealtimeFromSchemas() (hasRealtime bool, backend string, natsURL string) {
	schemasDir := "internal/schemas"

	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		return false, "", ""
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".firebird.yml") {
			continue
		}

		schemaPath := filepath.Join(schemasDir, entry.Name())
		def, err := schema.Parse(schemaPath)
		if err != nil {
			continue
		}

		if def.Spec.Realtime != nil && def.Spec.Realtime.Enabled {
			backend = def.Spec.Realtime.Backend
			if backend == "" {
				backend = "memory"
			}
			natsURL = def.Spec.Realtime.NatsURL
			if natsURL == "" {
				natsURL = "nats://localhost:4222"
			}
			return true, backend, natsURL
		}
	}

	return false, "", ""
}

// UpdateConfigWithRealtime updates firebird.yml with realtime settings
func UpdateConfigWithRealtime(backend, natsURL string, markInitialized bool) error {
	configPath := "firebird.yml" // In project root

	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	// Ensure realtime section exists
	if config["realtime"] == nil {
		config["realtime"] = make(map[string]interface{})
	}

	realtimeConfig := config["realtime"].(map[string]interface{})
	realtimeConfig["enabled"] = true
	realtimeConfig["initialized"] = markInitialized
	realtimeConfig["backend"] = backend

	if backend == "nats" {
		realtimeConfig["nats_url"] = natsURL
	}

	// Write back
	updatedContent, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(configPath, updatedContent, 0644)
}

// GetModulePath reads go.mod to get module path
func GetModulePath() (string, error) {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", fmt.Errorf("module path not found in go.mod")
}

// UpdateMainGo injects EventBus initialization into main.go
func UpdateMainGo(mainPath, modulePath, backend, natsURL string) error {
	content, err := os.ReadFile(mainPath)
	if err != nil {
		return fmt.Errorf("reading main.go: %w", err)
	}

	mainStr := string(content)

	// Check if already has realtime code
	if strings.Contains(mainStr, "events.NewMemoryBus") || strings.Contains(mainStr, "events.NewNATSBus") {
		return nil // Already integrated
	}

	// Add imports
	importBlock := fmt.Sprintf(`
	"%s/internal/events"
	"%s/internal/realtime"`, modulePath, modulePath)

	if !strings.Contains(mainStr, "internal/events") {
		importPos := strings.Index(mainStr, "import (")
		if importPos != -1 {
			closePos := strings.Index(mainStr[importPos:], ")")
			if closePos != -1 {
				insertPos := importPos + closePos
				mainStr = mainStr[:insertPos] + importBlock + "\n" + mainStr[insertPos:]
			}
		}
	}

	// Add EventBus initialization
	var eventBusInit string
	if backend == "nats" {
		eventBusInit = fmt.Sprintf(`eventBus = events.NewNATSBus("%s", logger)`, natsURL)
	} else {
		eventBusInit = `eventBus = events.NewMemoryBus(logger)`
	}

	eventBusCode := fmt.Sprintf(`
	// Initialize EventBus for real-time features
	var eventBus events.EventBus
	%s
	defer func() {
		if err := eventBus.Close(); err != nil {
			logger.Error("failed to close event bus", "error", err.Error())
		}
	}()

	// Initialize WebSocket infrastructure
	connManager := realtime.NewConnectionManager(eventBus, logger)

	logger.Info("real-time features enabled", "backend", "%s")
`, eventBusInit, backend)

	// Find where to insert (after logger setup)
	loggerPos := strings.Index(mainStr, "logger := ")
	if loggerPos == -1 {
		loggerPos = strings.Index(mainStr, "logger = ")
	}

	if loggerPos != -1 {
		// Find next double newline
		searchStart := loggerPos
		insertPos := strings.Index(mainStr[searchStart:], "\n\n")
		if insertPos != -1 {
			insertPos += searchStart + 2
			mainStr = mainStr[:insertPos] + eventBusCode + mainStr[insertPos:]
		}
	}

	return os.WriteFile(mainPath, []byte(mainStr), 0644)
}

// UpdateRoutesGo injects WebSocket endpoint registration
func UpdateRoutesGo(routesPath, modulePath string) error {
	content, err := os.ReadFile(routesPath)
	if err != nil {
		return fmt.Errorf("reading routes.go: %w", err)
	}

	routesStr := string(content)

	// Check if already has WebSocket endpoint
	if strings.Contains(routesStr, "/ws") {
		return nil
	}

	// Add imports
	if !strings.Contains(routesStr, "internal/realtime") {
		importBlock := fmt.Sprintf(`
	"%s/internal/realtime"
	"%s/internal/handlers"
	"log/slog"`, modulePath, modulePath)

		importPos := strings.Index(routesStr, "import (")
		if importPos != -1 {
			closePos := strings.Index(routesStr[importPos:], ")")
			if closePos != -1 {
				insertPos := importPos + closePos
				routesStr = routesStr[:insertPos] + importBlock + "\n" + routesStr[insertPos:]
			}
		}
	}

	// Update function signature
	oldSig := "func RegisterRoutes(logger *slog.Logger)"
	newSig := "func RegisterRoutes(logger *slog.Logger, connManager *realtime.ConnectionManager)"

	if strings.Contains(routesStr, oldSig) {
		routesStr = strings.Replace(routesStr, oldSig, newSig, 1)
	}

	// Add WebSocket endpoint
	wsCode := `
	// WebSocket endpoint for real-time features
	if connManager != nil {
		wsHandler := handlers.NewWebSocketHandler(connManager, logger)
		mux.HandleFunc("/ws", wsHandler.HandleWebSocket)
		logger.Info("WebSocket endpoint registered", "path", "/ws")
	}
`

	funcPos := strings.Index(routesStr, "func RegisterRoutes")
	if funcPos != -1 {
		bodyPos := strings.Index(routesStr[funcPos:], "{")
		if bodyPos != -1 {
			insertPos := funcPos + bodyPos + 1
			routesStr = routesStr[:insertPos] + wsCode + routesStr[insertPos:]
		}
	}

	return os.WriteFile(routesPath, []byte(routesStr), 0644)
}
