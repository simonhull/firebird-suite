package realtime

import (
	"embed"
	"path/filepath"

	"github.com/simonhull/firebird-suite/fledge/generator"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator generates real-time event system files
type Generator struct {
	projectPath string
	modulePath  string
	models      []ModelHelper
	renderer    *generator.Renderer
}

// ModelHelper represents a model for subscription helper generation
type ModelHelper struct {
	Name       string // Model name (e.g., "Post")
	NamePlural string // Plural name (e.g., "posts")
	PKType     string // Primary key type (e.g., "uuid.UUID", "int")
}

// New creates a new realtime generator
func New(projectPath string) *Generator {
	return &Generator{
		projectPath: projectPath,
		renderer:    generator.NewRenderer(),
	}
}

// NewWithModule creates a new realtime generator with module path
func NewWithModule(projectPath, modulePath string) *Generator {
	return &Generator{
		projectPath: projectPath,
		modulePath:  modulePath,
		renderer:    generator.NewRenderer(),
	}
}

// NewWithModels creates a new realtime generator with module path and models
func NewWithModels(projectPath, modulePath string, models []ModelHelper) *Generator {
	return &Generator{
		projectPath: projectPath,
		modulePath:  modulePath,
		models:      models,
		renderer:    generator.NewRenderer(),
	}
}

// Generate creates event system files
func (g *Generator) Generate() ([]generator.Operation, error) {
	var ops []generator.Operation

	// Generate events.go
	eventsOp, err := g.generateEvents()
	if err != nil {
		return nil, err
	}
	ops = append(ops, eventsOp)

	// Generate memory_bus.go
	memoryBusOp, err := g.generateMemoryBus()
	if err != nil {
		return nil, err
	}
	ops = append(ops, memoryBusOp)

	// Generate nats_bus.go
	natsBusOp, err := g.generateNATSBus()
	if err != nil {
		return nil, err
	}
	ops = append(ops, natsBusOp)

	// Generate WebSocket files only if modulePath is provided
	if g.modulePath != "" {
		// Generate connection_manager.go
		connManagerOp, err := g.generateConnectionManager()
		if err != nil {
			return nil, err
		}
		ops = append(ops, connManagerOp)

		// Generate websocket_handler.go
		wsHandlerOp, err := g.generateWebSocketHandler()
		if err != nil {
			return nil, err
		}
		ops = append(ops, wsHandlerOp)

		// Generate presence.go
		presenceOp, err := g.generatePresence()
		if err != nil {
			return nil, err
		}
		ops = append(ops, presenceOp)

		// Generate rooms.go
		roomsOp, err := g.generateRooms()
		if err != nil {
			return nil, err
		}
		ops = append(ops, roomsOp)

		// Generate subscription helpers if models are provided
		if len(g.models) > 0 {
			helpersOp, err := g.generateSubscriptionHelpers()
			if err != nil {
				return nil, err
			}
			ops = append(ops, helpersOp)
		}
	}

	return ops, nil
}

func (g *Generator) generateEvents() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "events", "events.go")

	data := map[string]interface{}{}

	content, err := g.renderer.RenderFS(templatesFS, "templates/events.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateMemoryBus() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "events", "memory_bus.go")

	data := map[string]interface{}{}

	content, err := g.renderer.RenderFS(templatesFS, "templates/memory_bus.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateNATSBus() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "events", "nats_bus.go")

	data := map[string]interface{}{}

	content, err := g.renderer.RenderFS(templatesFS, "templates/nats_bus.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateConnectionManager() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "realtime", "connection_manager.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/connection_manager.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateWebSocketHandler() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "handlers", "websocket_handler.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/websocket_handler.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generatePresence() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "realtime", "presence.go")

	data := map[string]interface{}{
		"ModulePath": g.modulePath,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/presence.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateRooms() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "realtime", "rooms.go")

	data := map[string]interface{}{}

	content, err := g.renderer.RenderFS(templatesFS, "templates/rooms.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

func (g *Generator) generateSubscriptionHelpers() (generator.Operation, error) {
	path := filepath.Join(g.projectPath, "internal", "realtime", "subscription_helpers.go")

	data := SubscriptionHelpersData{
		Models: g.models,
	}

	content, err := g.renderer.RenderFS(templatesFS, "templates/subscription_helpers.go.tmpl", data)
	if err != nil {
		return nil, err
	}

	return &generator.WriteFileOp{
		Path:    path,
		Content: content,
		Mode:    0644,
	}, nil
}

// SubscriptionHelpersData is the template data for subscription helpers
type SubscriptionHelpersData struct {
	Models []ModelHelper
}
