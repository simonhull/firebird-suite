package migrate

import (
	"testing"
)

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "PostgreSQL URL with password",
			input: "postgres://user:secret123@localhost:5432/mydb",
			want:  "postgres://user:****@localhost:5432/mydb",
		},
		{
			name:  "PostgreSQL URL with complex password",
			input: "postgres://user:p@ssw0rd!@example.com:5432/db",
			want:  "postgres://user:****@example.com:5432/db",
		},
		{
			name:  "PostgreSQL URL with no password",
			input: "postgres://user@localhost:5432/mydb",
			want:  "postgres://user@localhost:5432/mydb",
		},
		{
			name:  "PostgreSQL URL with query params",
			input: "postgres://admin:pass123@db.example.com:5432/production?sslmode=require",
			want:  "postgres://admin:****@db.example.com:5432/production?sslmode=require",
		},
		{
			name:  "MySQL DSN with password",
			input: "user:password@tcp(localhost:3306)/mydb",
			want:  "user:****@tcp(localhost:3306)/mydb",
		},
		{
			name:  "MySQL URL format (from connection.go)",
			input: "mysql://root:secret@tcp(localhost:3306)/db",
			want:  "mysql://root:****@tcp(localhost:3306)/db",
		},
		{
			name:  "MySQL DSN with unix socket",
			input: "user:pass@unix(/tmp/mysql.sock)/mydb",
			want:  "user:****@unix(/tmp/mysql.sock)/mydb",
		},
		{
			name:  "SQLite file path (no password)",
			input: "mydb.db",
			want:  "mydb.db",
		},
		{
			name:  "SQLite URL",
			input: "sqlite3://./data/mydb.db",
			want:  "sqlite3://./data/mydb.db",
		},
		{
			name:  "PostgreSQL with special characters in password",
			input: "postgres://user:p@ss:word@localhost:5432/db",
			want:  "postgres://user:****@localhost:5432/db",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskPassword(tt.input)
			if got != tt.want {
				t.Errorf("MaskPassword(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMaskURLPassword(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid URL with password",
			input: "postgres://admin:secretpass@localhost:5432/db",
			want:  "postgres://admin:****@localhost:5432/db",
		},
		{
			name:  "URL without password",
			input: "postgres://admin@localhost:5432/db",
			want:  "postgres://admin@localhost:5432/db",
		},
		{
			name:  "URL with empty password",
			input: "postgres://admin:@localhost:5432/db",
			want:  "postgres://admin:****@localhost:5432/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskURLPassword(tt.input)
			if got != tt.want {
				t.Errorf("maskURLPassword(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMaskMySQLDSN(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard MySQL DSN",
			input: "root:password@tcp(localhost:3306)/mydb",
			want:  "root:****@tcp(localhost:3306)/mydb",
		},
		{
			name:  "DSN with unix socket",
			input: "user:pass@unix(/var/run/mysqld/mysqld.sock)/db",
			want:  "user:****@unix(/var/run/mysqld/mysqld.sock)/db",
		},
		{
			name:  "DSN with special characters",
			input: "admin:p@ss:w0rd@tcp(db:3306)/prod",
			want:  "admin:****@tcp(db:3306)/prod",
		},
		{
			name:  "DSN without password (malformed)",
			input: "user@tcp(localhost:3306)/db",
			want:  "user@tcp(localhost:3306)/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskMySQLDSN(tt.input)
			if got != tt.want {
				t.Errorf("maskMySQLDSN(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
