package generator

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// DiffOptions configures how diffs are generated and displayed.
// All fields are optional with sensible defaults.
type DiffOptions struct {
	// ContextLines is the number of unchanged lines to show around changes.
	// Default: 3
	ContextLines int

	// TabWidth is the number of spaces each tab character expands to.
	// Default: 4
	TabWidth int

	// ShowLineNums displays line numbers in the left margin.
	// Default: false
	ShowLineNums bool
}

// DiffGenerator provides efficient diff generation with reusable allocations.
// Create once and reuse for multiple diff operations to reduce GC pressure.
//
// Example:
//
//	gen := NewDiffGenerator()
//	diff1 := gen.GenerateDiff("a.go", "a.go", old1, new1, nil)
//	diff2 := gen.GenerateDiff("b.go", "b.go", old2, new2, nil)
type DiffGenerator struct {
	v     map[int]int
	trace []map[int]int
}

// NewDiffGenerator creates a diff generator optimized for repeated use.
func NewDiffGenerator() *DiffGenerator {
	return &DiffGenerator{
		v:     make(map[int]int, 100),
		trace: make([]map[int]int, 0, 100),
	}
}

// GenerateDiffDefault is a convenience wrapper using default options.
func (dg *DiffGenerator) GenerateDiffDefault(oldPath, newPath string, old, newer []byte) string {
	return dg.GenerateDiff(oldPath, newPath, old, newer, nil)
}

// GenerateDiff generates a diff, reusing internal allocations for efficiency.
func (dg *DiffGenerator) GenerateDiff(oldPath, newPath string, old, newer []byte, opts *DiffOptions) string {
	// Apply defaults
	if opts == nil {
		opts = &DiffOptions{ContextLines: 3, TabWidth: 4}
	}
	if opts.ContextLines == 0 {
		opts.ContextLines = 3
	}
	if opts.TabWidth == 0 {
		opts.TabWidth = 4
	}

	// Check for binary files
	if isBinary(old) || isBinary(newer) {
		return "Binary files differ\n"
	}

	// Split into lines
	oldLines := splitLines(string(old))
	newLines := splitLines(string(newer))

	// Check if identical
	if len(oldLines) == len(newLines) {
		identical := true
		for i := range oldLines {
			if oldLines[i] != newLines[i] {
				identical = false
				break
			}
		}
		if identical {
			return ""
		}
	}

	// Handle very large files (>10k lines)
	if len(oldLines) > 10000 || len(newLines) > 10000 {
		return fmt.Sprintf("Files too large for diff (%d and %d lines)\n", len(oldLines), len(newLines))
	}

	// Compute diff using Myers algorithm
	diffLines := dg.computeEditScript(oldLines, newLines)

	// Build hunks with context
	hunks := buildHunks(diffLines, opts.ContextLines)

	// Format output
	if len(hunks) == 0 {
		return ""
	}

	var buf strings.Builder

	// Write header
	buf.WriteString(headerStyle.Render("--- "+oldPath) + "\n")
	buf.WriteString(headerStyle.Render("+++ "+newPath) + "\n")

	// Get terminal width for line wrapping
	termWidth := getTerminalWidth()

	// Write hunks
	for _, h := range hunks {
		buf.WriteString(formatHunk(h, opts, termWidth))
	}

	return buf.String()
}

// operation represents the type of diff operation
type operation int

const (
	opUnchanged operation = iota
	opAdded
	opRemoved
)

// diffLine represents a single line in the diff with its operation
type diffLine struct {
	oldLineNum int       // Line number in old file (0 if added)
	newLineNum int       // Line number in new file (0 if removed)
	content    string    // Line content
	op         operation // Operation type
}

// hunk represents a contiguous block of changes with surrounding context
type hunk struct {
	oldStart int        // Starting line in old file
	oldCount int        // Number of lines in old file
	newStart int        // Starting line in new file
	newCount int        // Number of lines in new file
	lines    []diffLine // Lines in this hunk
}

// Lipgloss styles for terminal output
var (
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hunkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("cyan")).Bold(true)
	addedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("22"))
	removedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("52"))
	lineNumStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Faint(true)
)

// GenerateDiffDefault uses default options (3 context lines).
// For repeated diff operations, use DiffGenerator for better performance.
func GenerateDiffDefault(oldPath, newPath string, old, newer []byte) string {
	return GenerateDiff(oldPath, newPath, old, newer, nil)
}

// GenerateDiff creates a unified diff (standalone convenience function).
// For repeated diff operations, use DiffGenerator for better performance.
func GenerateDiff(oldPath, newPath string, old, newer []byte, opts *DiffOptions) string {
	gen := NewDiffGenerator()
	return gen.GenerateDiff(oldPath, newPath, old, newer, opts)
}

// computeEditScript implements Myers diff algorithm (wrapper for standalone use).
func computeEditScript(old, newer []string) []diffLine {
	gen := NewDiffGenerator()
	return gen.computeEditScript(old, newer)
}

// computeEditScript implements Myers diff algorithm to compute the shortest edit script.
// Based on "An O(ND) Difference Algorithm and Its Variations" by Eugene W. Myers (1986).
func (dg *DiffGenerator) computeEditScript(old, newer []string) []diffLine {
	n := len(old)
	m := len(newer)
	maxD := n + m

	// Clear and reuse v map
	for k := range dg.v {
		delete(dg.v, k)
	}
	dg.v[1] = 0

	// Clear and reuse trace
	dg.trace = dg.trace[:0]

	// Forward pass: find shortest edit script
	var d int
	for d = 0; d <= maxD; d++ {
		// Save current V for backtracking
		vcopy := make(map[int]int)
		for k, val := range dg.v {
			vcopy[k] = val
		}
		dg.trace = append(dg.trace, vcopy)

		for k := -d; k <= d; k += 2 {
			var x int

			// Determine if we should move down or right
			// Move down if k == -d (bottom edge) or if the path from k-1 is better
			if k == -d || (k != d && dg.v[k-1] < dg.v[k+1]) {
				x = dg.v[k+1] // Move down (deletion in old)
			} else {
				x = dg.v[k-1] + 1 // Move right (insertion in new)
			}

			y := x - k

			// Follow diagonal as far as possible (matching lines)
			for x < n && y < m && old[x] == newer[y] {
				x++
				y++
			}

			dg.v[k] = x

			// Check if we've reached the end
			if x >= n && y >= m {
				goto backtrack
			}
		}
	}

backtrack:
	// Backtrack to build the edit script
	var result []diffLine
	x, y := n, m

	for d := len(dg.trace) - 1; d >= 0; d-- {
		v := dg.trace[d]
		k := x - y

		var prevK int
		if k == -d || (k != d && v[k-1] < v[k+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := v[prevK]
		prevY := prevX - prevK

		// Follow diagonal backwards
		for x > prevX && y > prevY {
			x--
			y--
			result = append([]diffLine{{
				oldLineNum: x + 1,
				newLineNum: y + 1,
				content:    old[x],
				op:         opUnchanged,
			}}, result...)
		}

		if d > 0 {
			if x == prevX {
				// Insertion
				y--
				result = append([]diffLine{{
					oldLineNum: 0,
					newLineNum: y + 1,
					content:    newer[y],
					op:         opAdded,
				}}, result...)
			} else {
				// Deletion
				x--
				result = append([]diffLine{{
					oldLineNum: x + 1,
					newLineNum: 0,
					content:    old[x],
					op:         opRemoved,
				}}, result...)
			}
		}
	}

	return result
}

// buildHunks groups diff lines into hunks with surrounding context
func buildHunks(lines []diffLine, contextLines int) []hunk {
	if len(lines) == 0 {
		return nil
	}

	var hunks []hunk
	var currentHunk *hunk

	for i, line := range lines {
		if line.op != opUnchanged {
			// Start a new hunk if needed
			if currentHunk == nil {
				// Find context start
				contextStart := i - contextLines
				if contextStart < 0 {
					contextStart = 0
				}

				currentHunk = &hunk{lines: []diffLine{}}

				// Add leading context
				for j := contextStart; j < i; j++ {
					currentHunk.lines = append(currentHunk.lines, lines[j])
				}
			}

			currentHunk.lines = append(currentHunk.lines, line)
		} else {
			// Context line
			if currentHunk != nil {
				currentHunk.lines = append(currentHunk.lines, line)

				// Check if we should close this hunk
				// Count consecutive context lines after the last change
				contextAfter := 1
				for j := i + 1; j < len(lines) && lines[j].op == opUnchanged; j++ {
					contextAfter++
				}

				// If we have enough context and more changes are coming, close hunk
				if contextAfter > contextLines*2 && i < len(lines)-1 {
					// Trim to only needed context
					trimCount := contextAfter - contextLines
					if trimCount > 0 && trimCount <= len(currentHunk.lines) {
						currentHunk.lines = currentHunk.lines[:len(currentHunk.lines)-trimCount]
					}

					finalizeHunk(currentHunk)
					hunks = append(hunks, *currentHunk)
					currentHunk = nil
				}
			}
		}
	}

	// Finalize last hunk
	if currentHunk != nil {
		finalizeHunk(currentHunk)
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

// finalizeHunk calculates the start and count values for a hunk
func finalizeHunk(h *hunk) {
	if len(h.lines) == 0 {
		return
	}

	// Find first and last line numbers
	for _, line := range h.lines {
		if line.oldLineNum > 0 && (h.oldStart == 0 || line.oldLineNum < h.oldStart) {
			h.oldStart = line.oldLineNum
		}
		if line.newLineNum > 0 && (h.newStart == 0 || line.newLineNum < h.newStart) {
			h.newStart = line.newLineNum
		}
	}

	// Count lines
	for _, line := range h.lines {
		if line.op == opRemoved || line.op == opUnchanged {
			h.oldCount++
		}
		if line.op == opAdded || line.op == opUnchanged {
			h.newCount++
		}
	}
}

// formatHunk formats a hunk as a unified diff string with styling
func formatHunk(h hunk, opts *DiffOptions, termWidth int) string {
	var buf strings.Builder

	// Hunk header
	header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.oldStart, h.oldCount, h.newStart, h.newCount)
	buf.WriteString(hunkStyle.Render(header) + "\n")

	// Format each line
	for _, line := range h.lines {
		content := expandTabs(line.content, opts.TabWidth)
		content = truncateLine(content, termWidth-10) // Leave room for prefix and line numbers

		var prefix string
		var style lipgloss.Style

		switch line.op {
		case opAdded:
			prefix = "+"
			style = addedStyle
		case opRemoved:
			prefix = "-"
			style = removedStyle
		case opUnchanged:
			prefix = " "
			style = lipgloss.NewStyle()
		}

		formatted := prefix + content
		if line.op == opAdded || line.op == opRemoved {
			formatted = style.Render(formatted)
		}

		if opts.ShowLineNums {
			lineNum := ""
			if line.oldLineNum > 0 {
				lineNum = fmt.Sprintf("%4d", line.oldLineNum)
			} else {
				lineNum = "    "
			}
			formatted = lineNumStyle.Render(lineNum) + " " + formatted
		}

		buf.WriteString(formatted + "\n")
	}

	return buf.String()
}

// isBinary checks if content appears to be binary (contains null bytes)
func isBinary(data []byte) bool {
	// Check first 8192 bytes for null bytes
	checkLen := len(data)
	if checkLen > 8192 {
		checkLen = 8192
	}
	return bytes.IndexByte(data[:checkLen], 0) != -1
}

// splitLines splits content into lines, preserving empty lines
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}

	lines := strings.Split(s, "\n")

	// Remove trailing empty line if present (from final newline)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// expandTabs replaces tabs with spaces
func expandTabs(s string, tabWidth int) string {
	var buf strings.Builder
	col := 0

	for _, r := range s {
		if r == '\t' {
			// Add spaces to next tab stop
			spaces := tabWidth - (col % tabWidth)
			buf.WriteString(strings.Repeat(" ", spaces))
			col += spaces
		} else {
			buf.WriteRune(r)
			col++
		}
	}

	return buf.String()
}

// truncateLine truncates a line if it's too long, adding "..." indicator
func truncateLine(s string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	if utf8.RuneCountInString(s) <= maxWidth {
		return s
	}

	// Truncate and add indicator
	runes := []rune(s)
	if maxWidth < 3 {
		return "..."[:maxWidth]
	}

	return string(runes[:maxWidth-3]) + "..."
}

// getTerminalWidth returns the terminal width, defaulting to 80 if unable to detect
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80
	}
	return width
}
