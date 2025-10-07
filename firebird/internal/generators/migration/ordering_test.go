package migration

import (
	"testing"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
)

func TestDependencyOrdering(t *testing.T) {
	tests := []struct {
		name      string
		resources []*schema.Definition
		expected  []string
		wantErr   bool
	}{
		{
			name: "simple dependency - comment depends on post",
			resources: []*schema.Definition{
				{
					Name: "Comment",
					Spec: schema.Spec{
						TableName: "comments",
						Fields: []schema.Field{
							{
								Name: "post_id",
								Type: "int64",
								Tags: map[string]string{"fk": "posts.id"},
							},
						},
					},
				},
				{
					Name: "Post",
					Spec: schema.Spec{
						TableName: "posts",
						Fields: []schema.Field{
							{Name: "id", Type: "int64"},
						},
					},
				},
			},
			expected: []string{"Post", "Comment"},
			wantErr:  false,
		},
		{
			name: "multiple dependencies - comment depends on post and user",
			resources: []*schema.Definition{
				{
					Name: "Comment",
					Spec: schema.Spec{
						TableName: "comments",
						Fields: []schema.Field{
							{Name: "post_id", Type: "int64", Tags: map[string]string{"fk": "posts.id"}},
							{Name: "user_id", Type: "int64", Tags: map[string]string{"fk": "users.id"}},
						},
					},
				},
				{
					Name: "Post",
					Spec: schema.Spec{TableName: "posts", Fields: []schema.Field{{Name: "id", Type: "int64"}}},
				},
				{
					Name: "User",
					Spec: schema.Spec{TableName: "users", Fields: []schema.Field{{Name: "id", Type: "int64"}}},
				},
			},
			// Comment must be last, Post and User can be in any order
			expected: []string{"Comment"}, // We'll check Comment is last
			wantErr:  false,
		},
		{
			name: "circular dependency",
			resources: []*schema.Definition{
				{
					Name: "A",
					Spec: schema.Spec{
						TableName: "a",
						Fields: []schema.Field{
							{Name: "b_id", Type: "int64", Tags: map[string]string{"fk": "b.id"}},
						},
					},
				},
				{
					Name: "B",
					Spec: schema.Spec{
						TableName: "b",
						Fields: []schema.Field{
							{Name: "a_id", Type: "int64", Tags: map[string]string{"fk": "a.id"}},
						},
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "no dependencies - all independent",
			resources: []*schema.Definition{
				{
					Name:      "Post",
					Spec:      schema.Spec{TableName: "posts", Fields: []schema.Field{{Name: "id", Type: "int64"}}},
				},
				{
					Name:      "User",
					Spec:      schema.Spec{TableName: "users", Fields: []schema.Field{{Name: "id", Type: "int64"}}},
				},
				{
					Name:      "Tag",
					Spec:      schema.Spec{TableName: "tags", Fields: []schema.Field{{Name: "id", Type: "int64"}}},
				},
			},
			expected: []string{"Post", "User", "Tag"}, // Any order is valid
			wantErr:  false,
		},
		{
			name: "chain dependency - A -> B -> C",
			resources: []*schema.Definition{
				{
					Name: "C",
					Spec: schema.Spec{
						TableName: "c",
						Fields: []schema.Field{
							{Name: "b_id", Type: "int64", Tags: map[string]string{"fk": "b.id"}},
						},
					},
				},
				{
					Name: "A",
					Spec: schema.Spec{
						TableName: "a",
						Fields:    []schema.Field{{Name: "id", Type: "int64"}},
					},
				},
				{
					Name: "B",
					Spec: schema.Spec{
						TableName: "b",
						Fields: []schema.Field{
							{Name: "a_id", Type: "int64", Tags: map[string]string{"fk": "a.id"}},
						},
					},
				},
			},
			expected: []string{"A", "B", "C"},
			wantErr:  false,
		},
		{
			name: "partial dependencies - mixed",
			resources: []*schema.Definition{
				{
					Name: "Comment",
					Spec: schema.Spec{
						TableName: "comments",
						Fields: []schema.Field{
							{Name: "post_id", Type: "int64", Tags: map[string]string{"fk": "posts.id"}},
						},
					},
				},
				{
					Name:      "Tag",
					Spec:      schema.Spec{TableName: "tags", Fields: []schema.Field{{Name: "id", Type: "int64"}}},
				},
				{
					Name:      "Post",
					Spec:      schema.Spec{TableName: "posts", Fields: []schema.Field{{Name: "id", Type: "int64"}}},
				},
			},
			// Post must come before Comment, Tag can be anywhere
			expected: []string{"Post", "Comment"}, // Check relative order
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := BuildDependencyGraph(tt.resources)
			if err != nil {
				t.Fatalf("BuildDependencyGraph failed: %v", err)
			}

			sorted, err := graph.TopologicalSort()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("TopologicalSort failed: %v", err)
			}

			if len(sorted) != len(tt.resources) {
				t.Errorf("expected %d resources, got %d", len(tt.resources), len(sorted))
			}

			// Special handling for tests with multiple valid orderings
			if tt.name == "multiple dependencies - comment depends on post and user" {
				// Check that Comment is last
				if sorted[len(sorted)-1] != "Comment" {
					t.Errorf("Comment should be last, but got: %v", sorted)
				}
				return
			}

			if tt.name == "no dependencies - all independent" {
				// Any order is valid, just check all are present
				if len(sorted) != 3 {
					t.Errorf("expected 3 resources, got %d", len(sorted))
				}
				return
			}

			if tt.name == "partial dependencies - mixed" {
				// Check that Post comes before Comment
				postIdx := -1
				commentIdx := -1
				for i, name := range sorted {
					if name == "Post" {
						postIdx = i
					}
					if name == "Comment" {
						commentIdx = i
					}
				}
				if postIdx == -1 || commentIdx == -1 {
					t.Errorf("Post and Comment not found in sorted list")
				}
				if postIdx >= commentIdx {
					t.Errorf("Post (index %d) should come before Comment (index %d)", postIdx, commentIdx)
				}
				return
			}

			// Verify order is correct (dependencies come before dependents)
			for i, name := range sorted {
				if i >= len(tt.expected) {
					break
				}
				if name != tt.expected[i] {
					t.Errorf("position %d: expected %s, got %s (full order: %v)", i, tt.expected[i], name, sorted)
				}
			}
		})
	}
}

func TestBuildDependencyGraph(t *testing.T) {
	resources := []*schema.Definition{
		{
			Name: "Comment",
			Spec: schema.Spec{
				TableName: "comments",
				Fields: []schema.Field{
					{Name: "post_id", Type: "int64", Tags: map[string]string{"fk": "posts.id"}},
					{Name: "user_id", Type: "int64", Tags: map[string]string{"fk": "users.id"}},
				},
			},
		},
		{
			Name:      "Post",
			Spec:      schema.Spec{TableName: "posts"},
		},
		{
			Name:      "User",
			Spec:      schema.Spec{TableName: "users"},
		},
	}

	graph, err := BuildDependencyGraph(resources)
	if err != nil {
		t.Fatalf("BuildDependencyGraph failed: %v", err)
	}

	// Check nodes were created
	if len(graph.nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(graph.nodes))
	}

	// Check Comment dependencies
	commentNode := graph.GetNode("Comment")
	if commentNode == nil {
		t.Fatal("Comment node not found")
	}
	if len(commentNode.DependsOn) != 2 {
		t.Errorf("expected Comment to have 2 dependencies, got %d", len(commentNode.DependsOn))
	}

	// Check Post has no dependencies
	postNode := graph.GetNode("Post")
	if postNode == nil {
		t.Fatal("Post node not found")
	}
	if len(postNode.DependsOn) != 0 {
		t.Errorf("expected Post to have 0 dependencies, got %d", len(postNode.DependsOn))
	}
}

func TestGetDependencies(t *testing.T) {
	resources := []*schema.Definition{
		{
			Name: "Comment",
			Spec: schema.Spec{
				TableName: "comments",
				Fields: []schema.Field{
					{Name: "post_id", Type: "int64", Tags: map[string]string{"fk": "posts.id"}},
				},
			},
		},
		{
			Name:      "Post",
			Spec:      schema.Spec{TableName: "posts"},
		},
	}

	graph, err := BuildDependencyGraph(resources)
	if err != nil {
		t.Fatalf("BuildDependencyGraph failed: %v", err)
	}

	// Comment should depend on Post
	deps := graph.GetDependencies("Comment")
	if len(deps) != 1 {
		t.Errorf("expected 1 dependency for Comment, got %d", len(deps))
	}
	if len(deps) > 0 && deps[0] != "Post" {
		t.Errorf("expected Comment to depend on Post, got %s", deps[0])
	}

	// Post should have no dependencies
	deps = graph.GetDependencies("Post")
	if len(deps) != 0 {
		t.Errorf("expected 0 dependencies for Post, got %d", len(deps))
	}

	// Non-existent resource should return empty slice
	deps = graph.GetDependencies("NonExistent")
	if len(deps) != 0 {
		t.Errorf("expected 0 dependencies for non-existent resource, got %d", len(deps))
	}
}

func TestHasCircularDependency(t *testing.T) {
	t.Run("circular dependency detected", func(t *testing.T) {
		resources := []*schema.Definition{
			{
				Name: "A",
				Spec: schema.Spec{
					TableName: "a",
					Fields: []schema.Field{
						{Name: "b_id", Type: "int64", Tags: map[string]string{"fk": "b.id"}},
					},
				},
			},
			{
				Name: "B",
				Spec: schema.Spec{
					TableName: "b",
					Fields: []schema.Field{
						{Name: "a_id", Type: "int64", Tags: map[string]string{"fk": "a.id"}},
					},
				},
			},
		}

		graph, err := BuildDependencyGraph(resources)
		if err != nil {
			t.Fatalf("BuildDependencyGraph failed: %v", err)
		}

		hasCycle, msg := graph.HasCircularDependency()
		if !hasCycle {
			t.Error("expected circular dependency to be detected")
		}
		if msg == "" {
			t.Error("expected error message for circular dependency")
		}
	})

	t.Run("no circular dependency", func(t *testing.T) {
		resources := []*schema.Definition{
			{
				Name: "Comment",
				Spec: schema.Spec{
					TableName: "comments",
					Fields: []schema.Field{
						{Name: "post_id", Type: "int64", Tags: map[string]string{"fk": "posts.id"}},
					},
				},
			},
			{
				Name:      "Post",
				Spec:      schema.Spec{TableName: "posts"},
			},
		}

		graph, err := BuildDependencyGraph(resources)
		if err != nil {
			t.Fatalf("BuildDependencyGraph failed: %v", err)
		}

		hasCycle, msg := graph.HasCircularDependency()
		if hasCycle {
			t.Errorf("unexpected circular dependency detected: %s", msg)
		}
	})
}

func TestDuplicateDependencies(t *testing.T) {
	// Resource with multiple fields referencing the same table
	resources := []*schema.Definition{
		{
			Name: "Comment",
			Spec: schema.Spec{
				TableName: "comments",
				Fields: []schema.Field{
					{Name: "author_id", Type: "int64", Tags: map[string]string{"fk": "users.id"}},
					{Name: "reviewer_id", Type: "int64", Tags: map[string]string{"fk": "users.id"}},
				},
			},
		},
		{
			Name:      "User",
			Spec:      schema.Spec{TableName: "users"},
		},
	}

	graph, err := BuildDependencyGraph(resources)
	if err != nil {
		t.Fatalf("BuildDependencyGraph failed: %v", err)
	}

	deps := graph.GetDependencies("Comment")
	if len(deps) != 1 {
		t.Errorf("expected 1 unique dependency (User), got %d: %v", len(deps), deps)
	}
}
