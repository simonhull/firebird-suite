package realtime

import (
	"testing"

	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresenceAndRoomGeneration(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)
	require.Len(t, ops, 7, "Should generate 7 files: events.go, memory_bus.go, nats_bus.go, connection_manager.go, websocket_handler.go, presence.go, rooms.go")

	// Verify presence and room files are generated
	presenceFound := false
	roomsFound := false

	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			content := string(writeOp.Content)

			switch writeOp.Path {
			case "/test/project/internal/realtime/presence.go":
				presenceFound = true
				// Verify presence manager has all required components
				assert.Contains(t, content, "type PresenceInfo struct")
				assert.Contains(t, content, "type PresenceManager struct")
				assert.Contains(t, content, "func NewPresenceManager")
				assert.Contains(t, content, "func (pm *PresenceManager) Track")
				assert.Contains(t, content, "func (pm *PresenceManager) Untrack")
				assert.Contains(t, content, "func (pm *PresenceManager) List")
				assert.Contains(t, content, "func (pm *PresenceManager) IsPresent")
				assert.Contains(t, content, "func (pm *PresenceManager) UpdateMetadata")
				assert.Contains(t, content, "github.com/test/project/internal/events")

			case "/test/project/internal/realtime/rooms.go":
				roomsFound = true
				// Verify room manager has all required components
				assert.Contains(t, content, "type Room struct")
				assert.Contains(t, content, "type RoomManager struct")
				assert.Contains(t, content, "func NewRoomManager")
				assert.Contains(t, content, "func (rm *RoomManager) CreateRoom")
				assert.Contains(t, content, "func (rm *RoomManager) GetOrCreateRoom")
				assert.Contains(t, content, "func (rm *RoomManager) Join")
				assert.Contains(t, content, "func (rm *RoomManager) Leave")
				assert.Contains(t, content, "func (rm *RoomManager) Broadcast")
				assert.Contains(t, content, "func (rm *RoomManager) BroadcastExcept")
			}
		}
	}

	assert.True(t, presenceFound, "presence.go should be generated")
	assert.True(t, roomsFound, "rooms.go should be generated")
}

func TestPresenceFeatures(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find presence.go
	var presenceContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/realtime/presence.go" {
				presenceContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, presenceContent)

	// Verify PresenceInfo structure
	assert.Contains(t, presenceContent, "UserID       string", "Should have UserID field")
	assert.Contains(t, presenceContent, "ConnectionID string", "Should have ConnectionID field")
	assert.Contains(t, presenceContent, "Metadata     map[string]interface{}", "Should have Metadata field")
	assert.Contains(t, presenceContent, "JoinedAt     time.Time", "Should have JoinedAt field")
	assert.Contains(t, presenceContent, "LastSeenAt   time.Time", "Should have LastSeenAt field")

	// Verify presence tracking features
	assert.Contains(t, presenceContent, "sync.RWMutex", "Should use mutex for thread safety")
	assert.Contains(t, presenceContent, "presence map[string]map[string][]*PresenceInfo", "Should track presence by topic and user")
	assert.Contains(t, presenceContent, "presence_join", "Should broadcast join events")
	assert.Contains(t, presenceContent, "presence_leave", "Should broadcast leave events")
	assert.Contains(t, presenceContent, "presence_update", "Should broadcast update events")
}

func TestPresenceEventBroadcasting(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find presence.go
	var presenceContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/realtime/presence.go" {
				presenceContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, presenceContent)

	// Verify event broadcasting
	assert.Contains(t, presenceContent, `pm.eventBus.Publish(ctx, topic+".presence_join"`, "Should publish join events")
	assert.Contains(t, presenceContent, `pm.eventBus.Publish(ctx, topic+".presence_leave"`, "Should publish leave events")
	assert.Contains(t, presenceContent, `pm.eventBus.Publish(ctx, topic+".presence_update"`, "Should publish update events")
}

func TestRoomFeatures(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find rooms.go
	var roomsContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/realtime/rooms.go" {
				roomsContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, roomsContent)

	// Verify Room structure
	assert.Contains(t, roomsContent, "Name        string", "Should have Name field")
	assert.Contains(t, roomsContent, "Connections map[string]*Connection", "Should track connections")
	assert.Contains(t, roomsContent, "Metadata    map[string]interface{}", "Should have Metadata field")

	// Verify room management features
	assert.Contains(t, roomsContent, "sync.RWMutex", "Should use mutex for thread safety")
	assert.Contains(t, roomsContent, "rooms  map[string]*Room", "Should track rooms by name")
	assert.Contains(t, roomsContent, "room already exists", "Should check for duplicate rooms")
	assert.Contains(t, roomsContent, "room auto-created", "Should auto-create rooms")
}

func TestRoomBroadcasting(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find rooms.go
	var roomsContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/realtime/rooms.go" {
				roomsContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, roomsContent)

	// Verify broadcasting
	assert.Contains(t, roomsContent, "func (rm *RoomManager) Broadcast", "Should have Broadcast method")
	assert.Contains(t, roomsContent, "func (rm *RoomManager) BroadcastExcept", "Should have BroadcastExcept method")
	assert.Contains(t, roomsContent, "for _, conn := range room.Connections", "Should iterate over connections")
	assert.Contains(t, roomsContent, "conn.Send <- message", "Should send to connection channel")
	assert.Contains(t, roomsContent, "if connID == excludeConnectionID", "Should exclude specific connection")
}

func TestRoomManagement(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find rooms.go
	var roomsContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/realtime/rooms.go" {
				roomsContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, roomsContent)

	// Verify room lifecycle methods
	assert.Contains(t, roomsContent, "func (rm *RoomManager) CreateRoom", "Should have CreateRoom")
	assert.Contains(t, roomsContent, "func (rm *RoomManager) GetOrCreateRoom", "Should have GetOrCreateRoom")
	assert.Contains(t, roomsContent, "func (rm *RoomManager) GetRoom", "Should have GetRoom")
	assert.Contains(t, roomsContent, "func (rm *RoomManager) DeleteRoom", "Should have DeleteRoom")
	assert.Contains(t, roomsContent, "func (rm *RoomManager) Join", "Should have Join")
	assert.Contains(t, roomsContent, "func (rm *RoomManager) Leave", "Should have Leave")
	assert.Contains(t, roomsContent, "func (rm *RoomManager) ListRooms", "Should have ListRooms")
	assert.Contains(t, roomsContent, "func (rm *RoomManager) RoomSize", "Should have RoomSize")
}

func TestConnectionManagerIntegration(t *testing.T) {
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

	// Verify presence and room integration
	assert.Contains(t, connManagerContent, "presence    *PresenceManager", "Should have presence manager")
	assert.Contains(t, connManagerContent, "rooms       *RoomManager", "Should have room manager")
	assert.Contains(t, connManagerContent, "NewPresenceManager(eventBus, logger)", "Should initialize presence manager")
	assert.Contains(t, connManagerContent, "NewRoomManager(logger)", "Should initialize room manager")
	assert.Contains(t, connManagerContent, "func (cm *ConnectionManager) Presence()", "Should have Presence accessor")
	assert.Contains(t, connManagerContent, "func (cm *ConnectionManager) Rooms()", "Should have Rooms accessor")
}

func TestClientMessageActions(t *testing.T) {
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

	// Verify new actions
	assert.Contains(t, connManagerContent, `case "join":`, "Should handle join action")
	assert.Contains(t, connManagerContent, `case "leave":`, "Should handle leave action")
	assert.Contains(t, connManagerContent, `case "track":`, "Should handle track action")
	assert.Contains(t, connManagerContent, `c.Manager.rooms.Join(msg.Topic, c)`, "Should join room")
	assert.Contains(t, connManagerContent, `c.Manager.rooms.Leave(msg.Topic, c.ID)`, "Should leave room")
	assert.Contains(t, connManagerContent, `c.Manager.presence.Track(ctx, msg.Topic, userID, c.ID, metadata)`, "Should track presence")
}

func TestPresenceMultipleConnections(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find presence.go
	var presenceContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/realtime/presence.go" {
				presenceContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, presenceContent)

	// Verify support for multiple connections per user
	assert.Contains(t, presenceContent, "map[string]map[string][]*PresenceInfo", "Should support multiple connections per user")
	assert.Contains(t, presenceContent, "append(pm.presence[topic][userID], info)", "Should append to user's connections")
	assert.Contains(t, presenceContent, "len(pm.presence[topic][userID]) == 0", "Should check if user has no more connections")
}
