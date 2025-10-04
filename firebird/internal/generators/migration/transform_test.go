package migration

import (
	"testing"

	"github.com/simonhull/firebird-suite/firebird/internal/schema"
)

func TestTransformIndexes(t *testing.T) {
	tests := []struct {
		name      string
		indexes   []schema.Index
		tableName string
		want      []IndexData
	}{
		{
			name: "multi-column index",
			indexes: []schema.Index{
				{
					Name:    "idx_email_status",
					Columns: []string{"email", "status"},
					Unique:  false,
				},
			},
			tableName: "users",
			want: []IndexData{
				{
					Name:    "idx_email_status",
					Columns: []string{"email", "status"},
					Unique:  false,
				},
			},
		},
		{
			name: "unique index with WHERE clause",
			indexes: []schema.Index{
				{
					Columns: []string{"email"},
					Unique:  true,
					Where:   "deleted_at IS NULL",
				},
			},
			tableName: "users",
			want: []IndexData{
				{
					Name:    "uniq_users_email",
					Columns: []string{"email"},
					Unique:  true,
					Where:   "deleted_at IS NULL",
				},
			},
		},
		{
			name: "index with type",
			indexes: []schema.Index{
				{
					Name:    "idx_status_active",
					Columns: []string{"status"},
					Type:    "btree",
				},
			},
			tableName: "users",
			want: []IndexData{
				{
					Name:    "idx_status_active",
					Columns: []string{"status"},
					Type:    "btree",
				},
			},
		},
		{
			name: "auto-generated name for regular index",
			indexes: []schema.Index{
				{
					Columns: []string{"createdAt"},
					Unique:  false,
				},
			},
			tableName: "posts",
			want: []IndexData{
				{
					Name:    "idx_posts_created_at",
					Columns: []string{"created_at"},
					Unique:  false,
				},
			},
		},
		{
			name: "auto-generated name for unique index",
			indexes: []schema.Index{
				{
					Columns: []string{"slug"},
					Unique:  true,
				},
			},
			tableName: "articles",
			want: []IndexData{
				{
					Name:    "uniq_articles_slug",
					Columns: []string{"slug"},
					Unique:  true,
				},
			},
		},
		{
			name: "multi-column auto-generated name",
			indexes: []schema.Index{
				{
					Columns: []string{"userId", "postId"},
					Unique:  true,
				},
			},
			tableName: "likes",
			want: []IndexData{
				{
					Name:    "uniq_likes_user_id_post_id",
					Columns: []string{"user_id", "post_id"},
					Unique:  true,
				},
			},
		},
		{
			name:      "empty indexes",
			indexes:   []schema.Index{},
			tableName: "users",
			want:      []IndexData{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transformIndexes(tt.indexes, tt.tableName)

			if len(got) != len(tt.want) {
				t.Fatalf("transformIndexes() returned %d indexes, want %d", len(got), len(tt.want))
			}

			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("index[%d].Name = %q, want %q", i, got[i].Name, tt.want[i].Name)
				}

				if len(got[i].Columns) != len(tt.want[i].Columns) {
					t.Errorf("index[%d] has %d columns, want %d", i, len(got[i].Columns), len(tt.want[i].Columns))
				}

				for j := range got[i].Columns {
					if got[i].Columns[j] != tt.want[i].Columns[j] {
						t.Errorf("index[%d].Columns[%d] = %q, want %q", i, j, got[i].Columns[j], tt.want[i].Columns[j])
					}
				}

				if got[i].Unique != tt.want[i].Unique {
					t.Errorf("index[%d].Unique = %v, want %v", i, got[i].Unique, tt.want[i].Unique)
				}

				if got[i].Where != tt.want[i].Where {
					t.Errorf("index[%d].Where = %q, want %q", i, got[i].Where, tt.want[i].Where)
				}

				if got[i].Type != tt.want[i].Type {
					t.Errorf("index[%d].Type = %q, want %q", i, got[i].Type, tt.want[i].Type)
				}
			}
		})
	}
}

func TestGenerateIndexName(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		columns   []string
		unique    bool
		want      string
	}{
		{
			name:      "regular index single column",
			tableName: "users",
			columns:   []string{"email"},
			unique:    false,
			want:      "idx_users_email",
		},
		{
			name:      "unique index single column",
			tableName: "users",
			columns:   []string{"email"},
			unique:    true,
			want:      "uniq_users_email",
		},
		{
			name:      "multi-column regular index",
			tableName: "posts",
			columns:   []string{"user_id", "created_at"},
			unique:    false,
			want:      "idx_posts_user_id_created_at",
		},
		{
			name:      "multi-column unique index",
			tableName: "likes",
			columns:   []string{"user_id", "post_id"},
			unique:    true,
			want:      "uniq_likes_user_id_post_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateIndexName(tt.tableName, tt.columns, tt.unique)
			if got != tt.want {
				t.Errorf("generateIndexName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTransformRelationshipToFK(t *testing.T) {
	rel := schema.Relationship{
		Name:       "Author",
		Type:       "belongs_to",
		Model:      "User",
		ForeignKey: "author_id",
	}

	def := &schema.Definition{Name: "Post"}
	fk := transformRelationshipToFK(rel, "posts", def)

	if fk.Name != "fk_posts_author_id" {
		t.Errorf("fk.Name = %q, want %q", fk.Name, "fk_posts_author_id")
	}
	if fk.Column != "author_id" {
		t.Errorf("fk.Column = %q, want %q", fk.Column, "author_id")
	}
	if fk.ReferenceTable != "users" {
		t.Errorf("fk.ReferenceTable = %q, want %q", fk.ReferenceTable, "users")
	}
	if fk.ReferenceColumn != "id" {
		t.Errorf("fk.ReferenceColumn = %q, want %q", fk.ReferenceColumn, "id")
	}
	if fk.OnDelete != "CASCADE" {
		t.Errorf("fk.OnDelete = %q, want %q", fk.OnDelete, "CASCADE")
	}
	if fk.OnUpdate != "CASCADE" {
		t.Errorf("fk.OnUpdate = %q, want %q", fk.OnUpdate, "CASCADE")
	}
}

func TestPrepareMigrationDataWithForeignKeys(t *testing.T) {
	def := &schema.Definition{
		Name: "Post",
		Spec: schema.Spec{
			Fields: []schema.Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
				{Name: "author_id", Type: "uuid.UUID", DBType: "UUID"},
				{Name: "title", Type: "string", DBType: "VARCHAR(255)"},
			},
			Relationships: []schema.Relationship{
				{Name: "Author", Type: "belongs_to", Model: "User", ForeignKey: "author_id"},
			},
		},
	}

	data := PrepareMigrationData(def, PostgreSQL)

	if data.TableName != "posts" {
		t.Errorf("TableName = %q, want %q", data.TableName, "posts")
	}

	if len(data.ForeignKeys) != 1 {
		t.Fatalf("len(ForeignKeys) = %d, want 1", len(data.ForeignKeys))
	}

	fk := data.ForeignKeys[0]
	if fk.Name != "fk_posts_author_id" {
		t.Errorf("fk.Name = %q, want %q", fk.Name, "fk_posts_author_id")
	}
	if fk.Column != "author_id" {
		t.Errorf("fk.Column = %q, want %q", fk.Column, "author_id")
	}
	if fk.ReferenceTable != "users" {
		t.Errorf("fk.ReferenceTable = %q, want %q", fk.ReferenceTable, "users")
	}
}

func TestPrepareMigrationDataWithoutBelongsTo(t *testing.T) {
	def := &schema.Definition{
		Name: "User",
		Spec: schema.Spec{
			Fields: []schema.Field{
				{Name: "id", Type: "uuid.UUID", DBType: "UUID", PrimaryKey: true},
			},
			Relationships: []schema.Relationship{
				{Name: "Posts", Type: "has_many", Model: "Post", ForeignKey: "author_id"},
			},
		},
	}

	data := PrepareMigrationData(def, PostgreSQL)

	// has_many should not create FK constraints in this table
	if len(data.ForeignKeys) != 0 {
		t.Errorf("len(ForeignKeys) = %d, want 0 (has_many should not create FKs in this table)", len(data.ForeignKeys))
	}
}
