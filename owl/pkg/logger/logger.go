package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents the logging level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelSilent
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelSilent:
		return "SILENT"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging with configurable levels
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	WithFields(fields ...Field) Logger
	SetLevel(level Level)
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value any
}

// F is a convenience function for creating fields
func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// standardLogger implements Logger interface
type standardLogger struct {
	level  Level
	out    io.Writer
	mu     sync.Mutex
	fields []Field
}

// NewLogger creates a new logger with the specified level and output
func NewLogger(level Level, out io.Writer) Logger {
	if out == nil {
		out = os.Stdout
	}
	return &standardLogger{
		level:  level,
		out:    out,
		fields: make([]Field, 0),
	}
}

// NewDefaultLogger creates a logger with Info level writing to stdout
func NewDefaultLogger() Logger {
	return NewLogger(LevelInfo, os.Stdout)
}

// NewSilentLogger creates a logger that outputs nothing
func NewSilentLogger() Logger {
	return NewLogger(LevelSilent, io.Discard)
}

// SetLevel sets the minimum logging level
func (l *standardLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// WithFields returns a new logger with additional fields
func (l *standardLogger) WithFields(fields ...Field) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make([]Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &standardLogger{
		level:  l.level,
		out:    l.out,
		fields: newFields,
	}
}

// Debug logs a debug message
func (l *standardLogger) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, fields...)
}

// Info logs an info message
func (l *standardLogger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, fields...)
}

// Warn logs a warning message
func (l *standardLogger) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, fields...)
}

// Error logs an error message
func (l *standardLogger) Error(msg string, fields ...Field) {
	l.log(LevelError, msg, fields...)
}

// log performs the actual logging
func (l *standardLogger) log(level Level, msg string, fields ...Field) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if this message should be logged
	if level < l.level {
		return
	}

	// Build the log message
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	icon := getIcon(level)

	// Start with timestamp and level
	output := fmt.Sprintf("%s [%s] %s %s", timestamp, level.String(), icon, msg)

	// Add persistent fields
	if len(l.fields) > 0 {
		output += " |"
		for _, field := range l.fields {
			output += fmt.Sprintf(" %s=%v", field.Key, field.Value)
		}
	}

	// Add message-specific fields
	if len(fields) > 0 {
		if len(l.fields) == 0 {
			output += " |"
		}
		for _, field := range fields {
			output += fmt.Sprintf(" %s=%v", field.Key, field.Value)
		}
	}

	output += "\n"

	// Write to output
	_, _ = l.out.Write([]byte(output))
}

// getIcon returns an emoji icon for each level
func getIcon(level Level) string {
	switch level {
	case LevelDebug:
		return "🔍"
	case LevelInfo:
		return "ℹ️"
	case LevelWarn:
		return "⚠️"
	case LevelError:
		return "❌"
	default:
		return "•"
	}
}

// Global default logger
var defaultLogger = NewDefaultLogger()

// SetDefault sets the global default logger
func SetDefault(l Logger) {
	defaultLogger = l
}

// Default returns the global default logger
func Default() Logger {
	return defaultLogger
}

// Convenience functions using the default logger
func Debug(msg string, fields ...Field) {
	defaultLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...Field) {
	defaultLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...Field) {
	defaultLogger.Warn(msg, fields...)
}

func Error(msg string, fields ...Field) {
	defaultLogger.Error(msg, fields...)
}
