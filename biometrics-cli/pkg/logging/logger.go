// Package logging provides structured logging capabilities for the BIOMETRICS CLI.
// It supports JSON and text output formats, multiple log levels, context propagation,
// and configurable output destinations.
package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry.
type LogLevel int

const (
	// DebugLevel is for detailed diagnostic information.
	DebugLevel LogLevel = iota
	// InfoLevel is for general operational information.
	InfoLevel
	// WarnLevel is for warning messages that don't require immediate action.
	WarnLevel
	// ErrorLevel is for error conditions that require attention.
	ErrorLevel
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return "unknown"
	}
}

// MarshalJSON returns the JSON representation of the log level.
func (l LogLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

// ParseLogLevel parses a string into a LogLevel.
func ParseLogLevel(s string) (LogLevel, error) {
	switch s {
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	default:
		return DebugLevel, fmt.Errorf("invalid log level: %s", s)
	}
}

// LogEntry represents a single structured log entry.
type LogEntry struct {
	// Timestamp is when the log entry was created.
	Timestamp time.Time `json:"timestamp"`
	// Level is the severity of the log entry.
	Level LogLevel `json:"level"`
	// Message is the human-readable log message.
	Message string `json:"message"`
	// Fields contains additional structured data.
	Fields map[string]interface{} `json:"fields,omitempty"`
	// Caller is the source location of the log call.
	Caller string `json:"caller,omitempty"`
	// TraceID is the distributed tracing identifier.
	TraceID string `json:"trace_id,omitempty"`
	// SpanID is the distributed tracing span identifier.
	SpanID string `json:"span_id,omitempty"`
}

// MarshalJSON returns the JSON representation of the log entry.
func (e *LogEntry) MarshalJSON() ([]byte, error) {
	type Alias LogEntry
	return json.Marshal(&struct {
		*Alias
		Level string `json:"level"`
	}{
		Alias: (*Alias)(e),
		Level: e.Level.String(),
	})
}

// UnmarshalJSON parses the JSON representation of a log entry.
func (e *LogEntry) UnmarshalJSON(data []byte) error {
	type Alias LogEntry
	aux := &struct {
		*Alias
		Level string `json:"level"`
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Level != "" {
		level, err := ParseLogLevel(aux.Level)
		if err != nil {
			return err
		}
		e.Level = level
	}
	return nil
}

// LoggerConfig contains configuration options for the Logger.
type LoggerConfig struct {
	// Level is the minimum log level to output.
	Level LogLevel
	// Format is the output format (JSON or text).
	Format LogFormat
	// Output is the destination for log output.
	Output io.Writer
	// ErrorOutput is the destination for error-level logs.
	ErrorOutput io.Writer
	// AddCaller adds the caller information to log entries.
	AddCaller bool
	// AddTimestamp adds timestamp to log entries (always true for JSON).
	AddTimestamp bool
}

// LogFormat specifies the output format for log entries.
type LogFormat int

const (
	// JSONFormat outputs logs as JSON objects.
	JSONFormat LogFormat = iota
	// TextFormat outputs logs as human-readable text.
	TextFormat
)

// Logger provides structured logging capabilities.
type Logger struct {
	mu         sync.Mutex
	config     LoggerConfig
	fields     map[string]interface{}
	out        io.Writer
	errOut     io.Writer
	bufferPool *sync.Pool
	callerSkip int
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates a 64-bit integer field.
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a boolean field.
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field.
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Err creates an error field.
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Any creates a field with any value type.
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field.
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Time creates a time field.
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value.Format(time.RFC3339)}
}

// Context keys for tracing.
type contextKey int

const (
	traceIDKey contextKey = iota
	spanIDKey
)

// NewLogger creates a new Logger with the given configuration.
func NewLogger(config *LoggerConfig) *Logger {
	if config == nil {
		config = DefaultConfig()
	}

	out := config.Output
	if out == nil {
		out = os.Stdout
	}

	errOut := config.ErrorOutput
	if errOut == nil {
		errOut = os.Stderr
	}

	return &Logger{
		config: *config,
		fields: make(map[string]interface{}),
		out:    out,
		errOut: errOut,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 256)
			},
		},
		callerSkip: 1,
	}
}

// DefaultConfig returns a default LoggerConfig.
func DefaultConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:        InfoLevel,
		Format:       JSONFormat,
		AddCaller:    true,
		AddTimestamp: true,
	}
}

// WithLevel creates a new Logger with the specified log level.
func (l *Logger) WithLevel(level LogLevel) *Logger {
	newLogger := l.clone()
	newLogger.config.Level = level
	return newLogger
}

// WithFormat creates a new Logger with the specified output format.
func (l *Logger) WithFormat(format LogFormat) *Logger {
	newLogger := l.clone()
	newLogger.config.Format = format
	return newLogger
}

// WithFields creates a new Logger with additional fields.
func (l *Logger) WithFields(fields ...Field) *Logger {
	newLogger := l.clone()
	newLogger.mu.Lock()
	defer newLogger.mu.Unlock()

	for _, field := range fields {
		newLogger.fields[field.Key] = field.Value
	}
	return newLogger
}

// WithField creates a new Logger with a single additional field.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.WithFields(Field{Key: key, Value: value})
}

// WithContext creates a new Logger with trace context from the context.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	newLogger := l.clone()

	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		newLogger.fields["trace_id"] = traceID
	}

	if spanID, ok := ctx.Value(spanIDKey).(string); ok {
		newLogger.fields["span_id"] = spanID
	}

	return newLogger
}

// clone creates a copy of the Logger.
func (l *Logger) clone() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(map[string]interface{}, len(l.fields))
	for k, v := range l.fields {
		newFields[k] = v
	}

	return &Logger{
		config:     l.config,
		fields:     newFields,
		out:        l.out,
		errOut:     l.errOut,
		bufferPool: l.bufferPool,
		callerSkip: l.callerSkip,
	}
}

// Debug logs a message at the debug level.
func (l *Logger) Debug(msg string, fields ...Field) {
	l.log(DebugLevel, msg, fields...)
}

// Debugf logs a formatted message at the debug level.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, fmt.Sprintf(format, args...))
}

// Info logs a message at the info level.
func (l *Logger) Info(msg string, fields ...Field) {
	l.log(InfoLevel, msg, fields...)
}

// Infof logs a formatted message at the info level.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, fmt.Sprintf(format, args...))
}

// Warn logs a message at the warn level.
func (l *Logger) Warn(msg string, fields ...Field) {
	l.log(WarnLevel, msg, fields...)
}

// Warnf logs a formatted message at the warn level.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, fmt.Sprintf(format, args...))
}

// Error logs a message at the error level.
func (l *Logger) Error(msg string, fields ...Field) {
	l.log(ErrorLevel, msg, fields...)
}

// Errorf logs a formatted message at the error level.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...))
}

// log is the internal logging method.
func (l *Logger) log(level LogLevel, msg string, fields ...Field) {
	if level < l.config.Level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := &LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   msg,
		Fields:    make(map[string]interface{}),
	}

	// Add global fields
	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	// Add local fields
	for _, field := range fields {
		entry.Fields[field.Key] = field.Value
	}

	// Add caller information
	if l.config.AddCaller {
		entry.Caller = l.getCaller()
	}

	// Add trace context if present
	if traceID, ok := l.fields["trace_id"].(string); ok {
		entry.TraceID = traceID
	}
	if spanID, ok := l.fields["span_id"].(string); ok {
		entry.SpanID = spanID
	}

	// Write the log entry
	l.write(entry, level)
}

// getCaller returns the caller information.
func (l *Logger) getCaller() string {
	_, file, line, ok := runtime.Caller(2 + l.callerSkip)
	if !ok {
		return "unknown"
	}

	// Extract just the filename
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			return fmt.Sprintf("%s:%d", file[i+1:], line)
		}
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// write outputs the log entry to the appropriate destination.
func (l *Logger) write(entry *LogEntry, level LogLevel) {
	var output io.Writer
	if level >= ErrorLevel {
		output = l.errOut
	} else {
		output = l.out
	}

	switch l.config.Format {
	case JSONFormat:
		l.writeJSON(entry, output)
	case TextFormat:
		l.writeText(entry, output)
	}
}

// writeJSON outputs the log entry as JSON.
func (l *Logger) writeJSON(entry *LogEntry, output io.Writer) {
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(output, "{\"error\": \"failed to marshal log entry: %v\"}\n", err)
		return
	}

	_, err = output.Write(append(data, '\n'))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write log: %v\n", err)
	}
}

// writeText outputs the log entry as human-readable text.
func (l *Logger) writeText(entry *LogEntry, output io.Writer) {
	buf := l.bufferPool.Get().([]byte)
	buf = buf[:0]
	defer l.bufferPool.Put(buf)

	// Format: [LEVEL] TIMESTAMP MESSAGE (caller) [fields]
	buf = append(buf, '[')
	buf = append(buf, []byte(entry.Level.String())...)
	buf = append(buf, "] "...)

	if l.config.AddTimestamp {
		buf = append(buf, []byte(entry.Timestamp.Format(time.RFC3339))...)
		buf = append(buf, ' ')
	}

	buf = append(buf, []byte(entry.Message)...)

	if entry.Caller != "" {
		buf = append(buf, " ("...)
		buf = append(buf, []byte(entry.Caller)...)
		buf = append(buf, ')')
	}

	if len(entry.Fields) > 0 {
		buf = append(buf, " "...)
		buf = append(buf, []byte(fmt.Sprintf("%v", entry.Fields))...)
	}

	buf = append(buf, '\n')

	_, err := output.Write(buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write log: %v\n", err)
	}
}

// Sync flushes any buffered log entries.
func (l *Logger) Sync() error {
	if syncer, ok := l.out.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	if syncer, ok := l.errOut.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	return nil
}

// SetOutput changes the output destination.
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
}

// SetErrorOutput changes the error output destination.
func (l *Logger) SetErrorOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errOut = w
}

// WithTraceContext adds trace context to the logger.
func WithTraceContext(ctx context.Context, traceID, spanID string) context.Context {
	ctx = context.WithValue(ctx, traceIDKey, traceID)
	ctx = context.WithValue(ctx, spanIDKey, spanID)
	return ctx
}

// TraceContextFromLogger extracts trace context from the logger's fields.
func TraceContextFromLogger(l *Logger) (traceID, spanID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	traceID, _ = l.fields["trace_id"].(string)
	spanID, _ = l.fields["span_id"].(string)
	return
}

// global is the global default logger.
var global = NewLogger(nil)

// Default returns the global default logger.
func Default() *Logger {
	return global
}

// SetDefault sets the global default logger.
func SetDefault(l *Logger) {
	global = l
}

// Convenience functions that use the global logger.

// Debug logs a message at the debug level using the global logger.
func Debug(msg string, fields ...Field) {
	global.Debug(msg, fields...)
}

// Info logs a message at the info level using the global logger.
func Info(msg string, fields ...Field) {
	global.Info(msg, fields...)
}

// Warn logs a message at the warn level using the global logger.
func Warn(msg string, fields ...Field) {
	global.Warn(msg, fields...)
}

// Error logs a message at the error level using the global logger.
func Error(msg string, fields ...Field) {
	global.Error(msg, fields...)
}

// WithFields creates a new logger with additional fields from the global logger.
func WithFields(fields ...Field) *Logger {
	return global.WithFields(fields...)
}

// WithContext creates a new logger with context from the global logger.
func WithContext(ctx context.Context) *Logger {
	return global.WithContext(ctx)
}
