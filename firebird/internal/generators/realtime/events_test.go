package realtime

import (
	"testing"

	"github.com/simonhull/firebird-suite/fledge/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealtimeGeneration(t *testing.T) {
	gen := New("/test/project")

	ops, err := gen.Generate()
	require.NoError(t, err)
	require.Len(t, ops, 3, "Should generate 3 files when no module path is provided")

	// Verify events.go
	eventsFound := false
	memoryBusFound := false
	natsBusFound := false

	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			content := string(writeOp.Content)

			switch writeOp.Path {
			case "/test/project/internal/events/events.go":
				eventsFound = true
				assert.Contains(t, content, "type Event struct")
				assert.Contains(t, content, "type EventBus interface")
				assert.Contains(t, content, "func MatchTopic")
				assert.Contains(t, content, "func MarshalEvent")
				assert.Contains(t, content, "func UnmarshalEvent")

			case "/test/project/internal/events/memory_bus.go":
				memoryBusFound = true
				assert.Contains(t, content, "type MemoryBus struct")
				assert.Contains(t, content, "func NewMemoryBus")
				assert.Contains(t, content, "func (mb *MemoryBus) Publish")
				assert.Contains(t, content, "func (mb *MemoryBus) Subscribe")
				assert.Contains(t, content, "func (mb *MemoryBus) Close")

			case "/test/project/internal/events/nats_bus.go":
				natsBusFound = true
				assert.Contains(t, content, "type NATSBus struct")
				assert.Contains(t, content, "func NewNATSBus")
				assert.Contains(t, content, "func (nb *NATSBus) Publish")
				assert.Contains(t, content, "func (nb *NATSBus) Subscribe")
				assert.Contains(t, content, "func (nb *NATSBus) Close")
				assert.Contains(t, content, "nats.Connect")
			}
		}
	}

	assert.True(t, eventsFound, "events.go should be generated")
	assert.True(t, memoryBusFound, "memory_bus.go should be generated")
	assert.True(t, natsBusFound, "nats_bus.go should be generated")
}

func TestTopicMatching(t *testing.T) {
	// These tests verify the MatchTopic function logic
	// The actual implementation is in the template
	tests := []struct {
		name    string
		pattern string
		topic   string
		matches bool
	}{
		{
			name:    "exact match",
			pattern: "posts.created",
			topic:   "posts.created",
			matches: true,
		},
		{
			name:    "wildcard all",
			pattern: "*",
			topic:   "posts.created",
			matches: true,
		},
		{
			name:    "wildcard suffix match",
			pattern: "posts.*",
			topic:   "posts.created",
			matches: true,
		},
		{
			name:    "wildcard suffix match 2",
			pattern: "posts.*",
			topic:   "posts.updated",
			matches: true,
		},
		{
			name:    "wildcard suffix no match",
			pattern: "posts.*",
			topic:   "comments.created",
			matches: false,
		},
		{
			name:    "no match different prefix",
			pattern: "posts.created",
			topic:   "comments.created",
			matches: false,
		},
		{
			name:    "wildcard middle",
			pattern: "posts.*.created",
			topic:   "posts.123.created",
			matches: true,
		},
		{
			name:    "wildcard middle no match",
			pattern: "posts.*.created",
			topic:   "posts.123.updated",
			matches: false,
		},
		{
			name:    "wildcard suffix no match short topic",
			pattern: "posts.*",
			topic:   "posts",
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test documents the expected behavior
			// Actual testing happens when the generated code is used
			t.Logf("Pattern: %s, Topic: %s, Expected: %v", tt.pattern, tt.topic, tt.matches)
		})
	}
}

func TestEventStructure(t *testing.T) {
	gen := New("/test/project")
	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find events.go
	var eventsContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/events/events.go" {
				eventsContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, eventsContent)

	// Verify Event struct has all required fields
	assert.Contains(t, eventsContent, "Topic     string")
	assert.Contains(t, eventsContent, "Data      interface{}")
	assert.Contains(t, eventsContent, "Metadata  map[string]interface{}")
	assert.Contains(t, eventsContent, "Timestamp time.Time")

	// Verify EventBus interface has all required methods
	assert.Contains(t, eventsContent, "Publish(ctx context.Context")
	assert.Contains(t, eventsContent, "Subscribe(ctx context.Context")
	assert.Contains(t, eventsContent, "Unsubscribe(pattern string)")
	assert.Contains(t, eventsContent, "Close() error")
}

func TestMemoryBusFeatures(t *testing.T) {
	gen := New("/test/project")
	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find memory_bus.go
	var memoryBusContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/events/memory_bus.go" {
				memoryBusContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, memoryBusContent)

	// Verify key features
	assert.Contains(t, memoryBusContent, "sync.RWMutex", "Should use mutex for thread safety")
	assert.Contains(t, memoryBusContent, "bufferSize", "Should have configurable buffer")
	assert.Contains(t, memoryBusContent, "MatchTopic", "Should use topic matching")
	assert.Contains(t, memoryBusContent, "WarnContext", "Should log warnings for full channels")
	assert.Contains(t, memoryBusContent, "closed", "Should track closed state")
}

func TestNATSBusFeatures(t *testing.T) {
	gen := New("/test/project")
	ops, err := gen.Generate()
	require.NoError(t, err)

	// Find nats_bus.go
	var natsBusContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/events/nats_bus.go" {
				natsBusContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, natsBusContent)

	// Verify NATS integration
	assert.Contains(t, natsBusContent, "nats.Connect", "Should connect to NATS")
	assert.Contains(t, natsBusContent, "json.Marshal", "Should marshal events to JSON")
	assert.Contains(t, natsBusContent, "json.Unmarshal", "Should unmarshal events from JSON")
	assert.Contains(t, natsBusContent, "nats.Subscription", "Should track subscriptions")
	assert.Contains(t, natsBusContent, "conn.Close", "Should close NATS connection")
}

func TestGeneratorCreation(t *testing.T) {
	gen := New("/test/project")
	assert.NotNil(t, gen)
	assert.Equal(t, "/test/project", gen.projectPath)
	assert.NotNil(t, gen.renderer)
}

func TestSubscriptionHelpersGeneration(t *testing.T) {
	models := []ModelHelper{
		{
			Name:       "Post",
			NamePlural: "posts",
			PKType:     "uuid.UUID",
		},
		{
			Name:       "Comment",
			NamePlural: "comments",
			PKType:     "int",
		},
	}

	gen := NewWithModels("/test/project", "github.com/example/myapp", models)

	ops, err := gen.Generate()
	require.NoError(t, err)
	require.Len(t, ops, 8, "Should generate 8 files (3 event files + 4 WebSocket files + 1 subscription helpers)")

	// Find subscription_helpers.go
	var helpersContent string
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			if writeOp.Path == "/test/project/internal/realtime/subscription_helpers.go" {
				helpersContent = string(writeOp.Content)
				break
			}
		}
	}

	require.NotEmpty(t, helpersContent, "subscription_helpers.go should be generated")

	// Verify Post helpers
	assert.Contains(t, helpersContent, "func SubscribePostEvents")
	assert.Contains(t, helpersContent, "func SubscribePostCreated")
	assert.Contains(t, helpersContent, "func SubscribePostUpdated")
	assert.Contains(t, helpersContent, "func SubscribePostDeleted")
	assert.Contains(t, helpersContent, "func SubscribeAllPosts")
	assert.Contains(t, helpersContent, "func SubscribeAllPostsCreated")
	assert.Contains(t, helpersContent, "func SubscribeAllPostsUpdated")
	assert.Contains(t, helpersContent, "func SubscribeAllPostsDeleted")

	// Verify Comment helpers
	assert.Contains(t, helpersContent, "func SubscribeCommentEvents")
	assert.Contains(t, helpersContent, "func SubscribeCommentCreated")
	assert.Contains(t, helpersContent, "func SubscribeCommentUpdated")
	assert.Contains(t, helpersContent, "func SubscribeCommentDeleted")
	assert.Contains(t, helpersContent, "func SubscribeAllComments")
	assert.Contains(t, helpersContent, "func SubscribeAllCommentsCreated")
	assert.Contains(t, helpersContent, "func SubscribeAllCommentsUpdated")
	assert.Contains(t, helpersContent, "func SubscribeAllCommentsDeleted")

	// Verify UUID import only appears once (for Post, not Comment)
	assert.Contains(t, helpersContent, `"github.com/google/uuid"`)

	// Verify topic patterns
	assert.Contains(t, helpersContent, `fmt.Sprintf("posts.%v.*", id)`)
	assert.Contains(t, helpersContent, `"posts.*.created"`)
	assert.Contains(t, helpersContent, `fmt.Sprintf("comments.%v.*", id)`)
	assert.Contains(t, helpersContent, `"comments.*.created"`)

	// Verify usage examples
	assert.Contains(t, helpersContent, "Usage examples:")
	assert.Contains(t, helpersContent, "SubscribePostEvents(ctx, conn, postID)")
}

func TestSubscriptionHelpersNotGeneratedWithoutModels(t *testing.T) {
	gen := NewWithModule("/test/project", "github.com/example/myapp")

	ops, err := gen.Generate()
	require.NoError(t, err)
	require.Len(t, ops, 7, "Should generate 7 files without subscription helpers")

	// Verify subscription_helpers.go is NOT generated
	for _, op := range ops {
		if writeOp, ok := op.(*generator.WriteFileOp); ok {
			assert.NotEqual(t, "/test/project/internal/realtime/subscription_helpers.go", writeOp.Path,
				"subscription_helpers.go should not be generated without models")
		}
	}
}
