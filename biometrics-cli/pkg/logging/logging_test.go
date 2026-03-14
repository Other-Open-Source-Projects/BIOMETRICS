package logging

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotationConfigDefaults(t *testing.T) {
	cfg := RotationConfig{
		Enabled:    true,
		MaxSize:    1024,
		MaxAge:     7,
		MaxBackups: 3,
		Compress:   true,
		LocalTime:  true,
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.MaxSize != 1024 {
		t.Errorf("MaxSize = %d, want 1024", cfg.MaxSize)
	}
	if cfg.MaxAge != 7 {
		t.Errorf("MaxAge = %d, want 7", cfg.MaxAge)
	}
	if cfg.MaxBackups != 3 {
		t.Errorf("MaxBackups = %d, want 3", cfg.MaxBackups)
	}
	if !cfg.Compress {
		t.Error("Compress should be true")
	}
	if !cfg.LocalTime {
		t.Error("LocalTime should be true")
	}
}

func TestNewRotatingWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := RotationConfig{
		Enabled:    true,
		MaxSize:    1024,
		MaxAge:     7,
		MaxBackups: 3,
	}

	writer, err := NewRotatingWriter(logFile, cfg)
	if err != nil {
		t.Fatalf("NewRotatingWriter() error = %v", err)
	}
	if writer == nil {
		t.Fatal("NewRotatingWriter() returned nil")
	}

	if err := writer.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNewRotatingWriterInvalidPath(t *testing.T) {
	logFile := "/nonexistent/path/test.log"

	cfg := RotationConfig{
		Enabled: true,
		MaxSize: 1024,
	}

	_, err := NewRotatingWriter(logFile, cfg)
	if err == nil {
		t.Error("NewRotatingWriter() should fail with invalid path")
	}
}

func TestRotatingWriterWrite(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := RotationConfig{
		Enabled:    true,
		MaxSize:    1024,
		MaxAge:     7,
		MaxBackups: 3,
	}

	writer, err := NewRotatingWriter(logFile, cfg)
	if err != nil {
		t.Fatalf("NewRotatingWriter() error = %v", err)
	}

	data := []byte("test log entry\n")
	n, err := writer.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() n = %d, want %d", n, len(data))
	}

	if err := writer.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file should exist: %v", err)
	}
}

func TestRotatingWriterRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := RotationConfig{
		Enabled:    true,
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
	}

	writer, err := NewRotatingWriter(logFile, cfg)
	if err != nil {
		t.Fatalf("NewRotatingWriter() error = %v", err)
	}

	data := []byte("0123456789\n")
	for i := 0; i < 5; i++ {
		_, err := writer.Write(data)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	matches, _ := filepath.Glob(logFile + ".*")
	if len(matches) == 0 {
		t.Error("Expected rotated log files")
	}
}

func TestRotatingWriterSync(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := RotationConfig{
		Enabled: true,
		MaxSize: 1024,
	}

	writer, err := NewRotatingWriter(logFile, cfg)
	if err != nil {
		t.Fatalf("NewRotatingWriter() error = %v", err)
	}

	if err := writer.Sync(); err != nil {
		t.Errorf("Sync() error = %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNewMultiWriter(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	multi := NewMultiWriter(buf1, buf2)

	data := []byte("test data")
	n, err := multi.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() n = %d, want %d", n, len(data))
	}

	if buf1.Len() != len(data) {
		t.Errorf("buf1.Len() = %d, want %d", buf1.Len(), len(data))
	}
	if buf2.Len() != len(data) {
		t.Errorf("buf2.Len() = %d, want %d", buf2.Len(), len(data))
	}
}

func TestMultiWriterAddWriter(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	multi := NewMultiWriter(buf1)
	multi.AddWriter(buf2)

	data := []byte("test data")
	multi.Write(data)

	if buf2.Len() != len(data) {
		t.Errorf("buf2.Len() = %d, want %d", buf2.Len(), len(data))
	}
}

func TestMultiWriterSync(t *testing.T) {
	buf := &syncedBuffer{data: &bytes.Buffer{}}
	multi := NewMultiWriter(buf)

	if err := multi.Sync(); err != nil {
		t.Errorf("Sync() error = %v", err)
	}
}

func TestNewLevelFilter(t *testing.T) {
	buf := &bytes.Buffer{}
	filter := NewLevelFilter(buf, InfoLevel)

	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     DebugLevel,
		Message:   "debug message",
	}

	data, _ := json.Marshal(entry)
	n, err := filter.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if n != len(data) {
		t.Errorf("Write() n = %d, want %d", n, len(data))
	}

	if buf.Len() != 0 {
		t.Error("Debug level should be filtered out")
	}
}

func TestLevelFilterAllowsMatchingLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	filter := NewLevelFilter(buf, InfoLevel)

	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     InfoLevel,
		Message:   "info message",
	}

	data, _ := json.Marshal(entry)
	filter.Write(data)

	if buf.Len() == 0 {
		t.Error("Info level should be allowed")
	}
}

func TestNewAdvancedLogger(t *testing.T) {
	cfg := &LoggerConfig{
		Level:  DebugLevel,
		Format: JSONFormat,
	}

	rotateCfg := &RotationConfig{
		Enabled: false,
	}

	logger, err := NewAdvancedLogger(cfg, rotateCfg)
	if err != nil {
		t.Fatalf("NewAdvancedLogger() error = %v", err)
	}
	if logger == nil {
		t.Fatal("NewAdvancedLogger() returned nil")
	}
}

func TestNewAdvancedLoggerNilConfig(t *testing.T) {
	logger, err := NewAdvancedLogger(nil, nil)
	if err != nil {
		t.Fatalf("NewAdvancedLogger() error = %v", err)
	}
	if logger == nil {
		t.Fatal("NewAdvancedLogger() returned nil")
	}
}

func TestNewFileLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := RotationConfig{
		Enabled:    true,
		MaxSize:    1024,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := NewFileLogger(logFile, InfoLevel, cfg)
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	if logger == nil {
		t.Fatal("NewFileLogger() returned nil")
	}

	logger.Info("test message")

	if err := logger.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Log file should exist: %v", err)
	}
}

func TestNewFileLoggerDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := RotationConfig{
		Enabled: false,
	}

	logger, err := NewFileLogger(logFile, InfoLevel, cfg)
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}

	if !logger.rotateCfg.Enabled {
		t.Error("Rotation should be enabled by default")
	}
	if logger.rotateCfg.MaxSize == 0 {
		t.Error("MaxSize should have default value")
	}
	if logger.rotateCfg.MaxBackups == 0 {
		t.Error("MaxBackups should have default value")
	}
	if logger.rotateCfg.MaxAge == 0 {
		t.Error("MaxAge should have default value")
	}

	logger.Close()
}

func TestNewMultiOutputLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	configs := []OutputConfig{
		{
			Type:     "file",
			Path:     logFile,
			MinLevel: DebugLevel,
		},
	}

	logger, err := NewMultiOutputLogger(configs)
	if err != nil {
		t.Fatalf("NewMultiOutputLogger() error = %v", err)
	}
	if logger == nil {
		t.Fatal("NewMultiOutputLogger() returned nil")
	}

	logger.Info("test message")

	if err := logger.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNewMultiOutputLoggerStdout(t *testing.T) {
	configs := []OutputConfig{
		{
			Type:     "stdout",
			MinLevel: InfoLevel,
		},
	}

	logger, err := NewMultiOutputLogger(configs)
	if err != nil {
		t.Fatalf("NewMultiOutputLogger() error = %v", err)
	}
	if logger == nil {
		t.Fatal("NewMultiOutputLogger() returned nil")
	}
}

func TestNewMultiOutputLoggerStderr(t *testing.T) {
	configs := []OutputConfig{
		{
			Type:     "stderr",
			MinLevel: ErrorLevel,
		},
	}

	logger, err := NewMultiOutputLogger(configs)
	if err != nil {
		t.Fatalf("NewMultiOutputLogger() error = %v", err)
	}
	if logger == nil {
		t.Fatal("NewMultiOutputLogger() returned nil")
	}
}

func TestAdvancedLoggerDebug(t *testing.T) {
	var buf bytes.Buffer
	cfg := &LoggerConfig{
		Level:  DebugLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewAdvancedLogger(cfg, nil)
	logger.Debug("debug message", String("key", "value"))

	if buf.Len() == 0 {
		t.Error("Debug should write output")
	}
}

func TestAdvancedLoggerInfo(t *testing.T) {
	var buf bytes.Buffer
	cfg := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewAdvancedLogger(cfg, nil)
	logger.Info("info message")

	if buf.Len() == 0 {
		t.Error("Info should write output")
	}
}

func TestAdvancedLoggerWarn(t *testing.T) {
	var buf bytes.Buffer
	cfg := &LoggerConfig{
		Level:  WarnLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewAdvancedLogger(cfg, nil)
	logger.Warn("warn message")

	if buf.Len() == 0 {
		t.Error("Warn should write output")
	}
}

func TestAdvancedLoggerError(t *testing.T) {
	var buf bytes.Buffer
	cfg := &LoggerConfig{
		Level:       ErrorLevel,
		Format:      JSONFormat,
		ErrorOutput: &buf,
	}

	logger, _ := NewAdvancedLogger(cfg, nil)
	logger.Error("error message")

	if buf.Len() == 0 {
		t.Error("Error should write output")
	}
}

func TestAdvancedLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	cfg := &LoggerConfig{
		Level:  DebugLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewAdvancedLogger(cfg, nil)
	logger = logger.WithFields(String("global", "value"))
	logger.Info("message")

	if buf.Len() == 0 {
		t.Error("WithFields should write output")
	}
}

func TestAdvancedLoggerWithContext(t *testing.T) {
	var buf bytes.Buffer
	cfg := &LoggerConfig{
		Level:  DebugLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewAdvancedLogger(cfg, nil)
	logger = logger.WithContext(nil)
	logger.Info("message")

	if buf.Len() == 0 {
		t.Error("WithContext should write output")
	}
}

func TestAdvancedLoggerSync(t *testing.T) {
	var buf bytes.Buffer
	cfg := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewAdvancedLogger(cfg, nil)
	if err := logger.Sync(); err != nil {
		t.Errorf("Sync() error = %v", err)
	}
}

func TestAdvancedLoggerClose(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := RotationConfig{
		Enabled:    true,
		MaxSize:    1024,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := NewFileLogger(logFile, InfoLevel, cfg)
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}

	if err := logger.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestAdvancedLoggerGetUnderlyingLogger(t *testing.T) {
	cfg := &LoggerConfig{
		Level: InfoLevel,
	}

	logger, _ := NewAdvancedLogger(cfg, nil)
	underlying := logger.GetUnderlyingLogger()

	if underlying == nil {
		t.Error("GetUnderlyingLogger() should not return nil")
	}
}

func TestAdvancedLoggerRotate(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := RotationConfig{
		Enabled:    true,
		MaxSize:    1024,
		MaxAge:     7,
		MaxBackups: 3,
	}

	logger, err := NewFileLogger(logFile, InfoLevel, cfg)
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}

	logger.Info("first message")

	if err := logger.Rotate(); err != nil {
		t.Errorf("Rotate() error = %v", err)
	}

	logger.Close()
}

func TestBufferedLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  DebugLevel,
		Format: JSONFormat,
		Output: &buf,
	})

	flushed := false
	buffered := NewBufferedLogger(logger, 1, func(data []byte) {
		flushed = true
	})

	buffered.Log(InfoLevel, "test message")

	if !flushed {
		t.Error("Callback should have been triggered")
	}

	if err := buffered.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestBufferedLoggerManualFlush(t *testing.T) {
	logger := NewLogger(&LoggerConfig{
		Level:  DebugLevel,
		Format: JSONFormat,
		Output: &bytes.Buffer{},
	})

	callCount := 0
	buffered := NewBufferedLogger(logger, 10000, func(data []byte) {
		callCount++
	})

	for i := 0; i < 5; i++ {
		buffered.Log(InfoLevel, "test message")
	}

	buffered.Flush()

	if callCount != 1 {
		t.Errorf("Expected 1 flush, got %d", callCount)
	}
}

func TestBufferedLoggerClose(t *testing.T) {
	logger := NewLogger(&LoggerConfig{
		Level:  DebugLevel,
		Format: JSONFormat,
		Output: &bytes.Buffer{},
	})

	buffered := NewBufferedLogger(logger, 10000, func(data []byte) {})

	buffered.Log(InfoLevel, "test message")

	if err := buffered.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

type syncedBuffer struct {
	data *bytes.Buffer
}

func (s *syncedBuffer) Write(p []byte) (int, error) {
	return s.data.Write(p)
}

func (s *syncedBuffer) Sync() error {
	return nil
}
