package generator

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*.tmpl
var testFS embed.FS

func TestNewRenderer(t *testing.T) {
	r := NewRenderer()
	assert.NotNil(t, r)
	assert.NotNil(t, r.funcMap)
	assert.NotNil(t, r.cache)
	assert.Empty(t, r.cache)
}

func TestRenderString(t *testing.T) {
	r := NewRenderer()

	tests := []struct {
		name        string
		templateStr string
		data        any
		expected    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "simple template with no data",
			templateStr: "Hello World",
			data:        nil,
			expected:    "Hello World",
		},
		{
			name:        "template with struct data",
			templateStr: "Hello, {{ .Name }}!",
			data:        struct{ Name string }{Name: "Alice"},
			expected:    "Hello, Alice!",
		},
		{
			name:        "template with map data",
			templateStr: "Count: {{ .count }}",
			data:        map[string]any{"count": 42},
			expected:    "Count: 42",
		},
		{
			name:        "template with syntax error",
			templateStr: "{{ .Name }",
			data:        nil,
			wantErr:     true,
			errContains: "failed to parse template",
		},
		{
			name:        "template with execution error",
			templateStr: "{{ .NonExistent }}",
			data:        struct{}{},
			wantErr:     true,
			errContains: "failed to render template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := r.RenderString(tt.name, tt.templateStr, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, string(output))
			}
		})
	}
}

func TestRenderFS(t *testing.T) {
	r := NewRenderer()

	tests := []struct {
		name        string
		path        string
		data        any
		expected    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "simple template",
			path:     "testdata/simple.tmpl",
			data:     struct{ Name string }{Name: "Bob"},
			expected: "Hello, Bob!",
		},
		{
			name:        "non-existent template",
			path:        "testdata/nonexistent.tmpl",
			data:        nil,
			wantErr:     true,
			errContains: "failed to read template from fs",
		},
		{
			name:        "invalid syntax template",
			path:        "testdata/invalid_syntax.tmpl",
			data:        nil,
			wantErr:     true,
			errContains: "failed to parse template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := r.RenderFS(testFS, tt.path, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, string(output))
			}
		})
	}
}

func TestRenderFile(t *testing.T) {
	r := NewRenderer()

	// Create a temporary file for testing
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.tmpl")
	err := os.WriteFile(tempFile, []byte("File: {{ .Name }}"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		path        string
		data        any
		expected    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "existing file",
			path:     tempFile,
			data:     struct{ Name string }{Name: "test.txt"},
			expected: "File: test.txt",
		},
		{
			name:        "non-existent file",
			path:        filepath.Join(tempDir, "nonexistent.tmpl"),
			data:        nil,
			wantErr:     true,
			errContains: "failed to read template file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := r.RenderFile(tt.path, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, string(output))
			}
		})
	}
}

func TestCaching(t *testing.T) {
	r := NewRenderer()

	// First render should cache the template
	output1, err := r.RenderString("cached", "{{ .Value }}", map[string]int{"Value": 1})
	require.NoError(t, err)
	assert.Equal(t, "1", string(output1))

	// Check cache has entry
	assert.Len(t, r.cache, 1)

	// Second render should use cache
	output2, err := r.RenderString("cached", "{{ .Value }}", map[string]int{"Value": 2})
	require.NoError(t, err)
	assert.Equal(t, "2", string(output2))

	// Cache should still have one entry
	assert.Len(t, r.cache, 1)

	// Clear cache
	r.ClearCache()
	assert.Empty(t, r.cache)

	// After clearing, it should parse again
	output3, err := r.RenderString("cached", "{{ .Value }}", map[string]int{"Value": 3})
	require.NoError(t, err)
	assert.Equal(t, "3", string(output3))
	assert.Len(t, r.cache, 1)
}

func TestConcurrency(t *testing.T) {
	r := NewRenderer()

	// Run multiple goroutines rendering templates concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			tmplName := "concurrent"
			tmplStr := "Number: {{ . }}"
			output, err := r.RenderString(tmplName, tmplStr, n)
			assert.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(string(output)), "Number: "+string(rune('0'+n)))
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cache should have one entry (all used the same template)
	assert.Len(t, r.cache, 1)
}

func TestPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic conversions
		{"", ""},
		{"user_name", "UserName"},
		{"blog_post", "BlogPost"},
		{"userName", "UserName"},
		{"UserName", "UserName"},
		{"user", "User"},

		// Acronyms
		{"id", "ID"},
		{"user_id", "UserID"},
		{"api_key", "APIKey"},
		{"http_server", "HTTPServer"},
		{"https_url", "HTTPSURL"},
		{"uuid", "UUID"},
		{"sql_query", "SQLQuery"},
		{"html_template", "HTMLTemplate"},
		{"css_style", "CSSStyle"},
		{"json_data", "JSONData"},
		{"xml_parser", "XMLParser"},
		{"ip_address", "IPAddress"},
		{"tcp_connection", "TCPConnection"},
		{"udp_packet", "UDPPacket"},
		{"tls_config", "TLSConfig"},
		{"ssl_cert", "SSLCert"},
		{"db_connection", "DBConnection"},
		{"ui_component", "UIComponent"},
		{"os_version", "OSVersion"},
		{"url_path", "URLPath"},
		{"uri_scheme", "URIScheme"},

		// Multiple acronyms
		{"api_url", "APIURL"},
		{"http_api", "HTTPAPI"},
		{"db_uuid", "DBUUID"},

		// Mixed regular words and acronyms
		{"user_api_key", "UserAPIKey"},
		{"server_url_path", "ServerURLPath"},
		{"database_id_field", "DatabaseIDField"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, PascalCase(tt.input))
		})
	}
}

func TestCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"user_name", "userName"},
		{"blog_post", "blogPost"},
		{"UserName", "userName"},
		{"userName", "userName"},
		{"api_key", "apiKey"},
		{"HTTP_SERVER", "httpServer"},
		{"id", "id"},
		{"ID", "iD"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, CamelCase(tt.input))
		})
	}
}

func TestSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"UserName", "user_name"},
		{"userName", "user_name"},
		{"user_name", "user_name"},
		{"HTTPServer", "http_server"},
		{"APIKey", "api_key"},
		{"ID", "id"},
		{"BlogPost", "blog_post"},
		{"XMLHttpRequest", "xml_http_request"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, SnakeCase(tt.input))
		})
	}
}

func TestIsPointer(t *testing.T) {
	assert.True(t, IsPointer("*string"))
	assert.True(t, IsPointer("*int"))
	assert.True(t, IsPointer("*time.Time"))
	assert.False(t, IsPointer("string"))
	assert.False(t, IsPointer("int"))
	assert.False(t, IsPointer("[]byte"))
}

func TestStripPointer(t *testing.T) {
	assert.Equal(t, "string", StripPointer("*string"))
	assert.Equal(t, "int", StripPointer("*int"))
	assert.Equal(t, "time.Time", StripPointer("*time.Time"))
	assert.Equal(t, "string", StripPointer("string"))
	assert.Equal(t, "[]byte", StripPointer("[]byte"))
}

func TestIsTime(t *testing.T) {
	assert.True(t, IsTime("time.Time"))
	assert.True(t, IsTime("*time.Time"))
	assert.False(t, IsTime("Time"))
	assert.False(t, IsTime("string"))
	assert.False(t, IsTime("*string"))
}

func TestQuote(t *testing.T) {
	assert.Equal(t, `"test"`, Quote("test"))
	assert.Equal(t, `"hello world"`, Quote("hello world"))
	assert.Equal(t, `""`, Quote(""))
	assert.Equal(t, `"with \"quotes\""`, Quote(`with "quotes"`))
	assert.Equal(t, `"line\nbreak"`, Quote("line\nbreak"))
}

func TestTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "Hello World"},
		{"HELLO WORLD", "Hello World"},
		{"hello", "Hello"},
		{"", ""},
		{"the quick brown fox", "The Quick Brown Fox"},
		{"multiple   spaces", "Multiple Spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, Title(tt.input))
		})
	}
}

func TestSliceString(t *testing.T) {
	tests := []struct {
		name     string
		start    int
		end      int
		input    string
		expected string
	}{
		{"normal slice", 0, 5, "hello world", "hello"},
		{"from middle", 6, 11, "hello world", "world"},
		{"negative start", -5, 5, "hello", "hello"},
		{"start beyond length", 20, 25, "hello", ""},
		{"end zero (to end)", 6, 0, "hello world", "world"},
		{"end beyond length", 0, 100, "hello", "hello"},
		{"start >= end", 5, 3, "hello", ""},
		{"empty string", 0, 5, "", ""},
		{"full string", 0, 0, "test", "test"},
		{"single character", 0, 1, "a", "a"},
		{"unicode string", 0, 5, "hello世界", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SliceString(tt.start, tt.end, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDict(t *testing.T) {
	// Valid dict
	result, err := Dict("key1", "value1", "key2", 42)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"key1": "value1", "key2": 42}, result)

	// Odd number of arguments
	_, err = Dict("key1", "value1", "key2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "even number of arguments")

	// Non-string key
	_, err = Dict(123, "value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "keys must be strings")

	// Empty dict
	result, err = Dict()
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDefault(t *testing.T) {
	// Nil value
	assert.Equal(t, "default", Default("default", nil))

	// Empty string
	assert.Equal(t, "default", Default("default", ""))

	// Non-empty string
	assert.Equal(t, "value", Default("default", "value"))

	// Zero value (0 is not considered empty for numbers)
	assert.Equal(t, 0, Default(42, 0))

	// Empty slice
	assert.Equal(t, "default", Default("default", []any{}))

	// Non-empty slice
	assert.Equal(t, []any{1}, Default("default", []any{1}))

	// Empty map
	assert.Equal(t, "default", Default("default", map[string]any{}))

	// Non-empty map
	assert.Equal(t, map[string]any{"key": "val"}, Default("default", map[string]any{"key": "val"}))
}

func TestHelperFunctionsInTemplate(t *testing.T) {
	r := NewRenderer()

	data := struct {
		Name  string
		Type  string
		Empty string
	}{
		Name: "user_profile",
		Type: "*string",
	}

	output, err := r.RenderFS(testFS, "testdata/with_helpers.tmpl", data)
	require.NoError(t, err)

	outputStr := string(output)
	assert.Contains(t, outputStr, "Original: user_profile")
	assert.Contains(t, outputStr, "PascalCase: UserProfile")
	assert.Contains(t, outputStr, "CamelCase: userProfile")
	assert.Contains(t, outputStr, "SnakeCase: user_profile")
	assert.Contains(t, outputStr, "Plural: user_profiles")
	assert.Contains(t, outputStr, "Upper: USER_PROFILE")
	assert.Contains(t, outputStr, "Lower: user_profile")
	assert.Contains(t, outputStr, "Title: User_profile")
	assert.Contains(t, outputStr, `Quoted: "user_profile"`)
	assert.Contains(t, outputStr, "Type: *string")
	assert.Contains(t, outputStr, "IsPointer: true")
	assert.Contains(t, outputStr, "Stripped: string")
	assert.Contains(t, outputStr, "IsTime: false")
	assert.Contains(t, outputStr, "Default: unknown")
	assert.Contains(t, outputStr, "Dict key1: value1")
	assert.Contains(t, outputStr, "Dict key2: 42")
}

func TestModelExampleTemplate(t *testing.T) {
	r := NewRenderer()

	type FieldInfo struct {
		Name    string
		Type    string
		DBType  string
		Tags    string
		Default string
	}

	type ModelData struct {
		Name   string
		Fields []FieldInfo
	}

	data := ModelData{
		Name: "User",
		Fields: []FieldInfo{
			{Name: "id", Type: "string", DBType: "UUID", Tags: "`json:\"id\" db:\"id\"`"},
			{Name: "email", Type: "string", DBType: "VARCHAR(255)", Tags: "`json:\"email\" db:\"email\"`"},
			{Name: "name", Type: "*string", DBType: "VARCHAR(100)", Tags: "`json:\"name,omitempty\" db:\"name\"`"},
			{Name: "active", Type: "bool", DBType: "BOOLEAN", Tags: "`json:\"active\" db:\"active\"`", Default: "true"},
			{Name: "created_at", Type: "time.Time", DBType: "TIMESTAMP", Tags: "`json:\"created_at\" db:\"created_at\"`"},
		},
	}

	output, err := r.RenderFS(testFS, "testdata/model_example.go.tmpl", data)
	require.NoError(t, err)

	outputStr := string(output)

	// Check struct definition
	assert.Contains(t, outputStr, "type User struct")
	assert.Contains(t, outputStr, "ID string")
	assert.Contains(t, outputStr, "Email string")
	assert.Contains(t, outputStr, "Name *string")
	assert.Contains(t, outputStr, "Active bool")
	assert.Contains(t, outputStr, "CreatedAt time.Time")

	// Check TableName method
	assert.Contains(t, outputStr, "func (u *User) TableName() string")
	assert.Contains(t, outputStr, `return "users"`)

	// Check constructor
	assert.Contains(t, outputStr, "func NewUser() *User")
	assert.Contains(t, outputStr, "Active: true,")

	// Check getter/setter for pointer field
	assert.Contains(t, outputStr, "func (u *User) GetName() string")
	assert.Contains(t, outputStr, "func (u *User) SetName(val string)")
}

func TestCacheKeyGeneration(t *testing.T) {
	r := NewRenderer()

	// Test different cache key types
	assert.Equal(t, "string:test", r.getCacheKey("string", "test"))
	assert.Equal(t, "fs:path/to/template", r.getCacheKey("fs", "path/to/template"))
	assert.Equal(t, "file:/tmp/test.tmpl", r.getCacheKey("file", "/tmp/test.tmpl"))
}

func TestTemplateCachePersistence(t *testing.T) {
	r := NewRenderer()

	// Render multiple templates
	_, err := r.RenderString("tmpl1", "{{ .Value }}", map[string]int{"Value": 1})
	require.NoError(t, err)

	_, err = r.RenderString("tmpl2", "{{ .Value }}", map[string]int{"Value": 2})
	require.NoError(t, err)

	// Both should be cached
	assert.Len(t, r.cache, 2)

	// Render again with same names
	_, err = r.RenderString("tmpl1", "different template", map[string]int{"Value": 3})
	require.NoError(t, err)

	// Should still use cached version (not the "different template" string)
	output, err := r.RenderString("tmpl1", "ignored", map[string]int{"Value": 4})
	require.NoError(t, err)
	assert.Equal(t, "4", string(output)) // Uses original cached template

	// Clear and verify
	r.ClearCache()
	assert.Empty(t, r.cache)

	// Now it should use the new template
	output, err = r.RenderString("tmpl1", "new: {{ .Value }}", map[string]int{"Value": 5})
	require.NoError(t, err)
	assert.Equal(t, "new: 5", string(output))
}

func BenchmarkRenderWithCache(b *testing.B) {
	r := NewRenderer()
	data := struct{ Name string }{Name: "Test"}

	// First render to populate cache
	_, _ = r.RenderString("bench", "Hello, {{ .Name }}!", data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.RenderString("bench", "Hello, {{ .Name }}!", data)
	}
}

func BenchmarkRenderWithoutCache(b *testing.B) {
	r := NewRenderer()
	data := struct{ Name string }{Name: "Test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ClearCache()
		_, _ = r.RenderString("bench", "Hello, {{ .Name }}!", data)
	}
}