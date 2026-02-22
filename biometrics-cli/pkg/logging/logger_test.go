package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level  LogLevel
		expect string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
		{LogLevel(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expect {
				t.Errorf("String() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestLogLevelMarshalJSON(t *testing.T) {
	level := InfoLevel
	data, err := level.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var result string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if result != "info" {
		t.Errorf("MarshalJSON() = %v, want %v", result, "info")
	}
}

func TestLogEntryMarshalJSON(t *testing.T) {
	entry := &LogEntry{
		Timestamp: time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC),
		Level:     InfoLevel,
		Message:   "test message",
		Fields: map[string]interface{}{
			"key": "value",
		},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if result["level"] != "info" {
		t.Errorf("level = %v, want %v", result["level"], "info")
	}
	if result["message"] != "test message" {
		t.Errorf("message = %v, want %v", result["message"], "test message")
	}
}

func TestField(t *testing.T) {
	err := &testError{"test error"}

	tests := []struct {
		name   string
		field  Field
		expect interface{}
	}{
		{"String", String("key", "value"), "value"},
		{"Int", Int("num", 42), 42},
		{"Int64", Int64("num", int64(42)), int64(42)},
		{"Bool", Bool("flag", true), true},
		{"Float64", Float64("pi", 3.14), 3.14},
		{"Err", Err(err), "test error"},
		{"ErrNil", Err(nil), nil},
		{"Any", Any("data", "test-value"), "test-value"},
		{"Duration", Duration("dur", time.Second), "1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Value != tt.expect {
				t.Errorf("%s Value = %v, want %v", tt.name, tt.field.Value, tt.expect)
			}
			if tt.field.Key == "" {
				t.Error("Key should not be empty")
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		logger := NewLogger(nil)
		if logger == nil {
			t.Fatal("NewLogger() returned nil")
		}
		if logger.config.Level != InfoLevel {
			t.Errorf("Level = %v, want %v", logger.config.Level, InfoLevel)
		}
		if logger.config.Format != JSONFormat {
			t.Errorf("Format = %v, want %v", logger.config.Format, JSONFormat)
		}
	})

	t.Run("CustomConfig", func(t *testing.T) {
		var buf bytes.Buffer
		config := &LoggerConfig{
			Level:        DebugLevel,
			Format:       TextFormat,
			Output:       &buf,
			AddCaller:    false,
			AddTimestamp: true,
		}
		logger := NewLogger(config)
		if logger.config.Level != DebugLevel {
			t.Errorf("Level = %v, want %v", logger.config.Level, DebugLevel)
		}
		if logger.config.Format != TextFormat {
			t.Errorf("Format = %v, want %v", logger.config.Format, TextFormat)
		}
	})
}

func TestLoggerWithLevel(t *testing.T) {
	logger := NewLogger(nil)
	newLogger := logger.WithLevel(DebugLevel)

	if logger.config.Level == newLogger.config.Level {
		t.Error("WithLevel() should create a new logger with different level")
	}
	if newLogger.config.Level != DebugLevel {
		t.Errorf("WithLevel() Level = %v, want %v", newLogger.config.Level, DebugLevel)
	}
}

func TestLoggerWithFormat(t *testing.T) {
	logger := NewLogger(nil)
	newLogger := logger.WithFormat(TextFormat)

	if logger.config.Format == newLogger.config.Format {
		t.Error("WithFormat() should create a new logger with different format")
	}
	if newLogger.config.Format != TextFormat {
		t.Errorf("WithFormat() Format = %v, want %v", newLogger.config.Format, TextFormat)
	}
}

func TestLoggerWithFields(t *testing.T) {
	logger := NewLogger(nil)
	newLogger := logger.WithFields(String("key1", "value1"), Int("key2", 42))

	if len(newLogger.fields) != 2 {
		t.Errorf("WithFields() fields count = %v, want %v", len(newLogger.fields), 2)
	}
	if newLogger.fields["key1"] != "value1" {
		t.Errorf("WithFields() key1 = %v, want %v", newLogger.fields["key1"], "value1")
	}
	if newLogger.fields["key2"] != 42 {
		t.Errorf("WithFields() key2 = %v, want %v", newLogger.fields["key2"], 42)
	}
}

func TestLoggerWithContext(t *testing.T) {
	ctx := WithTraceContext(context.Background(), "trace-123", "span-456")
	logger := NewLogger(nil)
	newLogger := logger.WithContext(ctx)

	if newLogger.fields["trace_id"] != "trace-123" {
		t.Errorf("WithContext() trace_id = %v, want %v", newLogger.fields["trace_id"], "trace-123")
	}
	if newLogger.fields["span_id"] != "span-456" {
		t.Errorf("WithContext() span_id = %v, want %v", newLogger.fields["span_id"], "span-456")
	}
}

func TestLoggerWithContextNil(t *testing.T) {
	logger := NewLogger(nil)
	newLogger := logger.WithContext(nil)

	if newLogger == nil {
		t.Error("WithContext(nil) should not return nil")
	}
}

func TestLoggerDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  DebugLevel,
		Format: JSONFormat,
		Output: &buf,
	})

	logger.Debug("debug message", String("key", "value"))

	if buf.Len() == 0 {
		t.Error("Debug() should write output")
	}

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Level != DebugLevel {
		t.Errorf("Level = %v, want %v", entry.Level, DebugLevel)
	}
	if entry.Message != "debug message" {
		t.Errorf("Message = %v, want %v", entry.Message, "debug message")
	}
	if entry.Fields["key"] != "value" {
		t.Errorf("Fields[key] = %v, want %v", entry.Fields["key"], "value")
	}
}

func TestLoggerDebugFiltered(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	})

	logger.Debug("debug message")

	if buf.Len() != 0 {
		t.Error("Debug() should not write when level is below threshold")
	}
}

func TestLoggerInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	})

	logger.Info("info message")

	if buf.Len() == 0 {
		t.Error("Info() should write output")
	}

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Level != InfoLevel {
		t.Errorf("Level = %v, want %v", entry.Level, InfoLevel)
	}
	if entry.Message != "info message" {
		t.Errorf("Message = %v, want %v", entry.Message, "info message")
	}
}

func TestLoggerWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  WarnLevel,
		Format: JSONFormat,
		Output: &buf,
	})

	logger.Warn("warn message", Int("code", 42))

	if buf.Len() == 0 {
		t.Error("Warn() should write output")
	}

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Level != WarnLevel {
		t.Errorf("Level = %v, want %v", entry.Level, WarnLevel)
	}
	if entry.Message != "warn message" {
		t.Errorf("Message = %v, want %v", entry.Message, "warn message")
	}
	// JSON unmarshal converts numbers to float64
	code, ok := entry.Fields["code"].(float64)
	if !ok || int(code) != 42 {
		t.Errorf("Fields[code] = %v, want 42", entry.Fields["code"])
	}
}

func TestLoggerError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:       ErrorLevel,
		Format:      JSONFormat,
		ErrorOutput: &buf,
	})

	logger.Error("error message", Err(&testError{"test"}))

	if buf.Len() == 0 {
		t.Error("Error() should write output")
	}

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Level != ErrorLevel {
		t.Errorf("Level = %v, want %v", entry.Level, ErrorLevel)
	}
	if entry.Message != "error message" {
		t.Errorf("Message = %v, want %v", entry.Message, "error message")
	}
}

func TestLoggerFormatted(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	})

	logger.Infof("info %s with %d", "message", 42)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Message != "info message with 42" {
		t.Errorf("Message = %v, want %v", entry.Message, "info message with 42")
	}
}

func TestLoggerTextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:        InfoLevel,
		Format:       TextFormat,
		Output:       &buf,
		AddTimestamp: true,
	})

	logger.Info("text message")

	output := buf.String()
	if !strings.Contains(output, "[info]") {
		t.Errorf("Text format should contain [info], got: %s", output)
	}
	if !strings.Contains(output, "text message") {
		t.Errorf("Text format should contain message, got: %s", output)
	}
}

func TestLoggerCaller(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:     InfoLevel,
		Format:    JSONFormat,
		Output:    &buf,
		AddCaller: true,
	})

	logger.Info("with caller")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Caller == "" {
		t.Error("Caller should be set when AddCaller is true")
	}
	if !strings.Contains(entry.Caller, "logger_test.go") {
		t.Errorf("Caller should contain test filename, got: %s", entry.Caller)
	}
}

func TestLoggerWithTraceContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceContext(ctx, "trace-abc", "span-def")

	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	})
	logger = logger.WithContext(ctx)

	logger.Info("traced message")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.TraceID != "trace-abc" {
		t.Errorf("TraceID = %v, want %v", entry.TraceID, "trace-abc")
	}
	if entry.SpanID != "span-def" {
		t.Errorf("SpanID = %v, want %v", entry.SpanID, "span-def")
	}
}

func TestLoggerSetOutput(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf1,
	})

	logger.Info("first output")
	if buf1.Len() == 0 {
		t.Error("Should write to initial output")
	}

	logger.SetOutput(&buf2)
	logger.Info("second output")

	if buf2.Len() == 0 {
		t.Error("Should write to new output")
	}
}

func TestLoggerSetErrorOutput(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:       ErrorLevel,
		Format:      JSONFormat,
		ErrorOutput: &buf1,
	})

	logger.Error("first error output")
	if buf1.Len() == 0 {
		t.Error("Should write to initial error output")
	}

	logger.SetErrorOutput(&buf2)
	logger.Error("second error output")

	if buf2.Len() == 0 {
		t.Error("Should write to new error output")
	}
}

func TestLoggerSync(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	})

	if err := logger.Sync(); err != nil {
		t.Errorf("Sync() error = %v", err)
	}
}

func TestGlobalLogger(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := Default()
	defer SetDefault(oldLogger)

	logger := NewLogger(&LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	})
	SetDefault(logger)

	Info("global message")

	if buf.Len() == 0 {
		t.Error("Global Info() should write output")
	}

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Message != "global message" {
		t.Errorf("Message = %v, want %v", entry.Message, "global message")
	}
}

func TestTraceContextFromLogger(t *testing.T) {
	logger := NewLogger(nil)
	logger = logger.WithFields(
		String("trace_id", "trace-xyz"),
		String("span_id", "span-789"),
	)

	traceID, spanID := TraceContextFromLogger(logger)

	if traceID != "trace-xyz" {
		t.Errorf("TraceContextFromLogger() traceID = %v, want %v", traceID, "trace-xyz")
	}
	if spanID != "span-789" {
		t.Errorf("TraceContextFromLogger() spanID = %v, want %v", spanID, "span-789")
	}
}

func TestFieldTime(t *testing.T) {
	now := time.Now()
	field := Time("timestamp", now)

	expected := now.Format(time.RFC3339)
	if field.Value != expected {
		t.Errorf("Time() Value = %v, want %v", field.Value, expected)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
