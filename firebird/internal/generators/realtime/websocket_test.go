package realtime

import (
	"os"
	"testing"

	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketGeneration(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)
	require.Len(t, ops, 7, "Should generate 7 files: events.go, memory_bus.go, nats_bus.go, connection_manager.go, websocket_handler.go, presence.go, rooms.go")

	// Verify all files are generated
	connManagerFound := false
	wsHandlerFound := false

	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			content := string(writeOp.Content)

			switch writeOp.Path {
			case "/test/project/internal/realtime/connection_manager.go":
				connManagerFound = true
				// Verify connection manager has all required components
				assert.Contains(t, content, "type Connection struct")
				assert.Contains(t, content, "type ConnectionManager struct")
				assert.Contains(t, content, "func NewConnectionManager")
				assert.Contains(t, content, "func (cm *ConnectionManager) Register")
				assert.Contains(t, content, "func (cm *ConnectionManager) Unregister")
				assert.Contains(t, content, "func (c *Connection) Subscribe")
				assert.Contains(t, content, "func (c *Connection) Unsubscribe")
				assert.Contains(t, content, "func (c *Connection) WritePump")
				assert.Contains(t, content, "func (c *Connection) ReadPump")
				assert.Contains(t, content, "type ClientMessage struct")
				assert.Contains(t, content, "github.com/test/project/internal/events")

			case "/test/project/internal/handlers/websocket_handler.go":
				wsHandlerFound = true
				// Verify WebSocket handler has all required components
				assert.Contains(t, content, "type WebSocketHandler struct")
				assert.Contains(t, content, "func NewWebSocketHandler")
				assert.Contains(t, content, "func (h *WebSocketHandler) HandleWebSocket")
				assert.Contains(t, content, "upgrader.Upgrade")
				assert.Contains(t, content, "github.com/test/project/internal/realtime")
			}
		}
	}

	assert.True(t, connManagerFound, "connection_manager.go should be generated")
	assert.True(t, wsHandlerFound, "websocket_handler.go should be generated")
}

func TestConnectionManagerFeatures(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find connection_manager.go
	var connManagerContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/realtime/connection_manager.go" {
				connManagerContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, connManagerContent)

	// Verify key features
	assert.Contains(t, connManagerContent, "sync.RWMutex", "Should use mutex for thread safety")
	assert.Contains(t, connManagerContent, "uuid.New()", "Should generate connection IDs")
	assert.Contains(t, connManagerContent, "make(chan []byte, 256)", "Should create buffered send channel")
	assert.Contains(t, connManagerContent, "Subscriptions map[string]bool", "Should track subscriptions")
	assert.Contains(t, connManagerContent, "Metadata", "Should support metadata")
	assert.Contains(t, connManagerContent, "forwardEvents", "Should forward events to WebSocket")
	assert.Contains(t, connManagerContent, "54 * time.Second", "Should ping every 54 seconds")
	assert.Contains(t, connManagerContent, "60 * time.Second", "Should have 60 second read deadline")
}

func TestWebSocketHandlerFeatures(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find websocket_handler.go
	var wsHandlerContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/handlers/websocket_handler.go" {
				wsHandlerContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, wsHandlerContent)

	// Verify key features
	assert.Contains(t, wsHandlerContent, "var upgrader = websocket.Upgrader", "Should define upgrader")
	assert.Contains(t, wsHandlerContent, "ReadBufferSize:  1024", "Should set read buffer size")
	assert.Contains(t, wsHandlerContent, "WriteBufferSize: 1024", "Should set write buffer size")
	assert.Contains(t, wsHandlerContent, "CheckOrigin", "Should have origin check")
	assert.Contains(t, wsHandlerContent, "upgrader.Upgrade(w, r, nil)", "Should upgrade connection")
	assert.Contains(t, wsHandlerContent, "go connection.WritePump()", "Should start write pump")
	assert.Contains(t, wsHandlerContent, "go connection.ReadPump()", "Should start read pump")
}

func TestClientMessageHandling(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find connection_manager.go
	var connManagerContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/realtime/connection_manager.go" {
				connManagerContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, connManagerContent)

	// Verify message handling
	assert.Contains(t, connManagerContent, `Action string`, "Should have Action field")
	assert.Contains(t, connManagerContent, `Topic  string`, "Should have Topic field")
	assert.Contains(t, connManagerContent, `Data   map[string]interface{}`, "Should have Data field")
	assert.Contains(t, connManagerContent, `case "subscribe"`, "Should handle subscribe action")
	assert.Contains(t, connManagerContent, `case "unsubscribe"`, "Should handle unsubscribe action")
	assert.Contains(t, connManagerContent, `case "publish"`, "Should handle publish action")
	assert.Contains(t, connManagerContent, `json.Unmarshal(message, &msg)`, "Should unmarshal JSON messages")
}

func TestConnectionLifecycle(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find connection_manager.go
	var connManagerContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/realtime/connection_manager.go" {
				connManagerContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, connManagerContent)

	// Verify connection lifecycle
	assert.Contains(t, connManagerContent, "connection registered", "Should log registration")
	assert.Contains(t, connManagerContent, "connection unregistered", "Should log unregistration")
	assert.Contains(t, connManagerContent, "close(connection.Send)", "Should close send channel")
	assert.Contains(t, connManagerContent, "c.Conn.Close()", "Should close WebSocket connection")
	assert.Contains(t, connManagerContent, "defer func() {", "Should use defer for cleanup")
	assert.Contains(t, connManagerContent, "websocket.CloseMessage", "Should send close message")
}

func TestPingPongKeepalive(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find connection_manager.go
	var connManagerContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/realtime/connection_manager.go" {
				connManagerContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, connManagerContent)

	// Verify ping/pong keepalive
	assert.Contains(t, connManagerContent, "time.NewTicker(54 * time.Second)", "Should create ping ticker")
	assert.Contains(t, connManagerContent, "websocket.PingMessage", "Should send ping messages")
	assert.Contains(t, connManagerContent, "SetPongHandler", "Should handle pong messages")
	assert.Contains(t, connManagerContent, "SetReadDeadline", "Should set read deadline")
	assert.Contains(t, connManagerContent, "SetWriteDeadline", "Should set write deadline")
}

func TestGeneratorWithoutModulePath(t *testing.T) {
	// Test backward compatibility with New() constructor
	gen := New("/test/project")
	assert.NotNil(t, gen)
	assert.Equal(t, "/test/project", gen.projectPath)
	assert.Empty(t, gen.modulePath)
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}
