package exec

import (
	"bufio"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StreamingWriter wraps output with beautiful formatting
type StreamingWriter struct {
	prefix string
	style  lipgloss.Style
	writer io.Writer
	// Buffer for incomplete lines
	buffer []byte
}

// NewStreamingWriter creates a formatted output writer
func NewStreamingWriter(writer io.Writer, prefix string, color lipgloss.Color) *StreamingWriter {
	return &StreamingWriter{
		prefix: prefix,
		style:  lipgloss.NewStyle().Foreground(color),
		writer: writer,
		buffer: make([]byte, 0),
	}
}

// Write formats and writes output line by line
func (s *StreamingWriter) Write(p []byte) (n int, err error) {
	// Append to buffer
	s.buffer = append(s.buffer, p...)

	// Process complete lines
	lines := strings.Split(string(s.buffer), "\n")

	// Keep the last incomplete line in buffer
	if len(lines) > 0 {
		s.buffer = []byte(lines[len(lines)-1])
		lines = lines[:len(lines)-1]
	}

	// Write complete lines
	for _, line := range lines {
		if line != "" || len(lines) > 1 { // Don't write empty lines unless they're intentional
			formatted := s.formatLine(line)
			if _, err := s.writer.Write([]byte(formatted + "\n")); err != nil {
				return 0, err
			}
		}
	}

	return len(p), nil
}

// Flush writes any remaining buffered content
func (s *StreamingWriter) Flush() error {
	if len(s.buffer) > 0 {
		formatted := s.formatLine(string(s.buffer))
		_, err := s.writer.Write([]byte(formatted + "\n"))
		s.buffer = s.buffer[:0]
		return err
	}
	return nil
}

// formatLine formats a single line with prefix and style
func (s *StreamingWriter) formatLine(line string) string {
	if s.prefix != "" {
		line = s.prefix + line
	}
	// Always apply style if it exists
	return s.style.Render(line)
}

// PrefixWriter adds a prefix to each line of output
type PrefixWriter struct {
	prefix string
	writer io.Writer
	buffer []byte
}

// NewPrefixWriter creates a writer that prefixes each line
func NewPrefixWriter(writer io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{
		prefix: prefix,
		writer: writer,
		buffer: make([]byte, 0),
	}
}

// Write adds prefix to each line
func (p *PrefixWriter) Write(data []byte) (n int, err error) {
	n = len(data)
	p.buffer = append(p.buffer, data...)

	// Process complete lines
	scanner := bufio.NewScanner(strings.NewReader(string(p.buffer)))
	var lastLine string
	var hasLastLine bool

	for scanner.Scan() {
		if hasLastLine {
			// Write the previous line (we know it's complete)
			if _, err := p.writer.Write([]byte(p.prefix + lastLine + "\n")); err != nil {
				return 0, err
			}
		}
		lastLine = scanner.Text()
		hasLastLine = true
	}

	// Check if we have a complete last line
	if hasLastLine {
		if strings.HasSuffix(string(data), "\n") {
			// Last line is complete, write it
			if _, err := p.writer.Write([]byte(p.prefix + lastLine + "\n")); err != nil {
				return 0, err
			}
			p.buffer = p.buffer[:0]
		} else {
			// Last line is incomplete, keep it in buffer
			p.buffer = []byte(lastLine)
		}
	}

	return n, nil
}

// TeeWriter writes to multiple writers simultaneously
type TeeWriter struct {
	writers []io.Writer
}

// NewTeeWriter creates a writer that duplicates output to multiple writers
func NewTeeWriter(writers ...io.Writer) *TeeWriter {
	return &TeeWriter{
		writers: writers,
	}
}

// Write writes to all underlying writers
func (t *TeeWriter) Write(p []byte) (n int, err error) {
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
	}
	return len(p), nil
}