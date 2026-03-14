// Package logging provides advanced logging capabilities with rotation support.
// This package extends the basic logger with file-based logging, log rotation,
// and multi-destination output support.
package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RotationConfig defines configuration for log file rotation.
type RotationConfig struct {
	// Enabled enables log rotation.
	Enabled bool
	// MaxSize is the maximum size of a single log file in bytes.
	MaxSize int64
	// MaxAge is the maximum number of days to retain old log files.
	MaxAge int
	// MaxBackups is the maximum number of old log files to retain.
	MaxBackups int
	// Compress determines if old log files should be compressed.
	Compress bool
	// LocalTime determines if the time used for formatting file names
	// is the local time (true) or UTC (false).
	LocalTime bool
}

// FileConfig defines configuration for file-based logging.
type FileConfig struct {
	// Filename is the path to the log file.
	Filename string
	// RotateConfig defines rotation settings.
	RotateConfig RotationConfig
}

// WriterConfig defines configuration for multi-output logging.
type WriterConfig struct {
	// Outputs defines multiple output destinations.
	Outputs []OutputConfig
}

// OutputConfig defines configuration for a single output destination.
type OutputConfig struct {
	// Type is the output type: "stdout", "stderr", or "file".
	Type string
	// Path is the file path for file outputs.
	Path string
	// MinLevel is the minimum log level for this output.
	MinLevel LogLevel
	// RotationConfig defines rotation settings for file outputs.
	RotationConfig *RotationConfig
}

// RotatingWriter implements a io.Writer with rotation support.
type RotatingWriter struct {
	mu        sync.Mutex
	filename  string
	rotConfig RotationConfig
	file      *os.File
	size      int64
	openTime  time.Time
}

// NewRotatingWriter creates a new RotatingWriter.
func NewRotatingWriter(filename string, config RotationConfig) (*RotatingWriter, error) {
	// Ensure directory exists even when rotation is disabled
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// When rotation is disabled, create a simple non-rotating writer
	if !config.Enabled {
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		return &RotatingWriter{
			filename:  filename,
			rotConfig: config,
			file:      f,
			openTime:  time.Now(),
		}, nil
	}

	// Non-rotating writer creation done above.

	writer := &RotatingWriter{
		filename:  filename,
		rotConfig: config,
		openTime:  time.Now(),
	}

	if err := writer.openFile(); err != nil {
		return nil, err
	}

	return writer, nil
}

// openFile opens the log file for writing.
func (r *RotatingWriter) openFile() error {
	f, err := os.OpenFile(r.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	r.file = f
	r.size = stat.Size()
	r.openTime = time.Now()

	return nil
}

// Write implements io.Writer.
func (r *RotatingWriter) Write(p []byte) (n int, err error) {
	if r.file == nil {
		return 0, os.ErrInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if rotation is needed
	if r.rotConfig.Enabled && r.size+int64(len(p)) > r.rotConfig.MaxSize {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = r.file.Write(p)
	if err == nil {
		r.size += int64(n)
	}

	return n, err
}

// rotate performs log file rotation.
func (r *RotatingWriter) rotate() error {
	// Close current file
	if err := r.file.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	// Generate timestamp for archive
	timestamp := r.openTime.Format("2006-01-02-150405")

	// Archive filename
	archiveName := fmt.Sprintf("%s.%s.log", r.filename, timestamp)

	// Rename current file to archive
	if err := os.Rename(r.filename, archiveName); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Open new file
	if err := r.openFile(); err != nil {
		return err
	}

	// Clean up old files
	r.cleanup()

	return nil
}

// cleanup removes old log files based on retention policy.
func (r *RotatingWriter) cleanup() {
	if r.rotConfig.MaxBackups == 0 && r.rotConfig.MaxAge == 0 {
		return
	}

	// Get list of log files
	pattern := fmt.Sprintf("%s.*.log", r.filename)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	// Sort by modification time (newest first)
	now := time.Now()

	var toDelete []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		// Check max age
		if r.rotConfig.MaxAge > 0 {
			age := now.Sub(info.ModTime()).Hours() / 24
			if age > float64(r.rotConfig.MaxAge) {
				toDelete = append(toDelete, match)
				continue
			}
		}

		// Check max backups (keep most recent)
		if r.rotConfig.MaxBackups > 0 && len(matches)-len(toDelete) > r.rotConfig.MaxBackups {
			toDelete = append(toDelete, match)
		}
	}

	// Delete old files
	for _, path := range toDelete {
		os.Remove(path)
	}
}

// Close closes the rotating writer.
func (r *RotatingWriter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// Sync flushes any buffered data.
func (r *RotatingWriter) Sync() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		return r.file.Sync()
	}
	return nil
}

// MultiWriter provides a writer that writes to multiple destinations.
type MultiWriter struct {
	writers []io.Writer
	mu      sync.Mutex
}

// NewMultiWriter creates a new MultiWriter.
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write implements io.Writer.
func (m *MultiWriter) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, w := range m.writers {
		if w == nil {
			continue
		}
		nw, we := w.Write(p)
		if err == nil {
			err = we
		}
		if nw > n {
			n = nw
		}
	}

	return n, err
}

// AddWriter adds a writer to the multi-writer.
func (m *MultiWriter) AddWriter(w io.Writer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writers = append(m.writers, w)
}

// Sync flushes all writers.
func (m *MultiWriter) Sync() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, w := range m.writers {
		if s, ok := w.(interface{ Sync() error }); ok {
			if err := s.Sync(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// LevelFilter provides a writer that filters by log level.
type LevelFilter struct {
	writer    io.Writer
	minLevel  LogLevel
	formatter Formatter
	mu        sync.Mutex
}

// Formatter defines a log entry formatter.
type Formatter interface {
	Format(entry *LogEntry) ([]byte, error)
}

// NewLevelFilter creates a new LevelFilter.
func NewLevelFilter(writer io.Writer, minLevel LogLevel) *LevelFilter {
	return &LevelFilter{
		writer:   writer,
		minLevel: minLevel,
	}
}

// SetFormatter sets the formatter for the level filter.
func (f *LevelFilter) SetFormatter(formatter Formatter) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.formatter = formatter
}

// Write implements io.Writer.
func (f *LevelFilter) Write(p []byte) (n int, err error) {
	// Try to parse as JSON log entry
	var entry LogEntry
	if err := json.Unmarshal(p, &entry); err != nil {
		// Not JSON, write as-is
		return f.writer.Write(p)
	}

	// Check if level meets minimum
	if entry.Level < f.minLevel {
		return len(p), nil
	}

	return f.writer.Write(p)
}

// AdvancedLogger provides advanced logging capabilities with rotation.
type AdvancedLogger struct {
	logger    *Logger
	rotateCfg *RotationConfig
	writer    *RotatingWriter
	multi     *MultiWriter
	mu        sync.RWMutex
}

// NewAdvancedLogger creates a new AdvancedLogger with rotation support.
func NewAdvancedLogger(config *LoggerConfig, rotateConfig *RotationConfig) (*AdvancedLogger, error) {
	logger := NewLogger(config)

	var writer *RotatingWriter
	var err error

	if rotateConfig != nil && rotateConfig.Enabled && config != nil && config.Output != nil {
		if fw, ok := config.Output.(*os.File); ok {
			writer, err = NewRotatingWriter(fw.Name(), *rotateConfig)
			if err != nil {
				return nil, err
			}
		}
	}

	return &AdvancedLogger{
		logger:    logger,
		rotateCfg: rotateConfig,
		writer:    writer,
	}, nil
}

// NewFileLogger creates a new logger that writes to a file with rotation.
func NewFileLogger(filename string, level LogLevel, rotateCfg RotationConfig) (*AdvancedLogger, error) {
	if !rotateCfg.Enabled {
		rotateCfg.Enabled = true
		rotateCfg.MaxSize = 10 * 1024 * 1024
		rotateCfg.MaxBackups = 5
		rotateCfg.MaxAge = 30
	}

	writer, err := NewRotatingWriter(filename, rotateCfg)
	if err != nil {
		return nil, err
	}

	config := &LoggerConfig{
		Level:        level,
		Format:       JSONFormat,
		Output:       writer,
		AddCaller:    true,
		AddTimestamp: true,
	}

	logger := NewLogger(config)

	return &AdvancedLogger{
		logger:    logger,
		rotateCfg: &rotateCfg,
		writer:    writer,
	}, nil
}

// NewMultiOutputLogger creates a logger with multiple outputs.
func NewMultiOutputLogger(configs []OutputConfig) (*AdvancedLogger, error) {
	var writers []io.Writer

	for _, cfg := range configs {
		var w io.Writer

		switch cfg.Type {
		case "stdout":
			w = os.Stdout
		case "stderr":
			w = os.Stderr
		case "file":
			rotateCfg := RotationConfig{Enabled: false}
			if cfg.RotationConfig != nil {
				rotateCfg = *cfg.RotationConfig
			}
			writer, err := NewRotatingWriter(cfg.Path, rotateCfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create rotating writer for %s: %w", cfg.Path, err)
			}
			w = writer
		}

		if cfg.MinLevel > 0 {
			w = NewLevelFilter(w, cfg.MinLevel)
		}

		writers = append(writers, w)
	}

	multi := NewMultiWriter(writers...)

	config := &LoggerConfig{
		Level:        DebugLevel,
		Format:       JSONFormat,
		Output:       multi,
		AddCaller:    true,
		AddTimestamp: true,
	}

	logger := NewLogger(config)

	return &AdvancedLogger{
		logger: logger,
		multi:  multi,
	}, nil
}

// Debug logs a message at debug level.
func (l *AdvancedLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, fields...)
}

// Info logs a message at info level.
func (l *AdvancedLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, fields...)
}

// Warn logs a message at warn level.
func (l *AdvancedLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, fields...)
}

// Error logs a message at error level.
func (l *AdvancedLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, fields...)
}

// WithFields creates a new logger with additional fields.
func (l *AdvancedLogger) WithFields(fields ...Field) *AdvancedLogger {
	newLogger := &AdvancedLogger{
		logger:    l.logger.WithFields(fields...),
		rotateCfg: l.rotateCfg,
		writer:    l.writer,
		multi:     l.multi,
	}

	return newLogger
}

// WithContext creates a new logger with context.
func (l *AdvancedLogger) WithContext(ctx context.Context) *AdvancedLogger {
	newLogger := &AdvancedLogger{
		logger:    l.logger.WithContext(ctx),
		rotateCfg: l.rotateCfg,
		writer:    l.writer,
		multi:     l.multi,
	}

	return newLogger
}

// Sync flushes any buffered log entries.
func (l *AdvancedLogger) Sync() error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var errs []error

	if l.logger != nil {
		if err := l.logger.Sync(); err != nil && !isIgnorableSyncError(err) {
			errs = append(errs, err)
		}
	}

	if l.writer != nil {
		if err := l.writer.Sync(); err != nil && !isIgnorableSyncError(err) {
			errs = append(errs, err)
		}
	}

	if l.multi != nil {
		if err := l.multi.Sync(); err != nil && !isIgnorableSyncError(err) {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// Close closes the logger and its resources.
func (l *AdvancedLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []error

	if l.writer != nil {
		if err := l.writer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// GetUnderlyingLogger returns the underlying basic Logger.
func (l *AdvancedLogger) GetUnderlyingLogger() *Logger {
	return l.logger
}

// Rotate forces log rotation.
func (l *AdvancedLogger) Rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.writer != nil {
		return l.writer.rotate()
	}
	return nil
}

// BufferedLogger provides buffered logging capabilities.
type BufferedLogger struct {
	logger   *Logger
	buffer   *bytes.Buffer
	mu       sync.Mutex
	flushAt  int
	callback func([]byte)
}

// NewBufferedLogger creates a new BufferedLogger.
func NewBufferedLogger(logger *Logger, flushAt int, callback func([]byte)) *BufferedLogger {
	return &BufferedLogger{
		logger:   logger,
		buffer:   new(bytes.Buffer),
		flushAt:  flushAt,
		callback: callback,
	}
}

// Log logs a message and adds it to the buffer.
func (b *BufferedLogger) Log(level LogLevel, msg string, fields ...Field) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry := &LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   msg,
		Fields:    make(map[string]interface{}),
	}

	for _, field := range fields {
		entry.Fields[field.Key] = field.Value
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	b.buffer.Write(data)
	b.buffer.WriteByte('\n')

	if b.buffer.Len() >= b.flushAt || b.callback == nil {
		b.flush()
	}
}

// Flush flushes the buffer.
func (b *BufferedLogger) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flush()
}

func (b *BufferedLogger) flush() {
	if b.callback != nil && b.buffer.Len() > 0 {
		b.callback(b.buffer.Bytes())
	}
	b.buffer.Reset()
}

// Close closes the buffered logger and flushes remaining entries.
func (b *BufferedLogger) Close() error {
	b.Flush()
	return nil
}
