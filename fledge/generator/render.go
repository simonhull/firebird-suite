package generator

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"strings"
	"sync"
	"text/template"
	"unicode"

	"github.com/simonhull/firebird-suite/fledge/schema"
)

// Renderer handles template parsing and rendering with caching
type Renderer struct {
	funcMap template.FuncMap
	cache   map[string]*template.Template
	mu      sync.RWMutex // Protect cache for concurrent access
}

// NewRenderer creates a renderer with built-in helper functions
func NewRenderer() *Renderer {
	return &Renderer{
		funcMap: defaultFuncMap(),
		cache:   make(map[string]*template.Template),
	}
}

// RenderString renders a template from a string
// The name is used for caching and error messages
func (r *Renderer) RenderString(name, templateStr string, data any) ([]byte, error) {
	cacheKey := r.getCacheKey("string", name)

	// Check cache with read lock
	r.mu.RLock()
	if tmpl, ok := r.cache[cacheKey]; ok {
		r.mu.RUnlock()
		return r.executeTemplate(tmpl, data)
	}
	r.mu.RUnlock()

	// Parse template
	tmpl, err := template.New(name).Funcs(r.funcMap).Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template '%s': %w", name, err)
	}

	// Cache with write lock
	r.mu.Lock()
	r.cache[cacheKey] = tmpl
	r.mu.Unlock()

	return r.executeTemplate(tmpl, data)
}

// RenderFS renders a template from an embedded filesystem
func (r *Renderer) RenderFS(fs embed.FS, path string, data any) ([]byte, error) {
	cacheKey := r.getCacheKey("fs", path)

	// Check cache with read lock
	r.mu.RLock()
	if tmpl, ok := r.cache[cacheKey]; ok {
		r.mu.RUnlock()
		return r.executeTemplate(tmpl, data)
	}
	r.mu.RUnlock()

	// Read template from embedded filesystem
	templateBytes, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template from fs '%s': %w", path, err)
	}

	// Parse template
	tmpl, err := template.New(path).Funcs(r.funcMap).Parse(string(templateBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template '%s': %w", path, err)
	}

	// Cache with write lock
	r.mu.Lock()
	r.cache[cacheKey] = tmpl
	r.mu.Unlock()

	return r.executeTemplate(tmpl, data)
}

// RenderFile renders a template from a file path (for --template overrides)
func (r *Renderer) RenderFile(path string, data any) ([]byte, error) {
	cacheKey := r.getCacheKey("file", path)

	// Check cache with read lock
	r.mu.RLock()
	if tmpl, ok := r.cache[cacheKey]; ok {
		r.mu.RUnlock()
		return r.executeTemplate(tmpl, data)
	}
	r.mu.RUnlock()

	// Read template from file
	templateBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file '%s': %w", path, err)
	}

	// Parse template
	tmpl, err := template.New(path).Funcs(r.funcMap).Parse(string(templateBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template '%s': %w", path, err)
	}

	// Cache with write lock
	r.mu.Lock()
	r.cache[cacheKey] = tmpl
	r.mu.Unlock()

	return r.executeTemplate(tmpl, data)
}

// ClearCache clears the template cache (useful for testing)
func (r *Renderer) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = make(map[string]*template.Template)
}

// executeTemplate executes a parsed template with the given data
func (r *Renderer) executeTemplate(tmpl *template.Template, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render template '%s': %w", tmpl.Name(), err)
	}
	return buf.Bytes(), nil
}

// getCacheKey generates a cache key for a template
func (r *Renderer) getCacheKey(typ, identifier string) string {
	return fmt.Sprintf("%s:%s", typ, identifier)
}

// defaultFuncMap returns the default template function map
func defaultFuncMap() template.FuncMap {
	return template.FuncMap{
		// Case conversion
		"pascalCase": PascalCase, // user_name → UserName
		"camelCase":  CamelCase,  // user_name → userName
		"snakeCase":  SnakeCase,  // UserName → user_name

		// String manipulation
		"plural":     schema.Pluralize, // user → users
		"quote":      Quote,             // test → "test"
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"title":      Title, // Custom title case function
		"trim":       strings.TrimSpace,
		"join":       strings.Join,
		"split":      strings.Split,
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"replace":    strings.ReplaceAll,
		"slice":      SliceString, // Get substring

		// Type helpers
		"isPointer":    IsPointer,    // *string → true
		"stripPointer": StripPointer, // *string → string
		"isTime":       IsTime,        // time.Time → true

		// Utilities
		"dict":    Dict,    // Create map for passing multiple values
		"default": Default, // Provide default value if nil/empty
	}
}

// PascalCase converts snake_case or camelCase to PascalCase
// Examples: user_name → UserName, userName → UserName, user_id → UserID
func PascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Handle snake_case
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		for i, part := range parts {
			if part != "" {
				parts[i] = capitalizeWord(part)
			}
		}
		return strings.Join(parts, "")
	}

	// Handle camelCase or already PascalCase
	if len(s) > 0 {
		// If first letter is lowercase, capitalize it
		if unicode.IsLower(rune(s[0])) {
			return capitalizeWord(s)
		}
	}

	return s
}

// capitalizeWord capitalizes a word with special handling for acronyms
func capitalizeWord(s string) string {
	if s == "" {
		return ""
	}

	// Common acronyms that should be all-caps
	acronyms := map[string]string{
		"id":    "ID",
		"url":   "URL",
		"uri":   "URI",
		"http":  "HTTP",
		"https": "HTTPS",
		"api":   "API",
		"uuid":  "UUID",
		"sql":   "SQL",
		"html":  "HTML",
		"css":   "CSS",
		"json":  "JSON",
		"xml":   "XML",
		"ip":    "IP",
		"tcp":   "TCP",
		"udp":   "UDP",
		"tls":   "TLS",
		"ssl":   "SSL",
		"db":    "DB",
		"ui":    "UI",
		"os":    "OS",
	}

	lower := strings.ToLower(s)
	if acronym, ok := acronyms[lower]; ok {
		return acronym
	}

	// Regular capitalization - capitalize first letter, keep rest as-is
	if len(s) > 0 {
		return strings.ToUpper(string(s[0])) + s[1:]
	}

	return s
}

// CamelCase converts snake_case or PascalCase to camelCase
// Examples: user_name → userName, UserName → userName
func CamelCase(s string) string {
	if s == "" {
		return ""
	}

	// Handle snake_case
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		for i, part := range parts {
			if part != "" {
				if i == 0 {
					parts[i] = strings.ToLower(part)
				} else {
					parts[i] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
				}
			}
		}
		return strings.Join(parts, "")
	}

	// Handle PascalCase or already camelCase
	if len(s) > 0 {
		// If first letter is uppercase, make it lowercase
		if unicode.IsUpper(rune(s[0])) {
			return strings.ToLower(string(s[0])) + s[1:]
		}
	}

	return s
}

// SnakeCase converts PascalCase or camelCase to snake_case
// Examples: UserName → user_name, userName → user_name, HTTPServer → http_server
func SnakeCase(s string) string {
	if s == "" {
		return ""
	}

	// Handle already snake_case
	if strings.Contains(s, "_") {
		return strings.ToLower(s)
	}

	var result strings.Builder
	for i, r := range s {
		// If current rune is uppercase
		if unicode.IsUpper(r) {
			// Add underscore before uppercase letter if:
			// - Not the first character
			// - Previous character is lowercase OR
			// - Previous character is uppercase but next is lowercase (handling acronyms)
			if i > 0 {
				prev := rune(s[i-1])
				if unicode.IsLower(prev) {
					result.WriteRune('_')
				} else if i+1 < len(s) && unicode.IsLower(rune(s[i+1])) {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// IsPointer checks if a type string is a pointer (starts with *)
func IsPointer(typeStr string) bool {
	return strings.HasPrefix(typeStr, "*")
}

// StripPointer removes the * prefix from a pointer type
// Example: *string → string
func StripPointer(typeStr string) string {
	return strings.TrimPrefix(typeStr, "*")
}

// IsTime checks if a type is time.Time or *time.Time
func IsTime(typeStr string) bool {
	return typeStr == "time.Time" || typeStr == "*time.Time"
}

// Quote wraps a string in double quotes
func Quote(s string) string {
	return fmt.Sprintf("%q", s)
}

// Title converts a string to title case (first letter of each word capitalized)
// This replaces the deprecated strings.Title
func Title(s string) string {
	if s == "" {
		return ""
	}

	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// SliceString returns a substring from start to end indices
// If end is 0 or greater than string length, it goes to the end of the string
func SliceString(start, end int, s string) string {
	if start < 0 {
		start = 0
	}
	if start >= len(s) {
		return ""
	}
	if end <= 0 || end > len(s) {
		end = len(s)
	}
	if start >= end {
		return ""
	}
	return s[start:end]
}

// Dict creates a map from alternating key-value pairs
// Usage in template: {{ template "partial" (dict "key1" val1 "key2" val2) }}
func Dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict requires an even number of arguments")
	}

	result := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings, got %T at position %d", values[i], i)
		}
		result[key] = values[i+1]
	}
	return result, nil
}

// Default returns the default value if the given value is nil or empty
func Default(defaultVal, val any) any {
	if val == nil {
		return defaultVal
	}

	// Check for empty string
	if s, ok := val.(string); ok && s == "" {
		return defaultVal
	}

	// Check for zero-length slices/maps
	switch v := val.(type) {
	case []any:
		if len(v) == 0 {
			return defaultVal
		}
	case map[string]any:
		if len(v) == 0 {
			return defaultVal
		}
	}

	// Note: We don't check for numeric zero values because 0 might be a valid value
	// Only nil, empty string, and empty collections are considered "empty"

	return val
}

// Pluralize converts a singular noun to plural form
// This is exported so it can be used by generators outside of templates
func Pluralize(s string) string {
	return schema.Pluralize(s)
}