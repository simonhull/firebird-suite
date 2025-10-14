package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		name          string
		level         Level
		logFunc       func(Logger, string)
		expectedInLog bool
	}{
		{
			name:          "debug message when level is debug",
			level:         LevelDebug,
			logFunc:       func(l Logger, msg string) { l.Debug(msg) },
			expectedInLog: true,
		},
		{
			name:          "debug message when level is info",
			level:         LevelInfo,
			logFunc:       func(l Logger, msg string) { l.Debug(msg) },
			expectedInLog: false,
		},
		{
			name:          "info message when level is info",
			level:         LevelInfo,
			logFunc:       func(l Logger, msg string) { l.Info(msg) },
			expectedInLog: true,
		},
		{
			name:          "warn message when level is error",
			level:         LevelError,
			logFunc:       func(l Logger, msg string) { l.Warn(msg) },
			expectedInLog: false,
		},
		{
			name:          "error message when level is error",
			level:         LevelError,
			logFunc:       func(l Logger, msg string) { l.Error(msg) },
			expectedInLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewLogger(tt.level, buf)

			testMsg := "test message"
			tt.logFunc(logger, testMsg)

			output := buf.String()
			contains := strings.Contains(output, testMsg)

			if contains != tt.expectedInLog {
				t.Errorf("expected message in log: %v, got: %v (output: %s)",
					tt.expectedInLog, contains, output)
			}
		})
	}
}

func TestLogger_Fields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LevelInfo, buf)

	logger.Info("test message",
		F("key1", "value1"),
		F("key2", 42),
	)

	output := buf.String()

	if !strings.Contains(output, "key1=value1") {
		t.Errorf("expected key1=value1 in output, got: %s", output)
	}

	if !strings.Contains(output, "key2=42") {
		t.Errorf("expected key2=42 in output, got: %s", output)
	}
}

func TestLogger_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	baseLogger := NewLogger(LevelInfo, buf)

	// Create a child logger with persistent fields
	childLogger := baseLogger.WithFields(
		F("requestID", "123"),
		F("userID", "456"),
	)

	childLogger.Info("test message", F("action", "create"))

	output := buf.String()

	// Check all fields are present
	expectedFields := []string{"requestID=123", "userID=456", "action=create"}
	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("expected %s in output, got: %s", field, output)
		}
	}
}

func TestLogger_SilentMode(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LevelSilent, buf)

	logger.Debug("debug msg")
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")

	if buf.Len() > 0 {
		t.Errorf("expected no output in silent mode, got: %s", buf.String())
	}
}

func TestLogger_SetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LevelError, buf)

	// Should not log at error level
	logger.Info("info message")
	if buf.Len() > 0 {
		t.Errorf("expected no output at error level for info, got: %s", buf.String())
	}

	// Change to info level
	logger.SetLevel(LevelInfo)
	buf.Reset()

	// Should now log
	logger.Info("info message")
	if buf.Len() == 0 {
		t.Error("expected output after changing to info level")
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelSilent, "SILENT"},
		{Level(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	logger := NewLogger(LevelInfo, bytes.NewBuffer(make([]byte, 0, 1024*1024)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			F("iteration", i),
			F("key", "value"),
		)
	}
}

func BenchmarkLogger_InfoNoFields(b *testing.B) {
	logger := NewLogger(LevelInfo, bytes.NewBuffer(make([]byte, 0, 1024*1024)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}
