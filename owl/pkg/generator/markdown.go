package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var md goldmark.Markdown

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,          // GitHub Flavored Markdown
			extension.Table,        // Tables
			extension.Strikethrough,
			extension.TaskList,     // - [ ] task lists
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Auto-generate heading IDs
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Allow raw HTML in markdown
		),
	)
}

// ParseMarkdown converts markdown bytes to HTML
func ParseMarkdown(source []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		return nil, fmt.Errorf("parsing markdown: %w", err)
	}
	return buf.Bytes(), nil
}

// ReadREADME reads and parses README.md from a directory
// Returns HTML content and true if found, empty string and false if not found
func ReadREADME(dir string) (string, bool) {
	// Try README.md first, then readme.md
	paths := []string{
		filepath.Join(dir, "README.md"),
		filepath.Join(dir, "readme.md"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		html, err := ParseMarkdown(data)
		if err != nil {
			// If parsing fails, return empty but log the error
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse %s: %v\n", path, err)
			return "", false
		}

		return string(html), true
	}

	return "", false
}

// ReadPackageDoc reads package documentation from doc.go or README.md
// Returns HTML content and true if found
func ReadPackageDoc(pkgPath string) (string, bool) {
	// Priority 1: README.md in package directory
	if content, found := ReadREADME(pkgPath); found {
		return content, true
	}

	// Priority 2: doc.go file (extract package comment)
	docPath := filepath.Join(pkgPath, "doc.go")
	data, err := os.ReadFile(docPath)
	if err == nil {
		// Extract package comment (everything before "package" keyword)
		content := extractPackageComment(string(data))
		if content != "" {
			html, err := ParseMarkdown([]byte(content))
			if err == nil {
				return string(html), true
			}
		}
	}

	return "", false
}

// extractPackageComment extracts the package-level comment from doc.go
func extractPackageComment(content string) string {
	lines := []string{}
	inComment := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		// Start of block comment
		if strings.HasPrefix(trimmed, "/*") {
			inComment = true
			// Remove /* and add content
			text := strings.TrimPrefix(trimmed, "/*")
			text = strings.TrimSpace(text)
			if text != "" && text != "*/" {
				lines = append(lines, text)
			}
			continue
		}

		// End of block comment
		if inComment && strings.Contains(trimmed, "*/") {
			// Remove */ and add content before it
			text := strings.TrimSuffix(trimmed, "*/")
			text = strings.TrimSpace(text)
			if text != "" && text != "/*" {
				lines = append(lines, text)
			}
			break
		}

		// Inside block comment
		if inComment {
			// Remove leading * from comment lines
			text := trimmed
			if strings.HasPrefix(text, "*") {
				text = strings.TrimPrefix(text, "*")
				text = strings.TrimSpace(text)
			}
			lines = append(lines, text)
			continue
		}

		// Line comment
		if strings.HasPrefix(trimmed, "//") {
			text := strings.TrimPrefix(trimmed, "//")
			text = strings.TrimSpace(text)
			lines = append(lines, text)
			continue
		}

		// Stop at package keyword
		if strings.HasPrefix(trimmed, "package ") {
			break
		}
	}

	return strings.Join(lines, "\n")
}
