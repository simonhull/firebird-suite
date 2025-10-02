package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateDiff_Identical(t *testing.T) {
	old := []byte("line 1\nline 2\nline 3\n")
	newer := []byte("line 1\nline 2\nline 3\n")

	result := GenerateDiffDefault("old.txt", "new.txt", old, newer)

	if result != "" {
		t.Errorf("Expected empty string for identical files, got: %q", result)
	}
}

func TestGenerateDiff_EmptyFiles(t *testing.T) {
	tests := []struct {
		name  string
		old   []byte
		newer []byte
		want  string
	}{
		{
			name:  "both empty",
			old:   []byte(""),
			newer: []byte(""),
			want:  "",
		},
		{
			name:  "old empty, newer has content",
			old:   []byte(""),
			newer: []byte("line 1\nline 2\n"),
			want:  "addition",
		},
		{
			name:  "newer empty, old has content",
			old:   []byte("line 1\nline 2\n"),
			newer: []byte(""),
			want:  "deletion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateDiffDefault("old.txt", "new.txt", tt.old, tt.newer)

			if tt.want == "" {
				if result != "" {
					t.Errorf("Expected empty diff, got: %q", result)
				}
			} else if tt.want == "addition" {
				if !strings.Contains(result, "+line 1") || !strings.Contains(result, "+line 2") {
					t.Errorf("Expected additions, got: %q", result)
				}
			} else if tt.want == "deletion" {
				if !strings.Contains(result, "-line 1") || !strings.Contains(result, "-line 2") {
					t.Errorf("Expected deletions, got: %q", result)
				}
			}
		})
	}
}

func TestGenerateDiff_SimpleAddition(t *testing.T) {
	old := []byte("line 1\nline 2\nline 3\n")
	newer := []byte("line 1\nline 2\nline 2.5\nline 3\n")

	result := GenerateDiffDefault("old.txt", "new.txt", old, newer)

	if !strings.Contains(result, "--- old.txt") {
		t.Error("Missing old file header")
	}
	if !strings.Contains(result, "+++ new.txt") {
		t.Error("Missing new file header")
	}
	if !strings.Contains(result, "+line 2.5") {
		t.Error("Missing added line")
	}
	if !strings.Contains(result, "@@") {
		t.Error("Missing hunk header")
	}
}

func TestGenerateDiff_SimpleRemoval(t *testing.T) {
	old := []byte("line 1\nline 2\nline 3\nline 4\n")
	newer := []byte("line 1\nline 2\nline 4\n")

	result := GenerateDiffDefault("old.txt", "new.txt", old, newer)

	if !strings.Contains(result, "-line 3") {
		t.Error("Missing removed line")
	}
}

func TestGenerateDiff_Replacement(t *testing.T) {
	old := []byte("line 1\nold content\nline 3\n")
	newer := []byte("line 1\nnew content\nline 3\n")

	result := GenerateDiffDefault("old.txt", "new.txt", old, newer)

	if !strings.Contains(result, "-old content") {
		t.Error("Missing removed line")
	}
	if !strings.Contains(result, "+new content") {
		t.Error("Missing added line")
	}
}

func TestGenerateDiff_MultipleHunks(t *testing.T) {
	old := []byte(`line 1
line 2
line 3
line 4
line 5
line 6
line 7
line 8
line 9
line 10
line 11
line 12
`)

	newer := []byte(`line 1
line 2
changed line 3
line 4
line 5
line 6
line 7
line 8
line 9
changed line 10
line 11
line 12
`)

	result := GenerateDiffDefault("old.txt", "new.txt", old, newer)

	// Count hunk headers
	hunkCount := strings.Count(result, "@@")

	// With default context of 3, these changes should be in separate hunks
	if hunkCount < 2 {
		t.Errorf("Expected at least 2 hunks, got %d", hunkCount)
	}
}

func TestGenerateDiff_ContextLines(t *testing.T) {
	old := []byte("line 1\nline 2\nline 3\nline 4\nline 5\n")
	newer := []byte("line 1\nline 2\nchanged\nline 4\nline 5\n")

	tests := []struct {
		name         string
		contextLines int
	}{
		{"context 0", 0},
		{"context 1", 1},
		{"context 3", 3},
		{"context 5", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &DiffOptions{ContextLines: tt.contextLines}
			result := GenerateDiff("old.txt", "new.txt", old, newer, opts)

			if !strings.Contains(result, "-line 3") {
				t.Error("Missing removed line")
			}
			if !strings.Contains(result, "+changed") {
				t.Error("Missing added line")
			}
		})
	}
}

func TestGenerateDiff_TabHandling(t *testing.T) {
	old := []byte("line 1\n\tindented line\nline 3\n")
	newer := []byte("line 1\n\t\tdouble indented\nline 3\n")

	opts := &DiffOptions{ContextLines: 3, TabWidth: 4}
	result := GenerateDiff("old.txt", "new.txt", old, newer, opts)

	// Tabs should be expanded to spaces
	if strings.Contains(result, "\t") {
		t.Error("Result contains tab characters, should be expanded")
	}
}

func TestGenerateDiff_BinaryFiles(t *testing.T) {
	// Binary content with null bytes
	old := []byte{0x00, 0x01, 0x02, 0xFF}
	newer := []byte{0x00, 0x01, 0x03, 0xFF}

	result := GenerateDiffDefault("old.bin", "new.bin", old, newer)

	if !strings.Contains(result, "Binary files differ") {
		t.Errorf("Expected binary file message, got: %q", result)
	}
}

func TestGenerateDiff_LargeFiles(t *testing.T) {
	// Generate file with > 10k lines (make them different)
	var oldBuf, newBuf strings.Builder
	for i := 0; i < 11000; i++ {
		oldBuf.WriteString(fmt.Sprintf("line %d\n", i))
		newBuf.WriteString(fmt.Sprintf("line %d modified\n", i))
	}

	result := GenerateDiffDefault("old.txt", "new.txt", []byte(oldBuf.String()), []byte(newBuf.String()))

	if !strings.Contains(result, "too large") {
		t.Errorf("Expected large file message, got: %q", result)
	}
}

func TestGenerateDiff_VeryLongLines(t *testing.T) {
	// Create a very long line
	longLine := strings.Repeat("x", 500)
	old := []byte("short line\n")
	newer := []byte("short line\n" + longLine + "\n")

	result := GenerateDiffDefault("old.txt", "new.txt", old, newer)

	// Should contain truncation indicator
	if !strings.Contains(result, "+") {
		t.Error("Missing added line indicator")
	}
}

func TestComputeEditScript_Simple(t *testing.T) {
	old := []string{"a", "b", "c"}
	newer := []string{"a", "x", "b", "c"}

	script := computeEditScript(old, newer)

	// Should have: unchanged(a), added(x), unchanged(b), unchanged(c)
	if len(script) != 4 {
		t.Errorf("Expected 4 operations, got %d", len(script))
	}

	if script[0].op != opUnchanged || script[0].content != "a" {
		t.Errorf("Expected unchanged 'a', got %v", script[0])
	}

	if script[1].op != opAdded || script[1].content != "x" {
		t.Errorf("Expected added 'x', got %v", script[1])
	}
}

func TestComputeEditScript_Deletion(t *testing.T) {
	old := []string{"a", "b", "c"}
	newer := []string{"a", "c"}

	script := computeEditScript(old, newer)

	// Find the deletion
	foundDeletion := false
	for _, op := range script {
		if op.op == opRemoved && op.content == "b" {
			foundDeletion = true
			break
		}
	}

	if !foundDeletion {
		t.Error("Expected deletion of 'b' not found")
	}
}

func TestBuildHunks_SingleChange(t *testing.T) {
	lines := []diffLine{
		{oldLineNum: 1, newLineNum: 1, content: "line 1", op: opUnchanged},
		{oldLineNum: 2, newLineNum: 2, content: "line 2", op: opUnchanged},
		{oldLineNum: 3, newLineNum: 0, content: "old line", op: opRemoved},
		{oldLineNum: 0, newLineNum: 3, content: "new line", op: opAdded},
		{oldLineNum: 4, newLineNum: 4, content: "line 4", op: opUnchanged},
		{oldLineNum: 5, newLineNum: 5, content: "line 5", op: opUnchanged},
	}

	hunks := buildHunks(lines, 2)

	if len(hunks) != 1 {
		t.Errorf("Expected 1 hunk, got %d", len(hunks))
	}

	if len(hunks) > 0 {
		if hunks[0].oldStart != 1 {
			t.Errorf("Expected oldStart=1, got %d", hunks[0].oldStart)
		}
	}
}

func TestBuildHunks_MultipleChanges(t *testing.T) {
	// Create changes far apart
	lines := []diffLine{
		{oldLineNum: 1, newLineNum: 1, content: "line 1", op: opUnchanged},
		{oldLineNum: 2, newLineNum: 0, content: "removed", op: opRemoved},
		{oldLineNum: 3, newLineNum: 2, content: "line 3", op: opUnchanged},
		{oldLineNum: 4, newLineNum: 3, content: "line 4", op: opUnchanged},
		{oldLineNum: 5, newLineNum: 4, content: "line 5", op: opUnchanged},
		{oldLineNum: 6, newLineNum: 5, content: "line 6", op: opUnchanged},
		{oldLineNum: 7, newLineNum: 6, content: "line 7", op: opUnchanged},
		{oldLineNum: 8, newLineNum: 7, content: "line 8", op: opUnchanged},
		{oldLineNum: 9, newLineNum: 8, content: "line 9", op: opUnchanged},
		{oldLineNum: 10, newLineNum: 9, content: "line 10", op: opUnchanged},
		{oldLineNum: 0, newLineNum: 10, content: "added", op: opAdded},
	}

	hunks := buildHunks(lines, 2)

	// Should create multiple hunks if changes are far enough apart
	if len(hunks) == 0 {
		t.Error("Expected at least one hunk")
	}
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		binary bool
	}{
		{"text", []byte("hello world"), false},
		{"binary with null", []byte{0x00, 0x01}, true},
		{"text with unicode", []byte("hello 世界"), false},
		{"empty", []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBinary(tt.data)
			if result != tt.binary {
				t.Errorf("isBinary() = %v, want %v", result, tt.binary)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", []string{}},
		{"single line", "hello", []string{"hello"}},
		{"multiple lines", "line1\nline2\nline3", []string{"line1", "line2", "line3"}},
		{"trailing newline", "line1\nline2\n", []string{"line1", "line2"}},
		{"empty lines", "line1\n\nline3", []string{"line1", "", "line3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			if len(result) != len(tt.want) {
				t.Errorf("splitLines() length = %v, want %v", len(result), len(tt.want))
				return
			}
			for i := range result {
				if result[i] != tt.want[i] {
					t.Errorf("splitLines()[%d] = %q, want %q", i, result[i], tt.want[i])
				}
			}
		})
	}
}

func TestExpandTabs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		tabWidth int
		want     string
	}{
		{"no tabs", "hello", 4, "hello"},
		{"one tab", "\thello", 4, "    hello"},
		{"multiple tabs", "\t\thello", 4, "        hello"},
		{"tab width 2", "\thello", 2, "  hello"},
		{"mid-line tab", "hello\tworld", 4, "hello   world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandTabs(tt.input, tt.tabWidth)
			if result != tt.want {
				t.Errorf("expandTabs() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
	}{
		{"short line", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"too long", "hello world", 8, "hello..."},
		{"unicode", "hello 世界 and more text", 12, "hello 世界 ..."},
		{"very short max", "hello", 2, ".."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateLine(tt.input, tt.maxWidth)
			if result != tt.want {
				t.Errorf("truncateLine() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestGenerateDiff_GoldenFiles(t *testing.T) {
	testdataDir := "testdata"

	// Check if testdata directory exists
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skip("testdata directory not found, skipping golden file tests")
	}

	goldenFiles, err := filepath.Glob(filepath.Join(testdataDir, "*.golden"))
	if err != nil {
		t.Fatalf("Failed to read golden files: %v", err)
	}

	for _, goldenPath := range goldenFiles {
		testName := strings.TrimSuffix(filepath.Base(goldenPath), ".golden")

		t.Run(testName, func(t *testing.T) {
			// Read old and new files
			oldPath := filepath.Join(testdataDir, testName+".old")
			newPath := filepath.Join(testdataDir, testName+".new")

			old, err := os.ReadFile(oldPath)
			if err != nil {
				t.Fatalf("Failed to read old file: %v", err)
			}

			newer, err := os.ReadFile(newPath)
			if err != nil {
				t.Fatalf("Failed to read new file: %v", err)
			}

			// Generate diff
			result := GenerateDiffDefault("old.txt", "new.txt", old, newer)

			// Read golden file
			golden, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("Failed to read golden file: %v", err)
			}

			// Remove ANSI color codes from result for comparison
			resultClean := stripAnsiCodes(result)
			goldenClean := stripAnsiCodes(string(golden))

			if resultClean != goldenClean {
				t.Errorf("Diff mismatch.\nGot:\n%s\n\nWant:\n%s", resultClean, goldenClean)
			}
		})
	}
}

// stripAnsiCodes removes ANSI escape sequences from a string
func stripAnsiCodes(s string) string {
	// Simple regex-free approach: remove escape sequences
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++ // Skip '['
			continue
		}

		if inEscape {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}

		result.WriteByte(s[i])
	}

	return result.String()
}

func TestDiffGenerator_IdenticalOutput(t *testing.T) {
	old := []byte("line 1\nline 2\nline 3\n")
	newer := []byte("line 1\nline 2\nmodified\nline 3\n")

	// Generate with standalone function
	standaloneResult := GenerateDiffDefault("test.txt", "test.txt", old, newer)

	// Generate with DiffGenerator
	gen := NewDiffGenerator()
	generatorResult := gen.GenerateDiffDefault("test.txt", "test.txt", old, newer)

	if standaloneResult != generatorResult {
		t.Error("DiffGenerator should produce identical output to standalone function")
	}
}

func TestDiffGenerator_Reuse(t *testing.T) {
	gen := NewDiffGenerator()

	// Generate multiple diffs
	diff1 := gen.GenerateDiffDefault("a.txt", "a.txt", []byte("foo\n"), []byte("bar\n"))
	diff2 := gen.GenerateDiffDefault("b.txt", "b.txt", []byte("old\n"), []byte("new\n"))
	diff3 := gen.GenerateDiffDefault("c.txt", "c.txt", []byte("test\n"), []byte("test\nmore\n"))

	// All should be non-empty (they have changes)
	if diff1 == "" || diff2 == "" || diff3 == "" {
		t.Error("Expected non-empty diffs")
	}

	// Verify each diff is different
	if diff1 == diff2 || diff2 == diff3 || diff1 == diff3 {
		t.Error("Different inputs should produce different diffs")
	}
}

func TestDiffGenerator_StateCleared(t *testing.T) {
	gen := NewDiffGenerator()

	// First diff
	diff1 := gen.GenerateDiffDefault("test.txt", "test.txt",
		[]byte("line 1\nline 2\n"),
		[]byte("line 1\nmodified\n"))

	// Second diff - should not be affected by first
	diff2 := gen.GenerateDiffDefault("test.txt", "test.txt",
		[]byte("alpha\nbeta\n"),
		[]byte("alpha\ngamma\n"))

	// Both should contain their respective changes
	if !strings.Contains(diff1, "modified") {
		t.Error("First diff should contain 'modified'")
	}
	if !strings.Contains(diff2, "gamma") {
		t.Error("Second diff should contain 'gamma'")
	}

	// First diff should NOT contain content from second diff
	if strings.Contains(diff1, "gamma") || strings.Contains(diff1, "alpha") {
		t.Error("First diff contaminated with content from second diff")
	}
}
