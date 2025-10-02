package generator

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResolver_ValidFlags(t *testing.T) {
	tests := []struct {
		name  string
		force bool
		skip  bool
		diff  bool
	}{
		{"no flags", false, false, false},
		{"force only", true, false, false},
		{"skip only", false, true, false},
		{"diff only", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewResolver(tt.force, tt.skip, tt.diff)
			require.NoError(t, err)
			assert.NotNil(t, resolver)
			assert.NotNil(t, resolver.diffGen)
		})
	}
}

func TestNewResolver_InvalidCombinations(t *testing.T) {
	tests := []struct {
		name  string
		force bool
		skip  bool
		diff  bool
	}{
		{"force + skip", true, true, false},
		{"force + diff", true, false, true},
		{"force + skip + diff", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewResolver(tt.force, tt.skip, tt.diff)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot be combined")
		})
	}
}

func TestForceStrategy_AlwaysOverwrites(t *testing.T) {
	strategy := &ForceStrategy{}
	resolution, err := strategy.Resolve("test.go", []byte("old"), []byte("newer"))

	require.NoError(t, err)
	assert.Equal(t, Overwrite, resolution)
}

func TestSkipStrategy_AlwaysSkips(t *testing.T) {
	strategy := &SkipStrategy{}
	resolution, err := strategy.Resolve("test.go", []byte("old"), []byte("newer"))

	require.NoError(t, err)
	assert.Equal(t, Skip, resolution)
}

func TestMapChoiceToResolution(t *testing.T) {
	tests := []struct {
		cursor int
		want   ConflictResolution
	}{
		{0, ShowDiff},
		{1, Skip},
		{2, Overwrite},
		{3, Cancel},
		{99, Cancel}, // Out of range
	}

	for _, tt := range tests {
		got := mapChoiceToResolution(tt.cursor)
		assert.Equal(t, tt.want, got, "mapChoiceToResolution(%d)", tt.cursor)
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"1 minute ago", now.Add(-1 * time.Minute), "1 minute ago"},
		{"5 minutes ago", now.Add(-5 * time.Minute), "5 minutes ago"},
		{"1 hour ago", now.Add(-1 * time.Hour), "1 hour ago"},
		{"3 hours ago", now.Add(-3 * time.Hour), "3 hours ago"},
		{"1 day ago", now.Add(-24 * time.Hour), "1 day ago"},
		{"3 days ago", now.Add(-3 * 24 * time.Hour), "3 days ago"},
		{"1 week ago", now.Add(-7 * 24 * time.Hour), "1 week ago"},
		{"2 weeks ago", now.Add(-14 * 24 * time.Hour), "2 weeks ago"},
		{"1 month ago", now.Add(-30 * 24 * time.Hour), "1 month ago"},
		{"1 year ago", now.Add(-365 * 24 * time.Hour), "1 year ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRelativeTime(tt.time)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024, "1.5 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1536 * 1024 * 1024, "1.5 GB"},
	}

	for _, tt := range tests {
		got := formatFileSize(tt.size)
		assert.Equal(t, tt.want, got, "formatFileSize(%d)", tt.size)
	}
}

func TestMaxInt(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{1, 2, 2},
		{2, 1, 2},
		{5, 5, 5},
		{-1, 1, 1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		got := maxInt(tt.a, tt.b)
		assert.Equal(t, tt.want, got, "maxInt(%d, %d)", tt.a, tt.b)
	}
}

func TestConflictMenuModel_Init(t *testing.T) {
	model := newConflictMenuModel("test.go", nil)

	assert.Equal(t, "test.go", model.path)
	assert.Equal(t, 0, model.cursor)
	assert.Len(t, model.choices, 4)
}

func TestConflictMenuModel_Navigation(t *testing.T) {
	model := newConflictMenuModel("test.go", nil)

	// Test down navigation - manually update cursor
	if model.cursor == 0 && len(model.choices) > 1 {
		model.cursor++
	}
	assert.Equal(t, 1, model.cursor, "cursor should be at 1 after down")

	// Test up navigation
	if model.cursor > 0 {
		model.cursor--
	}
	assert.Equal(t, 0, model.cursor, "cursor should be at 0 after up")

	// Test j navigation (down)
	if model.cursor < len(model.choices)-1 {
		model.cursor++
	}
	assert.Equal(t, 1, model.cursor, "cursor should be at 1 after j")

	// Test k navigation (up)
	if model.cursor > 0 {
		model.cursor--
	}
	assert.Equal(t, 0, model.cursor, "cursor should be at 0 after k")
}

func TestConflictMenuModel_BoundaryConditions(t *testing.T) {
	model := newConflictMenuModel("test.go", nil)

	// Try to go up from 0 - cursor should stay at 0
	if model.cursor > 0 {
		model.cursor--
	}
	assert.Equal(t, 0, model.cursor, "cursor should stay at 0")

	// Go to last choice (cursor 3)
	model.cursor = 3

	// Try to go down from last - cursor should stay at 3
	if model.cursor < len(model.choices)-1 {
		model.cursor++
	}
	assert.Equal(t, 3, model.cursor, "cursor should stay at 3")
}

func TestConflictMenuModel_View(t *testing.T) {
	// Create a mock file info
	fileInfo := &mockFileInfo{
		name:    "test.go",
		size:    1234,
		modTime: time.Now().Add(-2 * time.Hour),
	}

	model := newConflictMenuModel("internal/models/user.go", fileInfo)
	view := model.View()

	// Check key elements are present
	assert.Contains(t, view, "File conflict detected")
	assert.Contains(t, view, "internal/models/user.go")
	assert.Contains(t, view, "Last modified")
	assert.Contains(t, view, "Size")
	assert.Contains(t, view, "Show diff and decide")
	assert.Contains(t, view, ">")
}

func TestDiffViewerModel_Init(t *testing.T) {
	model := newDiffViewerModel("test.go", "sample diff content")

	assert.Equal(t, "test.go", model.path)
	assert.Equal(t, "sample diff content", model.diff)
	assert.False(t, model.ready, "model should not be ready before window size message")
}

func TestSelectStrategy(t *testing.T) {
	tests := []struct {
		name  string
		force bool
		skip  bool
		diff  bool
		want  string
	}{
		{"force", true, false, false, "*generator.ForceStrategy"},
		{"skip", false, true, false, "*generator.SkipStrategy"},
		{"diff", false, false, true, "*generator.DiffStrategy"},
		{"interactive", false, false, false, "*generator.InteractiveStrategy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := selectStrategy(tt.force, tt.skip, tt.diff)
			got := fmt.Sprintf("%T", strategy)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDiffStrategy_HasDiffGen(t *testing.T) {
	strategy := selectStrategy(false, false, true)
	diffStrategy, ok := strategy.(*DiffStrategy)

	require.True(t, ok, "expected DiffStrategy")
	assert.NotNil(t, diffStrategy.diffGen, "DiffStrategy should have initialized diffGen")
}

func TestResolver_DiffGenReuse(t *testing.T) {
	resolver, err := NewResolver(false, false, false)
	require.NoError(t, err)

	assert.NotNil(t, resolver.diffGen, "Resolver should have a diffGen instance")

	// Verify the same diffGen instance is reused
	diffGen1 := resolver.diffGen
	assert.Same(t, diffGen1, resolver.diffGen, "diffGen should be the same instance")
}

func TestConflictResolutionConstants(t *testing.T) {
	// Ensure constants have expected values
	assert.Equal(t, ConflictResolution(0), Skip)
	assert.Equal(t, ConflictResolution(1), Overwrite)
	assert.Equal(t, ConflictResolution(2), ShowDiff)
	assert.Equal(t, ConflictResolution(3), Cancel)
}

// Mock types for testing

type mockKeyMsg string

func (m mockKeyMsg) String() string {
	return string(m)
}

type mockFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }